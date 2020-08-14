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
	"go.dedis.ch/kyber/v3/pairing/bn256"
	dkgPedersen "go.dedis.ch/kyber/v3/share/dkg/pedersen"
	"log"
	"sync"
)

const (
	resultQRFolder = "result_qr_codes"
)

type AirgappedMachine struct {
	sync.Mutex

	dkgInstances map[string]*dkg.DKG
	qrProcessor  qr.Processor

	pubKey kyber.Point
	secKey kyber.Scalar
	suite  *bn256.Suite
}

func NewAirgappedMachine() *AirgappedMachine {
	am := &AirgappedMachine{
		dkgInstances: make(map[string]*dkg.DKG),
		qrProcessor:  qr.NewCameraProcessor(),
	}

	// TODO: leveldb
	am.suite = bn256.NewSuiteG2()
	am.secKey = am.suite.Scalar().Pick(am.suite.RandomStream())
	am.pubKey = am.suite.Point().Mul(am.secKey, nil)

	return am
}

func (am *AirgappedMachine) getParticipantID(dkgIdentifier string) (int, error) {
	dkgInstance, ok := am.dkgInstances[dkgIdentifier]
	if !ok {
		return 0, fmt.Errorf("invalid dkg identifier: %s", dkgIdentifier)
	}
	return dkgInstance.ParticipantID, nil
}

func (am *AirgappedMachine) encryptData(dkgIdentifier, to string, data []byte) ([]byte, error) {
	dkgInstance, ok := am.dkgInstances[dkgIdentifier]
	if !ok {
		return nil, fmt.Errorf("invalid dkg identifier: %s", dkgIdentifier)
	}

	pk, err := dkgInstance.GetPubKeyByParticipant(to)
	if err != nil {
		return nil, fmt.Errorf("failed to get pk for participant %s: %w", to, err)
	}

	encryptedData, err := ecies.Encrypt(am.suite, pk, data, am.suite.Hash)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt data: %w", err)
	}
	return encryptedData, nil
}

func (am *AirgappedMachine) decryptData(data []byte) ([]byte, error) {
	decryptedData, err := ecies.Decrypt(am.suite, am.secKey, data, am.suite.Hash)
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

	if _, ok := am.dkgInstances[o.DKGIdentifier]; ok {
		return fmt.Errorf("dkg instance %s already exists", o.DKGIdentifier)
	}

	if err = json.Unmarshal(o.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	if _, ok := am.dkgInstances[o.DKGIdentifier]; ok {
		return fmt.Errorf("dkg instance %s already exists", o.DKGIdentifier)
	}

	dkgInstance := dkg.Init(am.suite, am.pubKey, am.secKey)
	dkgInstance.Threshold = len(payload)

	am.dkgInstances[o.DKGIdentifier] = dkgInstance

	return nil
}

func (am *AirgappedMachine) GetPubKey() kyber.Point {
	return am.pubKey
}

type Commits struct {
	MarshaledCommit []byte
}

func (am *AirgappedMachine) handleStateDkgCommitsAwaitConfirmations(o *client.Operation) error {
	var (
		payload responses.SignatureProposalParticipantStatusResponse
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
		if err = pubKey.UnmarshalBinary(entry.DkgPubKey); err != nil {
			return fmt.Errorf("failed to unmarshal pubkey: %w", err)
		}
		dkgInstance.StorePubKey(entry.Title, entry.ParticipantId, pubKey)
	}

	if err = dkgInstance.InitDKGInstance(); err != nil {
		return fmt.Errorf("failed to init dkg instance: %w", err)
	}

	// TODO: come up with something better
	var commits []Commits
	dkgCommits := dkgInstance.GetCommits()
	for _, commit := range dkgCommits {
		commitBz, err := commit.MarshalBinary()
		if err != nil {
			return fmt.Errorf("failed to marshal commits: %w", err)
		}
		commits = append(commits, Commits{MarshaledCommit: commitBz})
	}
	commitsBz, err := json.Marshal(commits)

	am.dkgInstances[o.DKGIdentifier] = dkgInstance

	req := requests.DKGProposalCommitConfirmationRequest{
		ParticipantId: dkgInstance.ParticipantID,
		Commit:        commitsBz,
		CreatedAt:     o.CreatedAt,
	}
	reqBz, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to generate fsm request: %w", err)
	}
	o.Result = reqBz
	o.Event = dkg_proposal_fsm.EventDKGCommitConfirmationReceived
	return nil
}

// We have many deals which should be sent privately to a required participant, so func returns a slice of operations
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
		var commits []Commits
		if err = json.Unmarshal(entry.Commit, &commits); err != nil {
			return nil, fmt.Errorf("failed to unmarshal commits: %w", err)
		}
		dkgCommits := make([]kyber.Point, 0, len(commits))
		for _, c := range commits {
			commit := am.suite.Point()
			if err = commit.UnmarshalBinary(c.MarshaledCommit); err != nil {
				return nil, fmt.Errorf("failed to unmarshal commit: %w", err)
			}
			dkgCommits = append(dkgCommits, commit)
		}
		dkgInstance.StoreCommits(entry.Title, dkgCommits)
	}

	deals, err := dkgInstance.GetDeals()
	if err != nil {
		return nil, fmt.Errorf("failed to get deals: %w", err)
	}

	operations := make([]client.Operation, 0, len(deals))

	am.dkgInstances[o.DKGIdentifier] = dkgInstance

	// deals variable is a map, so every key is an index of participant we should send a deal
	for index, deal := range deals {
		dealBz, err := json.Marshal(deal)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal deal: %w", err)
		}
		toParticipant := dkgInstance.GetParticipantByIndex(index)
		encryptedDeal, err := am.encryptData(o.DKGIdentifier, toParticipant, dealBz)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt deal: %w", err)
		}
		req := requests.DKGProposalDealConfirmationRequest{
			ParticipantId: dkgInstance.ParticipantID,
			Deal:          encryptedDeal,
			CreatedAt:     o.CreatedAt,
		}
		o.To = toParticipant
		reqBz, err := json.Marshal(req)
		if err != nil {
			return nil, fmt.Errorf("failed to generate fsm request: %w", err)
		}
		o.Result = reqBz
		o.Event = dkg_proposal_fsm.EventDKGDealConfirmationReceived
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
		decryptedDealBz, err := am.decryptData(entry.Deal)
		if err != nil {
			return fmt.Errorf("failed to decrypt deal: %w", err)
		}
		var deal dkgPedersen.Deal
		if err = json.Unmarshal(decryptedDealBz, &deal); err != nil {
			return fmt.Errorf("failed to unmarshal deal")
		}
		dkgInstance.StoreDeal(entry.Title, &deal)
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
		CreatedAt:     o.CreatedAt,
	}

	reqBz, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to generate fsm request: %w", err)
	}

	o.Result = reqBz
	o.Event = dkg_proposal_fsm.EventDKGResponseConfirmationReceived
	return nil
}

func (am *AirgappedMachine) handleStateDkgMasterKeyAwaitConfirmations(o *client.Operation) error {
	var (
		payload responses.DKGProposalResponsesParticipantResponse
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
		var entryResponses []*dkgPedersen.Response
		if err = json.Unmarshal(entry.Responses, &entryResponses); err != nil {
			return fmt.Errorf("failed to unmarshal responses: %w", err)
		}
		dkgInstance.StoreResponses(entry.Title, entryResponses)
	}

	if err = dkgInstance.ProcessResponses(); err != nil {
		return fmt.Errorf("failed to process responses: %w", err)
	}

	//TODO: THIS BLOCK IS WRONG. @oopcode, what exactly should we broadcast?
	///////////////////////////////////////////////////////////////////////////////
	masterPubKey, err := dkgInstance.GetMasterPubKey()
	if err != nil {
		return fmt.Errorf("failed to get master pub key: %w", err)
	}

	masterPubKeyBz, err := json.Marshal(masterPubKey)
	if err != nil {
		return fmt.Errorf("failed to marshal master pub key: %w", err)
	}

	am.dkgInstances[o.DKGIdentifier] = dkgInstance

	req := requests.DKGProposalMasterKeyConfirmationRequest{
		ParticipantId: dkgInstance.ParticipantID,
		MasterKey:     masterPubKeyBz,
		CreatedAt:     o.CreatedAt,
	}
	reqBz, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to generate fsm request: %w", err)
	}
	///////////////////////////////////////////////////////////////////////////////

	o.Result = reqBz
	o.Event = dkg_proposal_fsm.EventDKGMasterKeyConfirmationReceived
	return nil
}

// TODO @oopcode: reconstruct key and sign handlers

func (am *AirgappedMachine) HandleOperation(operation client.Operation) ([]client.Operation, error) {
	var (
		err error
		// output operations (cause of deals)
		operations []client.Operation
	)

	am.Lock()
	defer am.Unlock()

	// handler gets a pointer to an operation, do necessary things
	// and write a result (or an error) to .Result field of operation
	switch fsm.State(operation.Type) {
	case signature_proposal_fsm.StateAwaitParticipantsConfirmations:
		err = am.handleStateAwaitParticipantsConfirmations(&operation)
	case dkg_proposal_fsm.StateDkgCommitsAwaitConfirmations:
		err = am.handleStateDkgCommitsAwaitConfirmations(&operation)
	case dkg_proposal_fsm.StateDkgDealsAwaitConfirmations:
		operations, err = am.handleStateDkgDealsAwaitConfirmations(operation)
	case dkg_proposal_fsm.StateDkgResponsesAwaitConfirmations:
		err = am.handleStateDkgResponsesAwaitConfirmations(&operation)
	case dkg_proposal_fsm.StateDkgMasterKeyAwaitConfirmations:
		err = am.handleStateDkgMasterKeyAwaitConfirmations(&operation)
	default:
		err = fmt.Errorf("invalid operation type: %s", operation.Type)
	}

	// if we have error after handling the operation, we write the error to the operation, so we can feed it to a FSM
	if err != nil {
		log.Println(fmt.Sprintf("failed to handle operation %s, returning response with errot to client: %v",
			operation.Type, err))
		if e := am.writeErrorRequestToOperation(&operation, err); e != nil {
			return nil, fmt.Errorf("failed to write error request to an operation: %w", e)
		}
	}

	if len(operation.Result) > 0 {
		operations = append(operations, operation)
	}

	return operations, nil
}

// HandleQR - gets an operation from a QR code, do necessary things for the operation and returns paths to QR-code images
func (am *AirgappedMachine) HandleQR() ([]string, error) {
	var (
		err error

		// input operation
		operation client.Operation
		qrData    []byte

		// output operations (cause of deals)
		operations []client.Operation
	)

	if qrData, err = am.qrProcessor.ReadQR(); err != nil {
		return nil, fmt.Errorf("failed to read QR: %w", err)
	}
	if err = json.Unmarshal(qrData, &operation); err != nil {
		return nil, fmt.Errorf("failed to unmarshal operation: %w", err)
	}

	if operations, err = am.HandleOperation(operation); err != nil {
		return nil, err
	}

	qrPath := "%s/%s_%s_%s.png"
	qrPaths := make([]string, 0, len(operations))
	for _, o := range operations {
		operationBz, err := json.Marshal(o)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal operation: %w", err)
		}

		if err = am.qrProcessor.WriteQR(fmt.Sprintf(qrPath, resultQRFolder, o.Type, o.ID, o.To), operationBz); err != nil {
			return nil, fmt.Errorf("failed to write QR")
		}
		qrPaths = append(qrPaths, qrPath)
	}

	return qrPaths, nil
}

func (am *AirgappedMachine) writeErrorRequestToOperation(o *client.Operation, handlerError error) error {
	// each type of request should have a required event even error
	// maybe should be global?
	eventToErrorMap := map[fsm.State]fsm.Event{
		dkg_proposal_fsm.StateDkgCommitsAwaitConfirmations:   dkg_proposal_fsm.EventDKGCommitConfirmationError,
		dkg_proposal_fsm.StateDkgDealsAwaitConfirmations:     dkg_proposal_fsm.EventDKGDealConfirmationError,
		dkg_proposal_fsm.StateDkgResponsesAwaitConfirmations: dkg_proposal_fsm.EventDKGResponseConfirmationError,
		dkg_proposal_fsm.StateDkgMasterKeyAwaitConfirmations: dkg_proposal_fsm.EventDKGMasterKeyConfirmationError,
	}
	pid, err := am.getParticipantID(o.DKGIdentifier)
	if err != nil {
		return fmt.Errorf("failed to get participant id: %w", err)
	}
	req := requests.DKGProposalConfirmationErrorRequest{
		Error:         handlerError,
		ParticipantId: pid,
		CreatedAt:     o.CreatedAt,
	}
	errorEvent := eventToErrorMap[fsm.State(o.Type)]
	reqBz, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to generate fsm request: %w", err)
	}
	o.Result = reqBz
	o.Event = errorEvent
	return nil
}
