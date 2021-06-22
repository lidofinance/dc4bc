package client

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/lidofinance/dc4bc/storage"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lidofinance/dc4bc/fsm/fsm"
	spf "github.com/lidofinance/dc4bc/fsm/state_machines/signature_proposal_fsm"
	"github.com/lidofinance/dc4bc/storage/file_storage"

	"github.com/lidofinance/dc4bc/airgapped"
	"github.com/lidofinance/dc4bc/client/types"
	"github.com/lidofinance/dc4bc/fsm/state_machines/dkg_proposal_fsm"
	"github.com/lidofinance/dc4bc/fsm/types/requests"
	"github.com/lidofinance/dc4bc/qr"
)

var (
	errSendingNode            string
	commitConfirmationErrSent bool
	stateReset                bool
	errEvent                  fsm.Event
	msgToIgnore               string
	roundToIgnore             string
	operationToIgnore         string
)

type node struct {
	client     Client
	storage    storage.Storage
	keyPair    *KeyPair
	air        *airgapped.Machine
	listenAddr string
}

type OperationsResponse struct {
	ErrorMessage string                      `json:"error_message,omitempty"`
	Result       map[string]*types.Operation `json:"result"`
}

func getOperations(url string) (*OperationsResponse, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get operations for node %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body %v", err)
	}

	var response OperationsResponse
	if err = json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}
	return &response, nil
}

func handleProcessedOperation(url string, operation types.Operation) error {
	operationBz, err := json.Marshal(operation)
	if err != nil {
		return fmt.Errorf("failed to marshal operation: %w", err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(operationBz))
	if err != nil {
		return fmt.Errorf("failed to handle processed operation %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read body %v", err)
	}

	var response Response
	if err = json.Unmarshal(responseBody, &response); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}
	if response.ErrorMessage != "" {
		return fmt.Errorf("failed to handle processed operation: %s", response.ErrorMessage)
	}
	return nil
}

func (n *node) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			n.client.GetLogger().Log("node.run() stopped for %s", n.client.GetUsername())
			return
		default:
			operationsResponse, err := getOperations(fmt.Sprintf("http://%s/getOperations", n.listenAddr))
			if err != nil {
				panic(fmt.Sprintf("failed to get operations: %v", err))
			}

			operations := operationsResponse.Result
			_, containsOperationToIgnore := operations[operationToIgnore]
			if len(operations) == 0 || (len(operations) == 1 && containsOperationToIgnore) {
				time.Sleep(1 * time.Second)
				continue
			}

			n.client.GetLogger().Log("Got %d Operations from pool", len(operations))
			for _, operation := range operations {

				if fsm.State(operation.Type) == dkg_proposal_fsm.StateDkgCommitsAwaitConfirmations &&
					((!commitConfirmationErrSent && n.client.GetUsername() != errSendingNode) ||
						(stateReset && n.client.GetUsername() == errSendingNode)) {
					time.Sleep(1 * time.Second)
					continue
				}

				if fsm.State(operation.Type) == spf.StateAwaitParticipantsConfirmations {
					payloadBz, err := json.Marshal(map[string]string{"operationID": operation.ID})
					if err != nil {
						panic(fmt.Sprintf("failed to marshal payload: %v", err))
					}

					resp, err := http.Post(fmt.Sprintf("http://%s/approveDKGParticipation", n.listenAddr), "application/json", bytes.NewReader(payloadBz))
					if err != nil {
						panic(fmt.Sprintf("failed to make HTTP request to get operation: %v", err))
					}

					responseBody, err := ioutil.ReadAll(resp.Body)
					if err != nil {
						panic(fmt.Sprintf("failed to read body %v", err))
					}
					resp.Body.Close()

					var response Response
					if err = json.Unmarshal(responseBody, &response); err != nil {
						panic(fmt.Sprintf("failed to unmarshal response: %v", err))
					}
					if response.ErrorMessage != "" {
						panic(fmt.Sprintf("failed to approve participation: %s", response.ErrorMessage))
					}
					continue
				}

				n.client.GetLogger().Log("Handling operation %s in airgapped", operation.Type)
				processedOperation, err := n.air.GetOperationResult(*operation)
				if err != nil {
					n.client.GetLogger().Log("Failed to handle operation: %v", err)
				}

				n.client.GetLogger().Log("Operation %s handled in airgapped, result event is %s",
					operation.Type, processedOperation.Event)

				// for integration tests
				if processedOperation.Event == dkg_proposal_fsm.EventDKGMasterKeyConfirmationReceived {
					msg := processedOperation.ResultMsgs[0]
					var pubKeyReq requests.DKGProposalMasterKeyConfirmationRequest
					if err = json.Unmarshal(msg.Data, &pubKeyReq); err != nil {
						panic(fmt.Sprintf("failed to unmarshal pubKey request: %v", err))
					}
					if err = ioutil.WriteFile(fmt.Sprintf("/tmp/participant_%d.pubkey",
						pubKeyReq.ParticipantId), []byte(hex.EncodeToString(pubKeyReq.MasterKey)), 0666); err != nil {
						panic(fmt.Sprintf("failed to write pubkey to temp file: %v", err))
					}
				}

				if n.client.GetUsername() == errSendingNode && !commitConfirmationErrSent &&
					fsm.State(operation.Type) == dkg_proposal_fsm.StateDkgCommitsAwaitConfirmations {
					processedOperation.Event = errEvent
					operationMsg := processedOperation.ResultMsgs[0]
					req := requests.DKGProposalConfirmationErrorRequest{
						Error:     requests.NewFSMError(errors.New("test error")),
						CreatedAt: time.Now(),
					}
					reqBz, err := json.Marshal(req)
					if err != nil {
						n.client.GetLogger().Log("failed to generate fsm request: %v", err)
					}
					errMsg := storage.Message{
						DkgRoundID: operationMsg.DkgRoundID,
						Offset:     operationMsg.Offset,
						Event:      errEvent.String(),
						Data:       reqBz,
					}

					processedOperation.ResultMsgs = append(processedOperation.ResultMsgs, errMsg)

					roundToIgnore = processedOperation.DKGIdentifier
					operationToIgnore = operation.ID
					commitConfirmationErrSent = true
				}

				if err = handleProcessedOperation(fmt.Sprintf("http://%s/handleProcessedOperationJSON", n.listenAddr),
					processedOperation); err != nil {
					n.client.GetLogger().Log("Failed to handle processed operation: %v", err)
				} else {
					n.client.GetLogger().Log("Successfully handled processed operation %s", processedOperation.Event)
				}

				time.Sleep(1 * time.Second)
			}
			time.Sleep(1 * time.Second)
		}
	}
}

func RemoveContents(dir, mask string) error {
	files, err := filepath.Glob(filepath.Join(dir, mask))
	if err != nil {
		return err
	}
	for _, file := range files {
		err = os.RemoveAll(file)
		if err != nil {
			return err
		}
	}
	return nil
}

func TestFullFlow(t *testing.T) {
	_ = RemoveContents("/tmp", "dc4bc_*")
	//defer func() { _ = RemoveContents("/tmp", "dc4bc_*") }()

	mnemonics := []string{
		"old hawk occur merry sun valve reunion crime gallery purse mule shove ramp federal achieve ahead slam thought arrow can visual body response feed",
		"gold echo rookie frequent film mistake cart return teach off describe bright copper crucial brush present airport clutch slight theory rigid rib rich street",
		"fence body struggle huge neutral couple inherit almost battle demand unlock sport lawn raise slim robot water case economy orange fit spawn danger inside",
		"tourist soap atom icon nominee walk hold armed uncle whip violin hawk phrase crisp mystery foster train angle ketchup elephant judge list mention afraid",
	}

	var numNodes = 4
	var threshold = 2
	var storagePath = "/tmp/dc4bc_storage"
	topic := "test_topic"
	var nodes = make([]*node, numNodes)
	startingPort := 8085
	for nodeID := 0; nodeID < numNodes; nodeID++ {
		var ctx = context.Background()
		var userName = fmt.Sprintf("node_%d", nodeID)
		var state, err = NewLevelDBState(fmt.Sprintf("/tmp/dc4bc_node_%d_state", nodeID), topic)
		if err != nil {
			t.Fatalf("node %d failed to init state: %v\n", nodeID, err)
		}

		stg, err := file_storage.NewFileStorage(storagePath)
		if err != nil {
			t.Fatalf("node %d failed to init storage: %v\n", nodeID, err)
		}

		keyStore, err := NewLevelDBKeyStore(userName, fmt.Sprintf("/tmp/dc4bc_node_%d_key_store", nodeID))
		if err != nil {
			t.Fatalf("Failed to init key store: %v", err)
		}

		keyPair := NewKeyPair()
		if err := keyStore.PutKeys(userName, keyPair); err != nil {
			t.Fatalf("Failed to PutKeys: %v\n", err)
		}

		airgappedMachine, err := airgapped.NewMachine(fmt.Sprintf("/tmp/dc4bc_node_%d_airgapped_db", nodeID))
		if err != nil {
			t.Fatalf("Failed to create airgapped machine: %v", err)
		}

		clt, err := NewClient(
			ctx,
			userName,
			state,
			stg,
			keyStore,
			qr.NewCameraProcessor(),
		)
		if err != nil {
			t.Fatalf("node %d failed to init client: %v\n", nodeID, err)
		}
		airgappedMachine.SetEncryptionKey([]byte("very_strong_password")) //just for testing

		if err = airgappedMachine.SetBaseSeed(mnemonics[nodeID]); err != nil {
			t.Errorf(err.Error())
		}

		if err = airgappedMachine.InitKeys(); err != nil {
			t.Errorf(err.Error())
		}

		nodes[nodeID] = &node{
			client:     clt,
			storage:    stg,
			keyPair:    keyPair,
			air:        airgappedMachine,
			listenAddr: fmt.Sprintf("localhost:%d", startingPort),
		}
		startingPort++
	}

	// node_3 will produce event_dkg_confirm_cancelled_by_error
	errEvent = dkg_proposal_fsm.EventDKGCommitConfirmationError
	errSendingNode = nodes[3].client.GetUsername()

	// Each node starts to Poll().
	runCtx, cancel := context.WithCancel(context.Background())
	for nodeID, n := range nodes {
		go func(nodeID int, node *node) {
			if err := node.client.StartHTTPServer(node.listenAddr); err != nil {
				panic(fmt.Sprintf("failed to start HTTP server for nodeID #%d: %v\n", nodeID, err))
			}
		}(nodeID, n)
		time.Sleep(1 * time.Second)
		go nodes[nodeID].run(runCtx)

		go func(nodeID int, node Client) {
			if err := node.Poll(); err != nil {
				panic(fmt.Sprintf("client %d poller failed: %v\n", nodeID, err))
			}
		}(nodeID, n.client)

		log.Printf("client %d started...\n", nodeID)
	}

	// Node1 tells other participants to start DKG.
	var participants []*requests.SignatureProposalParticipantsEntry
	for _, node := range nodes {
		dkgPubKey, err := node.air.GetPubKey().MarshalBinary()
		if err != nil {
			log.Fatalln("failed to get DKG pubKey:", err.Error())
		}
		participants = append(participants, &requests.SignatureProposalParticipantsEntry{
			Username:  node.client.GetUsername(),
			PubKey:    node.client.GetPubKey(),
			DkgPubKey: dkgPubKey,
		})
	}
	messageData := requests.SignatureProposalParticipantsListRequest{
		Participants:     participants,
		SigningThreshold: threshold,
		CreatedAt:        time.Now(),
	}
	messageDataBz, err := json.Marshal(messageData)
	if err != nil {
		t.Fatalf("failed to marshal SignatureProposalParticipantsListRequest: %v\n", err)
	}

	if _, err := http.Post(fmt.Sprintf("http://localhost:%d/startDKG", startingPort-1),
		"application/json", bytes.NewReader(messageDataBz)); err != nil {
		t.Fatalf("failed to send HTTP request to start DKG: %v\n", err)
	}

	time.Sleep(30 * time.Second)

	log.Print("\n\n\nStopping nodes and resetting their states\n\n\n")

	cancel()
	time.Sleep(10 * time.Second)

	msgs, err := nodes[0].storage.GetMessages(0)
	if err != nil {
		t.Fatalf("failed to get messages from storage: %v\n", err)
	}

	for _, msg := range msgs {
		if msg.Event == errEvent.String() && msg.DkgRoundID == roundToIgnore {
			msgToIgnore = msg.ID
		}
	}

	resetReq := ResetStateRequest{
		NewStateDBDSN: "",
		UseOffset:     false,
		Messages:      []string{msgToIgnore},
	}
	resetReqBz, err := json.Marshal(resetReq)
	if err != nil {
		t.Fatalf("failed to marshal ResetStateRequest: %v\n", err)
	}

	for i := startingPort - numNodes; i < startingPort; i++ {
		if _, err := http.Post(fmt.Sprintf("http://localhost:%d/resetState", i),
			"application/json", bytes.NewReader(resetReqBz)); err != nil {
			t.Fatalf("failed to send HTTP request to reset state: %v\n", err)
		}
	}

	time.Sleep(20 * time.Second)

	stateReset = true

	runCtx, cancel = context.WithCancel(context.Background())
	for _, n := range nodes {
		go n.run(runCtx)
	}

	log.Print("\n\n\nState recreated\n\n\n")

	time.Sleep(35 * time.Second)

	log.Println("Propose message to sign")

	dkgRoundID := md5.Sum(messageDataBz)
	messageDataBz, err = json.Marshal(map[string][]byte{"data": []byte("message to sign"),
		"dkgID": dkgRoundID[:]})
	if err != nil {
		t.Fatalf("failed to marshal SignatureProposalParticipantsListRequest: %v\n", err)
	}

	if _, err := http.Post(fmt.Sprintf("http://localhost:%d/proposeSignMessage", startingPort-1),
		"application/json", bytes.NewReader(messageDataBz)); err != nil {
		t.Fatalf("failed to send HTTP request to sign message: %v\n", err)
	}
	time.Sleep(10 * time.Second)

	fmt.Println("Sign message again")
	if _, err := http.Post(fmt.Sprintf("http://localhost:%d/proposeSignMessage", startingPort-1),
		"application/json", bytes.NewReader(messageDataBz)); err != nil {
		t.Fatalf("failed to send HTTP request to sign message: %v\n", err)
	}
	time.Sleep(10 * time.Second)

	//reinit DKG stage
	fmt.Println("Reinit DKG...")
	fmt.Println("-----------------------------------------------------------------------------------")
	var newNodes = make([]*node, numNodes)
	var newStoragePath = "/tmp/dc4bc_new_storage"
	for nodeID := 0; nodeID < numNodes; nodeID++ {
		var ctx = context.Background()
		var userName = fmt.Sprintf("node_%d", nodeID)
		var state, err = NewLevelDBState(fmt.Sprintf("/tmp/dc4bc_new_node_%d_state", nodeID), topic)
		if err != nil {
			t.Fatalf("node %d failed to init state: %v\n", nodeID, err)
		}

		stg, err := file_storage.NewFileStorage(newStoragePath)
		if err != nil {
			t.Fatalf("node %d failed to init storage: %v\n", nodeID, err)
		}

		keyStore, err := NewLevelDBKeyStore(userName, fmt.Sprintf("/tmp/dc4bc_new_node_%d_key_store", nodeID))
		if err != nil {
			t.Fatalf("Failed to init key store: %v", err)
		}

		keyPair := NewKeyPair()
		if err := keyStore.PutKeys(userName, keyPair); err != nil {
			t.Fatalf("Failed to PutKeys: %v\n", err)
		}

		airgappedMachine, err := airgapped.NewMachine(fmt.Sprintf("/tmp/dc4bc_new_node_%d_airgapped_db", nodeID))
		if err != nil {
			t.Fatalf("Failed to create airgapped machine: %v", err)
		}

		clt, err := NewClient(
			ctx,
			userName,
			state,
			stg,
			keyStore,
			qr.NewCameraProcessor(),
		)
		if err != nil {
			t.Fatalf("node %d failed to init client: %v\n", nodeID, err)
		}
		airgappedMachine.SetEncryptionKey([]byte("very_strong_password")) //just for testing

		if err = airgappedMachine.SetBaseSeed(mnemonics[nodeID]); err != nil {
			t.Errorf(err.Error())
		}

		if err = airgappedMachine.InitKeys(); err != nil {
			t.Errorf(err.Error())
		}

		newNodes[nodeID] = &node{
			client:     clt,
			keyPair:    keyPair,
			air:        airgappedMachine,
			listenAddr: fmt.Sprintf("localhost:%d", startingPort),
		}
		startingPort++
	}

	runCtx, cancel = context.WithCancel(context.Background())
	for nodeID, n := range newNodes {
		go func(nodeID int, node *node) {
			if err := node.client.StartHTTPServer(node.listenAddr); err != nil {
				panic(fmt.Sprintf("failed to start HTTP server for nodeID #%d: %v\n", nodeID, err))
			}
		}(nodeID, n)
		time.Sleep(1 * time.Second)
		go nodes[nodeID].run(runCtx)

		go func(nodeID int, node Client) {
			if err := node.Poll(); err != nil {
				panic(fmt.Sprintf("client %d poller failed: %v\n", nodeID, err))
			}
		}(nodeID, n.client)

		log.Printf("client %d started...\n", nodeID)
	}

	oldStorage, err := file_storage.NewFileStorage("/tmp/dc4bc_storage")
	if err != nil {
		t.Fatalf(err.Error())
	}
	oldMessages, err := oldStorage.GetMessages(0)
	if err != nil {
		t.Fatalf(err.Error())
	}

	reInitDKG, err := types.GenerateReDKGMessage(oldMessages)
	if err != nil {
		t.Fatalf(err.Error())
	}

	for _, node := range newNodes {
		for i, participant := range reInitDKG.Participants {
			if participant.Name == node.client.GetUsername() {
				reInitDKG.Participants[i].NewCommPubKey = node.client.GetPubKey()
			}
		}
	}

	reInitDKGBz, err := json.Marshal(reInitDKG)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if _, err := http.Post(fmt.Sprintf("http://%s/reinitDKG", newNodes[0].listenAddr),
		"application/json", bytes.NewReader(reInitDKGBz)); err != nil {
		t.Fatalf("failed to send HTTP request to reinit DKG: %v\n", err)
	}

	time.Sleep(10 * time.Second)

	log.Println("Propose message to sign")
	messageDataBz, err = json.Marshal(map[string][]byte{"data": []byte("another message to sign"),
		"dkgID": dkgRoundID[:]})
	if err != nil {
		t.Fatalf("failed to marshal SignatureProposalParticipantsListRequest: %v\n", err)
	}

	if _, err := http.Post(fmt.Sprintf("http://localhost:%d/proposeSignMessage", startingPort-1),
		"application/json", bytes.NewReader(messageDataBz)); err != nil {
		t.Fatalf("failed to send HTTP request to sign message: %v\n", err)
	}
	time.Sleep(10 * time.Second)

}

type ReDKG struct {
	Result []types.Operation `json:"result"`
}
