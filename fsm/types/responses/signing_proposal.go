package responses

// Event:  "event_signing_start"
// States: "state_signing_await_confirmations"
type SigningProposalParticipantInvitationsResponse struct {
	SigningId    string
	InitiatorId  int
	Participants []*SigningProposalParticipantInvitationEntry
	// Source message for signing
	SrcPayload []byte
}

type SigningProposalParticipantInvitationEntry struct {
	ParticipantId int
	Addr          string
	Status        uint8
}

// Event:  "event_signing_proposal_confirm_by_participant"
// States: "state_signing_await_partial_keys"
type SigningPartialSignsParticipantInvitationsResponse struct {
	SigningId   string
	InitiatorId int
	SrcPayload  []byte
}

// Event:  ""
// States: ""
type SigningProposalParticipantStatusResponse struct {
	SigningId    string
	Participants []*SignatureProposalParticipantStatusEntry
}

type SigningProposalParticipantStatusEntry struct {
	ParticipantId int
	Addr          string
	Status        uint8
}

// Event:  ""
// States: ""
type SigningProcessParticipantResponse struct {
	SigningId    string
	SrcPayload   []byte
	Participants []*SigningProcessParticipantEntry
}

type SigningProcessParticipantEntry struct {
	ParticipantId int
	Addr          string
	PartialSign   []byte
}
