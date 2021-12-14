package config

type HttpApiConfig struct {
	ListenAddr    string `mapstructure:"listen_addr"`
	Debug         bool   `mapstructure:"enable_http_debug"`
	EnableLogging bool   `mapstructure:"enable_http_logging"`
}

type KafkaStorageConfig struct {
	DBDSN               string `mapstructure:"storage_dbdsn"`
	Topic               string `mapstructure:"storage_topic"`
	ConsumerGroup       string `mapstructure:"kafka_consumer_group"`
	TlsConfig           string `mapstructure:"kafka_truststore_path"`
	ProducerCredentials string `mapstructure:"producer_credentials"`
	ConsumerCredentials string `mapstructure:"consumer_credentials"`
	ReadDuration        string `mapstructure:"kafka_read_duration"`
	Timeout             string `mapstructure:"kafka_timeout"`

	IgnoredMessages    string `mapstructure:"storage_ignore_messages"`
	UseOffsetInsteadId bool   `mapstructure:"offsets_to_ignore_messages"`
}

type Config struct {
	HttpApiConfig *HttpApiConfig

	KafkaStorageConfig *KafkaStorageConfig

	Username      string `mapstructure:"username"`
	StateDBSN     string `mapstructure:"state_dbdsn"`
	KeyStoreDBDSN string `mapstructure:"key_store_dbdsn"`
}
