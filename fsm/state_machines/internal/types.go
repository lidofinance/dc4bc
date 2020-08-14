package internal

import (
	"crypto/rsa"
	"time"
)

type SignatureConfirmation struct {
	Quorum    SignatureProposalQuorum
	CreatedAt time.Time
	ExpiresAt time.Time
}

type SignatureProposalParticipant struct {
	// Public title for address, such as name, nickname, organization
	ParticipantId int
	Title         string
	PubKey        *rsa.PublicKey
	DkgPubKey     []byte
	// For validation user confirmation: sign(InvitationSecret, PubKey) => user
	InvitationSecret string
	Status           ParticipantStatus
	UpdatedAt        time.Time
}

// Unique alias for map iteration - Public Key Fingerprint
// Excludes array merge and rotate operations
type SignatureProposalQuorum map[string]*SignatureProposalParticipant

type ParticipantStatus uint8

const (
	SignatureConfirmationAwaitConfirmation ParticipantStatus = iota
	SignatureConfirmationConfirmed
	SignatureConfirmationDeclined
	SignatureConfirmationError
	CommitAwaitConfirmation
	CommitConfirmed
	CommitConfirmationError
	DealAwaitConfirmation
	DealConfirmed
	DealConfirmationError
	ResponseAwaitConfirmation
	ResponseConfirmed
	ResponseConfirmationError
	MasterKeyAwaitConfirmation
	MasterKeyConfirmed
	MasterKeyConfirmationError
)

type DKGProposalParticipant struct {
	Title     string
	PubKey    []byte
	Commit    []byte
	Deal      []byte
	Response  []byte
	MasterKey []byte
	Status    ParticipantStatus
	Error     error
	UpdatedAt time.Time
}

type DKGProposalQuorum map[int]*DKGProposalParticipant

type DKGConfirmation struct {
	Quorum    DKGProposalQuorum
	CreatedAt *time.Time
	ExpiresAt *time.Time
}

type DKGProposalParticipantStatus uint8

func (s ParticipantStatus) String() string {
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
