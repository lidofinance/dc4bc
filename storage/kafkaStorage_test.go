package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// kafkacat -C -b localhost -t messages
func TestKafkaStorage_Send(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping long test")
	}

	req := require.New(t)
	stg, err := NewKafkaStorage(context.Background(), "localhost:9092")
	req.NoError(err)

	t.Run("test_kafka_storage_send", func(t *testing.T) {
		_, err = stg.Send(Message{
			ID:            "test_message_id",
			DkgRoundID:    "test_dkg_round_id",
			Event:         "test_event",
			Data:          []byte("test_data"),
			Signature:     []byte("test_signature"),
			SenderAddr:    "test_sender_addr",
			RecipientAddr: "test_recipient_addr",
		})

		req.NoError(err)
	})

	t.Run("test_kafka_storage_get_messages", func(t *testing.T) {
		msgs, err := stg.GetMessages(0)

		req.NoError(err)
		req.Len(msgs, 1)
	})
}
