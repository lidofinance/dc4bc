package responses

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

type SigningProposalParticipantStatusResponse struct {
	SigningId    string
	Participants []*SignatureProposalParticipantStatusEntry
}

type SigningProposalParticipantStatusEntry struct {
	ParticipantId int
	Addr          string
	Status        uint8
}

type SigningProcessParticipantResponse struct {
	SigningId    string
	SrcPayload   []byte
	Participants []*SigningProcessParticipantEntry
}

type SigningProcessParticipantEntry struct {
	ParticipantId int
	Addr          string
	PartialKey    []byte
}
