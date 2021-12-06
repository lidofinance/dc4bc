package airgapped

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	client "github.com/lidofinance/dc4bc/client/types"
	"github.com/lidofinance/dc4bc/fsm/fsm"
	"github.com/lidofinance/dc4bc/fsm/state_machines/dkg_proposal_fsm"
	"github.com/lidofinance/dc4bc/fsm/state_machines/signature_proposal_fsm"
	"github.com/lidofinance/dc4bc/fsm/state_machines/signing_proposal_fsm"
	"github.com/lidofinance/dc4bc/fsm/types/requests"
	"github.com/lidofinance/dc4bc/fsm/types/responses"
	"github.com/lidofinance/dc4bc/storage"
	"github.com/stretchr/testify/require"
)

const (
	DKGIdentifier            = "dkg_identifier"
	testDB                   = "test_level_db"
	testDir                  = "/tmp/airgapped_test"
	successfulBatchSigningID = "successful_batch_signing_id"
)

type Node struct {
	ParticipantID              int
	Participant                string
	Machine                    *Machine
	participationConfirmations []requests.SignatureProposalParticipantRequest
	commits                    []requests.DKGProposalCommitConfirmationRequest
	deals                      []requests.DKGProposalDealConfirmationRequest
	responses                  []requests.DKGProposalResponseConfirmationRequest
	masterKeys                 []requests.DKGProposalMasterKeyConfirmationRequest
	partialSigns               []requests.SigningProposalBatchPartialSignRequests
}

func (n *Node) storeOperation(msg storage.Message) error {
	switch fsm.Event(msg.Event) {
	case signature_proposal_fsm.EventConfirmSignatureProposal:
		var req requests.SignatureProposalParticipantRequest
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			return fmt.Errorf("failed to unmarshal fsm req: %w", err)
		}
		n.participationConfirmations = append(n.participationConfirmations, req)
	case dkg_proposal_fsm.EventDKGCommitConfirmationReceived:
		var req requests.DKGProposalCommitConfirmationRequest
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			return fmt.Errorf("failed to unmarshal fsm req: %w", err)
		}
		n.commits = append(n.commits, req)
	case dkg_proposal_fsm.EventDKGDealConfirmationReceived:
		var req requests.DKGProposalDealConfirmationRequest
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			return fmt.Errorf("failed to unmarshal fsm req: %w", err)
		}
		n.deals = append(n.deals, req)
	case dkg_proposal_fsm.EventDKGResponseConfirmationReceived:
		var req requests.DKGProposalResponseConfirmationRequest
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			return fmt.Errorf("failed to unmarshal fsm req: %w", err)
		}
		n.responses = append(n.responses, req)
	case dkg_proposal_fsm.EventDKGMasterKeyConfirmationReceived:
		var req requests.DKGProposalMasterKeyConfirmationRequest
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			return fmt.Errorf("failed to unmarshal fsm req: %w", err)
		}
		n.masterKeys = append(n.masterKeys, req)
	case signing_proposal_fsm.EventSigningPartialSignReceived:
		var req requests.SigningProposalBatchPartialSignRequests
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			return fmt.Errorf("failed to unmarshal fsm req: %w", err)
		}
		n.partialSigns = append(n.partialSigns, req)
	default:
		return fmt.Errorf("invalid event: %s", msg.Event)
	}
	return nil
}

type Transport struct {
	nodes []*Node
}

func (tr *Transport) BroadcastMessage(msg storage.Message) error {
	for _, node := range tr.nodes {
		if msg.RecipientAddr == "" || msg.RecipientAddr == node.Participant {
			if err := node.storeOperation(msg); err != nil {
				return fmt.Errorf("failed to store operation: %w", err)
			}
		}
	}
	return nil
}

func createTransport(participants []string) (*Transport, error) {
	tr := &Transport{}
	for i := 0; i < len(participants); i++ {
		am, err := NewMachine(fmt.Sprintf("%s/%s-%d", testDir, testDB, i))
		if err != nil {
			return nil, fmt.Errorf("failed to create airgapped machine: %w", err)
		}
		am.SetEncryptionKey([]byte(fmt.Sprintf(testDB+"%d", i)))
		if err = am.InitKeys(); err != nil {
			return nil, fmt.Errorf("failed to init keys: %w", err)
		}
		node := Node{
			ParticipantID: i,
			Participant:   participants[i],
			Machine:       am,
		}
		tr.nodes = append(tr.nodes, &node)
	}
	return tr, nil
}

func createOperation(opType string, to string, req interface{}) (*client.Operation, error) {
	reqBz, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}
	op := &client.Operation{
		ID:            uuid.New().String(),
		Type:          client.OperationType(opType),
		Payload:       reqBz,
		CreatedAt:     time.Now(),
		DKGIdentifier: DKGIdentifier,
		To:            to,
	}
	return op, nil
}

func (tr *Transport) processOperation(n *Node, op client.Operation) error {
	operation, err := n.Machine.GetOperationResult(op)
	if err != nil {
		return fmt.Errorf("%s: failed to handle operation %s: %w", n.Participant, op.Type, err)
	}
	if !operation.IsSigningState() {
		if err := n.Machine.storeOperation(operation); err != nil {
			return fmt.Errorf("failed to storeOperation: %w", err)
		}
	}
	for _, msg := range operation.ResultMsgs {
		if err := tr.BroadcastMessage(msg); err != nil {
			return fmt.Errorf("failed to broadcast message: %w", err)
		}
	}
	return nil
}

func (tr *Transport) initRequest(threshold int) error {
	var initReq responses.SignatureProposalParticipantInvitationsResponse
	for _, n := range tr.nodes {
		pubKey, err := n.Machine.pubKey.MarshalBinary()
		if err != nil {
			return fmt.Errorf("failed to marshal dkg pubkey: %v", err)
		}
		entry := &responses.SignatureProposalParticipantInvitationEntry{
			ParticipantId: n.ParticipantID,
			Username:      n.Participant,
			Threshold:     threshold,
			DkgPubKey:     pubKey,
		}
		initReq = append(initReq, entry)
	}
	op, err := createOperation(string(signature_proposal_fsm.StateAwaitParticipantsConfirmations), "", initReq)
	if err != nil {
		return fmt.Errorf("failed to create opration: %w", err)
	}
	return runStep(tr, func(n *Node, wg *sync.WaitGroup) error {
		defer wg.Done()

		if err := tr.processOperation(n, *op); err != nil {
			return fmt.Errorf("failed to process operation: %w", err)
		}
		return nil
	})
}

func (tr *Transport) commitsStep(threshold int) error {
	var getCommitsRequest responses.DKGProposalPubKeysParticipantResponse
	for _, n := range tr.nodes {
		pubKey, err := n.Machine.pubKey.MarshalBinary()
		if err != nil {
			return fmt.Errorf("%s: failed to marshal pubkey: %v", n.Participant, err)
		}
		entry := &responses.DKGProposalPubKeysParticipantEntry{
			ParticipantId: n.ParticipantID,
			Username:      n.Participant,
			DkgPubKey:     pubKey,
			Threshold:     threshold,
		}
		getCommitsRequest = append(getCommitsRequest, entry)
	}
	op, err := createOperation(string(dkg_proposal_fsm.StateDkgCommitsAwaitConfirmations), "", getCommitsRequest)
	if err != nil {
		return fmt.Errorf("failed to create operation: %w", err)
	}
	return runStep(tr, func(n *Node, wg *sync.WaitGroup) error {
		defer wg.Done()

		if err := tr.processOperation(n, *op); err != nil {
			return fmt.Errorf("failed to process operation: %w", err)
		}
		return nil
	})
}

func (tr *Transport) dealsStep() error {
	return runStep(tr, func(n *Node, wg *sync.WaitGroup) error {
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
		op, err := createOperation(string(dkg_proposal_fsm.StateDkgDealsAwaitConfirmations), "", payload)
		if err != nil {
			return fmt.Errorf("failed to create operation: %w", err)
		}

		if err := tr.processOperation(n, *op); err != nil {
			return fmt.Errorf("failed to process operation: %w", err)
		}
		return nil
	})
}

func (tr *Transport) responsesStep() error {
	return runStep(tr, func(n *Node, wg *sync.WaitGroup) error {
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
		op, err := createOperation(string(dkg_proposal_fsm.StateDkgResponsesAwaitConfirmations), "", payload)
		if err != nil {
			return fmt.Errorf("failed to create operation: %w", err)
		}

		if err := tr.processOperation(n, *op); err != nil {
			return fmt.Errorf("failed to process operation: %w", err)
		}
		return nil
	})
}

func (tr *Transport) masterKeysStep() error {
	return runStep(tr, func(n *Node, wg *sync.WaitGroup) error {
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
		op, err := createOperation(string(dkg_proposal_fsm.StateDkgMasterKeyAwaitConfirmations), "", payload)
		if err != nil {
			return fmt.Errorf("failed to create operation: %w", err)
		}

		if err := tr.processOperation(n, *op); err != nil {
			return fmt.Errorf("failed to process operation: %w", err)
		}
		return nil
	})
}

func (tr *Transport) partialSignsStep(batchID string, msgsToSign []requests.MessageToSign) error {
	return runStep(tr, func(n *Node, wg *sync.WaitGroup) error {
		defer wg.Done()

		msgs, err := json.Marshal(msgsToSign)
		if err != nil {
			return fmt.Errorf("failed to create operation: %w", err)
		}

		payload := responses.SigningPartialSignsParticipantInvitationsResponse{
			BatchID:    batchID,
			SrcPayload: msgs,
		}

		op, err := createOperation(string(signing_proposal_fsm.StateSigningAwaitPartialSigns), "", payload)
		if err != nil {
			return fmt.Errorf("failed to create operation: %w", err)
		}

		if err := tr.processOperation(n, *op); err != nil {
			return fmt.Errorf("failed to process operation: %w", err)
		}
		return nil
	})
}

func (tr *Transport) checkReconstructedMasterKeys() error {
	for _, n := range tr.nodes {
		for i := 0; i < len(n.masterKeys); i++ {
			if !bytes.Equal(n.masterKeys[0].MasterKey, n.masterKeys[i].MasterKey) {
				return fmt.Errorf("master keys is not equal")
			}
		}
	}
	return nil
}

func TestAirgappedAllSteps(t *testing.T) {
	nodesCount := 10
	threshold := 3
	participants := make([]string, nodesCount)
	for i := 0; i < nodesCount; i++ {
		participants[i] = fmt.Sprintf("Participant#%d", i)
	}

	tr, err := createTransport(participants)
	if err != nil {
		t.Fatalf("failed to create transport: %v", err)
	}
	defer os.RemoveAll(testDir)

	if err := tr.commitsStep(threshold); err != nil {
		t.Fatalf("failed to do commits step: %v", err)
	}

	if err := tr.dealsStep(); err != nil {
		t.Fatalf("failed to do deals step: %v", err)
	}

	if err := tr.responsesStep(); err != nil {
		t.Fatalf("failed to do responses step: %v", err)
	}

	if err := tr.masterKeysStep(); err != nil {
		t.Fatalf("failed to do master keys step: %v", err)
	}

	if err := tr.checkReconstructedMasterKeys(); err != nil {
		t.Fatalf("failed check master keys: %v", err)
	}

	msgToSign := []requests.MessageToSign{
		{
			MessageID: "s1",
			Payload:   []byte("i am a message"),
		},
	}

	if err := tr.partialSignsStep(successfulBatchSigningID, msgToSign); err != nil {
		t.Fatalf("failed to do master keys step: %v", err)
	}

	fmt.Println("DKG succeeded")
}

func TestAirgappedMachine_Replay(t *testing.T) {
	nodesCount := 2
	threshold := 2
	participants := make([]string, nodesCount)
	for i := 0; i < nodesCount; i++ {
		participants[i] = fmt.Sprintf("Participant#%d", i)
	}

	tr, err := createTransport(participants)
	if err != nil {
		t.Fatalf("failed to create transport: %v", err)
	}
	defer os.RemoveAll(testDir)

	if err := tr.commitsStep(threshold); err != nil {
		t.Fatalf("failed to do init request: %v", err)
	}

	if err := tr.dealsStep(); err != nil {
		t.Fatalf("failed to do init request: %v", err)
	}

	// At this point something goes wrong and we have to restart the machines.
	for _, node := range tr.nodes {
		_ = node.Machine.db.Close()
	}

	participants = make([]string, nodesCount)
	for i := 0; i < nodesCount; i++ {
		participants[i] = fmt.Sprintf("Participant#%d", i)
	}

	newTr, err := createTransport(participants)
	if err != nil {
		t.Errorf("failed to create transport: %v", err)
	}
	for i, n := range newTr.nodes {
		n.deals = tr.nodes[i].deals
	}

	for _, node := range newTr.nodes {
		err := node.Machine.ReplayOperationsLog(DKGIdentifier)
		require.NoError(t, err)
	}

	//oldTr := tr
	tr = newTr

	if err := tr.responsesStep(); err != nil {
		t.Fatalf("failed to do init request: %v", err)
	}

	if err := tr.masterKeysStep(); err != nil {
		t.Fatalf("failed to do init request: %v", err)
	}

	if err := tr.checkReconstructedMasterKeys(); err != nil {
		t.Fatalf("failed check master keys: %v", err)
	}

	msgToSign := []requests.MessageToSign{
		{
			MessageID: "s1",
			Payload:   []byte("i am a message"),
		},
	}

	if err := tr.partialSignsStep(successfulBatchSigningID, msgToSign); err != nil {
		t.Fatalf("failed to do init request: %v", err)
	}

	fmt.Println("DKG succeeded")
}

func runStep(transport *Transport, cb func(n *Node, wg *sync.WaitGroup) error) error {
	var wg = &sync.WaitGroup{}
	for _, node := range transport.nodes {
		wg.Add(1)
		n := node
		if err := cb(n, wg); err != nil {
			return err
		}
	}
	wg.Wait()
	return nil
}
