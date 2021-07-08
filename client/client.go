package client

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lidofinance/dc4bc/storage/kafka_storage"

	"github.com/lidofinance/dc4bc/fsm/types/responses"

	"github.com/lidofinance/dc4bc/fsm/state_machines/dkg_proposal_fsm"
	sipf "github.com/lidofinance/dc4bc/fsm/state_machines/signing_proposal_fsm"

	"github.com/lidofinance/dc4bc/client/types"
	"github.com/lidofinance/dc4bc/fsm/types/requests"

	spf "github.com/lidofinance/dc4bc/fsm/state_machines/signature_proposal_fsm"

	"github.com/lidofinance/dc4bc/fsm/state_machines"

	"github.com/lidofinance/dc4bc/fsm/fsm"
	dpf "github.com/lidofinance/dc4bc/fsm/state_machines/dkg_proposal_fsm"
	"github.com/lidofinance/dc4bc/qr"
	"github.com/lidofinance/dc4bc/storage"
)

const (
	pollingPeriod = time.Second
	QrCodesDir    = "/tmp"
)

type Client interface {
	Poll() error
	GetLogger() Logger
	GetPubKey() ed25519.PublicKey
	GetUsername() string
	SendMessage(message storage.Message) error
	ProcessMessage(message storage.Message) error
	GetOperations() (map[string]*types.Operation, error)
	GetOperationQRPath(operationID string) (string, error)
	StartHTTPServer(listenAddr string) error
	StopHTTPServer()
	SetSkipCommKeysVerification(bool)
	ResetState(newStateDBPath string, consumerGroup string, messages []string, useOffset bool) (string, error)
}

type BaseClient struct {
	sync.Mutex
	server                   *http.Server
	Logger                   Logger
	userName                 string
	pubKey                   ed25519.PublicKey
	ctx                      context.Context
	stateMu                  sync.RWMutex
	state                    State
	storage                  storage.Storage
	keyStore                 KeyStore
	qrProcessor              qr.Processor
	SkipCommKeysVerification bool
}

func NewClient(
	ctx context.Context,
	userName string,
	state State,
	storage storage.Storage,
	keyStore KeyStore,
	qrProcessor qr.Processor,
) (Client, error) {
	keyPair, err := keyStore.LoadKeys(userName, "")
	if err != nil {
		return nil, fmt.Errorf("failed to LoadKeys: %w", err)
	}

	return &BaseClient{
		ctx:         ctx,
		Logger:      newLogger(userName),
		userName:    userName,
		pubKey:      keyPair.Pub,
		state:       state,
		storage:     storage,
		keyStore:    keyStore,
		qrProcessor: qrProcessor,
	}, nil
}

func (c *BaseClient) getState() State {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()

	return c.state
}

func (c *BaseClient) GetLogger() Logger {
	return c.Logger
}

func (c *BaseClient) GetUsername() string {
	return c.userName
}

func (c *BaseClient) GetPubKey() ed25519.PublicKey {
	return c.pubKey
}

func (c *BaseClient) GetSkipCommKeysVerification() bool {
	c.Lock()
	defer c.Unlock()

	return c.SkipCommKeysVerification
}

func (c *BaseClient) SetSkipCommKeysVerification(f bool) {
	c.Lock()
	defer c.Unlock()

	c.SkipCommKeysVerification = f
}

// Poll is a main client loop, which gets new messages from an append-only log and processes them
func (c *BaseClient) Poll() error {
	tk := time.NewTicker(pollingPeriod)
	for {
		select {
		case <-tk.C:
			offset, err := c.getState().LoadOffset()
			if err != nil {
				return fmt.Errorf("failed to LoadOffset: %w", err)
			}

			messages, err := c.storage.GetMessages(offset)
			if err != nil {
				return fmt.Errorf("failed to GetMessages: %w", err)
			}

			for _, message := range messages {
				c.Logger.Log("Handling message with offset %d, type %s", message.Offset, message.Event)
				if message.RecipientAddr == "" || message.RecipientAddr == c.GetUsername() {
					if err := c.ProcessMessage(message); err != nil {
						c.Logger.Log("Failed to process message with offset %d: %v", message.Offset, err)
					} else {
						c.Logger.Log("Successfully processed message with offset %d, type %s",
							message.Offset, message.Event)
					}
				} else {
					c.Logger.Log("Message with offset %d, type %s is not intended for us, skip it",
						message.Offset, message.Event)
				}
				if err := c.getState().SaveOffset(message.Offset + 1); err != nil {
					c.Logger.Log("Failed to save offset: %v", err)
				}
			}
		case <-c.ctx.Done():
			log.Println("Context closed, stop polling...")
			return nil
		}
	}
}

func (c *BaseClient) SendMessage(message storage.Message) error {
	if err := c.storage.Send(message); err != nil {
		return fmt.Errorf("failed to post message: %w", err)
	}

	return nil
}

// processSignature saves a broadcasted reconstructed signature to a LevelDB
func (c *BaseClient) processSignature(message storage.Message) error {
	var (
		signature types.ReconstructedSignature
		err       error
	)
	if err = json.Unmarshal(message.Data, &signature); err != nil {
		return fmt.Errorf("failed to unmarshal reconstructed signature: %w", err)
	}
	signature.Username = message.SenderAddr
	signature.DKGRoundID = message.DkgRoundID
	return c.getState().SaveSignature(signature)
}

func (c *BaseClient) reinitDKG(message storage.Message) error {
	var req types.ReDKG
	if err := json.Unmarshal(message.Data, &req); err != nil {
		return fmt.Errorf("failed to umarshal request: %v", err)

	}

	// temporarily fix cause we can't verify patch messages
	// TODO: remove later
	if !c.GetSkipCommKeysVerification() {
		c.SetSkipCommKeysVerification(true)
		defer c.SetSkipCommKeysVerification(false)
	}

	operations := make([]*types.Operation, 0)
	for _, msg := range req.Messages {
		if fsm.Event(msg.Event) == sipf.EventSigningStart {
			break
		}
		if msg.RecipientAddr == "" || msg.RecipientAddr == c.GetUsername() {
			operation, err := c.processMessage(msg)
			if err != nil {
				c.Logger.Log("failed to process operation: %v", err)
			}
			if operation != nil {
				operations = append(operations, operation)
			}
		}
	}

	operationsBz, err := json.Marshal(operations)
	if err != nil {
		return fmt.Errorf("failed to marshall operations")
	}

	operation := types.NewOperation(req.DKGID, operationsBz, types.ReinitDKG)
	if err := c.getState().PutOperation(operation); err != nil {
		return fmt.Errorf("failed to PutOperation: %w", err)
	}

	// save new comm keys into FSM to verify future messages
	fsmInstance, err := c.getFSMInstance(req.DKGID)
	if err != nil {
		return fmt.Errorf("failed to get FSM instance: %w", err)
	}
	for _, reqParticipant := range req.Participants {
		fsmInstance.FSMDump().Payload.SetPubKeyUsername(reqParticipant.Name, reqParticipant.NewCommPubKey)
	}
	fsmDump, err := fsmInstance.Dump()
	if err != nil {
		return fmt.Errorf("failed to get FSM dump")
	}

	if err := c.getState().SaveFSM(message.DkgRoundID, fsmDump); err != nil {
		return fmt.Errorf("failed to SaveFSM: %w", err)
	}

	return nil
}

func (c *BaseClient) ProcessMessage(message storage.Message) error {
	if fsm.State(message.Event) == types.ReinitDKG {
		if err := c.reinitDKG(message); err != nil {
			return fmt.Errorf("failed to reinitDKG")
		}
		return nil
	}

	operation, err := c.processMessage(message)
	if err != nil {
		return err
	}
	if operation != nil {
		if err := c.getState().PutOperation(operation); err != nil {
			return fmt.Errorf("failed to PutOperation: %w", err)
		}
	}
	return nil
}

func (c *BaseClient) processMessage(message storage.Message) (*types.Operation, error) {
	fsmInstance, err := c.getFSMInstance(message.DkgRoundID)
	if err != nil {
		return nil, fmt.Errorf("failed to getFSMInstance: %w", err)
	}

	// we can't verify a message at this moment, cause we don't have public keys of participants
	if fsm.Event(message.Event) != spf.EventInitProposal {
		if err := c.verifyMessage(fsmInstance, message); err != nil {
			return nil, fmt.Errorf("failed to verifyMessage %+v: %w", message, err)
		}
	}

	switch fsm.Event(message.Event) {
	case types.SignatureReconstructed: // save broadcasted reconstructed signature
		if err := c.processSignature(message); err != nil {
			return nil, fmt.Errorf("failed to process signature: %w", err)
		}
		return nil, nil
	case types.SignatureReconstructionFailed:
		errorRequest, err := types.FSMRequestFromMessage(message)
		if err != nil {
			return nil, fmt.Errorf("failed to get FSMRequestFromMessage: %v", err)
		}
		errorRequestTyped, ok := errorRequest.(requests.SignatureProposalConfirmationErrorRequest)
		if !ok {
			return nil, fmt.Errorf("failed to convert request to SignatureProposalConfirmationErrorRequest: %v", err)
		}
		c.Logger.Log("Participant #%d got an error during signature reconstruction process: %v", errorRequestTyped.ParticipantId, errorRequestTyped.Error)
		return nil, nil
	}

	//TODO: refactor the following checks
	//handle common errors
	if strings.HasSuffix(string(fsmInstance.FSMDump().State), "_error") {
		if fsmInstance.FSMDump().Payload.DKGProposalPayload != nil {
			for _, participant := range fsmInstance.FSMDump().Payload.DKGProposalPayload.Quorum {
				if participant.Error != nil {
					c.Logger.Log("Participant %s got an error during DKG process: %s. DKG aborted\n",
						participant.Username, participant.Error.Error())
					// if we have an error during DKG, abort the whole DKG procedure.
					return nil, nil
				}
			}
		}
		if fsmInstance.FSMDump().Payload.SigningProposalPayload != nil {
			for _, participant := range fsmInstance.FSMDump().Payload.SigningProposalPayload.Quorum {
				if participant.Error != nil {
					c.Logger.Log("Participant %s got an error during signing procedure: %s. Signing procedure aborted\n",
						participant.Username, participant.Error.Error())
					break
				}
			}
			//if we have an error during signing procedure, start a new signing procedure
			_, fsmDump, err := fsmInstance.Do(sipf.EventSigningRestart, requests.DefaultRequest{
				CreatedAt: time.Now(),
			})
			if err != nil {
				return nil, fmt.Errorf("failed to Do operation in FSM: %w", err)
			}

			if err := c.getState().SaveFSM(message.DkgRoundID, fsmDump); err != nil {
				return nil, fmt.Errorf("failed to SaveFSM: %w", err)

			}
		}
	}

	//handle timeout errors
	if strings.HasSuffix(string(fsmInstance.FSMDump().State), "_timeout") {
		if strings.HasPrefix(string(fsmInstance.FSMDump().State), "state_sig_") ||
			strings.HasPrefix(string(fsmInstance.FSMDump().State), "state_dkg") {
			c.Logger.Log("DKG process with ID \"%s\" aborted cause of timeout\n",
				fsmInstance.FSMDump().Payload.DkgId)
			// if we have an error during DKG, abort the whole DKG procedure.
			return nil, nil
		}
		if strings.HasPrefix(string(fsmInstance.FSMDump().State), "state_signing_") {
			c.Logger.Log("Signing process with ID \"%s\" aborted cause of timeout\n",
				fsmInstance.FSMDump().Payload.SigningProposalPayload.SigningId)

			//if we have an error during signing procedure, start a new signing procedure
			_, fsmDump, err := fsmInstance.Do(sipf.EventSigningRestart, requests.DefaultRequest{
				CreatedAt: time.Now(),
			})
			if err != nil {
				return nil, fmt.Errorf("failed to Do operation in FSM: %w", err)
			}

			if err := c.getState().SaveFSM(message.DkgRoundID, fsmDump); err != nil {
				return nil, fmt.Errorf("failed to SaveFSM: %w", err)
			}
		}
	}

	fsmReq, err := types.FSMRequestFromMessage(message)
	if err != nil {
		return nil, fmt.Errorf("failed to get FSMRequestFromMessage: %v", err)
	}

	resp, fsmDump, err := fsmInstance.Do(fsm.Event(message.Event), fsmReq)
	if err != nil {
		return nil, fmt.Errorf("failed to Do operation in FSM: %w", err)
	}

	c.Logger.Log("message %s done successfully from %s", message.Event, message.SenderAddr)

	// switch FSM state by hand due to implementation specifics
	if resp.State == spf.StateSignatureProposalCollected {
		fsmInstance, err = state_machines.FromDump(fsmDump)
		if err != nil {
			return nil, fmt.Errorf("failed get state_machines from dump: %w", err)
		}
		resp, fsmDump, err = fsmInstance.Do(dpf.EventDKGInitProcess, requests.DefaultRequest{
			CreatedAt: time.Now(),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to Do operation in FSM: %w", err)
		}
	}
	if resp.State == dpf.StateDkgMasterKeyCollected {
		fsmInstance, err = state_machines.FromDump(fsmDump)
		if err != nil {
			return nil, fmt.Errorf("failed get state_machines from dump: %w", err)
		}
		resp, fsmDump, err = fsmInstance.Do(sipf.EventSigningInit, requests.DefaultRequest{
			CreatedAt: time.Now(),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to Do operation in FSM: %w", err)
		}
	}

	var operation *types.Operation
	switch resp.State {
	// if the new state is waiting for RPC to airgapped machine
	case
		spf.StateAwaitParticipantsConfirmations,
		dpf.StateDkgCommitsAwaitConfirmations,
		dpf.StateDkgDealsAwaitConfirmations,
		dpf.StateDkgResponsesAwaitConfirmations,
		dpf.StateDkgMasterKeyAwaitConfirmations,
		sipf.StateSigningAwaitPartialSigns,
		sipf.StateSigningPartialSignsCollected,
		sipf.StateSigningAwaitConfirmations:
		if resp.Data != nil {

			// if we are initiator of signing, then we don't need to confirm our participation
			if data, ok := resp.Data.(responses.SigningProposalParticipantInvitationsResponse); ok {
				initiator, err := fsmInstance.SigningQuorumGetParticipant(data.InitiatorId)
				if err != nil {
					return nil, fmt.Errorf("failed to get SigningQuorumParticipant: %w", err)
				}
				if initiator.Username == c.GetUsername() {
					break
				}
			}

			operationPayloadBz, err := json.Marshal(resp.Data)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal FSM response: %w", err)
			}

			operation = types.NewOperation(
				message.DkgRoundID,
				operationPayloadBz,
				resp.State,
			)
		}
	default:
		c.Logger.Log("State %s does not require an operation", resp.State)
	}

	// switch FSM state by hand due to implementation specifics
	if resp.State == sipf.StateSigningPartialSignsCollected {
		fsmInstance, err = state_machines.FromDump(fsmDump)
		if err != nil {
			return nil, fmt.Errorf("failed get state_machines from dump: %w", err)
		}
		resp, fsmDump, err = fsmInstance.Do(sipf.EventSigningRestart, requests.DefaultRequest{
			CreatedAt: time.Now(),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to Do operation in FSM: %w", err)
		}
	}

	// save signing data to the same storage as we save signatures
	// This allows easy to view signing data by CLI-command
	if fsm.Event(message.Event) == sipf.EventSigningStart {
		if err := c.processSignature(message); err != nil {
			return nil, fmt.Errorf("failed to process signature: %w", err)
		}
	}

	if err := c.getState().SaveFSM(message.DkgRoundID, fsmDump); err != nil {
		return nil, fmt.Errorf("failed to SaveFSM: %w", err)
	}

	return operation, nil
}

func (c *BaseClient) GetOperations() (map[string]*types.Operation, error) {
	return c.getState().GetOperations()
}

//GetSignatures returns all signatures for the given DKG round that were reconstructed on the airgapped machine and
// broadcasted by users
func (c *BaseClient) GetSignatures(dkgID string) (map[string][]types.ReconstructedSignature, error) {
	return c.getState().GetSignatures(dkgID)
}

//GetSignatureByDataHash returns a list of reconstructed signatures of the signed data broadcasted by users
func (c *BaseClient) GetSignatureByID(dkgID, sigID string) ([]types.ReconstructedSignature, error) {
	return c.getState().GetSignatureByID(dkgID, sigID)
}

// getOperationJSON returns a specific JSON-encoded operation
func (c *BaseClient) getOperationJSON(operationID string) ([]byte, error) {
	operation, err := c.getState().GetOperationByID(operationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get operation: %w", err)
	}

	operationJSON, err := json.Marshal(operation)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal operation: %w", err)
	}
	return operationJSON, nil
}

// GetOperationQRPath returns a path to the image with the QR generated
// for the specified operation. It is supposed that the user will open
// this file herself.
func (c *BaseClient) GetOperationQRPath(operationID string) (string, error) {
	operationJSON, err := c.getOperationJSON(operationID)
	if err != nil {
		return "", fmt.Errorf("failed to get operation in JSON: %w", err)
	}

	operationQRPath := filepath.Join(QrCodesDir, fmt.Sprintf("dc4bc_qr_%s", operationID))

	qrPath := fmt.Sprintf("%s.gif", operationQRPath)
	if err = c.qrProcessor.WriteQR(qrPath, operationJSON); err != nil {
		return "", err
	}

	return qrPath, nil
}

// handleProcessedOperation handles an operation which was processed by the airgapped machine
// It checks that the operation exists in an operation pool, signs the operation, sends it to an append-only log and
// deletes it from the pool.
func (c *BaseClient) handleProcessedOperation(operation types.Operation) error {
	if operation.Event.IsEmpty() {
		return errors.New("operation is request operation, provide result operation instead")
	}

	storedOperation, err := c.getState().GetOperationByID(operation.ID)
	if err != nil {
		return fmt.Errorf("failed to find matching operation: %w", err)
	}

	if err := storedOperation.Check(&operation); err != nil {
		return fmt.Errorf("processed operation does not match stored operation: %w", err)
	}

	// there are no result messages for OperationProcessed event type
	if operation.Event != types.OperationProcessed {
		for i, message := range operation.ResultMsgs {
			message.SenderAddr = c.GetUsername()

			sig, err := c.signMessage(message.Bytes())
			if err != nil {
				return fmt.Errorf("failed to sign a message: %w", err)
			}
			message.Signature = sig

			operation.ResultMsgs[i] = message
		}
		if err := c.storage.Send(operation.ResultMsgs...); err != nil {
			return fmt.Errorf("failed to post messages: %w", err)
		}
	}

	if err := c.getState().DeleteOperation(&operation); err != nil {
		return fmt.Errorf("failed to DeleteOperation: %w", err)
	}

	return nil
}

// getFSMInstance returns a FSM for a necessary DKG round.
func (c *BaseClient) getFSMInstance(dkgRoundID string) (*state_machines.FSMInstance, error) {
	var err error
	fsmInstance, ok, err := c.getState().LoadFSM(dkgRoundID)
	if err != nil {
		return nil, fmt.Errorf("failed to LoadFSM: %w", err)
	}

	if !ok {
		fsmInstance, err = state_machines.Create(dkgRoundID)
		if err != nil {
			return nil, fmt.Errorf("failed to create FSM instance: %w", err)
		}
		bz, err := fsmInstance.Dump()
		if err != nil {
			return nil, fmt.Errorf("failed to Dump FSM instance: %w", err)
		}
		if err := c.getState().SaveFSM(dkgRoundID, bz); err != nil {
			return nil, fmt.Errorf("failed to SaveFSM: %w", err)
		}
	}

	return fsmInstance, nil
}

func (c *BaseClient) signMessage(message []byte) ([]byte, error) {
	keyPair, err := c.keyStore.LoadKeys(c.userName, "")
	if err != nil {
		return nil, fmt.Errorf("failed to LoadKeys: %w", err)
	}

	return ed25519.Sign(keyPair.Priv, message), nil
}

func (c *BaseClient) verifyMessage(fsmInstance *state_machines.FSMInstance, message storage.Message) error {
	if c.GetSkipCommKeysVerification() {
		return nil
	}
	senderPubKey, err := fsmInstance.GetPubKeyByUsername(message.SenderAddr)
	if err != nil {
		return fmt.Errorf("failed to GetPubKeyByUsername: %w", err)
	}

	if !ed25519.Verify(senderPubKey, message.Bytes(), message.Signature) {
		return errors.New("signature is corrupt")
	}

	return nil
}

func (c *BaseClient) GetFSMDump(dkgID string) (*state_machines.FSMDump, error) {
	fsmInstance, err := c.getFSMInstance(dkgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get FSM instance for DKG round ID %s: %w", dkgID, err)
	}
	return fsmInstance.FSMDump(), nil
}

func (c *BaseClient) ResetState(newStateDBPath string, cg string, messages []string, useOffset bool) (stateDb string, err error) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	if err = c.storage.IgnoreMessages(messages, useOffset); err != nil {
		return stateDb, fmt.Errorf("failed to ignore messages while resetting state: %v", err)
	}

	switch c.storage.(type) {
	case *kafka_storage.KafkaStorage:
		stg := c.storage.(*kafka_storage.KafkaStorage)
		if err = stg.SetConsumerGroup(cg); err != nil {
			return stateDb, fmt.Errorf("failed to set consumer group while reseting state: %v", err)
		}
	}

	var newState State
	newState, stateDb, err = c.state.NewStateFromOld(newStateDBPath)
	if err != nil {
		return stateDb, fmt.Errorf("failed to create new state from old: %v", err)
	}

	c.state = newState

	return stateDb, err
}

func createMessage(origMesage storage.Message) (storage.Message, error) {
	fsmReq, err := types.FSMRequestFromMessage(origMesage)
	if err != nil {
		return storage.Message{}, fmt.Errorf("failed to get FSMRequestFromMessage: %v", err)
	}
	request, ok := fsmReq.(requests.DKGProposalDealConfirmationRequest)
	if !ok {
		return storage.Message{}, errors.New("cannot cast message request  to type {DKGProposalDealConfirmationRequest}")
	}
	req := requests.DKGProposalDealConfirmationRequest{
		ParticipantId: request.ParticipantId,
		Deal:          []byte("self-confirm"),
		CreatedAt:     request.CreatedAt,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return storage.Message{}, fmt.Errorf("failed to encode FSMRequest: %v", err)
	}
	newMsg := storage.Message{
		ID:            uuid.New().String(),
		DkgRoundID:    origMesage.DkgRoundID,
		Event:         string(dkg_proposal_fsm.EventDKGDealConfirmationReceived),
		Data:          data,
		SenderAddr:    origMesage.SenderAddr,
		RecipientAddr: origMesage.SenderAddr,
	}

	return newMsg, nil
}

func GetAdaptedReDKG(originalDKG types.ReDKG) (types.ReDKG, error) {
	adaptedReDKG := types.ReDKG{}

	adaptedReDKG.DKGID = originalDKG.DKGID
	adaptedReDKG.Participants = originalDKG.Participants
	adaptedReDKG.Threshold = originalDKG.Threshold
	adaptedReDKG.Messages = []storage.Message{}
	var newOffset uint64
	fixedSenders := map[string]struct{}{}
	for _, m := range originalDKG.Messages {
		if _, found := fixedSenders[m.SenderAddr]; !found && fsm.Event(m.Event) == dkg_proposal_fsm.EventDKGDealConfirmationReceived {
			fixedSenders[m.SenderAddr] = struct{}{}
			workAroundMessage, err := createMessage(m)
			if err != nil {
				return types.ReDKG{}, fmt.Errorf("failed to construct new message for adapted reinit DKG message: %v", err)
			}
			workAroundMessage.Offset = newOffset
			newOffset++
			adaptedReDKG.Messages = append(adaptedReDKG.Messages, workAroundMessage)
		}
		m.Offset = newOffset
		newOffset++
		adaptedReDKG.Messages = append(adaptedReDKG.Messages, m)
	}
	return adaptedReDKG, nil
}
