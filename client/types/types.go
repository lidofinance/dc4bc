package types

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lidofinance/dc4bc/fsm/state_machines/signing_proposal_fsm"

	"github.com/lidofinance/dc4bc/fsm/fsm"
	"github.com/lidofinance/dc4bc/fsm/state_machines/dkg_proposal_fsm"
	"github.com/lidofinance/dc4bc/fsm/state_machines/signature_proposal_fsm"
	"github.com/lidofinance/dc4bc/fsm/types/requests"
	"github.com/lidofinance/dc4bc/storage"
)

type OperationType string

const (
	DKGCommits                    OperationType = "dkg_commits"
	SignatureReconstructed        fsm.Event     = "signature_reconstructed"
	SignatureReconstructionFailed fsm.Event     = "signature_reconstruction_failed"
	ReinitDKG                     fsm.State     = "reinit_dkg"

	// OperationProcessed common event type for successfully processed operations but with an empty result
	OperationProcessed fsm.Event = "operation_processed_successfully"
)

type ReconstructedSignature struct {
	SigningID  string
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

	// field for some additional helping data
	ExtraData []byte
}

func NewOperation(
	dkgRoundID string,
	payload []byte,
	state fsm.State,
) *Operation {
	operationID := fmt.Sprintf(
		"%s_%s",
		dkgRoundID,
		base64.StdEncoding.EncodeToString(payload),
	)
	operationIDmd5 := md5.Sum([]byte(operationID))
	return &Operation{
		ID:            hex.EncodeToString(operationIDmd5[:]),
		Type:          OperationType(state),
		Payload:       payload,
		DKGIdentifier: dkgRoundID,
		CreatedAt:     time.Now(),
	}
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
	case signature_proposal_fsm.EventConfirmSignatureProposal, signature_proposal_fsm.EventDeclineProposal:
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
	case signing_proposal_fsm.EventConfirmSigningConfirmation, signing_proposal_fsm.EventDeclineSigningConfirmation:
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
	case dkg_proposal_fsm.EventDKGCommitConfirmationError, dkg_proposal_fsm.EventDKGDealConfirmationError,
		dkg_proposal_fsm.EventDKGResponseConfirmationError, dkg_proposal_fsm.EventDKGMasterKeyConfirmationError:
		var req requests.DKGProposalConfirmationErrorRequest
		if err := json.Unmarshal(message.Data, &req); err != nil {
			return fmt.Errorf("failed to unmarshal fsm req: %v", err), nil
		}
		resolvedValue = req
	case signing_proposal_fsm.EventSigningPartialSignError, SignatureReconstructionFailed:
		var req requests.SignatureProposalConfirmationErrorRequest
		if err := json.Unmarshal(message.Data, &req); err != nil {
			return fmt.Errorf("failed to unmarshal fsm req: %v", err), nil
		}
		resolvedValue = req
	default:
		return nil, fmt.Errorf("invalid event: %s", message.Event)
	}

	return resolvedValue, nil
}

type Participant struct {
	DKGPubKey     []byte `json:"dkg_pub_key"`
	OldCommPubKey []byte `json:"old_comm_pub_key"`
	NewCommPubKey []byte `json:"new_comm_pub_key"`
	Name          string `json:"name"`
}

type ReDKG struct {
	DKGID        string            `json:"dkg_id"`
	Threshold    int               `json:"threshold"`
	Participants []Participant     `json:"participants"`
	Messages     []storage.Message `json:"messages"`
}

func GenerateReDKGMessage(messages []storage.Message) (*ReDKG, error) {
	var reDKG ReDKG

	for _, msg := range messages {
		if fsm.Event(msg.Event) == signature_proposal_fsm.EventInitProposal {
			req, err := FSMRequestFromMessage(msg)
			if err != nil {
				return nil, fmt.Errorf("failed to get FSM request from message: %v", err)
			}
			request, ok := req.(requests.SignatureProposalParticipantsListRequest)
			if !ok {
				return nil, fmt.Errorf("invalid request")
			}
			reDKG.DKGID = msg.DkgRoundID
			reDKG.Threshold = request.SigningThreshold
			for _, participant := range request.Participants {
				reDKG.Participants = append(reDKG.Participants, Participant{
					DKGPubKey:     participant.DkgPubKey,
					OldCommPubKey: participant.PubKey,
					Name:          participant.Username,
				})
			}
		}
		if fsm.Event(msg.Event) == signing_proposal_fsm.EventSigningStart {
			break
		}

		reDKG.Messages = append(reDKG.Messages, msg)
	}
	return &reDKG, nil
}

func CalcStartReInitDKGMessageHash(payload []byte) ([]byte, error) {
	var msg ReDKG
	if err := json.Unmarshal(payload, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	hashPayload := bytes.NewBuffer([]byte(msg.DKGID))
	if _, err := hashPayload.Write([]byte(fmt.Sprintf("%d", msg.Threshold))); err != nil {
		return nil, err
	}
	for _, p := range msg.Participants {
		if _, err := hashPayload.Write(p.NewCommPubKey); err != nil {
			return nil, err
		}
		if _, err := hashPayload.Write(p.OldCommPubKey); err != nil {
			return nil, err
		}
		if _, err := hashPayload.Write(p.DKGPubKey); err != nil {
			return nil, err
		}
		if _, err := hashPayload.Write([]byte(p.Name)); err != nil {
			return nil, err
		}
	}
	for _, m := range msg.Messages {
		if _, err := hashPayload.Write(m.Data); err != nil {
			return nil, err
		}
		if _, err := hashPayload.Write(m.Signature); err != nil {
			return nil, err
		}
		if _, err := hashPayload.Write([]byte(m.RecipientAddr)); err != nil {
			return nil, err
		}
		if _, err := hashPayload.Write([]byte(m.Event)); err != nil {
			return nil, err
		}
		if _, err := hashPayload.Write([]byte(m.SenderAddr)); err != nil {
			return nil, err
		}
		if _, err := hashPayload.Write([]byte(m.DkgRoundID)); err != nil {
			return nil, err
		}
		if _, err := hashPayload.Write([]byte(fmt.Sprintf("%d", m.Offset))); err != nil {
			return nil, err
		}
	}
	hash := md5.Sum(hashPayload.Bytes())
	return hash[:], nil
}
