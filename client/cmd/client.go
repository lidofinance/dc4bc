package main

import (
	"fmt"
	"github.com/lidofinance/dc4bc/client/api"
	"github.com/lidofinance/dc4bc/client/config"
	"github.com/lidofinance/dc4bc/client/services"
	"github.com/lidofinance/dc4bc/storage/kafka_storage"
	"github.com/segmentio/kafka-go/sasl/plain"
	"log"
	"os"
	"strings"
)

func main() {
	conf := &config.Config{
		BaseUrl: "http://localhost",
		HttpApiConfig: &config.HttpApiConfig{
			Host:  "",
			Port:  8080,
			Debug: false,
		},
		QrProcessorConfig: &config.QrProcessorConfig{
			FramesDelay: 10,
			ChunkSize:   256,
		},
		KafkaStorageConfig: &config.KafkaStorageConfig{
			DBDSN:         "",
			Topic:         "",
			ConsumerGroup: "",
		},
		Username:      "",
		KeyStoreDBDSN: "",
	}

	conf.KafkaStorageConfig.TlsConfig, _ = kafka_storage.GetTLSConfig("")
	conf.KafkaStorageConfig.ProducerCredentials, _ = parseKafkaSaslPlain("")
	conf.KafkaStorageConfig.ConsumerCredentials, _ = parseKafkaSaslPlain("")

	err := services.InitServices(conf)

	checkErr("Cannot init services", err)

	api.Run(conf)
}

func checkErr(format string, err error) {
	if err != nil {
		log.Printf("[Application][Fatal] %s. ErrorMessage: %s\n", format, err.Error())
		os.Exit(1)
	}
}

func parseKafkaSaslPlain(creds string) (*plain.Mechanism, error) {
	credsSplit := strings.SplitN(creds, ":", 2)
	if len(credsSplit) == 1 {
		return nil, fmt.Errorf("failed to parse credentials")
	}
	return &plain.Mechanism{
		Username: credsSplit[0],
		Password: credsSplit[1],
	}, nil
}
