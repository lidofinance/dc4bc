package storage

import (
	"bytes"
	"crypto/ed25519"
)

type Message struct {
	ID        string `json:"id"`
	Offset    uint64 `json:"offset"`
	Event     string `json:"event"`
	Data      []byte `json:"data"`
	Signature []byte `json:"signature"`
	Sender    string `json:"sender"`
}

func (m *Message) Bytes() []byte {
	buf := bytes.NewBuffer(nil)

	buf.Write([]byte(m.Sender))
	buf.Write([]byte(m.Event))
	buf.Write(m.Data)

	return buf.Bytes()
}

func (m *Message) Verify(pubKey ed25519.PublicKey) bool {
	return ed25519.Verify(pubKey, m.Bytes(), m.Signature)
}

type Storage interface {
	Send(message Message) (Message, error)
	GetMessages(offset uint64) ([]Message, error)
	Close() error
}
