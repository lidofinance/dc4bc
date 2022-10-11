package requests

import "time"

// MessageToSign is a message to sign on airgapped machine.
// It can either contains a contant as Payload or a range of baked into airgapped messages
type MessageToSign struct {
	MessageID string
	File      string
	Payload   []byte
}

type SigningTask struct {
	MessageID  string
	File       string
	Payload    []byte
	RangeStart int
	RangeEnd   int
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
	MessageID string
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
