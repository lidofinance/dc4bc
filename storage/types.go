package storage

type Message struct {
	Data      []byte `json:"data"`
	Signature []byte `json:"signature"`
}

type Storage interface {
	Post(message Message) error
	GetMessages(offset int) ([]Message, error)
	Close() error
}
