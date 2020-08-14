package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/depools/dc4bc/fsm/fsm"
	"time"
)

type OperationType string

const (
	DKGCommits OperationType = "dkg_commits"
)

type Operation struct {
	ID            string // UUID4
	Type          OperationType
	Payload       []byte
	Result        []byte
	CreatedAt     time.Time
	DKGIdentifier string
	To            string
	Event         fsm.Event
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

type FSMRequest struct {
	Event fsm.Event
	Args  []interface{}
}

func FSMRequestFromBytes(data []byte) (FSMRequest, error) {
	var (
		r   FSMRequest
		err error
	)
	if err = json.Unmarshal(data, &r); err != nil {
		return r, err
	}
	return r, err
}

func FSMRequestToBytes(event fsm.Event, req interface{}) ([]byte, error) {
	fsmReq := FSMRequest{
		Event: event,
		Args:  []interface{}{req},
	}
	return json.Marshal(fsmReq)
}
