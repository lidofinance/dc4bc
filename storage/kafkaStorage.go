package storage

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/segmentio/kafka-go/sasl/plain"

	"github.com/segmentio/kafka-go"
	"gopkg.in/matryer/try.v1"
)

const (
	kafkaPartition    = 0
	maxRetries        = 30
	reconnectInterval = time.Second
)

func init() {
	try.MaxRetries = maxRetries
}

type KafkaStorage struct {
	ctx    context.Context
	writer *kafka.Conn
	reader *kafka.Reader

	kafkaEndpoint string
	kafkaTopic    string

	producerCreds *KafkaAuthCredentials
	consumerCreds *KafkaAuthCredentials
	tlsConfig     *tls.Config
}

type KafkaAuthCredentials struct {
	Username string
	Password string
}

func GetTLSConfig(trustStorePath string) (*tls.Config, error) {
	caCert, err := ioutil.ReadFile(trustStorePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read trustStorePath: %w", err)
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	config := &tls.Config{
		RootCAs: caCertPool,
	}
	return config, nil
}

func NewKafkaStorage(ctx context.Context, kafkaEndpoint string, kafkaTopic string, tlsConfig *tls.Config, producerCreds, consumerCreds *KafkaAuthCredentials) (Storage, error) {
	stg := &KafkaStorage{
		ctx:           ctx,
		kafkaEndpoint: kafkaEndpoint,
		kafkaTopic:    kafkaTopic,
		tlsConfig:     tlsConfig,
		producerCreds: producerCreds,
		consumerCreds: consumerCreds,
	}

	if err := stg.connect(); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	return stg, nil
}

func (s *KafkaStorage) Send(m Message) (Message, error) {
	err := try.Do(func(attempt int) (bool, error) {
		var err error
		m, err = s.send(m)
		if err != nil {
			log.Printf("failed while trying to send message (%v), trying to reconnect", err)
			if err := s.connect(); err != nil {
				log.Printf("failed to reconnect (%v), %d retries left", err, try.MaxRetries-attempt)
			}
		}
		time.Sleep(reconnectInterval)

		return attempt < try.MaxRetries, err
	})

	return m, err
}

func (s *KafkaStorage) SendBatch(msgs ...Message) ([]Message, error) {
	err := try.Do(func(attempt int) (bool, error) {
		var err error
		msgs, err = s.sendBatch(msgs...)
		if err != nil {
			log.Printf("failed while trying to send message (%v), trying to reconnect", err)
			if err := s.connect(); err != nil {
				log.Printf("failed to reconnect (%v), %d retries left", err, try.MaxRetries-attempt)
			}
		}
		time.Sleep(reconnectInterval)

		return attempt < try.MaxRetries, err
	})

	return msgs, err
}

func (s *KafkaStorage) send(m Message) (Message, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return m, fmt.Errorf("failed to marshal a message %v: %v", m, err)
	}

	if err := s.writer.SetWriteDeadline(time.Now().Add(time.Second)); err != nil {
		return m, fmt.Errorf("failed to SetWriteDeadline: %w", err)
	}

	if _, err := s.writer.WriteMessages(kafka.Message{Key: []byte(m.ID), Value: data}); err != nil {
		return m, fmt.Errorf("failed to WriteMessages: %w", err)
	}

	return m, nil
}

func (s *KafkaStorage) sendBatch(msgs ...Message) ([]Message, error) {
	kafkaMessages := make([]kafka.Message, len(msgs))
	for i, m := range msgs {
		data, err := json.Marshal(m)
		if err != nil {
			return msgs, fmt.Errorf("failed to marshal a message %v: %v", m, err)
		}
		kafkaMessages[i] = kafka.Message{Key: []byte(m.ID), Value: data}
	}

	if err := s.writer.SetWriteDeadline(time.Now().Add(time.Second)); err != nil {
		return msgs, fmt.Errorf("failed to SetWriteDeadline: %w", err)
	}

	if _, err := s.writer.WriteMessages(kafkaMessages...); err != nil {
		return msgs, fmt.Errorf("failed to WriteMessages: %w", err)
	}

	return msgs, nil
}

func (s *KafkaStorage) GetMessages(offset uint64) (messages []Message, err error) {
	err = try.Do(func(attempt int) (bool, error) {
		var err error
		messages, err = s.getMessages(offset)
		if err != nil {
			log.Printf("failed while trying to getMessages (%v), trying to reconnect", err)
			if err := s.connect(); err != nil {
				log.Printf("failed to reconnect (%v), %d retries left", err, try.MaxRetries-attempt)
			}
		}
		time.Sleep(reconnectInterval)

		return attempt < try.MaxRetries, err
	})

	return messages, err
}

func (s *KafkaStorage) getMessages(offset uint64) ([]Message, error) {
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
	if s.writer != nil {
		if err := s.writer.Close(); err != nil {
			return fmt.Errorf("failed to close writer: %w", err)
		}
	}

	if s.reader != nil {
		if err := s.reader.Close(); err != nil {
			return fmt.Errorf("failed to close reader: %w", err)
		}
	}

	return nil
}

func (s *KafkaStorage) connect() error {
	_ = s.Close()

	mechanismProducer := plain.Mechanism{s.producerCreds.Username, s.producerCreds.Password}
	mechanismConsumer := plain.Mechanism{s.consumerCreds.Username, s.consumerCreds.Password}

	dialerProducer := &kafka.Dialer{
		Timeout:       10 * time.Second,
		DualStack:     true,
		TLS:           s.tlsConfig,
		SASLMechanism: mechanismProducer,
	}
	dialerConsumer := &kafka.Dialer{
		Timeout:       10 * time.Second,
		DualStack:     true,
		TLS:           s.tlsConfig,
		SASLMechanism: mechanismConsumer,
	}

	conn, err := dialerProducer.DialLeader(s.ctx, "tcp", s.kafkaEndpoint, s.kafkaTopic, kafkaPartition)
	if err != nil {
		return fmt.Errorf("failed to init Kafka client: %w", err)
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{s.kafkaEndpoint},
		Topic:     s.kafkaTopic,
		Partition: kafkaPartition,
		MaxWait:   time.Second,
		Dialer:    dialerConsumer,
	})

	s.writer, s.reader = conn, reader

	return nil
}
