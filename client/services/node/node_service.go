package node

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/lidofinance/dc4bc/client/services/fsmservice"
	"github.com/lidofinance/dc4bc/client/services/operation"
	"github.com/lidofinance/dc4bc/client/services/signature"

	"github.com/google/uuid"
	"github.com/lidofinance/dc4bc/client/api/dto"
	"github.com/lidofinance/dc4bc/client/config"
	"github.com/lidofinance/dc4bc/client/modules/keystore"
	"github.com/lidofinance/dc4bc/client/modules/logger"
	"github.com/lidofinance/dc4bc/client/modules/state"
	"github.com/lidofinance/dc4bc/client/services"
	"github.com/lidofinance/dc4bc/client/types"
	"github.com/lidofinance/dc4bc/fsm/fsm"
	"github.com/lidofinance/dc4bc/fsm/state_machines"
	dpf "github.com/lidofinance/dc4bc/fsm/state_machines/dkg_proposal_fsm"
	spf "github.com/lidofinance/dc4bc/fsm/state_machines/signature_proposal_fsm"
	sif "github.com/lidofinance/dc4bc/fsm/state_machines/signing_proposal_fsm"
	"github.com/lidofinance/dc4bc/fsm/types/requests"
	"github.com/lidofinance/dc4bc/fsm/types/responses"
	"github.com/lidofinance/dc4bc/storage"
)

const (
	pollingPeriod      = time.Second
	jsonFilesDir       = "/tmp"
	emptyParticipantId = -1
)

type NodeService interface {
	Poll() error
	GetLogger() logger.Logger
	GetPubKey() ed25519.PublicKey
	GetUsername() string
	ApproveParticipation(dto *dto.OperationIdDTO) error
	SendMessage(dto *dto.MessageDTO) error
	ProcessMessage(message storage.Message) error
	ProcessOperation(dto *dto.OperationDTO) error
	StartDKG(dto *dto.StartDkgDTO) error
	ReInitDKG(dto *dto.ReInitDKGDTO) error
	SetSkipCommKeysVerification(bool)
	ProposeSignMessages(dto *dto.ProposeSignBatchMessagesDTO) error
	SaveOffset(dto *dto.StateOffsetDTO) error
	GetStateOffset() (uint64, error)
}

type BaseNodeService struct {
	sync.Mutex
	ctx                      context.Context
	userName                 string
	pubKey                   ed25519.PublicKey
	stateMu                  sync.RWMutex
	state                    state.State
	storage                  storage.Storage
	keyStore                 keystore.KeyStore
	Logger                   logger.Logger
	fsmService               fsmservice.FSMService
	opService                operation.OperationService
	sigService               signature.SignatureService
	SkipCommKeysVerification bool
}

func NewNode(ctx context.Context, config *config.Config, sp *services.ServiceProvider) (NodeService, error) {
	keyPair, err := sp.GetKeyStore().LoadKeys(config.Username, "")
	if err != nil {
		return nil, fmt.Errorf("failed to LoadKeys: %w", err)
	}

	return &BaseNodeService{
		ctx:        ctx,
		userName:   config.Username,
		pubKey:     keyPair.Pub,
		state:      sp.GetState(),
		storage:    sp.GetStorage(),
		keyStore:   sp.GetKeyStore(),
		Logger:     sp.GetLogger(),
		fsmService: sp.GetFSMService(),
		opService:  sp.GetOperationService(),
		sigService: sp.GetSignatureService(),
	}, nil
}

func (s *BaseNodeService) GetLogger() logger.Logger {
	return s.Logger
}

func (s *BaseNodeService) ProcessMessage(message storage.Message) error {
	if fsm.State(message.Event) == types.ReinitDKG {
		if err := s.reinitDKG(message); err != nil {
			return fmt.Errorf("failed to reinitDKG")
		}
		return nil
	}

	operation, err := s.processMessage(message)
	if err != nil {
		return err
	}

	if operation != nil {
		if err := s.opService.PutOperation(operation); err != nil {
			return fmt.Errorf("failed to PutOperation: %w", err)
		}
	}
	return nil
}

func (s *BaseNodeService) SetSkipCommKeysVerification(b bool) {
	s.Lock()
	defer s.Unlock()

	s.SkipCommKeysVerification = b
}

// Poll is a main node loop, which gets new messages from an append-only log and processes them
func (s *BaseNodeService) Poll() error {
	tk := time.NewTicker(pollingPeriod)
	for {
		select {
		case <-tk.C:
			offset, err := s.getState().LoadOffset()
			if err != nil {
				return fmt.Errorf("failed to LoadOffset: %w", err)
			}

			messages, err := s.storage.GetMessages(offset)
			if err != nil {
				return fmt.Errorf("failed to GetMessages: %w", err)
			}

			for _, message := range messages {
				s.Logger.Log("Handling message with offset %d, type %s", message.Offset, message.Event)
				if message.RecipientAddr == "" || message.RecipientAddr == s.GetUsername() {
					if err := s.ProcessMessage(message); err != nil {
						s.Logger.Log("Failed to process message with offset %d: %v", message.Offset, err)
					} else {
						s.Logger.Log("Successfully processed message with offset %d, type %s",
							message.Offset, message.Event)
					}
				} else {
					s.Logger.Log("Message with offset %d, type %s is not intended for us, skip it",
						message.Offset, message.Event)
				}
				if err := s.getState().SaveOffset(message.Offset + 1); err != nil {
					s.Logger.Log("Failed to save offset: %v", err)
				}
			}
		case <-s.ctx.Done():
			log.Println("Context closed, stop polling...")
			return nil
		}
	}
}

func (s *BaseNodeService) getState() state.State {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return s.state
}

func (s *BaseNodeService) GetPubKey() ed25519.PublicKey {
	return s.pubKey
}

func (s *BaseNodeService) GetUsername() string {
	return s.userName
}

func (s *BaseNodeService) GetStateOffset() (uint64, error) {
	return s.getState().LoadOffset()
}

func (s *BaseNodeService) GetSkipCommKeysVerification() bool {
	s.Lock()
	defer s.Unlock()

	return s.SkipCommKeysVerification
}

func (s *BaseNodeService) SendMessage(dto *dto.MessageDTO) error {
	if err := s.storage.Send(storage.Message{
		ID:            dto.ID,
		DkgRoundID:    dto.DkgRoundID,
		Offset:        dto.Offset,
		Event:         dto.Event,
		Data:          dto.Data,
		Signature:     dto.Signature,
		SenderAddr:    dto.SenderAddr,
		RecipientAddr: dto.RecipientAddr,
	}); err != nil {
		return fmt.Errorf("failed to post message: %w", err)
	}
	return nil
}

// GetOperation returns operation for current state, if exists
func (s *BaseNodeService) getOperation(operationID string) (*types.Operation, error) {
	operations, err := s.opService.GetOperations()

	if err != nil {
		return nil, fmt.Errorf("failed to get operations: %w", err)
	}

	operation, ok := operations[operationID]

	if !ok {
		return nil, fmt.Errorf("failed to get operation")
	}
	return operation, nil
}

// ProcessOperation handles an operation which was processed by the airgapped machine
// It checks that the operation exists in an operation pool, signs the operation, sends it to an append-only log and
// deletes it from the pool.
func (s *BaseNodeService) ProcessOperation(dto *dto.OperationDTO) error {
	operation := &types.Operation{
		ID:            dto.ID,
		Type:          types.OperationType(dto.Type),
		Payload:       dto.Payload,
		ResultMsgs:    dto.ResultMsgs,
		CreatedAt:     dto.CreatedAt,
		DKGIdentifier: dto.DkgID,
		To:            dto.To,
		Event:         dto.Event,
	}

	return s.executeOperation(operation)
}

func (s *BaseNodeService) executeOperation(operation *types.Operation) error {
	if operation.Event.IsEmpty() {
		return errors.New("operation is request operation, provide result operation instead")
	}

	storedOperation, err := s.opService.GetOperationByID(operation.ID)
	if err != nil {
		return fmt.Errorf("failed to find matching operation: %w", err)
	}

	if err := storedOperation.Equal(operation); err != nil {
		return fmt.Errorf("processed operation does not match stored operation: %w", err)
	}

	// there are no result messages for OperationProcessed event type
	if operation.Event != types.OperationProcessed {
		for i, message := range operation.ResultMsgs {
			message.SenderAddr = s.GetUsername()

			sig, err := s.signMessage(message.Bytes())
			if err != nil {
				return fmt.Errorf("failed to sign a message: %w", err)
			}
			message.Signature = sig

			operation.ResultMsgs[i] = message
		}
		if err := s.storage.Send(operation.ResultMsgs...); err != nil {
			return fmt.Errorf("failed to post messages: %w", err)
		}
	}

	if err := s.opService.DeleteOperation(operation); err != nil {
		return fmt.Errorf("failed to DeleteOperation: %w", err)
	}

	return nil
}

func (s *BaseNodeService) signMessage(message []byte) ([]byte, error) {
	keyPair, err := s.keyStore.LoadKeys(s.userName, "")
	if err != nil {
		return nil, fmt.Errorf("failed to LoadKeys: %w", err)
	}

	return ed25519.Sign(keyPair.Priv, message), nil
}

func (s *BaseNodeService) verifyMessage(fsmInstance *state_machines.FSMInstance, message storage.Message) error {
	if s.GetSkipCommKeysVerification() {
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

func (s *BaseNodeService) StartDKG(dto *dto.StartDkgDTO) error {
	dkgRoundID := sha256.Sum256(dto.Payload)
	message, err := s.buildMessage(hex.EncodeToString(dkgRoundID[:]), spf.EventInitProposal, dto.Payload)

	if err != nil {
		return err
	}

	if message == nil {
		return errors.New("cannot build message for init DKG")
	}

	return s.storage.Send(*message)
}

func (s *BaseNodeService) buildMessage(dkgRoundID string, event fsm.Event, data []byte) (*storage.Message, error) {
	message := storage.Message{
		ID:         uuid.New().String(),
		DkgRoundID: dkgRoundID,
		Event:      string(event),
		Data:       data,
		SenderAddr: s.GetUsername(),
	}
	signature, err := s.signMessage(message.Bytes())

	if err != nil {
		return nil, fmt.Errorf("failed to sign message: %w", err)
	}

	message.Signature = signature
	return &message, nil
}

func (s *BaseNodeService) ProposeSignMessages(dtoMsg *dto.ProposeSignBatchMessagesDTO) error {
	messagesToSign := make([]requests.MessageToSign, 0, len(dtoMsg.Data))
	for messageID, msg := range dtoMsg.Data {
		messageDataSign := requests.MessageToSign{
			MessageID: messageID,
			Payload:   msg,
		}

		messagesToSign = append(messagesToSign, messageDataSign)
	}

	encodedDkgID := hex.EncodeToString(dtoMsg.DkgID)
	fsmInstance, err := s.fsmService.GetFSMInstance(encodedDkgID, false)
	if err != nil {
		return fmt.Errorf("failed to get FSM instance: %w", err)
	}

	fsmState, err := fsmInstance.State()
	if err != nil {
		return fmt.Errorf("failed to determine FSM instance state: %w", err)
	}

	if fsmState != sif.StateSigningIdle {
		return fmt.Errorf("required FSM state is %s, but have %s", sif.StateSigningIdle, fsmState)
	}

	participantID, err := fsmInstance.GetIDByUsername(s.GetUsername())
	if err != nil {
		return fmt.Errorf("failed to get participantID: %w", err)
	}

	batch := requests.SigningBatchProposalStartRequest{
		BatchID:        uuid.New().String(),
		ParticipantId:  participantID,
		CreatedAt:      time.Now(), // Is better to use time from node?
		MessagesToSign: messagesToSign,
	}

	batchBz, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("failed to marshal SigningBatchProposalStartRequest: %w", err)
	}

	message, err := s.buildMessage(encodedDkgID, sif.EventSigningStart, batchBz)
	if err != nil {
		return fmt.Errorf("failed to build message: %w", err)
	}

	err = s.storage.Send(*message)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

func (s *BaseNodeService) ApproveParticipation(dto *dto.OperationIdDTO) error {
	operation, err := s.getOperation(dto.OperationID)

	if err != nil {
		return err
	}

	if fsm.State(operation.Type) != spf.StateAwaitParticipantsConfirmations {
		return fmt.Errorf("cannot approve participation with operationID %s", dto.OperationID)
	}

	var payload responses.SignatureProposalParticipantInvitationsResponse

	if err = json.Unmarshal(operation.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	pid := emptyParticipantId
	for _, p := range payload {
		if s.GetPubKey().Equal(ed25519.PublicKey(p.PubKey)) {
			pid = p.ParticipantId
			break
		}
	}

	if pid == emptyParticipantId {
		return errors.New("failed to determine participant id")
	}

	fsmRequest := requests.SignatureProposalParticipantRequest{
		ParticipantId: pid,
		CreatedAt:     operation.CreatedAt,
	}

	reqBz, err := json.Marshal(fsmRequest)
	if err != nil {
		return fmt.Errorf("failed to generate FSM request: %w", err)
	}

	operation.Event = spf.EventConfirmSignatureProposal
	operation.ResultMsgs = append(operation.ResultMsgs, storage.Message{
		Event:         string(operation.Event),
		Data:          reqBz,
		DkgRoundID:    operation.DKGIdentifier,
		RecipientAddr: operation.To,
	})

	return s.executeOperation(operation)

}

func (s *BaseNodeService) ReInitDKG(dto *dto.ReInitDKGDTO) error {

	message, err := s.buildMessage(dto.ID, fsm.Event(types.ReinitDKG), dto.Payload)

	if err != nil {
		return fmt.Errorf("failed to build message: %v", err)
	}

	err = s.storage.Send(*message)

	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}

	return nil
}

func (s *BaseNodeService) SaveOffset(dto *dto.StateOffsetDTO) error {
	err := s.getState().SaveOffset(dto.Offset)

	if err != nil {
		return fmt.Errorf("failed to save offset: %v", err)
	}

	return nil
}

func (s *BaseNodeService) reinitDKG(message storage.Message) error {
	var req types.ReDKG
	if err := json.Unmarshal(message.Data, &req); err != nil {
		return fmt.Errorf("failed to umarshal request: %v", err)
	}

	// temporarily fix cause we can't verify patch messages
	// TODO: remove later
	if !s.GetSkipCommKeysVerification() {
		s.SetSkipCommKeysVerification(true)
		defer s.SetSkipCommKeysVerification(false)
	}

	operations := make([]*types.Operation, 0)
	for _, msg := range req.Messages {
		if fsm.Event(msg.Event) == sif.EventSigningStart {
			break
		}
		if msg.RecipientAddr == "" || msg.RecipientAddr == s.GetUsername() {
			operation, err := s.processMessage(msg)
			if err != nil {
				s.Logger.Log("failed to process operation: %v", err)
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
	operation.ExtraData, err = types.CalcStartReInitDKGMessageHash(message.Data)
	if err != nil {
		return fmt.Errorf("failed to calculat reinitDKG message hash: %w", err)
	}
	if err := s.opService.PutOperation(operation); err != nil {
		return fmt.Errorf("failed to PutOperation: %w", err)
	}

	// save new comm keys into FSM to verify future messages
	fsmInstance, err := s.fsmService.GetFSMInstance(req.DKGID, true)
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

	if err := s.fsmService.SaveFSM(message.DkgRoundID, fsmDump); err != nil {
		return fmt.Errorf("failed to SaveFSM: %w", err)
	}

	return nil
}

// processSignature saves a broadcasted reconstructed signature to a LevelDB
func (s *BaseNodeService) processSignature(message storage.Message) error {
	var (
		signatures []types.ReconstructedSignature
		err        error
	)
	if err = json.Unmarshal(message.Data, &signatures); err != nil {
		return fmt.Errorf("failed to unmarshal reconstructed signature: %w", err)
	}
	for i := range signatures {
		signatures[i].Username = message.SenderAddr
		signatures[i].DKGRoundID = message.DkgRoundID
	}
	return s.sigService.SaveSignatures(signatures)
}

// processBatchSignature saves a broadcasted reconstructed batch signatures to a LevelDB
func (s *BaseNodeService) processSignatureProposal(message storage.Message) error {
	var (
		proposal requests.SigningBatchProposalStartRequest
		err      error
	)

	if err = json.Unmarshal(message.Data, &proposal); err != nil {
		return fmt.Errorf("failed to unmarshal reconstructed signature: %w", err)
	}
	signatures := make([]types.ReconstructedSignature, 0, len(proposal.MessagesToSign))
	for _, msg := range proposal.MessagesToSign {
		sig := types.ReconstructedSignature{
			MessageID:  msg.MessageID,
			Username:   message.SenderAddr,
			DKGRoundID: message.DkgRoundID,
			SrcPayload: msg.Payload,
		}
		signatures = append(signatures, sig)
	}

	err = s.sigService.SaveSignatures(signatures)
	if err != nil {
		return fmt.Errorf("failed to save signature: %w", err)
	}
	return nil
}

func (s *BaseNodeService) processMessage(message storage.Message) (*types.Operation, error) {
	fsmInstance, err := s.fsmService.GetFSMInstance(message.DkgRoundID, true)
	if err != nil {
		return nil, fmt.Errorf("failed to getFSMInstance: %w", err)
	}

	// we can't verify a message at this moment, cause we don't have public keys of participants
	if fsm.Event(message.Event) != spf.EventInitProposal {
		if err := s.verifyMessage(fsmInstance, message); err != nil {
			return nil, fmt.Errorf("failed to verifyMessage %+v: %w", message, err)
		}
	}

	switch fsm.Event(message.Event) {
	case types.SignatureReconstructed: // save broadcasted reconstructed signature
		if err := s.processSignature(message); err != nil {
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
		s.Logger.Log("Participant #%d got an error during signature reconstruction process: %v", errorRequestTyped.ParticipantId, errorRequestTyped.Error)
		return nil, nil
	}

	//TODO: refactor the following checks
	//handle common errors
	if strings.HasSuffix(string(fsmInstance.FSMDump().State), "_error") {
		if fsmInstance.FSMDump().Payload.DKGProposalPayload != nil {
			for _, participant := range fsmInstance.FSMDump().Payload.DKGProposalPayload.Quorum {
				if participant.Error != nil {
					s.Logger.Log("Participant %s got an error during DKG process: %s. DKG aborted\n",
						participant.Username, participant.Error.Error())
					// if we have an error during DKG, abort the whole DKG procedure.
					return nil, nil
				}
			}
		}
		if fsmInstance.FSMDump().Payload.SigningProposalPayload != nil {
			for _, participant := range fsmInstance.FSMDump().Payload.SigningProposalPayload.Quorum {
				if participant.Error != nil {
					s.Logger.Log("Participant %s got an error during signing procedure: %s. Signing procedure aborted\n",
						participant.Username, participant.Error.Error())
					break
				}
			}
			//if we have an error during signing procedure, start a new signing procedure
			_, fsmDump, err := fsmInstance.Do(sif.EventSigningRestart, requests.DefaultRequest{
				CreatedAt: time.Now(),
			})
			if err != nil {
				return nil, fmt.Errorf("failed to Do operation in FSM: %w", err)
			}

			if err := s.fsmService.SaveFSM(message.DkgRoundID, fsmDump); err != nil {
				return nil, fmt.Errorf("failed to SaveFSM: %w", err)

			}
		}
	}

	//handle timeout errors
	if strings.HasSuffix(string(fsmInstance.FSMDump().State), "_timeout") {
		if strings.HasPrefix(string(fsmInstance.FSMDump().State), "state_sig_") ||
			strings.HasPrefix(string(fsmInstance.FSMDump().State), "state_dkg") {
			s.Logger.Log("DKG process with ID \"%s\" aborted cause of timeout\n",
				fsmInstance.FSMDump().Payload.DkgId)
			// if we have an error during DKG, abort the whole DKG procedure.
			return nil, nil
		}
		if strings.HasPrefix(string(fsmInstance.FSMDump().State), "state_signing_") {
			s.Logger.Log("Signing process with ID \"%s\" aborted cause of timeout\n",
				fsmInstance.FSMDump().Payload.SigningProposalPayload.BatchID)

			//if we have an error during signing procedure, start a new signing procedure
			_, fsmDump, err := fsmInstance.Do(sif.EventSigningRestart, requests.DefaultRequest{
				CreatedAt: time.Now(),
			})
			if err != nil {
				return nil, fmt.Errorf("failed to Do operation in FSM: %w", err)
			}

			if err := s.fsmService.SaveFSM(message.DkgRoundID, fsmDump); err != nil {
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

	s.Logger.Log("message %s done successfully from %s", message.Event, message.SenderAddr)

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
		resp, fsmDump, err = fsmInstance.Do(sif.EventSigningInit, requests.DefaultRequest{
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
		sif.StateSigningAwaitPartialSigns,
		sif.StateSigningPartialSignsCollected:
		if resp.Data != nil {
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
		s.Logger.Log("State %s does not require an operation", resp.State)
	}

	// switch FSM state by hand due to implementation specifics
	if resp.State == sif.StateSigningPartialSignsCollected {
		fsmInstance, err = state_machines.FromDump(fsmDump)
		if err != nil {
			return nil, fmt.Errorf("failed get state_machines from dump: %w", err)
		}
		_, fsmDump, err = fsmInstance.Do(sif.EventSigningRestart, requests.DefaultRequest{
			CreatedAt: time.Now(),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to Do operation in FSM: %w", err)
		}
	}

	// save signing data to the same storage as we save signatures
	// This allows easy to view signing data by CLI-command
	if fsm.Event(message.Event) == sif.EventSigningStart {
		if err := s.processSignatureProposal(message); err != nil {
			return nil, fmt.Errorf("failed to process signature: %w", err)
		}
	}

	if err := s.fsmService.SaveFSM(message.DkgRoundID, fsmDump); err != nil {
		return nil, fmt.Errorf("failed to SaveFSM: %w", err)
	}

	return operation, nil
}
