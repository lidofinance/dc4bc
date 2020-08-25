package responses

type SigningProposalParticipantInvitationsResponse struct {
	InitiatorId  int
	Participants []*SigningProposalParticipantInvitationEntry
	SigningId    string
}

type SigningProposalParticipantInvitationEntry struct {
	ParticipantId int
	Addr          string
}

type SigningProposalParticipantStatusResponse []*SignatureProposalParticipantStatusEntry

type SigningProposalParticipantStatusEntry struct {
	ParticipantId int
	Addr          string
	Status        uint8
}
