package storage

type Message struct {
	Offset    uint64
	Data      []byte `json:"data"`
	Signature []byte `json:"signature"`
}

type Storage interface {
	Post(message Message) error
	GetMessages(offset uint64) ([]Message, error)
	Close() error
}
