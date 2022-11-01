package requests

import (
	"time"

	"github.com/lidofinance/dc4bc/client/types"
	"github.com/lidofinance/dc4bc/fsm/fsm"
	"github.com/lidofinance/dc4bc/storage"
)

type MessageForm struct {
	ID            string `json:"id" validate:"attr=id,min=32,max=512"`
	DkgRoundID    string `json:"dkg_round_id" validate:"attr=dkg_round_id,min=32,max=512"`
	Offset        uint64 `json:"offset" validate:"attr=offset,min=0"`
	Event         string `json:"event" validate:"attr=event,min=1s"`
	Data          []byte `json:"data" validate:"attr=data,min=1"`
	Signature     []byte `json:"signature" validate:"attr=signature,min=1"`
	SenderAddr    string `json:"sender"  validate:"attr=signature,min=1"`
	RecipientAddr string `json:"recipient"`
}

type OperationIdForm struct {
	OperationID string `query:"operationID" json:"operationID" validate:"attr=operationID,min=32,max=512"`
}

type DkgIdForm struct {
	DkgID string `query:"dkgID" json:"dkgID" validate:"attr=dkgID,min=32,max=512"`
}

type SignatureByIDForm struct {
	ID    string `query:"id" json:"id" validate:"attr=id,max=512"`
	DkgID string `query:"dkgID" json:"dkgID" validate:"attr=dkgID,min=32,max=512"`
}

type SignaturesByBatchIDForm struct {
	BatchID string `query:"batchID" json:"batchID" validate:"attr=id,min=32,max=512"`
	DkgID   string `query:"dkgID" json:"dkgID" validate:"attr=dkgID,min=32,max=512"`
}

type OperationForm struct {
	ID         string            `json:"ID" validate:"attr=ID,min=32,max=512"` // UUID4
	Type       string            `json:"Type" validate:"attr=Type,min=1,max=512"`
	Payload    []byte            `json:"Payload"`
	ResultMsgs []storage.Message `json:"ResultMsgs"`
	CreatedAt  time.Time         `json:"CreatedAt"`
	DkgID      string            `json:"DKGIdentifier" validate:"attr=DKGIdentifier,min=32,max=512"`
	To         string            `json:"To" validate:"attr=To,min=0"`
	Event      fsm.Event         `json:"Event" validate:"attr=Event,min=1"`

	ExtraData []byte `json:"ExtraData"`
}

type StartDKGForm struct {
	Payload []byte
}

type ProposeSignMessageForm struct {
	DkgID []byte `json:"dkgID"`
	Data  []byte `json:"data"`
}

type ProposeSignBatchMessagesForm struct {
	DkgID []byte            `json:"dkgID"`
	Data  map[string][]byte `json:"data"`
}

type ProposeSignBakedMessagesForm struct {
	DkgID      []byte `json:"dkgID"`
	RangeStart int    `json:"range_start" validate:"attr=range_start,min=0"`
	RangeEnd   int    `json:"range_end"`
}

type ReInitDKGForm struct {
	ID           string              `json:"dkg_id"`
	Threshold    int                 `json:"threshold"`
	Participants []types.Participant `json:"participants"`
	Messages     []storage.Message   `json:"messages"`
}

type StateOffsetForm struct {
	Offset uint64 `json:"offset"`
}

type ResetStateForm struct {
	NewStateDBDSN      string   `json:"new_state_dbdsn,omitempty"`
	UseOffset          bool     `json:"use_offset"`
	KafkaConsumerGroup string   `json:"kafka_consumer_group"`
	Messages           []string `json:"messages,omitempty"`
}
