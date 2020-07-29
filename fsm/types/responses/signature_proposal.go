package responses

// Responses

type ProposalParticipantInvitationsResponse []*ProposalParticipantInvitationEntryResponse

type ProposalParticipantInvitationEntryResponse struct {
	// Public title for address, such as name, nickname, organization
	Title string
	// Key for link invitations to participants
	PubKeyFingerprint string
	// Encrypted with public key secret
	EncryptedInvitation string
}
