package storage

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	kafkaEndpoint   = "94.130.57.249:9093"
	kafkaTopic      = "test_long"
	certificatePath = "../ca.crt"
)

var (
	kafkaAuthCredentials = &KafkaAuthCredentials{
		Username: "producer",
		Password: "producerpass",
	}
)

// kafkacat -C -b localhost -t messages
func TestKafkaStorage_GetMessages(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping long test")
	}

	N := 10
	var offset uint64 = 5

	tlsConfig, err := GetTLSConfig(certificatePath)
	if err != nil {
		t.Fatal(err.Error())
	}

	req := require.New(t)
	stg, err := NewKafkaStorage(
		context.Background(),
		kafkaEndpoint,
		kafkaTopic,
		tlsConfig,
		kafkaAuthCredentials,
		kafkaAuthCredentials,
		time.Second*10)
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
	tlsConfig, err := GetTLSConfig(certificatePath)
	if err != nil {
		t.Fatal(err.Error())
	}

	req := require.New(t)
	stg, err := NewKafkaStorage(
		context.Background(),
		kafkaEndpoint,
		kafkaTopic,
		tlsConfig,
		kafkaAuthCredentials,
		kafkaAuthCredentials,
		time.Second*10)
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

func TestKafkaStorage_Resets(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping long test")
	}

	tlsConfig, err := GetTLSConfig(certificatePath)
	if err != nil {
		t.Fatal(err.Error())
	}

	var testDuration = time.Minute * 5

	req := require.New(t)
	stg, err := NewKafkaStorage(
		context.Background(),
		kafkaEndpoint,
		kafkaTopic,
		tlsConfig,
		kafkaAuthCredentials,
		kafkaAuthCredentials,
		time.Second*10,
	)
	req.NoError(err)

	wg := &sync.WaitGroup{}
	wg.Add(2)

	tmWrite := time.NewTimer(testDuration)
	go func() {
		for {
			select {
			case <-tmWrite.C:
				wg.Done()
				return
			default:
				msgs := []Message{
					{
						Data:      randomBytes(500000),
						Signature: randomBytes(10),
					},
				}
				_, err := stg.SendBatch(msgs...)
				req.NoError(err)
				time.Sleep(time.Millisecond * 330)
			}
		}
	}()

	tmRead := time.NewTimer(testDuration)
	go func() {
		var offset uint64
		for {
			select {
			case <-tmRead.C:
				wg.Done()
				return
			default:
				msgs, err := stg.GetMessages(offset)
				require.NoError(t, err)
				if len(msgs) > 0 {
					offset = msgs[len(msgs)-1].Offset
				}
			}
		}
	}()

	wg.Wait()
}
