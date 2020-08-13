package responses

// Response

// Event: "event_sig_proposal_init"
// States: "__idle"

type SignatureProposalParticipantInvitationsResponse []*SignatureProposalParticipantInvitationEntry

type SignatureProposalParticipantInvitationEntry struct {
	ParticipantId int
	// Public title for address, such as name, nickname, organization
	Title string
	// Key for link invitations to participants
	PubKeyFingerprint string
	// Encrypted with public key secret
	EncryptedInvitation string
}

// Public lists for proposal confirmation process
// States: "validation_canceled_by_participant", "validation_canceled_by_timeout",
type SignatureProposalParticipantStatusResponse []*SignatureProposalParticipantStatusEntry

type SignatureProposalParticipantStatusEntry struct {
	ParticipantId int
	Title         string
	DkgPubKey     []byte
	Status        uint8
}
