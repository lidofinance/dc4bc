package airgapped

import (
	"encoding/json"
	"fmt"
	"github.com/corestario/kyber/pairing"
	"github.com/corestario/kyber/sign/bls"
	"github.com/corestario/kyber/sign/tbls"
	client "github.com/lidofinance/dc4bc/client/types"
	"github.com/lidofinance/dc4bc/fsm/state_machines/signing_proposal_fsm"
	"github.com/lidofinance/dc4bc/fsm/types/requests"
	"github.com/lidofinance/dc4bc/fsm/types/responses"
)

// handleStateSigningAwaitPartialSigns takes a data to sign as payload and returns a partial sign for the data to broadcast
func (am *Machine) handleStateSigningAwaitPartialSigns(o *client.Operation) error {
	var (
		payload        responses.SigningPartialSignsParticipantInvitationsResponse
		messagesToSign []requests.MessageToSign
		err            error
	)

	if err = json.Unmarshal(o.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	if err = json.Unmarshal(payload.SrcPayload, &messagesToSign); err != nil {
		return fmt.Errorf("failed to unmarshal messages to sign: %w", err)
	}

	signs := make([]requests.PartialSign, 0, len(messagesToSign))
	participantID, err := am.getParticipantID(o.DKGIdentifier)
	if err != nil {
		return fmt.Errorf("failed to get paricipant id: %w", err)
	}
	for _, m := range messagesToSign {
		partialSign, err := am.createPartialSign(m.Payload, o.DKGIdentifier)
		if err != nil {
			return fmt.Errorf("failed to create partialSign for msg: %w", err)
		}

		signs = append(signs, requests.PartialSign{
			MessageID: m.MessageID,
			Sign:      partialSign,
		})
	}

	req := requests.SigningProposalBatchPartialSignRequests{
		BatchID:       payload.BatchID,
		ParticipantId: participantID,
		PartialSigns:  signs,
		CreatedAt:     o.CreatedAt,
	}

	reqBz, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to generate fsm request: %w", err)
	}

	o.Event = signing_proposal_fsm.EventSigningPartialSignReceived
	o.ResultMsgs = append(o.ResultMsgs, createMessage(*o, reqBz))
	return nil
}

// createPartialSign returns a partial sign of a given message
// with using of a private part of the reconstructed DKG key of a given DKG round
func (am *Machine) createPartialSign(msg []byte, dkgIdentifier string) ([]byte, error) {
	blsKeyring, err := am.loadBLSKeyring(dkgIdentifier)
	if err != nil {
		return nil, fmt.Errorf("failed to load blsKeyring: %w", err)
	}

	return tbls.Sign(am.baseSuite.(pairing.Suite), blsKeyring.Share, msg)
}

// VerifySign verifies a signature of a message
func (am *Machine) VerifySign(msg []byte, fullSignature []byte, dkgIdentifier string) error {
	blsKeyring, err := am.loadBLSKeyring(dkgIdentifier)
	if err != nil {
		return fmt.Errorf("failed to load blsKeyring: %w", err)
	}

	return bls.Verify(am.baseSuite.(pairing.Suite), blsKeyring.PubPoly.Commit(), msg, fullSignature)
}
