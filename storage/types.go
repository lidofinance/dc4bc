package storage

import "time"

type Message struct {
	To        string    `json:"to"`
	Data      []byte    `json:"data"`
	Signature []byte    `json:"signature"`
	ID        string    `json:"id"`
	Offset    uint64    `json:"offset"`
	CreatedAt time.Time `json:"created_at"`
}

type Storage interface {
	Send(message Message) (Message, error)
	GetMessages(offset uint64) ([]Message, error)
	Close() error
}
