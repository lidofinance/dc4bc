package main

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/depools/dc4bc/client/types"
	"github.com/depools/dc4bc/fsm/fsm"
	"github.com/depools/dc4bc/fsm/state_machines/dkg_proposal_fsm"
	"github.com/depools/dc4bc/fsm/state_machines/signature_proposal_fsm"
	"github.com/depools/dc4bc/fsm/state_machines/signing_proposal_fsm"
	"github.com/depools/dc4bc/fsm/types/requests"
	"github.com/depools/dc4bc/fsm/types/responses"
	"sort"
)

type DKGInvitationResponse responses.SignatureProposalParticipantInvitationsResponse

func (d DKGInvitationResponse) Len() int           { return len(d) }
func (d DKGInvitationResponse) Swap(i, j int)      { d[i], d[j] = d[j], d[i] }
func (d DKGInvitationResponse) Less(i, j int) bool { return d[i].Addr < d[j].Addr }

type DKGParticipants []*requests.SignatureProposalParticipantsEntry

func (d DKGParticipants) Len() int           { return len(d) }
func (d DKGParticipants) Swap(i, j int)      { d[i], d[j] = d[j], d[i] }
func (d DKGParticipants) Less(i, j int) bool { return d[i].Addr < d[j].Addr }

type OperationsResponse struct {
	ErrorMessage string                      `json:"error_message,omitempty"`
	Result       map[string]*types.Operation `json:"result"`
}

type OperationQRPathsResponse struct {
	ErrorMessage string `json:"error_message,omitempty"`
	Result       string `json:"result"`
}

// calcStartDKGMessageHash returns hash of a StartDKGMessage to verify its correctness later
func calcStartDKGMessageHash(payload []byte) ([]byte, error) {
	var msg DKGInvitationResponse
	if err := json.Unmarshal(payload, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	sort.Sort(msg)

	hashPayload := bytes.NewBuffer(nil)
	// threshold is the same for everyone
	if _, err := hashPayload.Write([]byte(fmt.Sprintf("%d", msg[0].Threshold))); err != nil {
		return nil, err
	}
	for _, p := range msg {
		if _, err := hashPayload.Write(p.PubKey); err != nil {
			return nil, err
		}
		if _, err := hashPayload.Write(p.DkgPubKey); err != nil {
			return nil, err
		}
		if _, err := hashPayload.Write([]byte(p.Addr)); err != nil {
			return nil, err
		}
	}
	hash := md5.Sum(hashPayload.Bytes())
	return hash[:], nil
}

func getShortOperationDescription(operationType types.OperationType) string {
	switch fsm.State(operationType) {
	case signature_proposal_fsm.StateAwaitParticipantsConfirmations:
		return "confirm participation in the new DKG round"
	case dkg_proposal_fsm.StateDkgCommitsAwaitConfirmations:
		return "send commits for the DKG round"
	case dkg_proposal_fsm.StateDkgDealsAwaitConfirmations:
		return "send deals for the DKG round"
	case dkg_proposal_fsm.StateDkgResponsesAwaitConfirmations:
		return "send responses for the DKG round"
	case dkg_proposal_fsm.StateDkgMasterKeyAwaitConfirmations:
		return "reconstruct the public key and broadcast it"
	case signing_proposal_fsm.StateSigningAwaitConfirmations:
		return "confirm participation in a new message signing"
	case signing_proposal_fsm.StateSigningAwaitPartialSigns:
		return "send your partial sign for the message"
	case signing_proposal_fsm.StateSigningPartialSignsCollected:
		return "recover full signature for the message"
	default:
		return "unknown operation"
	}
}
