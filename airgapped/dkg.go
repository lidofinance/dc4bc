package airgapped

import (
	"encoding/json"
	"fmt"

	"github.com/corestario/kyber"
	dkgPedersen "github.com/corestario/kyber/share/dkg/pedersen"
	client "github.com/depools/dc4bc/client/types"
	"github.com/depools/dc4bc/dkg"
	"github.com/depools/dc4bc/fsm/state_machines/dkg_proposal_fsm"
	"github.com/depools/dc4bc/fsm/state_machines/signature_proposal_fsm"
	"github.com/depools/dc4bc/fsm/types/requests"
	"github.com/depools/dc4bc/fsm/types/responses"
	"github.com/depools/dc4bc/storage"
	bls12381 "github.com/depools/kyber-bls12381"
)

func createMessage(o client.Operation, data []byte) storage.Message {
	return storage.Message{
		Event:         string(o.Event),
		Data:          data,
		DkgRoundID:    o.DKGIdentifier,
		RecipientAddr: o.To,
	}
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

	pid := -1
	for _, r := range payload {
		if r.Addr == am.ParticipantAddress {
			pid = r.ParticipantId
			break
		}
	}
	if pid < 0 {
		return fmt.Errorf("failed to determine participant id for DKG with participant address %s", am.ParticipantAddress)
	}

	if _, ok := am.dkgInstances[o.DKGIdentifier]; ok {
		return fmt.Errorf("dkg instance %s already exists", o.DKGIdentifier)
	}

	dkgInstance := dkg.Init(am.suite, am.pubKey, am.secKey)
	dkgInstance.Threshold = payload[0].Threshold //same for everyone
	dkgInstance.N = len(payload)
	am.dkgInstances[o.DKGIdentifier] = dkgInstance

	req := requests.SignatureProposalParticipantRequest{
		ParticipantId: pid,
		CreatedAt:     o.CreatedAt,
	}
	reqBz, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to generate fsm request: %w", err)
	}

	o.Event = signature_proposal_fsm.EventConfirmSignatureProposal
	o.ResultMsgs = append(o.ResultMsgs, createMessage(*o, reqBz))
	return nil
}

func (am *AirgappedMachine) GetPubKey() kyber.Point {
	return am.pubKey
}

func (am *AirgappedMachine) handleStateDkgCommitsAwaitConfirmations(o *client.Operation) error {
	var (
		payload responses.DKGProposalPubKeysParticipantResponse
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
		pubKey := bls12381.NewBLS12381Suite().Point()
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

	o.Event = dkg_proposal_fsm.EventDKGCommitConfirmationReceived
	o.ResultMsgs = append(o.ResultMsgs, createMessage(*o, reqBz))
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
		var commitsBz [][]byte
		if err = json.Unmarshal(entry.DkgCommit, &commitsBz); err != nil {
			return fmt.Errorf("failed to unmarshal commits: %w", err)
		}
		dkgCommits := make([]kyber.Point, 0, len(commitsBz))
		for _, commitBz := range commitsBz {
			commit := am.suite.Point()
			if err = commit.UnmarshalBinary(commitBz); err != nil {
				return fmt.Errorf("failed to unmarshal commit: %w", err)
			}
			dkgCommits = append(dkgCommits, commit)
		}
		dkgInstance.StoreCommits(entry.Addr, dkgCommits)
	}

	deals, err := dkgInstance.GetDeals()
	if err != nil {
		return fmt.Errorf("failed to get deals: %w", err)
	}

	am.dkgInstances[o.DKGIdentifier] = dkgInstance

	// deals variable is a map, so every key is an index of participant we should send a deal
	for index, deal := range deals {
		dealBz, err := json.Marshal(deal)
		if err != nil {
			return fmt.Errorf("failed to marshal deal: %w", err)
		}
		toParticipant := dkgInstance.GetParticipantByIndex(index)
		encryptedDeal, err := am.encryptDataForParticipant(o.DKGIdentifier, toParticipant, dealBz)
		if err != nil {
			return fmt.Errorf("failed to encrypt deal: %w", err)
		}
		req := requests.DKGProposalDealConfirmationRequest{
			ParticipantId: dkgInstance.ParticipantID,
			Deal:          encryptedDeal,
			CreatedAt:     o.CreatedAt,
		}
		o.To = toParticipant
		reqBz, err := json.Marshal(req)
		if err != nil {
			return fmt.Errorf("failed to generate fsm request: %w", err)
		}
		o.Event = dkg_proposal_fsm.EventDKGDealConfirmationReceived
		o.ResultMsgs = append(o.ResultMsgs, createMessage(*o, reqBz))
	}
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
		decryptedDealBz, err := am.decryptDataFromParticipant(entry.DkgDeal)
		if err != nil {
			return fmt.Errorf("failed to decrypt deal: %w", err)
		}
		var deal dkgPedersen.Deal
		if err = json.Unmarshal(decryptedDealBz, &deal); err != nil {
			return fmt.Errorf("failed to unmarshal deal")
		}
		dkgInstance.StoreDeal(entry.Addr, &deal)
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

	o.Event = dkg_proposal_fsm.EventDKGResponseConfirmationReceived
	o.ResultMsgs = append(o.ResultMsgs, createMessage(*o, reqBz))
	return nil
}

func (am *AirgappedMachine) handleStateDkgMasterKeyAwaitConfirmations(o *client.Operation) error {
	var (
		payload responses.DKGProposalResponseParticipantResponse
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
		if err = json.Unmarshal(entry.DkgResponse, &entryResponses); err != nil {
			return fmt.Errorf("failed to unmarshal responses: %w", err)
		}
		dkgInstance.StoreResponses(entry.Addr, entryResponses)
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

	o.Event = dkg_proposal_fsm.EventDKGMasterKeyConfirmationReceived
	o.ResultMsgs = append(o.ResultMsgs, createMessage(*o, reqBz))

	return nil
}
