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
	ParticipantId int
	CreatedAt     time.Time
}

// States: "state_signing_await_partial_keys"
// Events: "event_signing_partial_key_received"
type SigningProposalPartialKeyRequest struct {
	ParticipantId int
	PartialKey    []byte
	CreatedAt     time.Time
}
