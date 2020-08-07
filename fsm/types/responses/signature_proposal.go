package responses

// Response

const (
	ProposalConfirmationStatusIdle = iota
	ProposalConfirmationStatusAccepted
	ProposalConfirmationStatusCanceled
	ProposalConfirmationStatusTimeout
)

// States: "validate_proposal"

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
	ParticipantId     int
	Title             string
	PubKeyFingerprint string
	Status            int
}
