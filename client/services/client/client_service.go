package client

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"github.com/lidofinance/dc4bc/client/api/dto"
	"github.com/lidofinance/dc4bc/client/config"
	"github.com/lidofinance/dc4bc/client/modules/keystore"
	"github.com/lidofinance/dc4bc/client/modules/logger"
	"github.com/lidofinance/dc4bc/client/modules/state"
	"github.com/lidofinance/dc4bc/client/types"
	"github.com/lidofinance/dc4bc/qr"
	"github.com/lidofinance/dc4bc/storage"
	"path/filepath"
	"sync"
	"time"
)

const (
	pollingPeriod = time.Second
	qrCodesDir    = "/tmp"
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

// GetOperationQRPath returns a path to the image with the QR generated
// for the specified operation. It is supposed that the user will open
// this file herself.
func (s *BaseClientService) GetOperationQRPath(operationID string) (string, error) {
	operationJSON, err := s.getOperationJSON(operationID)

	if err != nil {
		return "", fmt.Errorf("failed to get operation in JSON: %w", err)
	}

	operationQRPath := filepath.Join(qrCodesDir, fmt.Sprintf("dc4bc_qr_%s", operationID))

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
func (s *BaseClientService) GetSignatures(dkgID string) (map[string][]types.ReconstructedSignature, error) {
	return s.getState().GetSignatures(dkgID)
}
