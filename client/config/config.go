package config

type HttpApiConfig struct {
	ListenAddr string `mapstructure:"listen_addr"`
	Debug      bool   `mapstructure:"debug"`
}

type QrProcessorConfig struct {
	FramesDelay int `mapstructure:"frames_delay"`
	ChunkSize   int `mapstructure:"chunk_size"`
}

type KafkaStorageConfig struct {
	DBDSN               string `mapstructure:"storage_dbdsn"`
	Topic               string `mapstructure:"storage_topic"`
	ConsumerGroup       string `mapstructure:"kafka_consumer_group"`
	TlsConfig           string `mapstructure:"kafka_truststore_path"`
	ProducerCredentials string `mapstructure:"producer_credentials"`
	ConsumerCredentials string `mapstructure:"consumer_credentials"`
	Timeout             string `mapstructure:"kafka_timeout"`

	IgnoredMessages    string `mapstructure:"storage_ignore_messages"`
	UseOffsetInsteadId bool   `mapstructure:"offsets_to_ignore_messages"`
}

type Config struct {
	HttpApiConfig *HttpApiConfig

	QrProcessorConfig *QrProcessorConfig

	KafkaStorageConfig *KafkaStorageConfig

	Username      string `mapstructure:"username"`
	StateDBSN     string `mapstructure:"state_dbdsn"`
	KeyStoreDBDSN string `mapstructure:"key_store_dbdsn"`
}
