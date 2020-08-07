package internal

import "time"

const (
	SignatureAwaitConfirmation SignatureProposalParticipantStatus = iota
	SignatureConfirmed
)

type ConfirmationProposal struct {
	Quorum    SignatureProposalQuorum
	CreatedAt *time.Time
	ExpiresAt *time.Time
}

type SignatureProposalParticipant struct {
	// Public title for address, such as name, nickname, organization
	ParticipantId int
	Title         string
	PublicKey     []byte
	// For validation user confirmation: sign(InvitationSecret, PublicKey) => user
	InvitationSecret string
	Status           SignatureProposalParticipantStatus
	UpdatedAt        *time.Time
}

// Unique alias for map iteration - Public Key Fingerprint
// Excludes array merge and rotate operations
type SignatureProposalQuorum map[string]SignatureProposalParticipant

type SignatureProposalParticipantStatus uint8

const (
	PubKeyConAwaitConfirmation DKGProposalParticipantStatus = iota
	PubKeyConfirmed
	CommitAwaitConfirmation
	CommitConfirmed
	DealAwaitConfirmation
	DealConfirmed
)

type DKGProposal struct {
	Quorum    map[int]DKGProposalParticipant
	CreatedAt *time.Time
	ExpiresAt *time.Time
}

type DKGProposalParticipant struct {
	Title     string
	PublicKey []byte
	Commit    []byte
	Deal      []byte
	Status    DKGProposalParticipantStatus
	UpdatedAt *time.Time
}

type DKGProposalQuorum map[int]DKGProposalParticipant

type DKGProposalParticipantStatus uint8
