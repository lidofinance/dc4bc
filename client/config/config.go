package config

import (
	"crypto/tls"
	"github.com/lidofinance/dc4bc/client/modules/state"
	"github.com/segmentio/kafka-go/sasl/plain"
	"time"
)

type HttpApiConfig struct {
	Host  string `mapstructure:"host"`
	Port  int    `mapstructure:"port"`
	Debug bool   `mapstructure:"debug"`
}

type QrProcessorConfig struct {
	FramesDelay int `mapstructure:"frames_delay"`
	ChunkSize   int `mapstructure:"chunk_size"`
}

type KafkaStorageConfig struct {
	DBDSN               string           // storage_dbdsn
	Topic               string           // storage_topic
	ConsumerGroup       string           // kafka_consumer_group
	TlsConfig           *tls.Config      // kafka_truststore_path
	ProducerCredentials *plain.Mechanism // producer_credentials
	ConsumerCredentials *plain.Mechanism // consumer_credentials
	Timeout             time.Duration

	IgnoredMessages    []string
	UseOffsetInsteadId bool
}

// TODO: Add github.com/spf13/viper
type Config struct {
	BaseUrl string `mapstructure:"base_url"`

	HttpApiConfig *HttpApiConfig `mapstructure:"http_api_config"`

	QrProcessorConfig *QrProcessorConfig

	KafkaStorageConfig *KafkaStorageConfig

	Username      string `mapstructure:"username"`
	State         state.State
	KeyStoreDBDSN string `mapstructure:"key_store_dbdsn"`
}
