package requests

import "time"

// Requests

// States: "__idle"
// Events: "event_sig_proposal_init"
type SignatureProposalParticipantsListRequest struct {
	Participants []*SignatureProposalParticipantsEntry
	CreatedAt    *time.Time
}

type SignatureProposalParticipantsEntry struct {
	// Public title for address, such as name, nickname, organization
	Title     string
	PublicKey []byte
}

// States: "__idle"
// Events: "event_sig_proposal_confirm_by_participant"
// 		   "event_sig_proposal_decline_by_participant"
type SignatureProposalParticipantRequest struct {
	// Key for link invitations to participants
	PubKeyFingerprint   string
	EncryptedInvitation string
	CreatedAt           *time.Time
}
