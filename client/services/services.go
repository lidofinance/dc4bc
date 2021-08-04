package services

import (
	"fmt"
	"github.com/lidofinance/dc4bc/client/config"
	"github.com/lidofinance/dc4bc/client/modules/keystore"
	"github.com/lidofinance/dc4bc/storage/kafka_storage"
)

func InitServices(config *config.Config) error {
	//var err error

	keyStore, err := keystore.NewLevelDBKeyStore(config.Username, config.KeyStoreDBDSN)
	if err != nil {
		return err
	}

	storage, err := kafka_storage.NewKafkaStorage(
		config.KafkaStorageConfig.DBDSN,
		config.KafkaStorageConfig.Topic,
		config.KafkaStorageConfig.ConsumerGroup,
		config.KafkaStorageConfig.TlsConfig,
		config.KafkaStorageConfig.ProducerCredentials,
		config.KafkaStorageConfig.ConsumerCredentials,
		config.KafkaStorageConfig.Timeout,
	)

	if err != nil {
		return fmt.Errorf("failed to init storage client: %w", err)
	}

	if err := storage.IgnoreMessages(
		config.KafkaStorageConfig.IgnoredMessages,
		config.KafkaStorageConfig.UseOffsetInsteadId,
	); err != nil {
		return fmt.Errorf("failed to ignore messages in storage: %w", err)
	}

	provider.Init(config, storage, keyStore)

	return nil
}
