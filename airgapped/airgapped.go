package airgapped

import (
	"encoding/json"
	"fmt"
	"github.com/depools/dc4bc/client"
	"github.com/depools/dc4bc/dkg"
	"github.com/depools/dc4bc/fsm/fsm"
	"github.com/depools/dc4bc/fsm/state_machines/dkg_proposal_fsm"
	"github.com/depools/dc4bc/fsm/state_machines/signature_proposal_fsm"
	"github.com/depools/dc4bc/fsm/types/requests"
	"github.com/depools/dc4bc/fsm/types/responses"
	"github.com/depools/dc4bc/qr"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/encrypt/ecies"
	"go.dedis.ch/kyber/v3/group/edwards25519"
	"go.dedis.ch/kyber/v3/pairing/bn256"
	dkgPedersen "go.dedis.ch/kyber/v3/share/dkg/pedersen"
	"sync"
)

const (
	resultQRFolder = "result_qr_codes"
)

type AirgappedMachine struct {
	sync.Mutex

	dkgInstances map[string]*dkg.DKG
	qrProcessor  qr.Processor
}

func NewAirgappedMachine() *AirgappedMachine {
	machine := &AirgappedMachine{
		dkgInstances: make(map[string]*dkg.DKG),
		qrProcessor:  qr.NewCameraProcessor(),
	}
	return machine
}

func (am *AirgappedMachine) encryptData(dkgIdentifier, to string, data []byte) ([]byte, error) {
	suite := edwards25519.NewBlakeSHA256Ed25519()
	dkgInstance, ok := am.dkgInstances[dkgIdentifier]
	if !ok {
		return nil, fmt.Errorf("invalid dkg identifier: %s", dkgIdentifier)
	}

	pk, err := dkgInstance.GetPubKeyByParticipantID(to)
	if err != nil {
		return nil, fmt.Errorf("failed to get pk for participant %s: %w", to, err)
	}

	encryptedData, err := ecies.Encrypt(suite, pk, data, suite.Hash)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt data: %w", err)
	}
	return encryptedData, nil
}

func (am *AirgappedMachine) decryptData(dkgIdentifier string, data []byte) ([]byte, error) {
	suite := edwards25519.NewBlakeSHA256Ed25519()
	dkgInstance, ok := am.dkgInstances[dkgIdentifier]
	if !ok {
		return nil, fmt.Errorf("invalid dkg identifier: %s", dkgIdentifier)
	}

	privateKey := dkgInstance.GetSecKey()

	decryptedData, err := ecies.Decrypt(suite, privateKey, data, suite.Hash)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}
	return decryptedData, nil
}

func (am *AirgappedMachine) handleStateAwaitParticipantsConfirmations(o *client.Operation) error {
	var (
		payload responses.SignatureProposalParticipantInvitationsResponse
		err     error
	)

	if err = json.Unmarshal(o.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	if _, ok := am.dkgInstances[o.DKGIdentifier]; ok {
		return fmt.Errorf("dkg instance %s already exists", o.DKGIdentifier)
	}

	am.dkgInstances[o.DKGIdentifier] = dkg.Init()

	// sMaybe we should do something with Title and ParticipantID
	// And i think threshold should be here

	return nil
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

func (am *AirgappedMachine) handleStateDkgDealsAwaitConfirmations(o client.Operation) ([]client.Operation, error) {
	var (
		payload responses.DKGProposalCommitParticipantResponse
		err     error
	)

	dkgInstance, ok := am.dkgInstances[o.DKGIdentifier]
	if !ok {
		return nil, fmt.Errorf("dkg instance with identifier %s does not exist", o.DKGIdentifier)
	}

	if err = json.Unmarshal(o.Payload, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	for _, entry := range payload {
		var commits []kyber.Point
		if err = json.Unmarshal(entry.Commit, &commits); err != nil {
			return nil, fmt.Errorf("failed to unmarshal commits: %w", err)
		}
		dkgInstance.StoreCommits(entry.Title, commits)
	}

	deals, err := dkgInstance.GetDeals()
	if err != nil {
		return nil, fmt.Errorf("failed to get deals: %w", err)
	}

	operations := make([]client.Operation, 0, len(deals))

	am.dkgInstances[o.DKGIdentifier] = dkgInstance

	for index, deal := range deals {
		dealBz, err := json.Marshal(deal)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal dea;: %w", err)
		}
		toParticipant := dkgInstance.GetParticipantByIndex(index)
		encryptedDeal, err := am.encryptData(o.DKGIdentifier, toParticipant, dealBz)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt deal: %w", err)
		}
		req := requests.DKGProposalDealConfirmationRequest{
			ParticipantId: dkgInstance.ParticipantID,
			Deal:          encryptedDeal,
		}
		o.To = toParticipant
		reqBz, err := json.Marshal(req)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		o.Result = reqBz
		operations = append(operations, o)
	}
	return operations, nil
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
		var deals []dkgPedersen.Deal // mock, must be another logic, because of encryption
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

func (am *AirgappedMachine) HandleQR() ([]string, error) {
	var (
		err        error
		operation  client.Operation
		qrData     []byte
		operations []client.Operation
	)

	am.Lock()
	defer am.Unlock()

	if qrData, err = am.qrProcessor.ReadQR(); err != nil {
		return nil, fmt.Errorf("failed to read QR: %w", err)
	}
	if err = json.Unmarshal(qrData, &operation); err != nil {
		return nil, fmt.Errorf("failed to unmarshal operation: %w", err)
	}

	switch fsm.State(operation.Type) {
	case signature_proposal_fsm.StateAwaitParticipantsConfirmations:
		////////
	case dkg_proposal_fsm.StateDkgPubKeysAwaitConfirmations:
		if err = am.handleStateDkgPubKeysAwaitConfirmations(&operation); err != nil {
			return nil, err
		}
	case dkg_proposal_fsm.StateDkgCommitsAwaitConfirmations:
		if err = am.handleStateDkgCommitsAwaitConfirmations(&operation); err != nil {
			return nil, err
		}
	case dkg_proposal_fsm.StateDkgDealsAwaitConfirmations:
		if operations, err = am.handleStateDkgDealsAwaitConfirmations(operation); err != nil {
			return nil, err
		}
	case dkg_proposal_fsm.StateDkgResponsesAwaitConfirmations:
		if err = am.handleStateDkgResponsesAwaitConfirmations(&operation); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("invalid operation type: %s", operation.Type)
	}

	if len(operation.Result) > 0 {
		operations = append(operations, operation)
	}

	qrPath := "%s/%s_%s_%s.png"
	qrPaths := make([]string, 0, len(operations))
	for _, o := range operations {
		operationBz, err := json.Marshal(o)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal operation: %w", err)
		}

		if err := am.qrProcessor.WriteQR(fmt.Sprintf(qrPath, resultQRFolder, o.Type, o.ID, o.To), operationBz); err != nil {
			return nil, fmt.Errorf("failed to write QR")
		}
		qrPaths = append(qrPaths, qrPath)
	}

	return qrPaths, nil
}
