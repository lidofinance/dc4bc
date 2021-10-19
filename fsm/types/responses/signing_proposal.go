package responses

// Event:  "event_signing_start"
// States: "state_signing_await_partial_keys"
type SigningPartialSignsParticipantInvitationsResponse struct {
	BatchID      string
	InitiatorId  int
	Participants []*SigningPartialSignsParticipantInvitationEntry
	// Source message for signing
	SrcPayload []byte
}

type SigningPartialSignsParticipantInvitationEntry struct {
	ParticipantId int
	Username      string
	Status        uint8
}

// Event:  "event_signing_partial_key_received"
// States: "state_signing_partial_signatures_collected"
type SigningProcessParticipantResponse struct {
	BatchID      string
	SrcPayload   []byte
	Participants []*SigningProcessParticipantEntry
}

type SigningProcessParticipantEntry struct {
	ParticipantId int
	Username      string
	PartialSigns  map[string][]byte
}
