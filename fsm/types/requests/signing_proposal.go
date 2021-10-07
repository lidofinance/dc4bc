package requests

import "time"

// States: "state_signing_await_confirmations"
// Events: "event_signing_proposal_confirm_by_participant"
//		   "event_signing_proposal_decline_by_participant"
type SigningProposalParticipantRequest struct {
	BatchID       string
	ParticipantId int
	CreatedAt     time.Time
}

type MessageToSign struct {
	SigningID string
	Payload   []byte
}

// States: "stage_signing_idle"
// Events: "event_signing_start_batch"
type SigningBatchProposalStartRequest struct {
	BatchID        string
	ParticipantId  int
	CreatedAt      time.Time
	MessagesToSign []MessageToSign
}

type PartialSign struct {
	SigningID string
	Sign      []byte
}

// States: "state_signing_await_partial_signs"
// Events: "event_signing_partial_sign_received"
type SigningProposalBatchPartialSignRequests struct {
	BatchID       string
	ParticipantId int
	PartialSigns  []PartialSign
	CreatedAt     time.Time
}
