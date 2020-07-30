package storage

type Message struct {
	Data      []byte `json:"data"`
	Signature []byte `json:"signature"`
	ID        string `json:"id"`
	Offset    uint64 `json:"offset"`
}

type Storage interface {
	Send(message Message) (Message, error)
	GetMessages(offset int) ([]Message, error)
	Close() error
}
