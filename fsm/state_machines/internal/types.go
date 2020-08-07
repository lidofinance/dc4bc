package internal

import "time"

type ProposalParticipantPrivate struct {
	// Public title for address, such as name, nickname, organization
	ParticipantId int
	Title         string
	PublicKey     []byte
	// For validation user confirmation: sign(InvitationSecret, PublicKey) => user
	InvitationSecret string
	ConfirmedAt      *time.Time
}

// Unique alias for map iteration - Public Key Fingerprint
// Excludes array merge and rotate operations

type ConfirmationProposalPrivateQuorum map[string]ProposalParticipantPrivate

type ProposalDKGParticipantPrivate struct {
	Title     string
	PublicKey []byte
	Commit    []byte
	Deal      []byte
	UpdatedAt *time.Time
}

type DKGProposalPrivateQuorum map[int]ProposalParticipantPrivate
