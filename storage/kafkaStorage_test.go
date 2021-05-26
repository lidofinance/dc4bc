package storage

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// kafkacat -C -b localhost -t messages
func TestKafkaStorage_GetMessages(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping long test")
	}

	N := 10
	var offset uint64 = 5

	producerCreds := &KafkaAuthCredentials{
		Username: "producer",
		Password: "producerpass",
	}
	consumerCreds := &KafkaAuthCredentials{
		Username: "consumer",
		Password: "consumerpass",
	}

	tlsConfig, err := GetTLSConfig("../kafka-docker/certs/ca.crt")
	if err != nil {
		t.Fatal(err.Error())
	}

	req := require.New(t)
	stg, err := NewKafkaStorage(context.Background(), "localhost:9093", "test", tlsConfig, producerCreds, consumerCreds)
	req.NoError(err)

	msgs := make([]Message, 0, N)
	for i := 0; i < N; i++ {
		msg := Message{
			Data:      randomBytes(10),
			Signature: randomBytes(10),
		}
		msg, err = stg.Send(msg)
		if err != nil {
			t.Error(err)
		}
		msgs = append(msgs, msg)
	}

	offsetMsgs, err := stg.GetMessages(offset)
	if err != nil {
		t.Error(err)
	}

	expectedOffsetMsgs := msgs[offset:]

	for idx, msg := range expectedOffsetMsgs {
		req.Equal(msg.Signature, offsetMsgs[idx].Signature)
	}
}

func TestKafkaStorage_SendBatch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping long test")
	}

	N := 10
	var offset uint64 = 5

	producerCreds := &KafkaAuthCredentials{
		Username: "producer",
		Password: "producerpass",
	}
	consumerCreds := &KafkaAuthCredentials{
		Username: "consumer",
		Password: "consumerpass",
	}

	tlsConfig, err := GetTLSConfig("../kafka-docker/certs/ca.crt")
	if err != nil {
		t.Fatal(err.Error())
	}

	req := require.New(t)
	stg, err := NewKafkaStorage(context.Background(), "localhost:9093", "test", tlsConfig, producerCreds, consumerCreds)
	req.NoError(err)

	msgs := make([]Message, 0, N)
	for i := 0; i < N; i++ {
		msg := Message{
			Data:      randomBytes(10),
			Signature: randomBytes(10),
		}
		msgs = append(msgs, msg)
	}

	sentMsgs, err := stg.SendBatch(msgs...)
	req.NoError(err)

	offsetMsgs, err := stg.GetMessages(offset)
	if err != nil {
		t.Error(err)
	}

	expectedOffsetMsgs := sentMsgs[offset:]

	for idx, msg := range expectedOffsetMsgs {
		req.Equal(msg.Signature, offsetMsgs[idx].Signature)
	}
}

func TestKafkaStorage_SendResets(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping long test")
	}

	N := 10
	producerCreds := &KafkaAuthCredentials{
		Username: "producer",
		Password: "producerpass",
	}
	consumerCreds := &KafkaAuthCredentials{
		Username: "consumer",
		Password: "consumerpass",
	}

	tlsConfig, err := GetTLSConfig("../ca.crt")
	if err != nil {
		t.Fatal(err.Error())
	}

	req := require.New(t)
	stg, err := NewKafkaStorage(
		context.Background(),
		"94.130.57.249:9093",
		"test",
		tlsConfig,
		producerCreds,
		consumerCreds,
		time.Second*10,
	)
	req.NoError(err)

	for j := 0; j < 10; j++ {
		msgs := make([]Message, 0, N)

		msg := Message{
			Data:      randomBytes(500000),
			Signature: randomBytes(10),
		}
		msgs = append(msgs, msg)
		time.Sleep(time.Second * 10)

		_, err := stg.SendBatch(msgs...)
		req.NoError(err)
	}
}
