package client

import (
	"bytes"
	"fmt"
	"time"
)

type OperationType string

const (
	DKGCommits OperationType = "dkg_commits"
)

type Operation struct {
	ID        string // UUID4
	Type      OperationType
	Payload   []byte
	Result    []byte
	CreatedAt time.Time
}

func (o *Operation) Check(o2 *Operation) error {
	if o.ID != o2.ID {
		return fmt.Errorf("o1.ID (%s) != o2.ID (%s)", o.ID, o2.ID)
	}

	if o.Type != o2.Type {
		return fmt.Errorf("o1.Type (%s) != o2.Type (%s)", o.Type, o2.Type)
	}

	if !bytes.Equal(o.Payload, o2.Payload) {
		return fmt.Errorf("o1.Payload (%v) != o2.Payload (%v)", o.Payload, o2.Payload)
	}

	return nil
}
