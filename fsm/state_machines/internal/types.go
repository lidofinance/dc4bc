package internal

import "time"

type ProposalParticipantPrivate struct {
	// Public title for address, such as name, nickname, organization
	Title     string
	PublicKey []byte
	// For validation user confirmation: sign(InvitationSecret, PublicKey) => user
	InvitationSecret string
	ConfirmedAt      *time.Time
}

// Unique alias for map iteration - Public Key Fingerprint
// Excludes array merge and rotate operations

type ProposalConfirmationPrivateQuorum map[string]ProposalParticipantPrivate
