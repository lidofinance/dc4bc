package storage

import (
	"bytes"
	"crypto/ed25519"
)

type Message struct {
	ID            string `json:"id"`
	DkgRoundID    string `json:"dkg_round_id"`
	Offset        uint64 `json:"offset"`
	Event         string `json:"event"`
	Data          []byte `json:"data"`
	Signature     []byte `json:"signature"`
	SenderAddr    string `json:"sender"`
	RecipientAddr string `json:"recipient"`
}

func (m *Message) Bytes() []byte {
	buf := bytes.NewBuffer(nil)
	buf.Write(m.Data)

	return buf.Bytes()
}

func (m *Message) Verify(pubKey ed25519.PublicKey) bool {
	return ed25519.Verify(pubKey, m.Bytes(), m.Signature)
}

type Storage interface {
	Send(message Message) (Message, error)
	SendBatch(messages ...Message) ([]Message, error) //expected to be an atomic operation
	GetMessages(offset uint64) ([]Message, error)
	Close() error
}
