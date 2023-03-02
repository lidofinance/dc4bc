package dto

import (
	"time"

	"github.com/lidofinance/dc4bc/fsm/fsm"
	"github.com/lidofinance/dc4bc/storage"
)

// This packages contains DTO (Data Transfer Object) structures
// for providing validated and sanitized values to service layer

type MessageDTO struct {
	ID            string
	DkgRoundID    string
	Offset        uint64
	Event         string
	Data          []byte
	Signature     []byte
	SenderAddr    string
	RecipientAddr string
}

type OperationIdDTO struct {
	OperationID string
}

type DkgIdDTO struct {
	DkgID string
}

type SignatureByIdDTO struct {
	ID    string
	DkgID string
}

type SignaturesByBatchIdDTO struct {
	BatchID string
	DkgID   string
}

type OperationDTO struct {
	ID         string // UUID4
	Type       string
	Payload    []byte
	ResultMsgs []storage.Message
	CreatedAt  time.Time
	DkgID      string
	To         string
	Event      fsm.Event

	ExtraData []byte
}

type StartDkgDTO struct {
	Payload []byte
}

type ProposeSignMessageDTO struct {
	DkgID []byte
	Data  []byte
}

type Range struct {
	Start int
	End   int
}

type ProposeSignBatchMessagesDTO struct {
	DkgID []byte
	Data  map[string][]byte // use messageID as key
	Range *Range
}

type ProposeSignBakedMessagesDTO struct {
	DkgID      []byte
	RangeStart int
	RangeEnd   int
}

type ReInitDKGDTO struct {
	ID      string
	Payload []byte
}

type StateOffsetDTO struct {
	Offset uint64
}

type ResetStateDTO struct {
	NewStateDBDSN      string
	UseOffset          bool
	KafkaConsumerGroup string
	Messages           []string
}
