package airgapped

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/depools/dc4bc/client"
	"github.com/depools/dc4bc/fsm/state_machines/dkg_proposal_fsm"
	"github.com/depools/dc4bc/fsm/state_machines/signature_proposal_fsm"
	"github.com/depools/dc4bc/fsm/types/requests"
	"github.com/depools/dc4bc/fsm/types/responses"
	"github.com/google/uuid"
	"sync"
	"testing"
	"time"
)

const (
	DKGIdentifier = "dkg_identifier"
	testDB        = "test_level_db"
)

type Node struct {
	ParticipantID int
	Participant   string
	Machine       *AirgappedMachine
	commits       []requests.DKGProposalCommitConfirmationRequest
	deals         []requests.DKGProposalDealConfirmationRequest
	responses     []requests.DKGProposalResponseConfirmationRequest
	masterKeys    []requests.DKGProposalMasterKeyConfirmationRequest
}

func (n *Node) storeOperation(t *testing.T, o client.Operation) {
	switch o.Event {
	case dkg_proposal_fsm.EventDKGCommitConfirmationReceived:
		var req requests.DKGProposalCommitConfirmationRequest
		if err := json.Unmarshal(o.Result, &req); err != nil {
			t.Fatalf("failed to unmarshal fsm req: %v", err)
		}
		n.commits = append(n.commits, req)
	case dkg_proposal_fsm.EventDKGDealConfirmationReceived:
		var req requests.DKGProposalDealConfirmationRequest
		if err := json.Unmarshal(o.Result, &req); err != nil {
			t.Fatalf("failed to unmarshal fsm req: %v", err)
		}
		n.deals = append(n.deals, req)
	case dkg_proposal_fsm.EventDKGResponseConfirmationReceived:
		var req requests.DKGProposalResponseConfirmationRequest
		if err := json.Unmarshal(o.Result, &req); err != nil {
			t.Fatalf("failed to unmarshal fsm req: %v", err)
		}
		n.responses = append(n.responses, req)
	case dkg_proposal_fsm.EventDKGMasterKeyConfirmationReceived:
		var req requests.DKGProposalMasterKeyConfirmationRequest
		if err := json.Unmarshal(o.Result, &req); err != nil {
			t.Fatalf("failed to unmarshal fsm req: %v", err)
		}
		n.masterKeys = append(n.masterKeys, req)
	default:
		t.Fatalf("invalid event: %s", o.Event)
	}
}

type Transport struct {
	nodes []*Node
}

func (tr *Transport) BroadcastOperation(t *testing.T, operation client.Operation) {
	for _, node := range tr.nodes {
		if operation.To == "" || operation.To == node.Participant {
			node.storeOperation(t, operation)
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
		Result:        nil,
		CreatedAt:     time.Now(),
		DKGIdentifier: DKGIdentifier,
		To:            to,
	}
	return op
}

func TestAirgappedAllSteps(t *testing.T) {
	nodesCount := 13
	participants := make([]string, nodesCount)
	for i := 0; i < nodesCount; i++ {
		participants[i] = fmt.Sprintf("Participant#%d", i)
	}

	tr := &Transport{}
	for i := 0; i < nodesCount; i++ {
		am, err := NewAirgappedMachine(fmt.Sprintf(testDB+"%d", i))
		if err != nil {
			t.Fatalf("failed to create airgapped machine: %v", err)
		}
		node := Node{
			ParticipantID: i,
			Participant:   participants[i],
			Machine:       am,
		}
		tr.nodes = append(tr.nodes, &node)
	}

	var initReq responses.SignatureProposalParticipantInvitationsResponse
	for _, n := range tr.nodes {
		entry := &responses.SignatureProposalParticipantInvitationEntry{
			ParticipantId: n.ParticipantID,
			Title:         n.Participant,
		}
		initReq = append(initReq, entry)
	}
	op := createOperation(t, string(signature_proposal_fsm.StateAwaitParticipantsConfirmations), "", initReq)
	runStep(tr, func(n *Node, wg *sync.WaitGroup) {
		defer wg.Done()

		_, err := n.Machine.HandleOperation(op)
		if err != nil {
			t.Fatalf("%s: failed to handle operation %s: %v", n.Participant, op.Type, err)
		}
	})

	// get commits
	var getCommitsRequest responses.SignatureProposalParticipantStatusResponse
	for _, n := range tr.nodes {
		pubKey, err := n.Machine.pubKey.MarshalBinary()
		if err != nil {
			t.Fatalf("%s: failed to marshal pubkey: %v", n.Participant, err)
		}
		entry := &responses.SignatureProposalParticipantStatusEntry{
			ParticipantId: n.ParticipantID,
			Title:         n.Participant,
			DkgPubKey:     pubKey,
		}
		getCommitsRequest = append(getCommitsRequest, entry)
	}
	op = createOperation(t, string(dkg_proposal_fsm.StateDkgCommitsAwaitConfirmations), "", getCommitsRequest)
	runStep(tr, func(n *Node, wg *sync.WaitGroup) {
		defer wg.Done()

		operations, err := n.Machine.HandleOperation(op)
		if err != nil {
			t.Fatalf("%s: failed to handle operation %s: %v", n.Participant, op.Type, err)
		}
		for _, op := range operations {
			tr.BroadcastOperation(t, op)
		}
	})

	//deals
	runStep(tr, func(n *Node, wg *sync.WaitGroup) {
		defer wg.Done()

		var payload responses.DKGProposalCommitParticipantResponse
		for _, req := range n.commits {
			p := responses.DKGProposalCommitParticipantEntry{
				ParticipantId: req.ParticipantId,
				Title:         fmt.Sprintf("Participant#%d", req.ParticipantId),
				Commit:        req.Commit,
			}
			payload = append(payload, &p)
		}
		op := createOperation(t, string(dkg_proposal_fsm.StateDkgDealsAwaitConfirmations), "", payload)

		operations, err := n.Machine.HandleOperation(op)
		if err != nil {
			t.Fatalf("%s: failed to handle operation %s: %v", n.Participant, op.Type, err)
		}
		for _, op := range operations {
			tr.BroadcastOperation(t, op)
		}
	})

	//responses
	runStep(tr, func(n *Node, wg *sync.WaitGroup) {
		defer wg.Done()

		var payload responses.DKGProposalDealParticipantResponse
		for _, req := range n.deals {
			p := responses.DKGProposalDealParticipantEntry{
				ParticipantId: req.ParticipantId,
				Title:         fmt.Sprintf("Participant#%d", req.ParticipantId),
				Deal:          req.Deal,
			}
			payload = append(payload, &p)
		}
		op := createOperation(t, string(dkg_proposal_fsm.StateDkgResponsesAwaitConfirmations), "", payload)

		operations, err := n.Machine.HandleOperation(op)
		if err != nil {
			t.Fatalf("%s: failed to handle operation %s: %v", n.Participant, op.Type, err)
		}
		for _, op := range operations {
			tr.BroadcastOperation(t, op)
		}
	})

	//master key
	runStep(tr, func(n *Node, wg *sync.WaitGroup) {
		defer wg.Done()

		var payload responses.DKGProposalResponsesParticipantResponse
		for _, req := range n.responses {
			p := responses.DKGProposalResponsesParticipantEntry{
				ParticipantId: req.ParticipantId,
				Title:         fmt.Sprintf("Participant#%d", req.ParticipantId),
				Responses:     req.Response,
			}
			payload = append(payload, &p)
		}
		op := createOperation(t, string(dkg_proposal_fsm.StateDkgMasterKeyAwaitConfirmations), "", payload)

		operations, err := n.Machine.HandleOperation(op)
		if err != nil {
			t.Fatalf("%s: failed to handle operation %s: %v", n.Participant, op.Type, err)
		}
		for _, op := range operations {
			tr.BroadcastOperation(t, op)
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
	sigShares := make([][]byte, 0)
	for _, n := range tr.nodes {
		sigShare, err := n.Machine.createPartialSign(msgToSign, DKGIdentifier)
		if err != nil {
			t.Fatalf("failed to create sig share: %v", err.Error())
		}
		sigShares = append(sigShares, sigShare)
	}

	fullSign, err := tr.nodes[0].Machine.recoverFullSign(msgToSign, sigShares, DKGIdentifier)
	if err != nil {
		t.Fatalf("failed to recover full sign: %v", err.Error())
	}

	for _, n := range tr.nodes {
		if err = n.Machine.verifySign(msgToSign, fullSign, DKGIdentifier); err != nil {
			t.Fatalf("failed to verify signature: %v", err)
		}
	}

	fmt.Println("DKG succeeded, signature recovered and verified")
}

func runStep(transport *Transport, cb func(n *Node, wg *sync.WaitGroup)) {
	var wg = &sync.WaitGroup{}
	for _, node := range transport.nodes {
		wg.Add(1)
		n := node
		go cb(n, wg)
	}
	wg.Wait()
}
