package main

import (
	"encoding/json"
	"fmt"
	"github.com/lidofinance/dc4bc/client/types"
	"github.com/lidofinance/dc4bc/fsm/fsm"
	spf "github.com/lidofinance/dc4bc/fsm/state_machines/signature_proposal_fsm"
	"github.com/lidofinance/dc4bc/fsm/types/requests"
	"github.com/lidofinance/dc4bc/storage"
	"github.com/lidofinance/dc4bc/storage/file_storage"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/spf13/cobra"
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
)

var rootCmd = &cobra.Command{
	Use:   "dkg_reinitializer",
	Short: "",
}

func init() {
	rootCmd.PersistentFlags().String(flagStorageDBDSN, "./dc4bc_file_storage", "Storage DBDSN")
	rootCmd.PersistentFlags().String(flagKafkaProducerCredentials, "producer:producerpass", "Producer credentials for Kafka: username:password")
	rootCmd.PersistentFlags().String(flagKafkaConsumerCredentials, "consumer:consumerpass", "Consumer credentials for Kafka: username:password")
	rootCmd.PersistentFlags().String(flagKafkaTrustStorePath, "certs/ca.pem", "Path to kafka truststore")
	rootCmd.PersistentFlags().String(flagKafkaConsumerGroup, "testUser_consumer_group", "Kafka consumer group")
	rootCmd.PersistentFlags().String(flagKafkaTimeout, "60s", "Kafka I/O Timeout")
	rootCmd.PersistentFlags().String(flagStorageTopic, "messages", "Storage Topic (Kafka)")
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

type Participant struct {
	DKGPubKey     []byte `json:"dkg_pub_key"`
	OldCommPubKey []byte `json:"old_comm_pub_key"`
	NewCommPubKey []byte `json:"new_comm_pub_key"`
	Name          string `json:"name"`
}

type ReDKG struct {
	DKGID        string            `json:"dkg_id"`
	Threshold    int               `json:"threshold"`
	Participants []Participant     `json:"participants"`
	Messages     []storage.Message `json:"messages"`
}

func main() {
	//kafkaTrustStorePath := viper.GetString(flagKafkaTrustStorePath)
	//kafkaConsumerGroup := viper.GetString(flagKafkaConsumerGroup)
	//kafkaTimeout := viper.GetDuration(flagKafkaTimeout)
	//tlsConfig, err := kafka_storage.GetTLSConfig(kafkaTrustStorePath)
	//if err != nil {
	//	log.Fatalf("failed to create tls config: %v", err)
	//}
	//
	//storageTopic := viper.GetString(flagStorageTopic)
	//
	//producerCredentials := viper.GetString(flagKafkaProducerCredentials)
	//producerCreds, err := parseKafkaSaslPlain(producerCredentials)
	//if err != nil {
	//	log.Fatalf("failed to parse kafka credentials: %v", err)
	//}
	//
	//consumerCredentials := viper.GetString(flagKafkaConsumerCredentials)
	//consumerCreds, err := parseKafkaSaslPlain(consumerCredentials)
	//if err != nil {
	//	log.Fatalf("failed to parse kafka credentials: %v", err)
	//}
	//
	//storageDBDSN := viper.GetString(flagStorageDBDSN)
	//stg, err := kafka_storage.NewKafkaStorage(storageDBDSN, storageTopic, kafkaConsumerGroup, tlsConfig,
	//	producerCreds, consumerCreds, kafkaTimeout)
	stg, err := file_storage.NewFileStorage("/tmp/dc4bc_storage")
	if err != nil {
		log.Fatalf("failed to init storage: %v", err)
	}

	messages, err := stg.GetMessages(0)
	if err != nil {
		log.Fatalf("failed to get messages: %v", err)
	}

	var reDKG ReDKG

	for _, msg := range messages {
		if fsm.Event(msg.Event) == spf.EventInitProposal {
			req, err := types.FSMRequestFromMessage(msg)
			if err != nil {
				log.Fatalf("failed to get FSM request from message: %v", err)
			}
			request, ok := req.(requests.SignatureProposalParticipantsListRequest)
			if !ok {
				log.Fatalf("invalid request")
			}
			reDKG.DKGID = msg.DkgRoundID
			reDKG.Threshold = request.SigningThreshold
			for _, participant := range request.Participants {
				reDKG.Participants = append(reDKG.Participants, Participant{
					DKGPubKey:     participant.DkgPubKey,
					OldCommPubKey: participant.PubKey,
					Name:          participant.Username,
				})
			}
		}

		reDKG.Messages = append(reDKG.Messages, msg)
	}

	reDKGBz, err := json.Marshal(reDKG)
	if err != nil {
		log.Fatalf("failed to encode reinit DKG message: %v", err)
	}

	fmt.Println(string(reDKGBz))
}