package airgapped

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
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
	prysmBLS "github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/stretchr/testify/require"
)

const (
	DKGIdentifier       = "dkg_identifier"
	testDB              = "test_level_db"
	testDir             = "/tmp/airgapped_test"
	failedSigningID     = "failed_signing_id"
	successfulSigningID = "successful_signing_id"
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
	reconstructedSignatures    map[string][]client.ReconstructedSignature
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
	case client.SignatureReconstructed:
		var req []client.ReconstructedSignature
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			return fmt.Errorf("failed to unmarshal fsm req: %w", err)
		}
		if n.reconstructedSignatures == nil {
			n.reconstructedSignatures = make(map[string][]client.ReconstructedSignature)
		}
		n.reconstructedSignatures[req[0].SigningID] = append(n.reconstructedSignatures[req[0].SigningID], req...)
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
	if err := n.Machine.storeOperation(operation); err != nil {
		return fmt.Errorf("failed to storeOperation: %w", err)
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

func (tr *Transport) partialSignsStep(signingID string, msgsToSign []requests.MessageToSign) error {
	return runStep(tr, func(n *Node, wg *sync.WaitGroup) error {
		defer wg.Done()

		msgs, err := json.Marshal(msgsToSign)
		if err != nil {
			return fmt.Errorf("failed to create operation: %w", err)
		}

		payload := responses.SigningPartialSignsParticipantInvitationsResponse{
			SigningId:  signingID,
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

func (tr *Transport) recoverFullSignStep(signingID string, msgToSign []requests.MessageToSign) error {
	return runStep(tr, func(n *Node, wg *sync.WaitGroup) error {
		defer wg.Done()

		var payload responses.SigningProcessParticipantResponse

		for _, req := range n.partialSigns {
			partialSigns := make(map[string][]byte)
			for _, ps := range req.PartialSigns {
				partialSigns[ps.SigningID] = ps.Sign
			}
			p := responses.SigningProcessParticipantEntry{
				ParticipantId: req.ParticipantId,
				Username:      fmt.Sprintf("Participant#%d", req.ParticipantId),
				PartialSigns:  partialSigns,
			}
			payload.Participants = append(payload.Participants, &p)
		}

		msgs, err := json.Marshal(msgToSign)
		if err != nil {
			return fmt.Errorf("failed to marshal messages: %w", err)
		}

		payload.SrcPayload = msgs
		payload.SigningId = signingID
		op, err := createOperation(string(signing_proposal_fsm.StateSigningPartialSignsCollected), "", payload)
		if err != nil {
			return fmt.Errorf("failed to create operation: %w", err)
		}

		if err := tr.processOperation(n, *op); err != nil {
			return fmt.Errorf("failed to process operation: %w", err)
		}

		if fsm.State(op.Type) == signing_proposal_fsm.StateSigningPartialSignsCollected {
			if err := n.Machine.removeSignatureOperations(op); err != nil {
				return fmt.Errorf("failed to remove signature operations: %v", err)
			}
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

func (tr *Transport) checkReconstructedSignatures(msgsToSign []requests.MessageToSign) error {
	for _, msg := range msgsToSign {
		for _, n := range tr.nodes {
			for i := 0; i < len(n.reconstructedSignatures[msg.SigningID]); i++ {
				if !bytes.Equal(n.reconstructedSignatures[msg.SigningID][0].Signature, n.reconstructedSignatures[msg.SigningID][i].Signature) {
					return fmt.Errorf("signatures are not equal")
				}
				if err := n.Machine.VerifySign(msg.Payload, n.reconstructedSignatures[msg.SigningID][i].Signature, DKGIdentifier); err != nil {
					return fmt.Errorf("signature is not verified")
				}
			}
			if err := testKyberPrysm(n.masterKeys[0].MasterKey, n.reconstructedSignatures[msg.SigningID][0].Signature, msg.Payload); err != nil {
				return fmt.Errorf("failed to check signatures on prysm compatibility: %w", err)
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
			SigningID: "s1",
			Payload:   []byte("i am a message"),
		},
	}

	if err := tr.partialSignsStep(successfulSigningID, msgToSign); err != nil {
		t.Fatalf("failed to do master keys step: %v", err)
	}

	if err := tr.recoverFullSignStep(successfulSigningID, msgToSign); err != nil {
		t.Fatalf("failed to do master keys step: %v", err)
	}

	if err := tr.checkReconstructedSignatures(msgToSign); err != nil {
		t.Fatalf("failed to vefify signatures: %v", err)
	}

	fmt.Println("DKG succeeded, signature recovered and verified")
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
			SigningID: "s1",
			Payload:   []byte("i am a message"),
		},
	}

	if err := tr.partialSignsStep(successfulSigningID, msgToSign); err != nil {
		t.Fatalf("failed to do init request: %v", err)
	}

	if err := tr.recoverFullSignStep(successfulSigningID, msgToSign); err != nil {
		t.Fatalf("failed to do init request: %v", err)
	}

	if err := tr.checkReconstructedSignatures(msgToSign); err != nil {
		t.Fatalf("failed to vefify signatures: %v", err)
	}

	fmt.Println("DKG succeeded, signature recovered and verified")
}

func TestAirgappedMachine_ClearOperations(t *testing.T) {
	nodesCount := 10
	threshold := 2

	if err := os.RemoveAll(testDir); err != nil {
		t.Fatal("failed to remove test dir:", err.Error())
	}

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
			SigningID: "s1",
			Payload:   []byte("i am a message"),
		},
	}

	//partialSigns
	if err := tr.partialSignsStep(failedSigningID, msgToSign); err != nil {
		t.Fatalf("failed to do init request: %v", err)
	}

	// something went wrong and current signing process failed, so we will start a new one
	for _, n := range tr.nodes {
		n.partialSigns = nil
	}

	//save operation logs of all nodes
	//at this moment the log contains operations of unfinished signature process
	//and all operations of DKG process
	oldOperationLogs := make([][]client.Operation, 0, len(tr.nodes))
	for _, n := range tr.nodes {
		storedOperations, err := n.Machine.getOperationsLog(DKGIdentifier)
		if err != nil {
			t.Fatalf("failed to get operation log: %v", err)
		}
		oldOperationLogs = append(oldOperationLogs, storedOperations)
	}

	//start a new signing process
	//partialSigns
	if err := tr.partialSignsStep(successfulSigningID, msgToSign); err != nil {
		t.Fatalf("failed to do init request: %v", err)
	}

	//recover full signature
	if err := tr.recoverFullSignStep(successfulSigningID, msgToSign); err != nil {
		t.Fatalf("failed to do init request: %v", err)
	}

	if err := tr.checkReconstructedSignatures(msgToSign); err != nil {
		t.Fatalf("failed to vefify signatures: %v", err)
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
		if !reflect.DeepEqual(oldOperationLogs[i], storedOperations) {
			for _, op := range oldOperationLogs[i] {
				fmt.Println(op.ID, op.Type)
			}
			fmt.Println("-------------------------")
			for _, op := range storedOperations {
				fmt.Println(op.ID, op.Type)
			}
			t.Fatalf("old operation log is not equal to cleaned operation log")
		}
	}

	fmt.Println("DKG succeeded, signature recovered and verified")
}

func testKyberPrysm(pubkey, signature, msg []byte) error {
	prysmSig, err := prysmBLS.SignatureFromBytes(signature)
	if err != nil {
		return fmt.Errorf("failed to get prysm sig from bytes: %w", err)
	}
	prysmPubKey, err := prysmBLS.PublicKeyFromBytes(pubkey)
	if err != nil {
		return fmt.Errorf("failed to get prysm pubkey from bytes: %w", err)
	}
	if !prysmSig.Verify(prysmPubKey, msg) {
		return fmt.Errorf("failed to verify prysm signature")
	}
	return nil
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
