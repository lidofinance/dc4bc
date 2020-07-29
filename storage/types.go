package storage

type Message struct {
	Offset    int64
	Data      []byte `json:"data"`
	Signature []byte `json:"signature"`
}

type Storage interface {
	Post(message Message) error
	GetMessages(offset int) ([]Message, error)
	Close() error
}
