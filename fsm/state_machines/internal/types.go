package internal

import (
	"crypto/ed25519"
	"sort"
	"time"

	"github.com/lidofinance/dc4bc/fsm/types/requests"
)

type ParticipantStatus interface {
	String() string
}

type ConfirmationParticipantStatus uint8

const (
	SigConfirmationAwaitConfirmation ConfirmationParticipantStatus = iota
	SigConfirmationConfirmed
	SigConfirmationDeclined
	SigConfirmationError
)

func (s ConfirmationParticipantStatus) String() string {
	var str = "undefined"
	switch s {
	case SigConfirmationAwaitConfirmation:
		str = "SigConfirmationAwaitConfirmation"
	case SigConfirmationConfirmed:
		str = "SigConfirmationConfirmed"
	case SigConfirmationDeclined:
		str = "SigConfirmationDeclined"
	case SigConfirmationError:
		str = "SigConfirmationError"
	}
	return str
}

type SignatureConfirmation struct {
	Quorum    SignatureProposalQuorum
	CreatedAt time.Time
	UpdatedAt time.Time
	ExpiresAt time.Time
}

type SignatureProposalParticipant struct {
	ParticipantID int
	Username      string
	PubKey        ed25519.PublicKey
	DkgPubKey     []byte
	// For validation user confirmation: sign(InvitationSecret, PubKey) => user
	InvitationSecret string
	Status           ConfirmationParticipantStatus
	Threshold        int
	UpdatedAt        time.Time
}

func (sigP SignatureProposalParticipant) GetStatus() ParticipantStatus {
	return sigP.Status
}

func (sigP SignatureProposalParticipant) GetUsername() string {
	return sigP.Username
}

func (c *SignatureConfirmation) IsExpired() bool {
	return c.ExpiresAt.Before(c.UpdatedAt)
}

// Unique alias for map iteration - Public Key Fingerprint
// Excludes array merge and rotate operations
type SignatureProposalQuorum map[int]*SignatureProposalParticipant

func (q SignatureProposalQuorum) GetOrderedParticipants() []*SignatureProposalParticipant {
	var sortedParticipantIDs []int
	for participantID := range q {
		sortedParticipantIDs = append(sortedParticipantIDs, participantID)
	}

	sort.Ints(sortedParticipantIDs)

	var out []*SignatureProposalParticipant
	for _, participantID := range sortedParticipantIDs {
		var participant = q[participantID]
		participant.ParticipantID = participantID
		out = append(out, participant)
	}

	return out
}

// DKG proposal

type DKGParticipantStatus uint8

const (
	CommitAwaitConfirmation DKGParticipantStatus = iota
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
	ParticipantID int
	Username      string
	DkgPubKey     []byte
	DkgCommit     []byte
	DkgDeal       []byte
	DkgResponse   []byte
	DkgMasterKey  []byte
	Status        DKGParticipantStatus
	Error         *requests.FSMError
	UpdatedAt     time.Time
}

func (dkgP DKGProposalParticipant) GetStatus() ParticipantStatus {
	return dkgP.Status
}

func (dkgP DKGProposalParticipant) GetUsername() string {
	return dkgP.Username
}

type DKGProposalQuorum map[int]*DKGProposalParticipant

func (q DKGProposalQuorum) GetOrderedParticipants() []*DKGProposalParticipant {
	var sortedParticipantIDs []int
	for participantID := range q {
		sortedParticipantIDs = append(sortedParticipantIDs, participantID)
	}

	sort.Ints(sortedParticipantIDs)

	var out []*DKGProposalParticipant
	for _, participantID := range sortedParticipantIDs {
		var participant = q[participantID]
		participant.ParticipantID = participantID
		out = append(out, participant)
	}

	return out
}

type DKGConfirmation struct {
	Quorum    DKGProposalQuorum
	CreatedAt time.Time
	UpdatedAt time.Time
	ExpiresAt time.Time
}

func (c *DKGConfirmation) IsExpired() bool {
	return c.ExpiresAt.Before(c.UpdatedAt)
}

type DKGProposalParticipantStatus uint8

func (s DKGParticipantStatus) String() string {
	var str = "undefined"
	switch s {
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
	case MasterKeyAwaitConfirmation:
		str = "MasterKeyAwaitConfirmation"
	case MasterKeyConfirmed:
		str = "MasterKeyConfirmed"
	case MasterKeyConfirmationError:
		str = "MasterKeyConfirmationError"
	}
	return str
}

// Signing proposal

type SigningConfirmation struct {
	BatchID          string
	InitiatorId      int
	Quorum           SigningProposalQuorum
	RecoveredKey     []byte
	SrcPayload       []byte
	EncryptedPayload []byte
	CreatedAt        time.Time
	UpdatedAt        time.Time
	ExpiresAt        time.Time
}

func (c *SigningConfirmation) IsExpired() bool {
	return c.ExpiresAt.Before(c.UpdatedAt)
}

type SigningProposalQuorum map[int]*SigningProposalParticipant

func (q SigningProposalQuorum) GetOrderedParticipants() []*SigningProposalParticipant {
	var sortedParticipantIDs []int
	for participantID := range q {
		sortedParticipantIDs = append(sortedParticipantIDs, participantID)
	}

	sort.Ints(sortedParticipantIDs)

	var out []*SigningProposalParticipant
	for _, participantID := range sortedParticipantIDs {
		var participant = q[participantID]
		participant.ParticipantID = participantID
		out = append(out, participant)
	}

	return out
}

type SigningParticipantStatus uint8

const (
	SigningAwaitConfirmation SigningParticipantStatus = iota
	SigningConfirmed
	SigningDeclined
	SigningAwaitPartialSigns
	SigningPartialSignsConfirmed
	SigningError
	SigningProcess
)

func (s SigningParticipantStatus) String() string {
	var str = "undefined"
	switch s {
	case SigningAwaitConfirmation:
		str = "SigningAwaitConfirmation"
	case SigningConfirmed:
		str = "SigningConfirmed"
	case SigningAwaitPartialSigns:
		str = "SigningAwaitPartialSigns"
	case SigningPartialSignsConfirmed:
		str = "SigningPartialSignsConfirmed"
	case SigningError:
		str = "SigningError"
	case SigningProcess:
		str = "SigningProcess"
	}
	return str
}

type SigningProposalParticipant struct {
	ParticipantID int
	Username      string
	Status        SigningParticipantStatus
	PartialSigns  map[string][]byte
	Error         *requests.FSMError
	UpdatedAt     time.Time
}

func (signingP SigningProposalParticipant) GetStatus() ParticipantStatus {
	return signingP.Status
}

func (signingP SigningProposalParticipant) GetUsername() string {
	return signingP.Username
}
