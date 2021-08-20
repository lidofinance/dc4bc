package requests

import (
	"github.com/lidofinance/dc4bc/client/types"
	"github.com/lidofinance/dc4bc/fsm/fsm"
	"github.com/lidofinance/dc4bc/storage"
	"time"
)

type MessageForm struct {
	ID            string `json:"id"`
	DkgRoundID    string `json:"dkg_round_id" validate:"attr=dkg_round_id,min=3"`
	Offset        uint64 `json:"offset"`
	Event         string `json:"event"`
	Data          []byte `json:"data"`
	Signature     []byte `json:"signature"`
	SenderAddr    string `json:"sender"`
	RecipientAddr string `json:"recipient"`
}

type OperationIdForm struct {
	OperationID string `query:"operationID" json:"operationID"`
}

type DkgIdForm struct {
	DkgID string `query:"dkgID" json:"dkgID"`
}

type SignatureByIDForm struct {
	ID    string `query:"id" json:"id"`
	DkgID string `query:"dkgID" json:"dkgID"`
}

type OperationForm struct {
	ID            string // UUID4
	Type          string
	Payload       []byte
	ResultMsgs    []storage.Message
	CreatedAt     time.Time
	DKGIdentifier string
	To            string
	Event         fsm.Event

	ExtraData []byte
}

type StartDKGForm struct {
	Payload []byte
}

type ProposeSignDataForm struct {
	DkgId string `json:"dkgID"`
	Data  []byte `json:"data"`
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
