package airgapped

import (
	"fmt"
	"go.dedis.ch/kyber/v3/sign/bls"
	"go.dedis.ch/kyber/v3/sign/tbls"
)

// commented because fsm is not ready
//func (am *AirgappedMachine) handleStateSigningAwaitPartialKeys(o *client.Operation) error {
//	var (
//		payload responses.DKGProposalResponsesParticipantResponse
//		err     error
//	)
//
//	if err = json.Unmarshal(o.Payload, &payload); err != nil {
//		return fmt.Errorf("failed to unmarshal payload: %w", err)
//	}
//
//	partialSign, err := am.createPartialSign(nil, o.DKGIdentifier)
//	if err != nil {
//		return fmt.Errorf("failed to create partialSign for msg: %w", err)
//	}
//
//	req := requests.SigningProposalPartialKeyRequest{
//		ParticipantId: 0, // TODO: from where?
//		PartialSign:    partialSign,
//		CreatedAt:     o.CreatedAt,
//	}
//	reqBz, err := json.Marshal(req)
//	if err != nil {
//		return fmt.Errorf("failed to generate fsm request: %w", err)
//	}
//
//	o.Result = reqBz
//	o.Event = signing_proposal_fsm.EventSigningPartialKeyReceived
//	return nil
//}

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
