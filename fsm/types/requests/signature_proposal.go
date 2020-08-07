package requests

// Requests

type SignatureProposalParticipantsListRequest []SignatureProposalParticipantsEntry

type SignatureProposalParticipantsEntry struct {
	// Public title for address, such as name, nickname, organization
	Title     string
	PublicKey []byte
}

type SignatureProposalParticipantRequest struct {
	// Key for link invitations to participants
	PubKeyFingerprint   string
	EncryptedInvitation string
}
