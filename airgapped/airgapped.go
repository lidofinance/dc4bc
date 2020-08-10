package airgapped

import (
	"encoding/json"
	"fmt"
	"github.com/depools/dc4bc/client"
	"github.com/depools/dc4bc/dkg"
	"github.com/depools/dc4bc/fsm/fsm"
	"github.com/depools/dc4bc/fsm/state_machines/dkg_proposal_fsm"
	"github.com/depools/dc4bc/fsm/types/requests"
	"github.com/depools/dc4bc/fsm/types/responses"
	"github.com/depools/dc4bc/qr"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/pairing/bn256"
	dkg2 "go.dedis.ch/kyber/v3/share/dkg/pedersen"
)

type AirgappedMachine struct {
	dkgInstance *dkg.DKG // should be a map or something to distinguish different rounds
	qrProcessor qr.Processor
}

func NewAirgappedMachine() *AirgappedMachine {
	machine := &AirgappedMachine{
		dkgInstance: dkg.Init(),
		qrProcessor: qr.NewCameraProcessor(),
	}
	return machine
}

func (am *AirgappedMachine) handleStateDkgPubKeysAwaitConfirmations(o *client.Operation) error {
	pubKeyBz, err := am.dkgInstance.GetPubKey().MarshalBinary()
	if err != nil {
		return fmt.Errorf("failed to marshal pubkey: %w", err)
	}
	req := requests.DKGProposalPubKeyConfirmationRequest{
		ParticipantId: am.dkgInstance.ParticipantID,
		PubKey:        pubKeyBz,
		CreatedAt:     nil,
	}
	reqBz, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	o.Result = reqBz
	return nil
}

func (am *AirgappedMachine) handleStateDkgCommitsAwaitConfirmations(o *client.Operation) error {
	var (
		payload responses.DKGProposalPubKeyParticipantResponse
		err     error
	)
	if err = json.Unmarshal(o.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	for _, entry := range payload {
		pubKey := bn256.NewSuiteG2().Point()
		if err = pubKey.UnmarshalBinary(entry.PubKey); err != nil {
			return fmt.Errorf("failed to unmarshal pubkey: %w", err)
		}
		am.dkgInstance.StorePubKey(entry.Title, pubKey)
	}

	if err = am.dkgInstance.InitDKGInstance(3); err != nil { // TODO: threshold
		return fmt.Errorf("failed to init dkg instance: %w", err)
	}

	commits, err := json.Marshal(am.dkgInstance.GetCommits())
	if err != nil {
		return fmt.Errorf("failed to marshal commits: %w", err)
	}

	req := requests.DKGProposalCommitConfirmationRequest{
		ParticipantId: am.dkgInstance.ParticipantID,
		Commit:        commits,
	}
	reqBz, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	o.Result = reqBz
	return nil
}

func (am *AirgappedMachine) handleStateDkgDealsAwaitConfirmations(o *client.Operation) error {
	var (
		payload responses.DKGProposalCommitParticipantResponse
		err     error
	)
	if err = json.Unmarshal(o.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	for _, entry := range payload {
		var commits []kyber.Point
		if err = json.Unmarshal(entry.Commit, &commits); err != nil {
			return fmt.Errorf("failed to unmarshal commits: %w", err)
		}
		am.dkgInstance.StoreCommits(entry.Title, commits)
	}

	deals, err := am.dkgInstance.GetDeals()
	if err != nil {
		return fmt.Errorf("failed to get deals: %w", err)
	}

	// Here we should create N=len(deals) private (encrypted) messages to participants but i don't know how to it yet
	//-------------------------------------------------------
	dealsBz, err := json.Marshal(deals)
	if err != nil {
		return fmt.Errorf("failed to marshal deals")
	}

	req := requests.DKGProposalDealConfirmationRequest{
		ParticipantId: am.dkgInstance.ParticipantID,
		Deal:          dealsBz,
	}
	//-------------------------------------------------------

	reqBz, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	o.Result = reqBz
	return nil
}

func (am *AirgappedMachine) handleStateDkgResponsesAwaitConfirmations(o *client.Operation) error {
	var (
		payload responses.DKGProposalDealParticipantResponse
		err     error
	)
	if err = json.Unmarshal(o.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	for _, entry := range payload {
		var deals []dkg2.Deal // mock, must be another logic, because of encryption
		if err = json.Unmarshal(entry.Deal, &deals); err != nil {
			return fmt.Errorf("failed to unmarshal commits: %w", err)
		}
		for _, deal := range deals {
			am.dkgInstance.StoreDeal(entry.Title, &deal)
		}
	}

	processedResponses, err := am.dkgInstance.ProcessDeals()
	if err != nil {
		return fmt.Errorf("failed to process deals: %w", err)
	}

	responsesBz, err := json.Marshal(processedResponses)
	if err != nil {
		return fmt.Errorf("failed to marshal deals")
	}

	req := requests.DKGProposalResponseConfirmationRequest{
		ParticipantId: am.dkgInstance.ParticipantID,
		Response:      responsesBz,
	}

	reqBz, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	o.Result = reqBz
	return nil
}

func (am *AirgappedMachine) HandleQR() error {
	var (
		err       error
		operation client.Operation
		qrData    []byte
	)

	if qrData, err = am.qrProcessor.ReadQR(); err != nil {
		return fmt.Errorf("failed to read QR: %w", err)
	}
	if err = json.Unmarshal(qrData, &operation); err != nil {
		return fmt.Errorf("failed to unmarshal operation: %w", err)
	}
	switch fsm.State(operation.Type) {
	case dkg_proposal_fsm.StateDkgPubKeysAwaitConfirmations:
		if err = am.handleStateDkgPubKeysAwaitConfirmations(&operation); err != nil {
			return err
		}
	case dkg_proposal_fsm.StateDkgCommitsAwaitConfirmations:
		if err = am.handleStateDkgCommitsAwaitConfirmations(&operation); err != nil {
			return err
		}
	case dkg_proposal_fsm.StateDkgDealsAwaitConfirmations:
		if err = am.handleStateDkgCommitsAwaitConfirmations(&operation); err != nil {
			return err
		}
	case dkg_proposal_fsm.StateDkgResponsesAwaitConfirmations:
		if err = am.handleStateDkgResponsesAwaitConfirmations(&operation); err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid operation type: %s", operation.Type)
	}

	operationBz, err := json.Marshal(operation)
	if err != nil {
		return fmt.Errorf("failed to marshal operation: %w", err)
	}

	//TODO: return path
	if err := am.qrProcessor.WriteQR(fmt.Sprintf("%s.png", operation.ID), operationBz); err != nil {
		return fmt.Errorf("failed to write QR")
	}

	return nil
}
