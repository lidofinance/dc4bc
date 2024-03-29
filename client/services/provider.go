package services

import (
	"fmt"
	"strconv"
	"strings"

	oprepo "github.com/lidofinance/dc4bc/client/repositories/operation"
	sigrepo "github.com/lidofinance/dc4bc/client/repositories/signature"
	"github.com/lidofinance/dc4bc/client/services/fsmservice"
	"github.com/lidofinance/dc4bc/client/services/operation"
	"github.com/lidofinance/dc4bc/client/services/signature"

	"github.com/lidofinance/dc4bc/client/config"
	"github.com/lidofinance/dc4bc/client/modules/keystore"
	"github.com/lidofinance/dc4bc/client/modules/logger"
	"github.com/lidofinance/dc4bc/client/modules/state"
	"github.com/lidofinance/dc4bc/storage"
	"github.com/lidofinance/dc4bc/storage/kafka_storage"
)

type ServiceProvider struct {
	storage    storage.Storage
	ks         keystore.KeyStore
	l          logger.Logger
	state      state.State
	fsm        fsmservice.FSMService
	opService  operation.OperationService
	sigService signature.SignatureService
}

func (s *ServiceProvider) GetStorage() storage.Storage {
	return s.storage
}

func (s *ServiceProvider) SetStorage(stg storage.Storage) {
	s.storage = stg
}

func (s *ServiceProvider) GetKeyStore() keystore.KeyStore {
	return s.ks
}

func (s *ServiceProvider) SetKeyStore(ks keystore.KeyStore) {
	s.ks = ks
}

func (s *ServiceProvider) GetLogger() logger.Logger {
	return s.l
}

func (s *ServiceProvider) SetLogger(l logger.Logger) {
	s.l = l
}

func (s *ServiceProvider) GetState() state.State {
	return s.state
}

func (s *ServiceProvider) SetState(st state.State) {
	s.state = st
}

func (s *ServiceProvider) GetFSMService() fsmservice.FSMService {
	return s.fsm
}

func (s *ServiceProvider) SetFSMService(fsm fsmservice.FSMService) {
	s.fsm = fsm
}

func (s *ServiceProvider) GetOperationService() operation.OperationService {
	return s.opService
}

func (s *ServiceProvider) SetOperationService(opService operation.OperationService) {
	s.opService = opService
}

func (s *ServiceProvider) GetSignatureService() signature.SignatureService {
	return s.sigService
}

func (s *ServiceProvider) SetSignatureService(sigService signature.SignatureService) {
	s.sigService = sigService
}

func parseMessagesToIgnore(cfg *config.KafkaStorageConfig) (msgs []string, err error) {
	if cfg == nil {
		return msgs, err
	}
	if len(cfg.IgnoredMessages) == 0 {
		return msgs, err
	}

	msgs = strings.Split(cfg.IgnoredMessages, ",")
	if cfg.UseOffsetInsteadId {
		for _, msg := range msgs {
			if _, err = strconv.ParseUint(msg, 10, 64); err != nil {
				return nil, fmt.Errorf("when %s flag is specified, values provided in %s flag should be"+
					" parsable into uint64. error: %w", "storage_ignore_messages", "offsets_to_ignore_messages", err)
			}
		}
	}

	return msgs, nil
}

func CreateServiceProviderWithCfg(cfg *config.Config) (*ServiceProvider, error) {
	var err error
	sp := ServiceProvider{}

	sp.storage, err = kafka_storage.NewKafkaStorage(cfg.KafkaStorageConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize fafka storgae: %w", err)
	}

	ignoredMsgs, err := parseMessagesToIgnore(cfg.KafkaStorageConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to ignore messages in storage: %w", err)
	}
	if err := sp.storage.IgnoreMessages(ignoredMsgs, cfg.KafkaStorageConfig.UseOffsetInsteadId); err != nil {
		return nil, fmt.Errorf("failed to ignore messages in storage: %w", err)
	}

	sp.ks, err = keystore.NewLevelDBKeyStore(cfg.Username, cfg.KeyStoreDBDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to init key store: %w", err)
	}

	sp.l = logger.NewLogger(cfg.Username)

	sp.state, err = state.NewLevelDBState(cfg.StateDBSN, cfg.KafkaStorageConfig.Topic)
	if err != nil {
		return nil, fmt.Errorf("failed to init state: %w", err)
	}

	sigRepo := sigrepo.NewSignatureRepo(sp.state)
	opRepo, err := oprepo.NewOperationRepo(sp.state, cfg.KafkaStorageConfig.Topic)
	if err != nil {
		return nil, fmt.Errorf("failed to init operation repo: %w", err)
	}

	sp.fsm = fsmservice.NewFSMService(sp.state, sp.storage, cfg.KafkaStorageConfig.Topic)
	sp.sigService = signature.NewSignatureService(sigRepo)
	sp.opService = operation.NewOperationService(opRepo)

	return &sp, nil
}
