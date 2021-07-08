package client

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/lidofinance/dc4bc/storage"

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
	errEvent                  fsm.Event
	msgToIgnore               string
	roundToIgnore             string
)

var sigReconstructedRegexp = regexp.MustCompile(`(?m)\[node_\d] Successfully processed message with offset \d{0,3}, type signature_reconstructed`)

var dkgAbortedRegexp = regexp.MustCompile(`(?m)\[node_\d] Participant node_\d got an error during DKG process: test error\. DKG aborted`)

type node struct {
	client       Client
	clientCancel context.CancelFunc
	clientLogger *savingLogger
	storage      storage.Storage
	keyPair      *KeyPair
	air          *airgapped.Machine
	listenAddr   string
}

type OperationsResponse struct {
	ErrorMessage string                      `json:"error_message,omitempty"`
	Result       map[string]*types.Operation `json:"result"`
}

type processedOperationCallback func(n *node, processedOperation *types.Operation)

type savingLogger struct {
	userName string
	logs     []string
}

func (l *savingLogger) Log(format string, args ...interface{}) {
	str := fmt.Sprintf("[%s] %s\n", l.userName, fmt.Sprintf(format, args...))
	l.logs = append(l.logs, str)
	fmt.Print(str)
}

func (l *savingLogger) checkLogsWithRegexp(re *regexp.Regexp, batchSize int) (matches int) {
	logs := l.logs[len(l.logs)-batchSize:]
	for _, str := range logs {
		if len(re.FindString(str)) > 0 {
			matches++
		}
	}

	return matches
}

func initNodes(numNodes int, startingPort int, storagePath string, topic string, mnemonics []string) (nodes []*node, err error) {
	nodes = make([]*node, numNodes)
	for nodeID := 0; nodeID < numNodes; nodeID++ {
		var ctx, cancel = context.WithCancel(context.Background())
		var userName = fmt.Sprintf("node_%d", nodeID)
		var state, err = NewLevelDBState(fmt.Sprintf("/tmp/dc4bc_node_%d_state", nodeID), topic)
		if err != nil {
			return nodes, fmt.Errorf("node %d failed to init state: %v\n", nodeID, err)
		}

		stg, err := file_storage.NewFileStorage(storagePath)
		if err != nil {
			return nodes, fmt.Errorf("node %d failed to init storage: %v\n", nodeID, err)
		}

		keyStore, err := NewLevelDBKeyStore(userName, fmt.Sprintf("/tmp/dc4bc_node_%d_key_store", nodeID))
		if err != nil {
			return nodes, fmt.Errorf("Failed to init key store: %v", err)
		}

		keyPair := NewKeyPair()
		if err := keyStore.PutKeys(userName, keyPair); err != nil {
			return nodes, fmt.Errorf("Failed to PutKeys: %v\n", err)
		}

		airgappedMachine, err := airgapped.NewMachine(fmt.Sprintf("/tmp/dc4bc_node_%d_airgapped_db", nodeID))
		if err != nil {
			return nodes, fmt.Errorf("failed to create airgapped machine: %v", err)
		}

		kp, err := keyStore.LoadKeys(userName, "")
		if err != nil {
			return nodes, fmt.Errorf("node %d failed to init client: %v\n", nodeID, err)
		}

		logger := &savingLogger{userName: userName}

		clt := &BaseClient{
			ctx:         ctx,
			Logger:      logger,
			userName:    userName,
			pubKey:      kp.Pub,
			state:       state,
			storage:     stg,
			keyStore:    keyStore,
			qrProcessor: qr.NewCameraProcessor(),
		}

		airgappedMachine.SetEncryptionKey([]byte("very_strong_password")) //just for testing

		if len(mnemonics) != 0 {
			if err = airgappedMachine.SetBaseSeed(mnemonics[nodeID]); err != nil {
				return nodes, err
			}
		}

		if err = airgappedMachine.InitKeys(); err != nil {
			return nodes, err
		}

		nodes[nodeID] = &node{
			client:       clt,
			clientCancel: cancel,
			clientLogger: logger,
			storage:      stg,
			keyPair:      keyPair,
			air:          airgappedMachine,
			listenAddr:   fmt.Sprintf("localhost:%d", startingPort),
		}
		startingPort++
	}

	return nodes, err
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

func startDkg(nodes []*node, threshold int) ([]byte, error) {
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
		return nil, fmt.Errorf("failed to marshal SignatureProposalParticipantsListRequest: %v\n", err)
	}

	if _, err := http.Post(fmt.Sprintf("http://%s/startDKG", nodes[len(nodes)-1].listenAddr),
		"application/json", bytes.NewReader(messageDataBz)); err != nil {
		return nil, fmt.Errorf("failed to send HTTP request to start DKG: %v\n", err)
	}

	return messageDataBz, nil
}

func signMessage(dkgID []byte, msg, addr string) ([]byte, error) {
	messageDataBz, err := json.Marshal(map[string][]byte{"data": []byte(msg),
		"dkgID": dkgID})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SignatureProposalParticipantsListRequest: %v\n", err)
	}

	resp, err := http.Post(fmt.Sprintf("http://%s/proposeSignMessage", addr),
		"application/json", bytes.NewReader(messageDataBz))
	if err != nil {
		return nil, fmt.Errorf("failed to send HTTP request to sign message: %v\n", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read HTTP response body: %w", err)
	}

	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal HTTP response body: %w", err)
	}

	if len(response.ErrorMessage) != 0 {
		return nil, fmt.Errorf("failed to propose sign message: %s", response.ErrorMessage)
	}
	return messageDataBz, nil
}

func (n *node) run(callback processedOperationCallback, ctx context.Context) {
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
			if len(operations) == 0 {
				time.Sleep(1 * time.Second)
				continue
			}

			n.client.GetLogger().Log("Got %d Operations from pool", len(operations))
			for _, operation := range operations {
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

				if callback != nil {
					callback(n, &processedOperation)
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

func startServerRunAndPoll(nodes []*node, callback processedOperationCallback) context.CancelFunc {
	runCtx, runCancel := context.WithCancel(context.Background())
	for nodeID, n := range nodes {
		go func(nodeID int, node *node) {
			if err := node.client.StartHTTPServer(node.listenAddr); err != nil && err != http.ErrServerClosed {
				panic(fmt.Sprintf("failed to start HTTP server for nodeID #%d: %v\n", nodeID, err))
			}
		}(nodeID, n)
		time.Sleep(1 * time.Second)
		go nodes[nodeID].run(callback, runCtx)

		go func(nodeID int, node Client) {
			if err := node.Poll(); err != nil {
				panic(fmt.Sprintf("client %d poller failed: %v\n", nodeID, err))
			}
		}(nodeID, n.client)

		log.Printf("client %d started...\n", nodeID)
	}

	return runCancel
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

func TestStandardFlow(t *testing.T) {
	_ = RemoveContents("/tmp", "dc4bc_*")
	defer func() { _ = RemoveContents("/tmp", "dc4bc_*") }()

	numNodes := 4
	threshold := 2
	startingPort := 8085
	topic := "test_topic"
	storagePath := "/tmp/dc4bc_storage"
	nodes, err := initNodes(numNodes, startingPort, storagePath, topic, nil)
	if err != nil {
		t.Fatalf("Failed to init nodes, err: %v", err)
	}

	// Each node starts to Poll().
	runCancel := startServerRunAndPoll(nodes, nil)

	// Last node tells other participants to start DKG.
	messageDataBz, err := startDkg(nodes, threshold)
	if err != nil {
		t.Fatal(err.Error())
	}
	time.Sleep(10 * time.Second)

	log.Println("Propose message to sign")

	dkgID := md5.Sum(messageDataBz)
	messageDataBz, err = signMessage(dkgID[:], "message to sign", nodes[len(nodes)-1].listenAddr)
	if err != nil {
		t.Fatal(err.Error())
	}
	time.Sleep(15 * time.Second)

	for _, n := range nodes {
		if matches := n.clientLogger.checkLogsWithRegexp(sigReconstructedRegexp, 70); matches != 4 {
			t.Fatalf("not enough checks: %d", matches)
		} else {
			fmt.Println("messaged signed successfully")
		}
	}

	fmt.Println("Sign message again")
	if _, err := http.Post(fmt.Sprintf("http://%s/proposeSignMessage", nodes[len(nodes)-1].listenAddr),
		"application/json", bytes.NewReader(messageDataBz)); err != nil {
		t.Fatalf("failed to send HTTP request to sign message: %v\n", err)
	}
	time.Sleep(15 * time.Second)

	for _, n := range nodes {
		if matches := n.clientLogger.checkLogsWithRegexp(sigReconstructedRegexp, 100); matches != 8 {
			t.Fatalf("not enough checks: %d", matches)
		} else {
			fmt.Println("messaged signed successfully")
		}
	}

	runCancel()
	for _, node := range nodes {
		node.client.StopHTTPServer()
		node.clientCancel()
	}
}

func TestResetStateFlow(t *testing.T) {
	_ = RemoveContents("/tmp", "dc4bc_*")
	defer func() { _ = RemoveContents("/tmp", "dc4bc_*") }()

	numNodes := 4
	threshold := 2
	startingPort := 8090
	topic := "test_topic"
	storagePath := "/tmp/dc4bc_storage"
	nodes, err := initNodes(numNodes, startingPort, storagePath, topic, nil)
	if err != nil {
		t.Fatalf("Failed to init nodes, err: %v", err)
	}

	// node_3 will produce event_dkg_confirm_cancelled_by_error
	errEvent = dkg_proposal_fsm.EventDKGCommitConfirmationError
	errSendingNode = nodes[0].client.GetUsername()

	// injecting error into processedOperation from airgapped machine to abort DKG
	processedOperationCallback := func(n *node, processedOperation *types.Operation) {
		if n.client.GetUsername() == errSendingNode && !commitConfirmationErrSent &&
			fsm.State(processedOperation.Type) == dkg_proposal_fsm.StateDkgCommitsAwaitConfirmations {
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

			processedOperation.ResultMsgs = []storage.Message{errMsg, operationMsg}

			roundToIgnore = processedOperation.DKGIdentifier
			commitConfirmationErrSent = true
		}
	}

	// Each node starts to Poll().
	runCancel := startServerRunAndPoll(nodes, processedOperationCallback)
	// Last node tells other participants to start DKG.
	messageDataBz, err := startDkg(nodes, threshold)
	if err != nil {
		t.Fatal(err.Error())
	}
	dkgID := md5.Sum(messageDataBz)

	time.Sleep(20 * time.Second)

	for _, n := range nodes {
		if matches := n.clientLogger.checkLogsWithRegexp(dkgAbortedRegexp, 30); matches < 1 {
			t.Fatalf("not enough checks: %d", matches)
		}
	}

	log.Print("\n\n\nStopping nodes and resetting their states\n\n\n")

	runCancel()
	time.Sleep(10 * time.Second)

	// Searching for an injected error message to ignore it and eventually recover aborted DKG
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

	for i := startingPort; i < startingPort+numNodes; i++ {
		if _, err := http.Post(fmt.Sprintf("http://localhost:%d/resetState", i),
			"application/json", bytes.NewReader(resetReqBz)); err != nil {
			t.Fatalf("failed to send HTTP request to reset state: %v\n", err)
		}
	}

	time.Sleep(10 * time.Second)

	runCtx, runCancel := context.WithCancel(context.Background())
	for _, n := range nodes {
		go n.run(nil, runCtx)
	}

	log.Print("\n\n\nState recreated\n\n\n")

	time.Sleep(20 * time.Second)

	log.Println("Propose message to sign")

	messageDataBz, err = signMessage(dkgID[:], "message to sign", nodes[len(nodes)-1].listenAddr)
	if err != nil {
		t.Fatal(err.Error())
	}
	time.Sleep(15 * time.Second)

	for _, n := range nodes {
		if matches := n.clientLogger.checkLogsWithRegexp(sigReconstructedRegexp, 70); matches != 4 {
			t.Fatalf("not enough checks: %d", matches)
		} else {
			fmt.Println("messaged signed successfully")
		}
	}

	fmt.Println("Sign message again")
	if _, err := http.Post(fmt.Sprintf("http://%s/proposeSignMessage", nodes[len(nodes)-1].listenAddr),
		"application/json", bytes.NewReader(messageDataBz)); err != nil {
		t.Fatalf("failed to send HTTP request to sign message: %v\n", err)
	}
	time.Sleep(15 * time.Second)

	for _, n := range nodes {
		if matches := n.clientLogger.checkLogsWithRegexp(sigReconstructedRegexp, 70); matches != 8 {
			t.Fatalf("not enough checks: %d", matches)
		} else {
			fmt.Println("messaged signed successfully")
		}
	}

	runCancel()
	for _, node := range nodes {
		node.client.StopHTTPServer()
		node.clientCancel()
	}
}

func convertDKGMessageto0_1_4(orig types.ReDKG) types.ReDKG {
	newDKG := types.ReDKG{}
	newDKG.DKGID = orig.DKGID
	newDKG.Participants = orig.Participants
	newDKG.Threshold = orig.Threshold
	newDKG.Messages = []storage.Message{}
	var newOffset uint64
	for _, m := range orig.Messages {
		if fsm.Event(m.Event) == dkg_proposal_fsm.EventDKGDealConfirmationReceived && m.SenderAddr == m.RecipientAddr {
			continue
		}
		m.Offset = newOffset
		newDKG.Messages = append(newDKG.Messages, m)
		newOffset++
	}
	return newDKG
}

func testReinitDKGFlow(t *testing.T, convertDKGTo10_1_4 bool) {
	_ = RemoveContents("/tmp", "dc4bc_*")
	defer func() { _ = RemoveContents("/tmp", "dc4bc_*") }()

	mnemonics := []string{
		"old hawk occur merry sun valve reunion crime gallery purse mule shove ramp federal achieve ahead slam thought arrow can visual body response feed",
		"gold echo rookie frequent film mistake cart return teach off describe bright copper crucial brush present airport clutch slight theory rigid rib rich street",
		"fence body struggle huge neutral couple inherit almost battle demand unlock sport lawn raise slim robot water case economy orange fit spawn danger inside",
		"tourist soap atom icon nominee walk hold armed uncle whip violin hawk phrase crisp mystery foster train angle ketchup elephant judge list mention afraid",
	}

	numNodes := 4
	threshold := 2
	startingPort := 8095
	topic := "test_topic"
	storagePath := "/tmp/dc4bc_storage"
	nodes, err := initNodes(numNodes, startingPort, storagePath, topic, mnemonics)
	if err != nil {
		t.Fatalf("Failed to init nodes, err: %v", err)
	}

	// Each node starts to Poll().
	runCancel := startServerRunAndPoll(nodes, nil)

	// Last node tells other participants to start DKG.
	messageDataBz, err := startDkg(nodes, threshold)
	if err != nil {
		t.Fatal(err.Error())
	}
	dkgID := md5.Sum(messageDataBz)

	time.Sleep(30 * time.Second)

	log.Println("Propose message to sign")

	messageDataBz, err = signMessage(dkgID[:], "message to sign", nodes[len(nodes)-1].listenAddr)
	if err != nil {
		t.Fatal(err.Error())
	}
	time.Sleep(15 * time.Second)

	for _, n := range nodes {
		if matches := n.clientLogger.checkLogsWithRegexp(sigReconstructedRegexp, 70); matches != 4 {
			t.Fatalf("not enough checks: %d", matches)
		} else {
			fmt.Println("messaged signed successfully")
		}
	}

	fmt.Println("Reinit DKG...")
	fmt.Println("-----------------------------------------------------------------------------------")
	runCancel()
	for _, node := range nodes {
		node.client.StopHTTPServer()
		node.clientCancel()
	}

	oldStorage, err := file_storage.NewFileStorage(storagePath)
	if err != nil {
		t.Fatalf(err.Error())
	}

	err = oldStorage.IgnoreMessages([]string{msgToIgnore}, false)
	if err != nil {
		t.Fatalf(err.Error())
	}

	oldMessages, err := oldStorage.GetMessages(0)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if err = RemoveContents("/tmp", "dc4bc_*"); err != nil {
		t.Fatalf(err.Error())
	}

	time.Sleep(10 * time.Second)

	var newNodes = make([]*node, numNodes)
	var newStoragePath = "/tmp/dc4bc_new_storage"
	newNodes, err = initNodes(numNodes, startingPort, newStoragePath, topic, mnemonics)
	if err != nil {
		t.Fatalf("Failed to init nodes, err: %v", err)
	}

	// Each node starts to Poll().
	runCancel = startServerRunAndPoll(newNodes, nil)

	reInitDKG, err := types.GenerateReDKGMessage(oldMessages)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if convertDKGTo10_1_4 {
		// removing dkg_proposal_fsm.EventDKGDealConfirmationReceived self-confirm messages
		// to make reInitDKG message looks like in release 0.1.4
		newDKG := convertDKGMessageto0_1_4(*reInitDKG)

		// adding back self-confirm messages
		// this is our test target
		adaptedReDKG, err := GetAdaptedReDKG(newDKG)
		if err != nil {
			t.Fatalf(err.Error())
		}
		reInitDKG = &adaptedReDKG

		// skip messages signature verification, since we are unable to sign self-confirm messages by old priv key
		//for _, node := range newNodes {
		//	node.client.SetSkipCommKeysVerification(true)
		//}
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

	messageDataBz, err = signMessage(dkgID[:], "message to sign", newNodes[len(newNodes)-1].listenAddr)
	if err != nil {
		t.Fatal(err.Error())
	}
	time.Sleep(15 * time.Second)

	for _, n := range newNodes {
		if matches := n.clientLogger.checkLogsWithRegexp(sigReconstructedRegexp, 40); matches != 4 {
			t.Fatalf("not enough checks: %d", matches)
		} else {
			fmt.Println("message signed successfully")
		}
	}

	fmt.Println("Sign message again")
	if _, err := http.Post(fmt.Sprintf("http://%s/proposeSignMessage", newNodes[len(newNodes)-1].listenAddr),
		"application/json", bytes.NewReader(messageDataBz)); err != nil {
		t.Fatalf("failed to send HTTP request to sign message: %v\n", err)
	}
	time.Sleep(15 * time.Second)

	for _, n := range newNodes {
		if matches := n.clientLogger.checkLogsWithRegexp(sigReconstructedRegexp, 40); matches < 4 {
			t.Fatalf("not enough checks: %d", matches)
		} else {
			fmt.Println("messaged signed successfully")
		}
	}

	runCancel()
	for _, node := range newNodes {
		node.client.StopHTTPServer()
		node.clientCancel()
	}
}

func TestReinitDKGFlow(t *testing.T) {
	testReinitDKGFlow(t, false)
}

func TestReinitDKGFlowWithDump0_1_4(t *testing.T) {
	testReinitDKGFlow(t, true)
}
