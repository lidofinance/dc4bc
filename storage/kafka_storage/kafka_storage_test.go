package kafka_storage

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/lidofinance/dc4bc/storage"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/stretchr/testify/require"
)

var (
	testBrokerEndpoint      = "94.130.57.249:9093"
	testTopic               = "long_test_topic"
	testConsumerGroup       = "test_consumer_group"
	testTruststorePath      = "../../ca.crt"
	testTimeout             = time.Second * 10
	testProducerCredentials = &plain.Mechanism{
		Username: "producer",
		Password: "producerpass",
	}
	testConsumerCredentials = &plain.Mechanism{
		Username: "consumer",
		Password: "consumerpass",
	}
)

func getTestStorage() storage.Storage {
	tlsConfig, err := GetTLSConfig(testTruststorePath)
	if err != nil {
		panic(err)
	}

	stg, err := NewKafkaStorage(testBrokerEndpoint, testTopic, testConsumerGroup,
		tlsConfig, testProducerCredentials, testConsumerCredentials, testTimeout)
	if err != nil {
		panic(err)
	}

	msgs, err := stg.GetMessages(0)
	if err != nil {
		panic(err)
	}

	msgIdsToIgnore := make([]string, 0, len(msgs))
	for _, msg := range msgs {
		msgIdsToIgnore = append(msgIdsToIgnore, msg.ID)
	}

	if err = stg.IgnoreMessages(msgIdsToIgnore, false); err != nil {
		panic(err)
	}

	msgs, err = stg.GetMessages(0)
	if err != nil {
		panic(err)
	}
	if len(msgs) > 0 {
		panic(fmt.Errorf("GetMessages() should not return any messages but it did"))
	}

	return stg
}

func TestKafkaStorage_Send(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping long test")
	}

	var (
		numMessages = 10
		stg         = getTestStorage()
		req         = require.New(t)
	)

	msgs := make([]storage.Message, 0, numMessages)
	for i := 0; i < numMessages; i++ {
		msg := storage.Message{
			Data:      randomBytes(10),
			Signature: randomBytes(10),
		}
		msgs = append(msgs, msg)
	}

	err := stg.Send(msgs...)
	req.NoError(err)

	offsetMsgs, err := stg.GetMessages(0)
	if err != nil {
		t.Error(err)
	}

	req.Len(offsetMsgs, len(msgs))
}

func randomBytes(n int) []byte {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil
	}
	return b
}
