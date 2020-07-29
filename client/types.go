package client

type OperationType int

const (
	DKGGetCommits OperationType = iota
)

type Operation struct {
	ID   string // UUID4
	Type OperationType
	Data []byte
}
