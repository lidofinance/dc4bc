package storage

import (
	"context"
	"testing"

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
	req := require.New(t)
	stg, err := NewKafkaStorage(context.Background(), "localhost:9093", "test_topic", producerCreds, consumerCreds)
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
