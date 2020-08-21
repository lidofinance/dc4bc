package airgapped

import (
	"encoding/json"
	"fmt"
	client "github.com/depools/dc4bc/client/types"
	"github.com/depools/dc4bc/dkg"
	"github.com/depools/dc4bc/fsm/state_machines/dkg_proposal_fsm"
	"github.com/depools/dc4bc/fsm/state_machines/signature_proposal_fsm"
	"github.com/depools/dc4bc/fsm/types/requests"
	"github.com/depools/dc4bc/fsm/types/responses"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/pairing/bn256"
	dkgPedersen "go.dedis.ch/kyber/v3/share/dkg/pedersen"
	"log"
)

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

	pid := 0

	for _, r := range payload {
		if r.Addr == am.ParticipantAddress {
			pid = r.ParticipantId
		}
	}

	req := requests.SignatureProposalParticipantRequest{
		ParticipantId: pid,
		CreatedAt:     o.CreatedAt,
	}
	reqBz, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to generate fsm request: %w", err)
	}
	o.Result = reqBz
	o.Event = signature_proposal_fsm.EventConfirmSignatureProposal

	return nil
}

func (am *AirgappedMachine) GetPubKey() kyber.Point {
	return am.pubKey
}

func (am *AirgappedMachine) handleStateDkgCommitsAwaitConfirmations(o *client.Operation) error {
	var (
		payload responses.SignatureProposalParticipantStatusResponse
		err     error
	)

	dkgInstance, ok := am.dkgInstances[o.DKGIdentifier]
	if !ok {
		log.Printf("dkg instance for dkg round %s does not exist, create new\n", o.DKGIdentifier)
		dkgInstance = dkg.Init(am.suite, am.pubKey, am.secKey)
	}

	dkgInstance.Threshold = len(payload)

	if err = json.Unmarshal(o.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	for _, entry := range payload {
		pubKey := bn256.NewSuiteG2().Point()
		if err = pubKey.UnmarshalBinary(entry.DkgPubKey); err != nil {
			return fmt.Errorf("failed to unmarshal pubkey: %w", err)
		}
		dkgInstance.StorePubKey(entry.Addr, entry.ParticipantId, pubKey)
	}

	if err = dkgInstance.InitDKGInstance(); err != nil {
		return fmt.Errorf("failed to init dkg instance: %w", err)
	}

	dkgCommits := dkgInstance.GetCommits()
	marshaledCommits := make([][]byte, 0, len(dkgCommits))
	for _, commit := range dkgCommits {
		commitBz, err := commit.MarshalBinary()
		if err != nil {
			return fmt.Errorf("failed to marshal commits: %w", err)
		}
		marshaledCommits = append(marshaledCommits, commitBz)
	}
	commitsBz, err := json.Marshal(marshaledCommits)

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
		var commitsBz [][]byte
		if err = json.Unmarshal(entry.Commit, &commitsBz); err != nil {
			return nil, fmt.Errorf("failed to unmarshal commits: %w", err)
		}
		dkgCommits := make([]kyber.Point, 0, len(commitsBz))
		for _, commitBz := range commitsBz {
			commit := am.suite.Point()
			if err = commit.UnmarshalBinary(commitBz); err != nil {
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

	pubKey, err := dkgInstance.GetDistributedPublicKey()
	if err != nil {
		return fmt.Errorf("failed to get master pub key: %w", err)
	}

	masterPubKeyBz, err := pubKey.MarshalBinary()
	if err != nil {
		return fmt.Errorf("failed to marshal master pub key: %w", err)
	}

	am.dkgInstances[o.DKGIdentifier] = dkgInstance

	blsKeyring, err := dkgInstance.GetBLSKeyring()
	if err != nil {
		return fmt.Errorf("failed to get BLSKeyring: %w", err)
	}

	if err = am.saveBLSKeyring(o.DKGIdentifier, blsKeyring); err != nil {
		return fmt.Errorf("failed to save BLSKeyring: %w", err)
	}

	req := requests.DKGProposalMasterKeyConfirmationRequest{
		ParticipantId: dkgInstance.ParticipantID,
		MasterKey:     masterPubKeyBz,
		CreatedAt:     o.CreatedAt,
	}
	reqBz, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to generate fsm request: %w", err)
	}

	o.Result = reqBz
	o.Event = dkg_proposal_fsm.EventDKGMasterKeyConfirmationReceived
	return nil
}
