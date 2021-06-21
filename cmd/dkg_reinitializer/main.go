package main

import (
	"encoding/json"
	"fmt"
	"github.com/lidofinance/dc4bc/client/types"
	"github.com/lidofinance/dc4bc/storage/kafka_storage"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"strings"
)

const (
	flagStorageDBDSN             = "storage_dbdsn"
	flagKafkaProducerCredentials = "producer_credentials"
	flagKafkaConsumerCredentials = "consumer_credentials"
	flagKafkaTrustStorePath      = "kafka_truststore_path"
	flagKafkaConsumerGroup       = "kafka_consumer_group"
	flagKafkaTimeout             = "kafka_timeout"
	flagStorageTopic             = "storage_topic"
	flagOutputFile               = "output"
)

var rootCmd = &cobra.Command{
	Use:   "dkg_reinitializer",
	Short: "DKG reinitializer tool",
}

func init() {
	rootCmd.PersistentFlags().String(flagStorageDBDSN, "./dc4bc_file_storage", "Storage DBDSN")
	rootCmd.PersistentFlags().String(flagKafkaProducerCredentials, "producer:producerpass", "Producer credentials for Kafka: username:password")
	rootCmd.PersistentFlags().String(flagKafkaConsumerCredentials, "consumer:consumerpass", "Consumer credentials for Kafka: username:password")
	rootCmd.PersistentFlags().String(flagKafkaTrustStorePath, "certs/ca.pem", "Path to kafka truststore")
	rootCmd.PersistentFlags().String(flagKafkaConsumerGroup, "testUser_consumer_group", "Kafka consumer group")
	rootCmd.PersistentFlags().String(flagKafkaTimeout, "60s", "Kafka I/O Timeout")
	rootCmd.PersistentFlags().String(flagStorageTopic, "messages", "Storage Topic (Kafka)")
	rootCmd.PersistentFlags().StringP(flagOutputFile, "o", "", "Output file")
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

func reinit() *cobra.Command {
	return &cobra.Command{
		Use:   "reinit",
		Short: "reads a Kafka storage, gets all messages from there and returns DKG reinit JSON (to stdout by default).",
		RunE: func(cmd *cobra.Command, args []string) error {
			kafkaTrustStorePath, _ := cmd.Flags().GetString(flagKafkaTrustStorePath)
			kafkaConsumerGroup, _ := cmd.Flags().GetString(flagKafkaConsumerGroup)
			kafkaTimeout, _ := cmd.Flags().GetDuration(flagKafkaTimeout)
			tlsConfig, err := kafka_storage.GetTLSConfig(kafkaTrustStorePath)
			if err != nil {
				return fmt.Errorf("failed to create tls config: %v", err)
			}

			storageTopic, _ := cmd.Flags().GetString(flagStorageTopic)

			producerCredentials, _ := cmd.Flags().GetString(flagKafkaProducerCredentials)
			producerCreds, err := parseKafkaSaslPlain(producerCredentials)
			if err != nil {
				return fmt.Errorf("failed to parse kafka credentials: %v", err)
			}

			consumerCredentials, _ := cmd.Flags().GetString(flagKafkaConsumerCredentials)
			consumerCreds, err := parseKafkaSaslPlain(consumerCredentials)
			if err != nil {
				return fmt.Errorf("failed to parse kafka credentials: %v", err)
			}

			storageDBDSN, _ := cmd.Flags().GetString(flagStorageDBDSN)
			stg, err := kafka_storage.NewKafkaStorage(storageDBDSN, storageTopic, kafkaConsumerGroup, tlsConfig,
				producerCreds, consumerCreds, kafkaTimeout)
			if err != nil {
				return fmt.Errorf("failed to init storage: %v", err)
			}

			messages, err := stg.GetMessages(0)
			if err != nil {
				return fmt.Errorf("failed to get messages: %v", err)
			}

			reDKG, err := types.GenerateReDKGMessage(messages)
			if err != nil {
				return fmt.Errorf("failed to generate reDKG message: %v", err)
			}

			reDKGBz, err := json.Marshal(reDKG)
			if err != nil {
				return fmt.Errorf("failed to encode reinit DKG message: %v", err)
			}

			outputFile, _ := cmd.Flags().GetString(flagOutputFile)
			if len(outputFile) == 0 {
				fmt.Println(string(reDKGBz))
				return nil
			}

			if err = ioutil.WriteFile(outputFile, reDKGBz, 0666); err != nil {
				return fmt.Errorf("failed to save reinit DKG JSON: %v", err)
			}
			return nil
		},
	}
}

func main() {
	rootCmd.AddCommand(
		reinit(),
	)
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Failed to execute root command: %v", err)
	}
}
