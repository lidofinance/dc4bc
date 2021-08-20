package client

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/lidofinance/dc4bc/client/api/dto"
	"github.com/lidofinance/dc4bc/client/config"
	"github.com/lidofinance/dc4bc/client/modules/keystore"
	"github.com/lidofinance/dc4bc/client/modules/logger"
	"github.com/lidofinance/dc4bc/client/modules/state"
	"github.com/lidofinance/dc4bc/client/types"
	"github.com/lidofinance/dc4bc/fsm/fsm"
	"github.com/lidofinance/dc4bc/fsm/state_machines"
	spf "github.com/lidofinance/dc4bc/fsm/state_machines/signature_proposal_fsm"
	sif "github.com/lidofinance/dc4bc/fsm/state_machines/signing_proposal_fsm"
	"github.com/lidofinance/dc4bc/fsm/types/requests"
	"github.com/lidofinance/dc4bc/fsm/types/responses"
	"github.com/lidofinance/dc4bc/qr"
	"github.com/lidofinance/dc4bc/storage"
	"github.com/lidofinance/dc4bc/storage/kafka_storage"
	"path/filepath"
	"sync"
	"time"
)

const (
	pollingPeriod      = time.Second
	qrCodesDir         = "/tmp"
	emptyParticipantId = -1
)

type IBaseClientService interface {
	//CreateBlog(userId int, dto *dto.BlogCreateDTO) (blog *BlogModel, serr *i18n.I18nError)
	Poll() error
	GetLogger() logger.Logger
	// GetPubKey() ed25519.PublicKey
	// GetUsername() string
	// SendMessage(message storage.Message) error
	ProcessMessage(message storage.Message) error
	// GetOperations() (map[string]*types.Operation, error)
	// GetOperationQRPath(operationID string) (string, error)
	// StartHTTPServer(listenAddr string) error
	StopHTTPServer()
	SetSkipCommKeysVerification(bool)
	ResetState(newStateDBPath string, consumerGroup string, messages []string, useOffset bool) (string, error)
}

type BaseClientService struct {
	sync.Mutex
	config                   *config.Config
	userName                 string
	pubKey                   ed25519.PublicKey
	stateMu                  sync.RWMutex
	state                    state.State
	storage                  storage.Storage
	keyStore                 keystore.KeyStore
	qrProcessor              qr.Processor
	Logger                   logger.Logger
	SkipCommKeysVerification bool
}

func Init(config *config.Config, storage storage.Storage, ks keystore.KeyStore) (*BaseClientService, error) {
	keyPair, err := ks.LoadKeys(config.Username, "")

	if err != nil {
		return nil, fmt.Errorf("failed to LoadKeys: %w", err)
	}

	qrProcessor := qr.NewCameraProcessor()
	qrProcessor.SetDelay(config.QrProcessorConfig.FramesDelay)
	qrProcessor.SetChunkSize(config.QrProcessorConfig.ChunkSize)

	return &BaseClientService{
		config:      config,
		userName:    config.Username,
		pubKey:      keyPair.Pub,
		state:       config.State,
		storage:     storage,
		keyStore:    ks,
		qrProcessor: qrProcessor,
		Logger:      logger.NewLogger(config.Username),
	}, nil
}

func (s *BaseClientService) getState() state.State {
	s.stateMu.RLock()
	defer s.stateMu.Unlock()
	return s.state
}

func (s *BaseClientService) GetPubKey() ed25519.PublicKey {
	return s.pubKey
}

func (s *BaseClientService) GetUsername() string {
	return s.userName
}

func (s *BaseClientService) GetStateOffset() (uint64, error) {
	return s.getState().LoadOffset()
}

func (s *BaseClientService) GetSkipCommKeysVerification() bool {
	s.Lock()
	defer s.Unlock()

	return s.SkipCommKeysVerification
}

func (s *BaseClientService) SendMessage(dto *dto.MessageDTO) error {
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

// GetOperations returns available operations for current state
func (s *BaseClientService) GetOperations() (map[string]*types.Operation, error) {
	return s.getState().GetOperations()
}

// GetOperation returns operation for current state, if exists
func (s *BaseClientService) getOperation(operationID string) (*types.Operation, error) {

	operations, err := s.getState().GetOperations()

	if err != nil {
		return nil, fmt.Errorf("failed to get operations: %v", err)
	}

	operation, ok := operations[operationID]

	if !ok {
		return nil, fmt.Errorf("failed to get operation")
	}
	return operation, nil
}

// GetOperationQRPath returns a path to the image with the QR generated
// for the specified operation. It is supposed that the user will open
// this file herself.
func (s *BaseClientService) GetOperationQRPath(dto *dto.OperationIdDTO) (string, error) {
	operationJSON, err := s.getOperationJSON(dto.OperationID)

	if err != nil {
		return "", fmt.Errorf("failed to get operation in JSON: %w", err)
	}

	operationQRPath := filepath.Join(qrCodesDir, fmt.Sprintf("dc4bc_qr_%s", dto.OperationID))

	qrPath := fmt.Sprintf("%s.gif", operationQRPath)
	if err = s.qrProcessor.WriteQR(qrPath, operationJSON); err != nil {
		return "", err
	}

	return qrPath, nil
}

// getOperationJSON returns a specific JSON-encoded operation
func (s *BaseClientService) getOperationJSON(operationID string) ([]byte, error) {
	operation, err := s.getState().GetOperationByID(operationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get operation: %w", err)
	}

	operationJSON, err := json.Marshal(operation)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal operation: %w", err)
	}
	return operationJSON, nil
}

// GetSignatures returns all signatures for the given DKG round that were reconstructed on the airgapped machine and
// broadcasted by users
func (s *BaseClientService) GetSignatures(dto *dto.DkgIdDTO) (map[string][]types.ReconstructedSignature, error) {
	return s.getState().GetSignatures(dto.DkgID)
}

func (s *BaseClientService) GetSignatureByID(dto *dto.SignatureByIdDTO) ([]types.ReconstructedSignature, error) {
	return s.getState().GetSignatureByID(dto.DkgID, dto.ID)
}

func (s *BaseClientService) GetOperationQRFile(dto *dto.OperationIdDTO) ([]byte, error) {
	operationJSON, err := s.getOperationJSON(dto.OperationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get operation in JSON: %v", err)
	}

	encodedData, err := qr.EncodeQR(operationJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to encode operation: %v", err)
	}

	return encodedData, nil
}

// handleProcessedOperation handles an operation which was processed by the airgapped machine
// It checks that the operation exists in an operation pool, signs the operation, sends it to an append-only log and
// deletes it from the pool.
func (s *BaseClientService) ProcessOperation(dto *dto.OperationDTO) error {
	operation := &types.Operation{
		ID:            dto.ID,
		Type:          types.OperationType(dto.Type),
		Payload:       dto.Payload,
		ResultMsgs:    dto.ResultMsgs,
		CreatedAt:     dto.CreatedAt,
		DKGIdentifier: dto.DKGIdentifier,
		To:            dto.To,
		Event:         dto.Event,
	}

	return s.executeOperation(operation)
}

func (s *BaseClientService) executeOperation(operation *types.Operation) error {
	if operation.Event.IsEmpty() {
		return errors.New("operation is request operation, provide result operation instead")
	}

	storedOperation, err := s.getState().GetOperationByID(operation.ID)
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

	if err := s.getState().DeleteOperation(operation); err != nil {
		return fmt.Errorf("failed to DeleteOperation: %w", err)
	}

	return nil
}

func (s *BaseClientService) signMessage(message []byte) ([]byte, error) {
	keyPair, err := s.keyStore.LoadKeys(s.userName, "")
	if err != nil {
		return nil, fmt.Errorf("failed to LoadKeys: %w", err)
	}

	return ed25519.Sign(keyPair.Priv, message), nil
}

func (s *BaseClientService) verifyMessage(fsmInstance *state_machines.FSMInstance, message storage.Message) error {
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

func (s *BaseClientService) GetOperation(dto *dto.OperationIdDTO) ([]byte, error) {
	return s.getOperationJSON(dto.OperationID)
}

func (s *BaseClientService) StartDKG(dto *dto.StartDkgDTO) error {
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

func (s *BaseClientService) buildMessage(dkgRoundID string, event fsm.Event, data []byte) (*storage.Message, error) {
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

func (s *BaseClientService) ProposeSignData(dto *dto.ProposeSignDataDTO) error {
	fsmInstance, err := s.getFSMInstance(dto.DkgID)

	if err != nil {
		return fmt.Errorf("failed to get FSM instance: %v", err)
	}

	participantID, err := fsmInstance.GetIDByUsername(s.GetUsername())

	if err != nil {
		return fmt.Errorf("failed to get participantID: %v", err)
	}

	messageDataSign := requests.SigningProposalStartRequest{
		SigningID:     uuid.New().String(),
		ParticipantId: participantID,
		SrcPayload:    dto.Data,
		CreatedAt:     time.Now(), // Is better to use time from client?
	}

	messageDataSignBz, err := json.Marshal(messageDataSign)

	if err != nil {
		return fmt.Errorf("failed to marshal SigningProposalStartRequest: %v", err)
	}

	message, err := s.buildMessage(dto.DkgID, sif.EventSigningStart, messageDataSignBz)

	if err != nil {
		return fmt.Errorf("failed to build message: %v", err)
	}

	err = s.storage.Send(*message)

	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}

	return nil
}

// getFSMInstance returns FSM for a necessary DKG round.
func (s *BaseClientService) getFSMInstance(dkgRoundID string) (*state_machines.FSMInstance, error) {
	var err error
	fsmInstance, ok, err := s.getState().LoadFSM(dkgRoundID)
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
		if err := s.getState().SaveFSM(dkgRoundID, bz); err != nil {
			return nil, fmt.Errorf("failed to SaveFSM: %w", err)
		}
	}

	return fsmInstance, nil
}

func (s *BaseClientService) ApproveParticipation(dto *dto.OperationIdDTO) error {
	operation, err := s.getOperation(dto.OperationID)

	if err != nil {
		return err
	}

	if fsm.State(operation.Type) != spf.StateAwaitParticipantsConfirmations {
		return fmt.Errorf("cannot approve participation with operationID %s", dto.OperationID)
	}

	var payload responses.SignatureProposalParticipantInvitationsResponse

	if err = json.Unmarshal(operation.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %v", err)
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
		return fmt.Errorf("failed to generate FSM request: %v", err)
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

func (s *BaseClientService) ReInitDKG(dto *dto.ReInitDKGDTO) error {

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

func (s *BaseClientService) SaveOffset(dto *dto.StateOffsetDTO) error {
	err := s.getState().SaveOffset(dto.Offset)

	if err != nil {
		return fmt.Errorf("failed to save offset: %v", err)
	}

	return nil
}

func (s *BaseClientService) GetFSMDump(dto *dto.DkgIdDTO) (*state_machines.FSMDump, error) {
	fsmInstance, err := s.getFSMInstance(dto.DkgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get FSM instance for DKG round ID %s: %w", dto.DkgID, err)
	}
	return fsmInstance.FSMDump(), nil
}

func (s *BaseClientService) GetFSMList() (map[string]string, error) {
	fsmInstances, err := s.getState().GetAllFSM()

	if err != nil {
		return nil, fmt.Errorf("failed to get all FSM instances: %v", err)
	}

	fsmInstancesStates := make(map[string]string, len(fsmInstances))
	for k, v := range fsmInstances {
		fsmState, err := v.State()
		if err != nil {
			return nil, fmt.Errorf("failed to get FSM state: %v", err)
		}
		fsmInstancesStates[k] = fsmState.String()
	}

	return fsmInstancesStates, nil
}

func (s *BaseClientService) ResetFSMState(dto *dto.ResetStateDTO) (string, error) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	if err := s.storage.IgnoreMessages(dto.Messages, dto.UseOffset); err != nil {
		return "", fmt.Errorf("failed to ignore messages while resetting state: %v", err)
	}

	switch s.storage.(type) {
	case *kafka_storage.KafkaStorage:
		stg := s.storage.(*kafka_storage.KafkaStorage)
		if err := stg.SetConsumerGroup(dto.KafkaConsumerGroup); err != nil {
			return "", fmt.Errorf("failed to set consumer group while reseting state: %v", err)
		}
	}

	newState, newStateDbPath, err := s.state.NewStateFromOld(dto.NewStateDBDSN)

	if err != nil {
		return "", fmt.Errorf("failed to create new state from old: %v", err)
	}

	s.state = newState

	return newStateDbPath, err
}
