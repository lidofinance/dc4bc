package requests

import (
	"encoding/json"
	"time"
)

// States: "state_dkg_commits_sending_await_confirmations"
// Events: "event_dkg_commit_confirm_received"
type DKGProposalCommitConfirmationRequest struct {
	ParticipantId int
	Commit        []byte
	CreatedAt     time.Time
}

// States: "state_dkg_deals_await_confirmations"
// Events: "event_dkg_deal_confirm_received"
type DKGProposalDealConfirmationRequest struct {
	ParticipantId int
	Deal          []byte
	CreatedAt     time.Time
}

// States: "state_dkg_responses_await_confirmations"
// Events: "event_dkg_response_confirm_received"
type DKGProposalResponseConfirmationRequest struct {
	ParticipantId int
	Response      []byte
	CreatedAt     time.Time
}

// States: "state_dkg_master_key_await_confirmations"
// Events: "event_dkg_master_key_confirm_received"
type DKGProposalMasterKeyConfirmationRequest struct {
	ParticipantId int
	MasterKey     []byte
	CreatedAt     time.Time
}

// States:  "state_dkg_pub_keys_await_confirmations"
// 			"state_dkg_commits_sending_await_confirmations"
//			"state_dkg_deals_await_confirmations"
//			"state_dkg_responses_await_confirmations"
// 			"state_dkg_master_key_await_confirmations"
//
// Events:  "event_dkg_pub_key_confirm_canceled_by_error",
//			"event_dkg_commit_confirm_canceled_by_error"
//			"event_dkg_deal_confirm_canceled_by_error"
// 			"event_dkg_response_confirm_canceled_by_error"
//			"event_dkg_master_key_confirm_canceled_by_error"
type DKGProposalConfirmationErrorRequest struct {
	ParticipantId int
	Error         *FSMError
	CreatedAt     time.Time
}

type FSMError struct {
	ErrorMsg string
}

func NewFSMError(err error) *FSMError {
	return &FSMError{
		ErrorMsg: err.Error(),
	}
}

func (re FSMError) Error() string {
	return re.ErrorMsg
}

func (re FSMError) MarshalJSON() ([]byte, error) {
	return json.Marshal(re.ErrorMsg)
}

func (re *FSMError) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &re.ErrorMsg)
}
