package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/depools/dc4bc/fsm/state_machines/signing_proposal_fsm"

	"github.com/depools/dc4bc/fsm/fsm"
	"github.com/depools/dc4bc/fsm/state_machines/dkg_proposal_fsm"
	"github.com/depools/dc4bc/fsm/state_machines/signature_proposal_fsm"
	"github.com/depools/dc4bc/fsm/types/requests"
	"github.com/depools/dc4bc/storage"
)

type OperationType string

const (
	DKGCommits             OperationType = "dkg_commits"
	SignatureReconstructed fsm.Event     = "signature_reconstructed"
)

type ReconstructedSignature struct {
	SrcPayload []byte
	Signature  []byte
	Username   string
	DKGRoundID string
}

// Operation is the type for any Operation that might be required for
// both DKG and signing process (e.g.,
type Operation struct {
	ID            string // UUID4
	Type          OperationType
	Payload       []byte
	ResultMsgs    []storage.Message
	CreatedAt     time.Time
	DKGIdentifier string
	To            string
	Event         fsm.Event
}

func (o *Operation) Check(o2 *Operation) error {
	if o.ID != o2.ID {
		return fmt.Errorf("o1.ID (%s) != o2.ID (%s)", o.ID, o2.ID)
	}

	if o.Type != o2.Type {
		return fmt.Errorf("o1.Type (%s) != o2.Type (%s)", o.Type, o2.Type)
	}

	if !bytes.Equal(o.Payload, o2.Payload) {
		return fmt.Errorf("o1.Payload (%v) != o2.Payload (%v)", o.Payload, o2.Payload)
	}

	return nil
}

// FSMRequestFromMessage converts a message data to a necessary FSM struct
func FSMRequestFromMessage(message storage.Message) (interface{}, error) {
	var resolvedValue interface{}
	switch fsm.Event(message.Event) {
	case signature_proposal_fsm.EventConfirmSignatureProposal:
		var req requests.SignatureProposalParticipantRequest
		if err := json.Unmarshal(message.Data, &req); err != nil {
			return fmt.Errorf("failed to unmarshal fsm req: %v", err), nil
		}
		resolvedValue = req
	case signature_proposal_fsm.EventInitProposal:
		var req requests.SignatureProposalParticipantsListRequest
		if err := json.Unmarshal(message.Data, &req); err != nil {
			return fmt.Errorf("failed to unmarshal fsm req: %v", err), nil
		}
		resolvedValue = req
	case dkg_proposal_fsm.EventDKGCommitConfirmationReceived:
		var req requests.DKGProposalCommitConfirmationRequest
		if err := json.Unmarshal(message.Data, &req); err != nil {
			return fmt.Errorf("failed to unmarshal fsm req: %v", err), nil
		}
		resolvedValue = req
	case dkg_proposal_fsm.EventDKGDealConfirmationReceived:
		var req requests.DKGProposalDealConfirmationRequest
		if err := json.Unmarshal(message.Data, &req); err != nil {
			return fmt.Errorf("failed to unmarshal fsm req: %v", err), nil
		}
		resolvedValue = req
	case dkg_proposal_fsm.EventDKGResponseConfirmationReceived:
		var req requests.DKGProposalResponseConfirmationRequest
		if err := json.Unmarshal(message.Data, &req); err != nil {
			return fmt.Errorf("failed to unmarshal fsm req: %v", err), nil
		}
		resolvedValue = req
	case dkg_proposal_fsm.EventDKGMasterKeyConfirmationReceived:
		var req requests.DKGProposalMasterKeyConfirmationRequest
		if err := json.Unmarshal(message.Data, &req); err != nil {
			return fmt.Errorf("failed to unmarshal fsm req: %v", err), nil
		}
		resolvedValue = req
	case signing_proposal_fsm.EventSigningPartialSignReceived:
		var req requests.SigningProposalPartialSignRequest
		if err := json.Unmarshal(message.Data, &req); err != nil {
			return fmt.Errorf("failed to unmarshal fsm req: %v", err), nil
		}
		resolvedValue = req
	case signing_proposal_fsm.EventConfirmSigningConfirmation:
		var req requests.SigningProposalParticipantRequest
		if err := json.Unmarshal(message.Data, &req); err != nil {
			return fmt.Errorf("failed to unmarshal fsm req: %v", err), nil
		}
		resolvedValue = req
	case signing_proposal_fsm.EventSigningStart:
		var req requests.SigningProposalStartRequest
		if err := json.Unmarshal(message.Data, &req); err != nil {
			return fmt.Errorf("failed to unmarshal fsm req: %v", err), nil
		}
		resolvedValue = req
	default:
		return nil, fmt.Errorf("invalid event: %s", message.Event)
	}

	return resolvedValue, nil
}
