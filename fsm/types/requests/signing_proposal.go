package requests

import "time"

// States: "stage_signing_idle"
// Events: "event_signing_start"
type SigningProposalStartRequest struct {
	ParticipantId int
	SrcPayload    []byte
	CreatedAt     time.Time
}

// States: "state_signing_await_confirmations"
// Events: "event_signing_proposal_confirm_by_participant"
//		   "event_signing_proposal_decline_by_participant"
type SigningProposalParticipantRequest struct {
	SigningId     string
	ParticipantId int
	CreatedAt     time.Time
}

// States: "state_signing_await_partial_keys"
// Events: "event_signing_partial_key_received"
type SigningProposalPartialSignRequest struct {
	SigningId     string
	ParticipantId int
	PartialSign   []byte
	CreatedAt     time.Time
}
