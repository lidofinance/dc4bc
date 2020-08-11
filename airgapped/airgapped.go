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
	dkgInstances map[string]*dkg.DKG // should be a map or something to distinguish different rounds
	qrProcessor  qr.Processor
}

func NewAirgappedMachine() *AirgappedMachine {
	machine := &AirgappedMachine{
		dkgInstances: make(map[string]*dkg.DKG),
		qrProcessor:  qr.NewCameraProcessor(),
	}
	return machine
}

func (am *AirgappedMachine) handleStateDkgPubKeysAwaitConfirmations(o *client.Operation) error {
	dkgInstance, ok := am.dkgInstances[o.DKGIdentifier]
	if !ok {
		return fmt.Errorf("dkg instance with identifier %s does not exist", o.DKGIdentifier)
	}

	pubKeyBz, err := dkgInstance.GetPubKey().MarshalBinary()
	if err != nil {
		return fmt.Errorf("failed to marshal pubkey: %w", err)
	}
	req := requests.DKGProposalPubKeyConfirmationRequest{
		ParticipantId: dkgInstance.ParticipantID,
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

	dkgInstance, ok := am.dkgInstances[o.DKGIdentifier]
	if !ok {
		return fmt.Errorf("dkg instance with identifier %s does not exist", o.DKGIdentifier)
	}

	if err = json.Unmarshal(o.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	for _, entry := range payload {
		pubKey := bn256.NewSuiteG2().Point()
		if err = pubKey.UnmarshalBinary(entry.PubKey); err != nil {
			return fmt.Errorf("failed to unmarshal pubkey: %w", err)
		}
		dkgInstance.StorePubKey(entry.Title, pubKey)
	}

	if err = dkgInstance.InitDKGInstance(3); err != nil { // TODO: threshold
		return fmt.Errorf("failed to init dkg instance: %w", err)
	}

	commits, err := json.Marshal(dkgInstance.GetCommits())
	if err != nil {
		return fmt.Errorf("failed to marshal commits: %w", err)
	}

	am.dkgInstances[o.DKGIdentifier] = dkgInstance

	req := requests.DKGProposalCommitConfirmationRequest{
		ParticipantId: dkgInstance.ParticipantID,
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

	dkgInstance, ok := am.dkgInstances[o.DKGIdentifier]
	if !ok {
		return fmt.Errorf("dkg instance with identifier %s does not exist", o.DKGIdentifier)
	}

	if err = json.Unmarshal(o.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	for _, entry := range payload {
		var commits []kyber.Point
		if err = json.Unmarshal(entry.Commit, &commits); err != nil {
			return fmt.Errorf("failed to unmarshal commits: %w", err)
		}
		dkgInstance.StoreCommits(entry.Title, commits)
	}

	deals, err := dkgInstance.GetDeals()
	if err != nil {
		return fmt.Errorf("failed to get deals: %w", err)
	}

	am.dkgInstances[o.DKGIdentifier] = dkgInstance

	// Here we should create N=len(deals) private (encrypted) messages to participants but i don't know how to it yet
	//-------------------------------------------------------
	dealsBz, err := json.Marshal(deals)
	if err != nil {
		return fmt.Errorf("failed to marshal deals")
	}

	req := requests.DKGProposalDealConfirmationRequest{
		ParticipantId: dkgInstance.ParticipantID,
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

	dkgInstance, ok := am.dkgInstances[o.DKGIdentifier]
	if !ok {
		return fmt.Errorf("dkg instance with identifier %s does not exist", o.DKGIdentifier)
	}

	if err = json.Unmarshal(o.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	for _, entry := range payload {
		var deals []dkg2.Deal // mock, must be another logic, because of encryption
		if err = json.Unmarshal(entry.Deal, &deals); err != nil {
			return fmt.Errorf("failed to unmarshal commits: %w", err)
		}
		for _, deal := range deals {
			dkgInstance.StoreDeal(entry.Title, &deal)
		}
	}

	processedResponses, err := dkgInstance.ProcessDeals()
	if err != nil {
		return fmt.Errorf("failed to process deals: %w", err)
	}

	am.dkgInstances[o.DKGIdentifier] = dkgInstance

	responsesBz, err := json.Marshal(processedResponses)
	if err != nil {
		return fmt.Errorf("failed to marshal deals")
	}

	req := requests.DKGProposalResponseConfirmationRequest{
		ParticipantId: dkgInstance.ParticipantID,
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
