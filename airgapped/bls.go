package airgapped

import (
	"encoding/json"
	"fmt"
	client "github.com/depools/dc4bc/client/types"
	"github.com/depools/dc4bc/fsm/state_machines/signing_proposal_fsm"
	"github.com/depools/dc4bc/fsm/types/requests"
	"github.com/depools/dc4bc/fsm/types/responses"
	"go.dedis.ch/kyber/v3/sign/bls"
	"go.dedis.ch/kyber/v3/sign/tbls"
)

// commented because fsm is not ready
func (am *AirgappedMachine) handleStateSigningAwaitPartialSigns(o *client.Operation) error {
	var (
		payload responses.SigningProposalParticipantInvitationsResponse
		err     error
	)

	if err = json.Unmarshal(o.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	partialSign, err := am.createPartialSign(payload.SrcPayload, o.DKGIdentifier)
	if err != nil {
		return fmt.Errorf("failed to create partialSign for msg: %w", err)
	}

	participantID, err := am.getParticipantID(o.DKGIdentifier)
	if err != nil {
		return fmt.Errorf("failed to get paricipant id: %w", err)
	}
	req := requests.SigningProposalPartialKeyRequest{
		ParticipantId: participantID,
		PartialSign:   partialSign,
		CreatedAt:     o.CreatedAt,
	}
	reqBz, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to generate fsm request: %w", err)
	}

	o.Event = signing_proposal_fsm.EventSigningPartialKeyReceived
	o.ResultMsgs = append(o.ResultMsgs, createMessage(*o, reqBz))
	return nil
}

func (am *AirgappedMachine) reconstructThresholdSignature(o *client.Operation) error {
	var (
		payload responses.SigningProcessParticipantResponse
		err     error
	)

	if err = json.Unmarshal(o.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	partialSignatures := make([][]byte, 0, len(payload.Participants))
	for _, participant := range payload.Participants {
		partialSignatures = append(partialSignatures, participant.PartialSign)
	}

	reconstructedSignature, err := am.recoverFullSign(payload.SrcPayload, partialSignatures, o.DKGIdentifier)
	if err != nil {
		return fmt.Errorf("failed to reconsruct full signature for msg: %w", err)
	}
	fmt.Println(reconstructedSignature)
	return nil
}

func (am *AirgappedMachine) createPartialSign(msg []byte, dkgIdentifier string) ([]byte, error) {
	blsKeyring, err := am.loadBLSKeyring(dkgIdentifier)
	if err != nil {
		return nil, fmt.Errorf("failed to load blsKeyring: %w", err)
	}

	return tbls.Sign(am.suite, blsKeyring.Share, msg)
}

func (am *AirgappedMachine) recoverFullSign(msg []byte, sigShares [][]byte, dkgIdentifier string) ([]byte, error) {
	blsKeyring, err := am.loadBLSKeyring(dkgIdentifier)
	if err != nil {
		return nil, fmt.Errorf("failed to load blsKeyring: %w", err)
	}

	return tbls.Recover(am.suite, blsKeyring.PubPoly, msg, sigShares, len(sigShares), len(sigShares))
}

func (am *AirgappedMachine) verifySign(msg []byte, fullSignature []byte, dkgIdentifier string) error {
	blsKeyring, err := am.loadBLSKeyring(dkgIdentifier)
	if err != nil {
		return fmt.Errorf("failed to load blsKeyring: %w", err)
	}

	return bls.Verify(am.suite, blsKeyring.PubPoly.Commit(), msg, fullSignature)
}
