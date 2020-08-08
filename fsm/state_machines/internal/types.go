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
	SignatureConfirmationAwaitConfirmation DKGProposalParticipantStatus = iota
	SignatureConfirmationConfirmed
	SignatureConfirmationDeclined
	SignatureConfirmationError
	PubKeyConAwaitConfirmation
	PubKeyConfirmed
	PubKeyConfirmationError
	CommitAwaitConfirmation
	CommitConfirmed
	CommitConfirmationError
	DealAwaitConfirmation
	DealConfirmed
	DealConfirmationError
	ResponseAwaitConfirmation
	ResponseConfirmed
	ResponseConfirmationError
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
	Response  []byte
	Status    DKGProposalParticipantStatus
	UpdatedAt *time.Time
}

type DKGProposalQuorum map[int]DKGProposalParticipant

type DKGProposalParticipantStatus uint8

func (s DKGProposalParticipantStatus) String() string {
	var str = "undefined"
	switch s {
	case SignatureConfirmationAwaitConfirmation:
		str = "SignatureConfirmationAwaitConfirmation"
	case SignatureConfirmationConfirmed:
		str = "SignatureConfirmationConfirmed"
	case SignatureConfirmationDeclined:
		str = "SignatureConfirmationDeclined"
	case SignatureConfirmationError:
		str = "SignatureConfirmationError"
	case PubKeyConAwaitConfirmation:
		str = "PubKeyConAwaitConfirmation"
	case PubKeyConfirmed:
		str = "PubKeyConfirmed"
	case PubKeyConfirmationError:
		str = "PubKeyConfirmationError"
	case CommitAwaitConfirmation:
		str = "CommitAwaitConfirmation"
	case CommitConfirmed:
		str = "CommitConfirmed"
	case CommitConfirmationError:
		str = "CommitConfirmationError"
	case DealAwaitConfirmation:
		str = "DealAwaitConfirmation"
	case DealConfirmed:
		str = "DealConfirmed"
	case DealConfirmationError:
		str = "DealConfirmationError"
	case ResponseAwaitConfirmation:
		str = "ResponseAwaitConfirmation"
	case ResponseConfirmed:
		str = "ResponseConfirmed"
	case ResponseConfirmationError:
		str = "ResponseConfirmationError"
	}
	return str
}
