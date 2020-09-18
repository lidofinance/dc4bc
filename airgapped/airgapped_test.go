package airgapped

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	client "github.com/depools/dc4bc/client/types"
	"github.com/depools/dc4bc/fsm/fsm"
	"github.com/depools/dc4bc/fsm/state_machines/dkg_proposal_fsm"
	"github.com/depools/dc4bc/fsm/state_machines/signature_proposal_fsm"
	"github.com/depools/dc4bc/fsm/state_machines/signing_proposal_fsm"
	"github.com/depools/dc4bc/fsm/types/requests"
	"github.com/depools/dc4bc/fsm/types/responses"
	"github.com/depools/dc4bc/storage"
	"github.com/google/uuid"
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
	partialSigns  []requests.SigningProposalPartialSignRequest
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
	nodesCount := 10
	threshold := 3
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
		am.SetAddress(participants[i])
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

	var initReq responses.SignatureProposalParticipantInvitationsResponse
	for _, n := range tr.nodes {
		entry := &responses.SignatureProposalParticipantInvitationEntry{
			ParticipantId: n.ParticipantID,
			Addr:          n.Participant,
			Threshold:     threshold,
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
	var getCommitsRequest responses.DKGProposalPubKeysParticipantResponse
	for _, n := range tr.nodes {
		pubKey, err := n.Machine.pubKey.MarshalBinary()
		if err != nil {
			t.Fatalf("%s: failed to marshal pubkey: %v", n.Participant, err)
		}
		entry := &responses.DKGProposalPubKeysParticipantEntry{
			ParticipantId: n.ParticipantID,
			Addr:          n.Participant,
			DkgPubKey:     pubKey,
		}
		getCommitsRequest = append(getCommitsRequest, entry)
	}
	op = createOperation(t, string(dkg_proposal_fsm.StateDkgCommitsAwaitConfirmations), "", getCommitsRequest)
	runStep(tr, func(n *Node, wg *sync.WaitGroup) {
		defer wg.Done()

		operation, err := n.Machine.HandleOperation(op)
		if err != nil {
			t.Fatalf("%s: failed to handle operation %s: %v", n.Participant, op.Type, err)
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
				Addr:          fmt.Sprintf("Participant#%d", req.ParticipantId),
				DkgCommit:     req.Commit,
			}
			payload = append(payload, &p)
		}
		op := createOperation(t, string(dkg_proposal_fsm.StateDkgDealsAwaitConfirmations), "", payload)

		operation, err := n.Machine.HandleOperation(op)
		if err != nil {
			t.Fatalf("%s: failed to handle operation %s: %v", n.Participant, op.Type, err)
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
				Addr:          fmt.Sprintf("Participant#%d", req.ParticipantId),
				DkgDeal:       req.Deal,
			}
			payload = append(payload, &p)
		}
		op := createOperation(t, string(dkg_proposal_fsm.StateDkgResponsesAwaitConfirmations), "", payload)

		operation, err := n.Machine.HandleOperation(op)
		if err != nil {
			t.Fatalf("%s: failed to handle operation %s: %v", n.Participant, op.Type, err)
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
				Addr:          fmt.Sprintf("Participant#%d", req.ParticipantId),
				DkgResponse:   req.Response,
			}
			payload = append(payload, &p)
		}
		op := createOperation(t, string(dkg_proposal_fsm.StateDkgMasterKeyAwaitConfirmations), "", payload)

		operation, err := n.Machine.HandleOperation(op)
		if err != nil {
			t.Fatalf("%s: failed to handle operation %s: %v", n.Participant, op.Type, err)
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

		operation, err := n.Machine.HandleOperation(op)
		if err != nil {
			t.Fatalf("%s: failed to handle operation %s: %v", n.Participant, op.Type, err)
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
				Addr:          fmt.Sprintf("Participant#%d", req.ParticipantId),
				PartialSign:   req.PartialSign,
			}
			payload.Participants = append(payload.Participants, &p)
		}
		payload.SrcPayload = msgToSign
		op := createOperation(t, string(signing_proposal_fsm.StateSigningPartialSignsCollected), "", payload)

		operation, err := n.Machine.HandleOperation(op)
		if err != nil {
			t.Fatalf("%s: failed to handle operation %s: %v", n.Participant, op.Type, err)
		}
		for _, msg := range operation.ResultMsgs {
			tr.BroadcastMessage(t, msg)
		}
	})

	fmt.Println("DKG succeeded, signature recovered")
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
