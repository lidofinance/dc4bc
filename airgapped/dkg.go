package airgapped

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	bls "github.com/corestario/kyber/pairing/bls12381"

	"github.com/corestario/kyber"
	dkgPedersen "github.com/corestario/kyber/share/dkg/pedersen"
	client "github.com/lidofinance/dc4bc/client/types"
	"github.com/lidofinance/dc4bc/dkg"
	"github.com/lidofinance/dc4bc/fsm/state_machines/dkg_proposal_fsm"
	"github.com/lidofinance/dc4bc/fsm/state_machines/signature_proposal_fsm"
	"github.com/lidofinance/dc4bc/fsm/types/requests"
	"github.com/lidofinance/dc4bc/fsm/types/responses"
	"github.com/lidofinance/dc4bc/storage"
)

func createMessage(o client.Operation, data []byte) storage.Message {
	return storage.Message{
		Event:         string(o.Event),
		Data:          data,
		DkgRoundID:    o.DKGIdentifier,
		RecipientAddr: o.To,
	}
}

// handleStateAwaitParticipantsConfirmations inits DKG instance for a new DKG round and returns a confirmation of
// participation in the round
func (am *Machine) handleStateAwaitParticipantsConfirmations(o *client.Operation) error {
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
		pubkey := am.baseSuite.Point()
		if err := pubkey.UnmarshalBinary(r.DkgPubKey); err != nil {
			return fmt.Errorf("failed to unmarshal dkg pubkey: %w", err)
		}
		if am.pubKey.Equal(pubkey) {
			pid = r.ParticipantId
			break
		}
	}
	if pid < 0 {
		return fmt.Errorf("failed to determine participant id for DKG #%s", o.DKGIdentifier)
	}

	if _, ok := am.dkgInstances[o.DKGIdentifier]; ok {
		return fmt.Errorf("dkg instance %s already exists", o.DKGIdentifier)
	}

	// Here we create a new seeded suite for the new DKG round with seed =
	// sha256.Sum256(baseSeed + DKGIdentifier). We need this to avoid identical
	// DKG rounds.
	var (
		dkgSeed = sha256.Sum256(append([]byte(o.DKGIdentifier), am.baseSeed...))
		suite   = bls.NewBLS12381Suite(dkgSeed[:])
	)
	dkgInstance := dkg.Init(suite, am.pubKey, am.secKey)
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

func (am *Machine) GetPubKey() kyber.Point {
	return am.pubKey
}

// handleStateDkgCommitsAwaitConfirmations takes a list of participants DKG pub keys as payload and
// returns DKG commits to broadcast
func (am *Machine) handleStateDkgCommitsAwaitConfirmations(o *client.Operation) error {
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
		pubKey := am.baseSuite.Point()
		if err = pubKey.UnmarshalBinary(entry.DkgPubKey); err != nil {
			return fmt.Errorf("failed to unmarshal pubkey: %w", err)
		}
		dkgInstance.StorePubKey(entry.Username, entry.ParticipantId, pubKey)
	}

	if err = dkgInstance.InitDKGInstance(am.baseSeed); err != nil {
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
	if err != nil {
		return fmt.Errorf("failed to marshal marshaledCommits: %w", err)
	}

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

// handleStateDkgDealsAwaitConfirmations takes broadcasted participants commits as payload and
// returns a private deal for every participant.
// Each deal is encrypted with a participant's public key which received on the previous step
func (am *Machine) handleStateDkgDealsAwaitConfirmations(o *client.Operation) error {
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
			commit := am.baseSuite.Point()
			if err = commit.UnmarshalBinary(commitBz); err != nil {
				return fmt.Errorf("failed to unmarshal commit: %w", err)
			}
			dkgCommits = append(dkgCommits, commit)
		}
		dkgInstance.StoreCommits(entry.Username, dkgCommits)
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

// handleStateDkgResponsesAwaitConfirmations takes deals sent to us as payload, decrypt and process them and
// returns responses to broadcast
func (am *Machine) handleStateDkgResponsesAwaitConfirmations(o *client.Operation) error {
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
		dkgInstance.StoreDeal(entry.Username, &deal)
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

// handleStateDkgMasterKeyAwaitConfirmations takes broadcasted responses from the previous step, process them,
// reconstructs a distributed DKG public key to broadcast and saves a private part of the key
func (am *Machine) handleStateDkgMasterKeyAwaitConfirmations(o *client.Operation) error {
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
		dkgInstance.StoreResponses(entry.Username, entryResponses)
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
