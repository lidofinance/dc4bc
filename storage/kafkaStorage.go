package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	kafkaTopic     = "messages"
	kafkaPartition = 0
)

type KafkaStorage struct {
	writer *kafka.Conn
	reader *kafka.Reader
}

func NewKafkaStorage(ctx context.Context, kafkaEndpoint string) (Storage, error) {
	conn, err := kafka.DialLeader(ctx, "tcp", kafkaEndpoint, kafkaTopic, kafkaPartition)
	if err != nil {
		return nil, fmt.Errorf("failed to init Kafka client: %w", err)
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{kafkaEndpoint},
		Topic:     kafkaTopic,
		Partition: kafkaPartition,
		MaxWait:   time.Second,
	})

	return &KafkaStorage{
		writer: conn,
		reader: reader,
	}, nil
}

func (s *KafkaStorage) Send(m Message) (Message, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return m, fmt.Errorf("failed to marshal a message %v: %v", m, err)
	}

	if err := s.writer.SetWriteDeadline(time.Now().Add(time.Second)); err != nil {
		return Message{}, fmt.Errorf("failed to SetWriteDeadline: %w", err)
	}

	if _, err := s.writer.WriteMessages(kafka.Message{Key: []byte(m.ID), Value: data}); err != nil {
		return Message{}, fmt.Errorf("failed to WriteMessages: %w", err)
	}

	return Message{}, nil
}

func (s *KafkaStorage) GetMessages(offset uint64) ([]Message, error) {
	if err := s.reader.SetOffset(int64(offset)); err != nil {
		return nil, fmt.Errorf("failed to SetOffset: %w", err)
	}

	lag, err := s.reader.ReadLag(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to ReadLag: %w", err)
	}
	var (
		message  Message
		messages []Message
		i        int64
	)
	for i = 0; i < lag; i++ {
		kafkaMessage, err := s.reader.ReadMessage(context.Background())
		if err != nil {
			break
		}

		if err = json.Unmarshal(kafkaMessage.Value, &message); err != nil {
			return nil, fmt.Errorf("failed to unmarshal a message %s: %v",
				string(kafkaMessage.Value), err)
		}

		message.Offset = uint64(kafkaMessage.Offset)
		messages = append(messages, message)
	}

	return messages, nil
}

func (s *KafkaStorage) Close() error {
	if err := s.reader.Close(); err != nil {
		return fmt.Errorf("failed to close reader: %w", err)
	}

	if err := s.writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	return nil
}
