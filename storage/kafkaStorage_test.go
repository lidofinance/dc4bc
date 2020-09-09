package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// kafkacat -C -b localhost -t messages
//func TestKafkaStorage_Send(t *testing.T) {
//	if testing.Short() {
//		t.Skip("skipping long test")
//	}
//
//	req := require.New(t)
//	stg, err := NewKafkaStorage(context.Background(), "localhost:9092")
//	req.NoError(err)
//
//	t.Run("test_kafka_storage_send", func(t *testing.T) {
//		_, err = stg.Send(Message{
//			ID:            "test_message_id",
//			DkgRoundID:    "test_dkg_round_id",
//			Event:         "test_event",
//			Data:          []byte("test_data"),
//			Signature:     []byte("test_signature"),
//			SenderAddr:    "test_sender_addr",
//			RecipientAddr: "test_recipient_addr",
//		})
//
//		req.NoError(err)
//	})
//}

func TestKafkaStorage_GetMessages(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping long test")
	}

	N := 10
	var offset uint64 = 5

	req := require.New(t)
	stg, err := NewKafkaStorage(context.Background(), "localhost:9092")
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
