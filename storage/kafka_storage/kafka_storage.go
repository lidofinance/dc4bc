package kafka_storage

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"

	"github.com/lidofinance/dc4bc/client/config"
	"github.com/lidofinance/dc4bc/storage"
)

const (
	kafkaMinBytes    = 10
	kafkaMaxBytes    = 10e6
	kafkaMaxAttempts = 16
	kafkaBatchBytes  = 10e6
)

type KafkaAuthCredentials struct {
	Username string
	Password string
}

type KafkaStorage struct {
	readerCtx                            context.Context
	readerCtxCancel                      context.CancelFunc
	readDuration                         time.Duration
	reader                               *kafka.Reader
	writer                               *kafka.Writer
	tlsConfig                            *tls.Config
	producerCreds, consumerCreds         *plain.Mechanism
	brokerEndpoint, consumerGroup, topic string
	timeout                              time.Duration

	idIgnoreList     map[string]struct{}
	offsetIgnoreList map[uint64]struct{}
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

func NewKafkaStorage(cfg *config.KafkaStorageConfig) (*KafkaStorage, error) {
	if cfg == nil {
		return nil, errors.New("kafka cfg should not be nil value")
	}

	producerCreds, err := parseKafkaSaslPlain(cfg.ProducerCredentials)
	if err != nil {
		return nil, fmt.Errorf("failed to parse producer credentials: %w", err)
	}

	consumerCreds, err := parseKafkaSaslPlain(cfg.ConsumerCredentials)
	if err != nil {
		return nil, fmt.Errorf("failed to parse consumer credentials: %w", err)
	}

	tlsConfig, err := GetTLSConfig(cfg.TlsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to init tsl config: %w", err)
	}

	readDuration, err := time.ParseDuration(cfg.ReadDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to parse read duration: %w", err)
	}

	timeout, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timeout duration: %w", err)
	}

	ks := &KafkaStorage{
		readDuration:   readDuration,
		brokerEndpoint: cfg.DBDSN,
		topic:          cfg.Topic,
		consumerGroup:  cfg.ConsumerGroup,
		tlsConfig:      tlsConfig,
		producerCreds:  producerCreds,
		consumerCreds:  consumerCreds,
		timeout:        timeout,

		idIgnoreList:     map[string]struct{}{},
		offsetIgnoreList: map[uint64]struct{}{},
	}
	if err := ks.reset(); err != nil {
		return nil, fmt.Errorf("failed to create a NewKafkaStorage: %w", err)
	}

	return ks, nil
}

func (ks *KafkaStorage) Close() error {
	if ks.reader != nil {
		ks.readerCtxCancel()
		if err := ks.reader.Close(); err != nil {
			return fmt.Errorf("failed to Close reader: %w", err)
		}
	}

	if ks.writer != nil {
		if err := ks.writer.Close(); err != nil {
			return fmt.Errorf("failed to Close writer: %w", err)
		}
	}

	return nil
}

func (ks *KafkaStorage) Send(messages ...storage.Message) error {
	kafkaMessages, err := ks.storageToKafkaMessages(messages...)
	if err != nil {
		return fmt.Errorf("failed to storageToKafkaMessages: %w", err)
	}

	if err := ks.writer.WriteMessages(context.Background(), kafkaMessages...); err != nil {
		return fmt.Errorf("failed to WriteMessages: %w", err)
	}

	return nil
}

func (ks *KafkaStorage) GetMessages(_ uint64) ([]storage.Message, error) {
	ctx, cancel := context.WithDeadline(ks.readerCtx, time.Now().Add(ks.readDuration))
	defer cancel()

	var (
		message  storage.Message
		messages []storage.Message
	)
	for {
		kafkaMessage, err := ks.reader.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				break
			} else {
				return nil, fmt.Errorf("failed to ReadMessage: %w", err)
			}
		}

		if err = json.Unmarshal(kafkaMessage.Value, &message); err != nil {
			log.Printf("failed to unmarshal a message %s: %s", string(kafkaMessage.Value), err.Error())
			continue
		}

		message.Offset = uint64(kafkaMessage.Offset)

		_, idOk := ks.idIgnoreList[message.ID]
		_, offsetOk := ks.offsetIgnoreList[message.Offset]
		if !idOk && !offsetOk {
			messages = append(messages, message)
		}
	}

	return messages, nil
}

func (ks *KafkaStorage) IgnoreMessages(messages []string, useOffset bool) error {
	for _, msg := range messages {
		if useOffset {
			offset, err := strconv.ParseUint(msg, 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse message offset: %v", err)
			}
			ks.offsetIgnoreList[offset] = struct{}{}

			continue
		}

		ks.idIgnoreList[msg] = struct{}{}
	}

	return nil
}

func (ks *KafkaStorage) UnignoreMessages() {
	ks.idIgnoreList = map[string]struct{}{}
	ks.offsetIgnoreList = map[uint64]struct{}{}
}

func (ks *KafkaStorage) SetConsumerGroup(cg string) error {
	ks.consumerGroup = cg
	if err := ks.reset(); err != nil {
		return fmt.Errorf("failed to reset kafka storage after setting consumer group: %w", err)
	}

	return nil
}

func (ks *KafkaStorage) storageToKafkaMessages(messages ...storage.Message) ([]kafka.Message, error) {
	kafkaMessages := make([]kafka.Message, len(messages))
	for i, m := range messages {
		data, err := json.Marshal(m)
		if err != nil {
			return kafkaMessages, fmt.Errorf("failed to marshal a message %v: %v", m, err)
		}
		kafkaMessages[i] = kafka.Message{Key: []byte(m.ID), Value: data}
	}

	return kafkaMessages, nil
}

func (ks *KafkaStorage) reset() error {
	if err := ks.Close(); err != nil {
		return fmt.Errorf("failed to Close connections: %w", err)
	}

	ks.reader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{ks.brokerEndpoint},
		GroupID:     ks.consumerGroup,
		Topic:       ks.topic,
		MinBytes:    kafkaMinBytes,
		MaxBytes:    kafkaMaxBytes,
		MaxAttempts: kafkaMaxAttempts,
		Dialer: &kafka.Dialer{
			Timeout:       ks.timeout,
			DualStack:     true,
			TLS:           ks.tlsConfig,
			SASLMechanism: ks.consumerCreds,
		},
	})
	ks.readerCtx, ks.readerCtxCancel = context.WithCancel(context.Background())

	kafka.DefaultTransport = &kafka.Transport{
		Dial: (&net.Dialer{
			Timeout: ks.timeout,
		}).DialContext,
		TLS:  ks.tlsConfig,
		SASL: ks.producerCreds,
	}
	ks.writer = &kafka.Writer{
		Addr:         kafka.TCP(ks.brokerEndpoint),
		Topic:        ks.topic,
		Balancer:     &kafka.LeastBytes{},
		MaxAttempts:  kafkaMaxAttempts,
		BatchBytes:   kafkaBatchBytes,
		BatchTimeout: ks.timeout,
		ReadTimeout:  ks.timeout,
		WriteTimeout: ks.timeout,
	}

	return nil
}
