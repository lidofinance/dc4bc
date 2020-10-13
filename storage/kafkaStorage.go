package storage

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"golang.org/x/crypto/pkcs12"
	"io/ioutil"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	kafkaPartition = 0
)

type KafkaStorage struct {
	writer *kafka.Conn
	reader *kafka.Reader
}

type KafkaAuthCredentials struct {
	Username string
	Password string
}

func tlsConfig() *tls.Config {

	// Keystore
	keys, _ := ioutil.ReadFile("../kafka-docker/certs/client.p12")
	blocks, err := pkcs12.ToPEM(keys, "test1234")
	if err != nil {
		log.Fatal(err.Error())
	}

	var pemData []byte
	for test, b := range blocks {
		_ = test
		pemData = append(pemData, pem.EncodeToMemory(b)...)
	}

	cert, err := tls.X509KeyPair(pemData, pemData)
	if err != nil {
		log.Fatal(err.Error())
	}
	//Truststore
	caCert, err := ioutil.ReadFile("../kafka-docker/certs/ca.pem")
	if err != nil {
		log.Fatal(err)
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}
	return config
}

func NewKafkaStorage(ctx context.Context, kafkaEndpoint string, kafkaTopic string, producerCreds, ConsumerCreds *KafkaAuthCredentials) (Storage, error) {

	dialerProducer := &kafka.Dialer{
		Timeout:   10 * time.Second,
		DualStack: true,
		TLS:       tlsConfig(),
	}
	dialerConsumer := &kafka.Dialer{
		Timeout:   10 * time.Second,
		DualStack: true,
		TLS:       tlsConfig(),
	}
	conn, err := dialerProducer.DialLeader(ctx, "tcp", kafkaEndpoint, kafkaTopic, kafkaPartition)
	if err != nil {
		return nil, fmt.Errorf("failed to init Kafka client: %w", err)
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{kafkaEndpoint},
		Topic:     kafkaTopic,
		Partition: kafkaPartition,
		MaxWait:   time.Second,
		Dialer:    dialerConsumer,
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

	return m, nil
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
