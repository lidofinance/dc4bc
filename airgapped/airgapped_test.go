package airgapped

import (
	"bytes"
	"encoding/json"
	"fmt"
	prysmBLS "github.com/prysmaticlabs/prysm/shared/bls"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/google/uuid"
	client "github.com/lidofinance/dc4bc/client/types"
	"github.com/lidofinance/dc4bc/fsm/fsm"
	"github.com/lidofinance/dc4bc/fsm/state_machines/dkg_proposal_fsm"
	"github.com/lidofinance/dc4bc/fsm/state_machines/signature_proposal_fsm"
	"github.com/lidofinance/dc4bc/fsm/state_machines/signing_proposal_fsm"
	"github.com/lidofinance/dc4bc/fsm/types/requests"
	"github.com/lidofinance/dc4bc/fsm/types/responses"
	"github.com/lidofinance/dc4bc/storage"
)

const (
	DKGIdentifier = "dkg_identifier"
	testDB        = "test_level_db"
)

type Node struct {
	ParticipantID           int
	Participant             string
	Machine                 *Machine
	commits                 []requests.DKGProposalCommitConfirmationRequest
	deals                   []requests.DKGProposalDealConfirmationRequest
	responses               []requests.DKGProposalResponseConfirmationRequest
	masterKeys              []requests.DKGProposalMasterKeyConfirmationRequest
	partialSigns            []requests.SigningProposalPartialSignRequest
	reconstructedSignatures []client.ReconstructedSignature
}

func (n *Node) storeOperation(t *testing.T, msg storage.Message) {
	switch fsm.Event(msg.Event) {
	case dkg_proposal_fsm.EventDKGCommitConfirmationReceived:
		var req requests.DKGProposalCommitConfirmationRequest
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			t.Fatalf("failed to unmarshal fsm req: %v", err)
		}
		n.commits = append(n.commits, req)
	case dkg_proposal_fsm.EventDKGDealConfirmationReceived:
		var req requests.DKGProposalDealConfirmationRequest
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			t.Fatalf("failed to unmarshal fsm req: %v", err)
		}
		n.deals = append(n.deals, req)
	case dkg_proposal_fsm.EventDKGResponseConfirmationReceived:
		var req requests.DKGProposalResponseConfirmationRequest
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			t.Fatalf("failed to unmarshal fsm req: %v", err)
		}
		n.responses = append(n.responses, req)
	case dkg_proposal_fsm.EventDKGMasterKeyConfirmationReceived:
		var req requests.DKGProposalMasterKeyConfirmationRequest
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			t.Fatalf("failed to unmarshal fsm req: %v", err)
		}
		n.masterKeys = append(n.masterKeys, req)
	case signing_proposal_fsm.EventSigningPartialSignReceived:
		var req requests.SigningProposalPartialSignRequest
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			t.Fatalf("failed to unmarshal fsm req: %v", err)
		}
		n.partialSigns = append(n.partialSigns, req)
	case client.SignatureReconstructed:
		var req client.ReconstructedSignature
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			t.Fatalf("failed to unmarshal fsm req: %v", err)
		}
		n.reconstructedSignatures = append(n.reconstructedSignatures, req)
	default:
		t.Fatalf("invalid event: %s", msg.Event)
	}
}

type Transport struct {
	nodes []*Node
}

func (tr *Transport) BroadcastMessage(t *testing.T, msg storage.Message) {
	for _, node := range tr.nodes {
		if msg.RecipientAddr == "" || msg.RecipientAddr == node.Participant {
			node.storeOperation(t, msg)
		}
	}
}

func createOperation(t *testing.T, opType string, to string, req interface{}) client.Operation {
	reqBz, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}
	op := client.Operation{
		ID:            uuid.New().String(),
		Type:          client.OperationType(opType),
		Payload:       reqBz,
		CreatedAt:     time.Now(),
		DKGIdentifier: DKGIdentifier,
		To:            to,
	}
	return op
}

func TestAirgappedAllSteps(t *testing.T) {
	testDir := "/tmp/airgapped_test"
	nodesCount := 10
	threshold := 3
	participants := make([]string, nodesCount)
	for i := 0; i < nodesCount; i++ {
		participants[i] = fmt.Sprintf("Participant#%d", i)
	}

	tr := &Transport{}
	for i := 0; i < nodesCount; i++ {
		am, err := NewMachine(fmt.Sprintf("%s/%s-%d", testDir, testDB, i))
		if err != nil {
			t.Fatalf("failed to create airgapped machine: %v", err)
		}
		am.SetEncryptionKey([]byte(fmt.Sprintf(testDB+"%d", i)))
		if err = am.InitKeys(); err != nil {
			t.Fatalf(err.Error())
		}
		node := Node{
			ParticipantID: i,
			Participant:   participants[i],
			Machine:       am,
		}
		tr.nodes = append(tr.nodes, &node)
	}
	defer os.RemoveAll(testDir)

	var initReq responses.SignatureProposalParticipantInvitationsResponse
	for _, n := range tr.nodes {
		pubKey, err := n.Machine.pubKey.MarshalBinary()
		if err != nil {
			t.Fatalf("failed to marshal dkg pubkey: %v", err)
		}
		entry := &responses.SignatureProposalParticipantInvitationEntry{
			ParticipantId: n.ParticipantID,
			Username:      n.Participant,
			Threshold:     threshold,
			DkgPubKey:     pubKey,
		}
		initReq = append(initReq, entry)
	}
	op := createOperation(t, string(signature_proposal_fsm.StateAwaitParticipantsConfirmations), "", initReq)
	runStep(tr, func(n *Node, wg *sync.WaitGroup) {
		defer wg.Done()

		operation, err := n.Machine.GetOperationResult(op)
		if err != nil {
			t.Fatalf("%s: failed to handle operation %s: %v", n.Participant, op.Type, err)
		}
		if err := n.Machine.storeOperation(operation); err != nil {
			t.Fatalf("failed to storeOperation: %v", err)
		}
	})

	// get commits
	var getCommitsRequest responses.DKGProposalPubKeysParticipantResponse
	for _, n := range tr.nodes {
		pubKey, err := n.Machine.pubKey.MarshalBinary()
		if err != nil {
			t.Fatalf("%s: failed to marshal pubkey: %v", n.Participant, err)
		}
		entry := &responses.DKGProposalPubKeysParticipantEntry{
			ParticipantId: n.ParticipantID,
			Username:      n.Participant,
			DkgPubKey:     pubKey,
		}
		getCommitsRequest = append(getCommitsRequest, entry)
	}
	op = createOperation(t, string(dkg_proposal_fsm.StateDkgCommitsAwaitConfirmations), "", getCommitsRequest)
	runStep(tr, func(n *Node, wg *sync.WaitGroup) {
		defer wg.Done()

		operation, err := n.Machine.GetOperationResult(op)
		if err != nil {
			t.Fatalf("%s: failed to handle operation %s: %v", n.Participant, op.Type, err)
		}
		if err := n.Machine.storeOperation(operation); err != nil {
			t.Fatalf("failed to storeOperation: %v", err)
		}
		for _, msg := range operation.ResultMsgs {
			tr.BroadcastMessage(t, msg)
		}
	})

	//deals
	runStep(tr, func(n *Node, wg *sync.WaitGroup) {
		defer wg.Done()

		var payload responses.DKGProposalCommitParticipantResponse
		for _, req := range n.commits {
			p := responses.DKGProposalCommitParticipantEntry{
				ParticipantId: req.ParticipantId,
				Username:      fmt.Sprintf("Participant#%d", req.ParticipantId),
				DkgCommit:     req.Commit,
			}
			payload = append(payload, &p)
		}
		op := createOperation(t, string(dkg_proposal_fsm.StateDkgDealsAwaitConfirmations), "", payload)

		operation, err := n.Machine.GetOperationResult(op)
		if err != nil {
			t.Fatalf("%s: failed to handle operation %s: %v", n.Participant, op.Type, err)
		}
		if err := n.Machine.storeOperation(operation); err != nil {
			t.Fatalf("failed to storeOperation: %v", err)
		}
		for _, msg := range operation.ResultMsgs {
			tr.BroadcastMessage(t, msg)
		}
	})

	//responses
	runStep(tr, func(n *Node, wg *sync.WaitGroup) {
		defer wg.Done()

		var payload responses.DKGProposalDealParticipantResponse
		for _, req := range n.deals {
			p := responses.DKGProposalDealParticipantEntry{
				ParticipantId: req.ParticipantId,
				Username:      fmt.Sprintf("Participant#%d", req.ParticipantId),
				DkgDeal:       req.Deal,
			}
			payload = append(payload, &p)
		}
		op := createOperation(t, string(dkg_proposal_fsm.StateDkgResponsesAwaitConfirmations), "", payload)

		operation, err := n.Machine.GetOperationResult(op)
		if err != nil {
			t.Fatalf("%s: failed to handle operation %s: %v", n.Participant, op.Type, err)
		}
		if err := n.Machine.storeOperation(operation); err != nil {
			t.Fatalf("failed to storeOperation: %v", err)
		}
		for _, msg := range operation.ResultMsgs {
			tr.BroadcastMessage(t, msg)
		}
	})

	//master key
	runStep(tr, func(n *Node, wg *sync.WaitGroup) {
		defer wg.Done()

		var payload responses.DKGProposalResponseParticipantResponse
		for _, req := range n.responses {
			p := responses.DKGProposalResponseParticipantEntry{
				ParticipantId: req.ParticipantId,
				Username:      fmt.Sprintf("Participant#%d", req.ParticipantId),
				DkgResponse:   req.Response,
			}
			payload = append(payload, &p)
		}
		op := createOperation(t, string(dkg_proposal_fsm.StateDkgMasterKeyAwaitConfirmations), "", payload)

		operation, err := n.Machine.GetOperationResult(op)
		if err != nil {
			t.Fatalf("%s: failed to handle operation %s: %v", n.Participant, op.Type, err)
		}
		if err := n.Machine.storeOperation(operation); err != nil {
			t.Fatalf("failed to storeOperation: %v", err)
		}
		for _, msg := range operation.ResultMsgs {
			tr.BroadcastMessage(t, msg)
		}
	})

	// check that all master keys are equal
	for _, n := range tr.nodes {
		for i := 0; i < len(n.masterKeys); i++ {
			if !bytes.Equal(n.masterKeys[0].MasterKey, n.masterKeys[i].MasterKey) {
				t.Fatalf("master keys is not equal!")
			}
		}
	}

	msgToSign := []byte("i am a message")

	//partialSigns
	runStep(tr, func(n *Node, wg *sync.WaitGroup) {
		defer wg.Done()

		payload := responses.SigningPartialSignsParticipantInvitationsResponse{
			SrcPayload: msgToSign,
		}

		op := createOperation(t, string(signing_proposal_fsm.StateSigningAwaitPartialSigns), "", payload)

		operation, err := n.Machine.GetOperationResult(op)
		if err != nil {
			t.Fatalf("%s: failed to handle operation %s: %v", n.Participant, op.Type, err)
		}
		if err := n.Machine.storeOperation(operation); err != nil {
			t.Fatalf("failed to storeOperation: %v", err)
		}
		for _, msg := range operation.ResultMsgs {
			tr.BroadcastMessage(t, msg)
		}
	})

	//recover full signature
	runStep(tr, func(n *Node, wg *sync.WaitGroup) {
		defer wg.Done()

		var payload responses.SigningProcessParticipantResponse
		for _, req := range n.partialSigns {
			p := responses.SigningProcessParticipantEntry{
				ParticipantId: req.ParticipantId,
				Username:      fmt.Sprintf("Participant#%d", req.ParticipantId),
				PartialSign:   req.PartialSign,
			}
			payload.Participants = append(payload.Participants, &p)
		}
		payload.SrcPayload = msgToSign
		op := createOperation(t, string(signing_proposal_fsm.StateSigningPartialSignsCollected), "", payload)

		operation, err := n.Machine.GetOperationResult(op)
		if err != nil {
			t.Fatalf("%s: failed to handle operation %s: %v", n.Participant, op.Type, err)
		}
		if err := n.Machine.storeOperation(operation); err != nil {
			t.Fatalf("failed to storeOperation: %v", err)
		}
		for _, msg := range operation.ResultMsgs {
			tr.BroadcastMessage(t, msg)
		}
	})

	//verify signatures
	for _, n := range tr.nodes {
		for i := 0; i < len(n.reconstructedSignatures); i++ {
			if !bytes.Equal(n.reconstructedSignatures[0].Signature, n.reconstructedSignatures[i].Signature) {
				t.Fatalf("signatures are not equal!")
			}
			if err := n.Machine.VerifySign(msgToSign, n.reconstructedSignatures[i].Signature, DKGIdentifier); err != nil {
				t.Fatal("signature is not verified!")
			}
		}
	}

	//keys and signatures are equal, so let's test it on prysm compatibility
	testKyberPrysm(t, tr.nodes[0].masterKeys[0].MasterKey, tr.nodes[0].reconstructedSignatures[0].Signature, msgToSign)

	fmt.Println("DKG succeeded, signature recovered and verified")
}

func TestAirgappedMachine_Replay(t *testing.T) {
	testDir := "/tmp/airgapped_test"
	nodesCount := 2
	threshold := 2
	participants := make([]string, nodesCount)
	for i := 0; i < nodesCount; i++ {
		participants[i] = fmt.Sprintf("Participant#%d", i)
	}

	tr := &Transport{}
	for i := 0; i < nodesCount; i++ {
		am, err := NewMachine(fmt.Sprintf("%s/%s-%d", testDir, testDB, i))
		if err != nil {
			t.Fatalf("failed to create airgapped machine: %v", err)
		}
		am.SetEncryptionKey([]byte(fmt.Sprintf(testDB+"%d", i)))
		if err = am.InitKeys(); err != nil {
			t.Fatalf(err.Error())
		}
		node := Node{
			ParticipantID: i,
			Participant:   participants[i],
			Machine:       am,
		}
		tr.nodes = append(tr.nodes, &node)
	}
	defer os.RemoveAll(testDir)

	var initReq responses.SignatureProposalParticipantInvitationsResponse
	for _, n := range tr.nodes {
		pubKey, err := n.Machine.pubKey.MarshalBinary()
		if err != nil {
			t.Fatalf("failed to marshal dkg pubkey: %v", err)
		}
		entry := &responses.SignatureProposalParticipantInvitationEntry{
			ParticipantId: n.ParticipantID,
			Username:      n.Participant,
			Threshold:     threshold,
			DkgPubKey:     pubKey,
		}
		initReq = append(initReq, entry)
	}
	op := createOperation(t, string(signature_proposal_fsm.StateAwaitParticipantsConfirmations), "", initReq)
	runStep(tr, func(n *Node, wg *sync.WaitGroup) {
		defer wg.Done()

		operation, err := n.Machine.GetOperationResult(op)
		if err != nil {
			t.Fatalf("%s: failed to handle operation %s: %v", n.Participant, op.Type, err)
		}
		if err := n.Machine.storeOperation(operation); err != nil {
			t.Fatalf("failed to storeOperation: %v", err)
		}
	})

	// get commits
	var getCommitsRequest responses.DKGProposalPubKeysParticipantResponse
	for _, n := range tr.nodes {
		pubKey, err := n.Machine.pubKey.MarshalBinary()
		if err != nil {
			t.Fatalf("%s: failed to marshal pubkey: %v", n.Participant, err)
		}
		entry := &responses.DKGProposalPubKeysParticipantEntry{
			ParticipantId: n.ParticipantID,
			Username:      n.Participant,
			DkgPubKey:     pubKey,
		}
		getCommitsRequest = append(getCommitsRequest, entry)
	}
	op = createOperation(t, string(dkg_proposal_fsm.StateDkgCommitsAwaitConfirmations), "", getCommitsRequest)
	runStep(tr, func(n *Node, wg *sync.WaitGroup) {
		defer wg.Done()

		operation, err := n.Machine.GetOperationResult(op)
		if err != nil {
			t.Fatalf("%s: failed to handle operation %s: %v", n.Participant, op.Type, err)
		}
		if err := n.Machine.storeOperation(operation); err != nil {
			t.Fatalf("failed to storeOperation: %v", err)
		}
		for _, msg := range operation.ResultMsgs {
			tr.BroadcastMessage(t, msg)
		}
	})

	//deals
	runStep(tr, func(n *Node, wg *sync.WaitGroup) {
		defer wg.Done()

		var payload responses.DKGProposalCommitParticipantResponse
		for _, req := range n.commits {
			p := responses.DKGProposalCommitParticipantEntry{
				ParticipantId: req.ParticipantId,
				Username:      fmt.Sprintf("Participant#%d", req.ParticipantId),
				DkgCommit:     req.Commit,
			}
			payload = append(payload, &p)
		}
		op := createOperation(t, string(dkg_proposal_fsm.StateDkgDealsAwaitConfirmations), "", payload)

		operation, err := n.Machine.GetOperationResult(op)
		if err != nil {
			t.Fatalf("%s: failed to handle operation %s: %v", n.Participant, op.Type, err)
		}
		if err := n.Machine.storeOperation(operation); err != nil {
			t.Fatalf("failed to storeOperation: %v", err)
		}
		for _, msg := range operation.ResultMsgs {
			tr.BroadcastMessage(t, msg)
		}
	})

	// At this point something goes wrong and we have to restart the machines.
	for _, node := range tr.nodes {
		_ = node.Machine.db.Close()
	}

	participants = make([]string, nodesCount)
	for i := 0; i < nodesCount; i++ {
		participants[i] = fmt.Sprintf("Participant#%d", i)
	}

	newTr := &Transport{}
	for i := 0; i < nodesCount; i++ {
		am, err := NewMachine(fmt.Sprintf("%s/%s-%d", testDir, testDB, i))
		if err != nil {
			t.Fatalf("failed to create airgapped machine: %v", err)
		}
		am.SetEncryptionKey([]byte(fmt.Sprintf(testDB+"%d", i)))
		if err = am.InitKeys(); err != nil {
			t.Fatalf(err.Error())
		}
		node := Node{
			ParticipantID: i,
			Participant:   participants[i],
			Machine:       am,
			deals:         tr.nodes[i].deals,
		}
		newTr.nodes = append(newTr.nodes, &node)
	}
	defer os.RemoveAll(testDir)

	for _, node := range newTr.nodes {
		err := node.Machine.ReplayOperationsLog(DKGIdentifier)
		require.NoError(t, err)
	}

	//oldTr := tr
	tr = newTr

	//responses
	runStep(tr, func(n *Node, wg *sync.WaitGroup) {
		defer wg.Done()

		var payload responses.DKGProposalDealParticipantResponse
		for _, req := range n.deals {
			p := responses.DKGProposalDealParticipantEntry{
				ParticipantId: req.ParticipantId,
				Username:      fmt.Sprintf("Participant#%d", req.ParticipantId),
				DkgDeal:       req.Deal,
			}
			payload = append(payload, &p)
		}
		op := createOperation(t, string(dkg_proposal_fsm.StateDkgResponsesAwaitConfirmations), "", payload)

		operation, err := n.Machine.GetOperationResult(op)
		if err != nil {
			t.Fatalf("%s: failed to handle operation %s: %v", n.Participant, op.Type, err)
		}
		if err := n.Machine.storeOperation(operation); err != nil {
			t.Fatalf("failed to storeOperation: %v", err)
		}
		for _, msg := range operation.ResultMsgs {
			tr.BroadcastMessage(t, msg)
		}
	})

	//master key
	runStep(tr, func(n *Node, wg *sync.WaitGroup) {
		defer wg.Done()

		var payload responses.DKGProposalResponseParticipantResponse
		for _, req := range n.responses {
			p := responses.DKGProposalResponseParticipantEntry{
				ParticipantId: req.ParticipantId,
				Username:      fmt.Sprintf("Participant#%d", req.ParticipantId),
				DkgResponse:   req.Response,
			}
			payload = append(payload, &p)
		}
		op := createOperation(t, string(dkg_proposal_fsm.StateDkgMasterKeyAwaitConfirmations), "", payload)

		operation, err := n.Machine.GetOperationResult(op)
		if err != nil {
			t.Fatalf("%s: failed to handle operation %s: %v", n.Participant, op.Type, err)
		}
		if err := n.Machine.storeOperation(operation); err != nil {
			t.Fatalf("failed to storeOperation: %v", err)
		}
		for _, msg := range operation.ResultMsgs {
			tr.BroadcastMessage(t, msg)
		}
	})

	// check that all master keys are equal
	for _, n := range tr.nodes {
		for i := 0; i < len(n.masterKeys); i++ {
			if !bytes.Equal(n.masterKeys[0].MasterKey, n.masterKeys[i].MasterKey) {
				t.Fatalf("master keys is not equal!")
			}
		}
	}

	msgToSign := []byte("i am a message")

	//partialSigns
	runStep(tr, func(n *Node, wg *sync.WaitGroup) {
		defer wg.Done()

		payload := responses.SigningPartialSignsParticipantInvitationsResponse{
			SrcPayload: msgToSign,
		}

		op := createOperation(t, string(signing_proposal_fsm.StateSigningAwaitPartialSigns), "", payload)

		operation, err := n.Machine.GetOperationResult(op)
		if err != nil {
			t.Fatalf("%s: failed to handle operation %s: %v", n.Participant, op.Type, err)
		}
		if err := n.Machine.storeOperation(operation); err != nil {
			t.Fatalf("failed to storeOperation: %v", err)
		}
		for _, msg := range operation.ResultMsgs {
			tr.BroadcastMessage(t, msg)
		}
	})

	//recover full signature
	runStep(tr, func(n *Node, wg *sync.WaitGroup) {
		defer wg.Done()

		var payload responses.SigningProcessParticipantResponse
		for _, req := range n.partialSigns {
			p := responses.SigningProcessParticipantEntry{
				ParticipantId: req.ParticipantId,
				Username:      fmt.Sprintf("Participant#%d", req.ParticipantId),
				PartialSign:   req.PartialSign,
			}
			payload.Participants = append(payload.Participants, &p)
		}
		payload.SrcPayload = msgToSign
		op := createOperation(t, string(signing_proposal_fsm.StateSigningPartialSignsCollected), "", payload)

		operation, err := n.Machine.GetOperationResult(op)
		if err != nil {
			t.Fatalf("%s: failed to handle operation %s: %v", n.Participant, op.Type, err)
		}
		if err := n.Machine.storeOperation(operation); err != nil {
			t.Fatalf("failed to storeOperation: %v", err)
		}
		for _, msg := range operation.ResultMsgs {
			tr.BroadcastMessage(t, msg)
		}
	})

	//verify signatures
	for _, n := range tr.nodes {
		for i := 0; i < len(n.reconstructedSignatures); i++ {
			if !bytes.Equal(n.reconstructedSignatures[0].Signature, n.reconstructedSignatures[i].Signature) {
				t.Fatalf("signatures are not equal!")
			}
			if err := n.Machine.VerifySign(msgToSign, n.reconstructedSignatures[i].Signature, DKGIdentifier); err != nil {
				t.Fatal("signature is not verified!")
			}
		}
	}

	//keys and signatures are equal, so let's test it on prysm compatibility
	testKyberPrysm(t, tr.nodes[0].masterKeys[0].MasterKey, tr.nodes[0].reconstructedSignatures[0].Signature, msgToSign)

	fmt.Println("DKG succeeded, signature recovered and verified")
}

func TestAirgappedMachine_ClearOperations(t *testing.T) {
	testDir := "/tmp/airgapped_test"
	nodesCount := 2
	threshold := 2

	if err := os.RemoveAll(testDir); err != nil {
		t.Fatal("failed to remove test dir:", err.Error())
	}

	participants := make([]string, nodesCount)
	for i := 0; i < nodesCount; i++ {
		participants[i] = fmt.Sprintf("Participant#%d", i)
	}

	tr := &Transport{}
	for i := 0; i < nodesCount; i++ {
		am, err := NewMachine(fmt.Sprintf("%s/%s-%d", testDir, testDB, i))
		if err != nil {
			t.Fatalf("failed to create airgapped machine: %v", err)
		}
		am.SetEncryptionKey([]byte(fmt.Sprintf(testDB+"%d", i)))
		if err = am.InitKeys(); err != nil {
			t.Fatalf(err.Error())
		}
		node := Node{
			ParticipantID: i,
			Participant:   participants[i],
			Machine:       am,
		}
		tr.nodes = append(tr.nodes, &node)
	}
	defer os.RemoveAll(testDir)

	var initReq responses.SignatureProposalParticipantInvitationsResponse
	for _, n := range tr.nodes {
		pubKey, err := n.Machine.pubKey.MarshalBinary()
		if err != nil {
			t.Fatalf("failed to marshal dkg pubkey: %v", err)
		}
		entry := &responses.SignatureProposalParticipantInvitationEntry{
			ParticipantId: n.ParticipantID,
			Username:      n.Participant,
			Threshold:     threshold,
			DkgPubKey:     pubKey,
		}
		initReq = append(initReq, entry)
	}
	op := createOperation(t, string(signature_proposal_fsm.StateAwaitParticipantsConfirmations), "", initReq)
	runStep(tr, func(n *Node, wg *sync.WaitGroup) {
		defer wg.Done()

		operation, err := n.Machine.GetOperationResult(op)
		if err != nil {
			t.Fatalf("%s: failed to handle operation %s: %v", n.Participant, op.Type, err)
		}
		if err := n.Machine.storeOperation(operation); err != nil {
			t.Fatalf("failed to storeOperation: %v", err)
		}
	})

	// get commits
	var getCommitsRequest responses.DKGProposalPubKeysParticipantResponse
	for _, n := range tr.nodes {
		pubKey, err := n.Machine.pubKey.MarshalBinary()
		if err != nil {
			t.Fatalf("%s: failed to marshal pubkey: %v", n.Participant, err)
		}
		entry := &responses.DKGProposalPubKeysParticipantEntry{
			ParticipantId: n.ParticipantID,
			Username:      n.Participant,
			DkgPubKey:     pubKey,
		}
		getCommitsRequest = append(getCommitsRequest, entry)
	}
	op = createOperation(t, string(dkg_proposal_fsm.StateDkgCommitsAwaitConfirmations), "", getCommitsRequest)
	runStep(tr, func(n *Node, wg *sync.WaitGroup) {
		defer wg.Done()

		operation, err := n.Machine.GetOperationResult(op)
		if err != nil {
			t.Fatalf("%s: failed to handle operation %s: %v", n.Participant, op.Type, err)
		}
		if err := n.Machine.storeOperation(operation); err != nil {
			t.Fatalf("failed to storeOperation: %v", err)
		}
		for _, msg := range operation.ResultMsgs {
			tr.BroadcastMessage(t, msg)
		}
	})

	//deals
	runStep(tr, func(n *Node, wg *sync.WaitGroup) {
		defer wg.Done()

		var payload responses.DKGProposalCommitParticipantResponse
		for _, req := range n.commits {
			p := responses.DKGProposalCommitParticipantEntry{
				ParticipantId: req.ParticipantId,
				Username:      fmt.Sprintf("Participant#%d", req.ParticipantId),
				DkgCommit:     req.Commit,
			}
			payload = append(payload, &p)
		}
		op := createOperation(t, string(dkg_proposal_fsm.StateDkgDealsAwaitConfirmations), "", payload)

		operation, err := n.Machine.GetOperationResult(op)
		if err != nil {
			t.Fatalf("%s: failed to handle operation %s: %v", n.Participant, op.Type, err)
		}
		if err := n.Machine.storeOperation(operation); err != nil {
			t.Fatalf("failed to storeOperation: %v", err)
		}
		for _, msg := range operation.ResultMsgs {
			tr.BroadcastMessage(t, msg)
		}
	})

	//responses
	runStep(tr, func(n *Node, wg *sync.WaitGroup) {
		defer wg.Done()

		var payload responses.DKGProposalDealParticipantResponse
		for _, req := range n.deals {
			p := responses.DKGProposalDealParticipantEntry{
				ParticipantId: req.ParticipantId,
				Username:      fmt.Sprintf("Participant#%d", req.ParticipantId),
				DkgDeal:       req.Deal,
			}
			payload = append(payload, &p)
		}
		op := createOperation(t, string(dkg_proposal_fsm.StateDkgResponsesAwaitConfirmations), "", payload)

		operation, err := n.Machine.GetOperationResult(op)
		if err != nil {
			t.Fatalf("%s: failed to handle operation %s: %v", n.Participant, op.Type, err)
		}
		if err := n.Machine.storeOperation(operation); err != nil {
			t.Fatalf("failed to storeOperation: %v", err)
		}
		for _, msg := range operation.ResultMsgs {
			tr.BroadcastMessage(t, msg)
		}
	})

	//master key
	runStep(tr, func(n *Node, wg *sync.WaitGroup) {
		defer wg.Done()

		var payload responses.DKGProposalResponseParticipantResponse
		for _, req := range n.responses {
			p := responses.DKGProposalResponseParticipantEntry{
				ParticipantId: req.ParticipantId,
				Username:      fmt.Sprintf("Participant#%d", req.ParticipantId),
				DkgResponse:   req.Response,
			}
			payload = append(payload, &p)
		}
		op := createOperation(t, string(dkg_proposal_fsm.StateDkgMasterKeyAwaitConfirmations), "", payload)

		operation, err := n.Machine.GetOperationResult(op)
		if err != nil {
			t.Fatalf("%s: failed to handle operation %s: %v", n.Participant, op.Type, err)
		}
		if err := n.Machine.storeOperation(operation); err != nil {
			t.Fatalf("failed to storeOperation: %v", err)
		}
		for _, msg := range operation.ResultMsgs {
			tr.BroadcastMessage(t, msg)
		}
	})

	// check that all master keys are equal
	for _, n := range tr.nodes {
		for i := 0; i < len(n.masterKeys); i++ {
			if !bytes.Equal(n.masterKeys[0].MasterKey, n.masterKeys[i].MasterKey) {
				t.Fatalf("master keys is not equal!")
			}
		}
	}

	msgToSign := []byte("i am a message")

	failedSigningID := "failed_signing_id"
	successfulSigningID := "successful_signing_id"

	//partialSigns
	runStep(tr, func(n *Node, wg *sync.WaitGroup) {
		defer wg.Done()

		payload := responses.SigningPartialSignsParticipantInvitationsResponse{
			SigningId:  failedSigningID,
			SrcPayload: msgToSign,
		}

		op := createOperation(t, string(signing_proposal_fsm.StateSigningAwaitPartialSigns), "", payload)

		operation, err := n.Machine.GetOperationResult(op)
		if err != nil {
			t.Fatalf("%s: failed to handle operation %s: %v", n.Participant, op.Type, err)
		}
		if err := n.Machine.storeOperation(operation); err != nil {
			t.Fatalf("failed to storeOperation: %v", err)
		}
		for _, msg := range operation.ResultMsgs {
			tr.BroadcastMessage(t, msg)
		}
	})

	// something went wrong and current signing process failed, so we will start a new one
	for _, n := range tr.nodes {
		n.partialSigns = nil
	}

	//save operation logs of all nodes
	//at this moment the log contains operations of unfinished signature process
	//and all operations of DKG process
	oldOperationLogs := make([][]client.Operation, len(tr.nodes))
	for _, n := range tr.nodes {
		storedOperations, err := n.Machine.getOperationsLog(DKGIdentifier)
		if err != nil {
			t.Fatalf("failed to get operation log: %v", err)
		}
		oldOperationLogs = append(oldOperationLogs, storedOperations)
	}

	//start a new signing process
	//partialSigns
	runStep(tr, func(n *Node, wg *sync.WaitGroup) {
		defer wg.Done()

		payload := responses.SigningPartialSignsParticipantInvitationsResponse{
			SigningId:  successfulSigningID,
			SrcPayload: msgToSign,
		}

		op := createOperation(t, string(signing_proposal_fsm.StateSigningAwaitPartialSigns), "", payload)

		operation, err := n.Machine.GetOperationResult(op)
		if err != nil {
			t.Fatalf("%s: failed to handle operation %s: %v", n.Participant, op.Type, err)
		}
		if err := n.Machine.storeOperation(operation); err != nil {
			t.Fatalf("failed to storeOperation: %v", err)
		}
		for _, msg := range operation.ResultMsgs {
			tr.BroadcastMessage(t, msg)
		}
	})

	//recover full signature
	runStep(tr, func(n *Node, wg *sync.WaitGroup) {
		defer wg.Done()

		var payload responses.SigningProcessParticipantResponse
		for _, req := range n.partialSigns {
			p := responses.SigningProcessParticipantEntry{
				ParticipantId: req.ParticipantId,
				Username:      fmt.Sprintf("Participant#%d", req.ParticipantId),
				PartialSign:   req.PartialSign,
			}
			payload.Participants = append(payload.Participants, &p)
			payload.SigningId = successfulSigningID
		}
		payload.SrcPayload = msgToSign
		op := createOperation(t, string(signing_proposal_fsm.StateSigningPartialSignsCollected), "", payload)

		operation, err := n.Machine.GetOperationResult(op)
		if err != nil {
			t.Fatalf("%s: failed to handle operation %s: %v", n.Participant, op.Type, err)
		}

		if err := n.Machine.storeOperation(operation); err != nil {
			t.Fatalf("failed to storeOperation: %v", err)
		}

		if fsm.State(operation.Type) == signing_proposal_fsm.StateSigningPartialSignsCollected {
			if err := n.Machine.removeSignatureOperations(&operation); err != nil {
				t.Fatalf("failed to remove signature operations: %v", err)
			}
		}
		for _, msg := range operation.ResultMsgs {
			tr.BroadcastMessage(t, msg)
		}
	})

	//verify signatures
	for _, n := range tr.nodes {
		for i := 0; i < len(n.reconstructedSignatures); i++ {
			if !bytes.Equal(n.reconstructedSignatures[0].Signature, n.reconstructedSignatures[i].Signature) {
				t.Fatalf("signatures are not equal!")
			}
			if err := n.Machine.VerifySign(msgToSign, n.reconstructedSignatures[i].Signature, DKGIdentifier); err != nil {
				t.Fatal("signature is not verified!")
			}
		}
	}

	//compare oldOperationLog with current operation log to check following things:
	// * unfinished signatures operations were NOT removed
	// * finished signatures operations were removed
	// * DKG operations were not changed
	for i, n := range tr.nodes {
		storedOperations, err := n.Machine.getOperationsLog(DKGIdentifier)
		if err != nil {
			t.Fatal("failed to get operations log: ", err.Error())
		}
		reflect.DeepEqual(oldOperationLogs[i], storedOperations)
	}

	//keys and signatures are equal, so let's test it on prysm compatibility
	testKyberPrysm(t, tr.nodes[0].masterKeys[0].MasterKey, tr.nodes[0].reconstructedSignatures[0].Signature, msgToSign)

	fmt.Println("DKG succeeded, signature recovered and verified")
}

func testKyberPrysm(t *testing.T, pubkey, signature, msg []byte) {
	prysmSig, err := prysmBLS.SignatureFromBytes(signature)
	if err != nil {
		t.Fatalf("failed to get prysm sig from bytes: %v", err)
	}
	prysmPubKey, err := prysmBLS.PublicKeyFromBytes(pubkey)
	if err != nil {
		t.Fatalf("failed to get prysm pubkey from bytes: %v", err)
	}
	if !prysmSig.Verify(prysmPubKey, msg) {
		t.Fatalf("failed to verify prysm signature")
	}
}

func runStep(transport *Transport, cb func(n *Node, wg *sync.WaitGroup)) {
	var wg = &sync.WaitGroup{}
	for _, node := range transport.nodes {
		wg.Add(1)
		n := node
		cb(n, wg)
	}
	wg.Wait()
}
