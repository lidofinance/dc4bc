package client

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/lidofinance/dc4bc/airgapped"
	"github.com/lidofinance/dc4bc/pkg/prysm"
	"github.com/lidofinance/dc4bc/pkg/utils"

	"github.com/lidofinance/dc4bc/client/api/dto"
	"github.com/lidofinance/dc4bc/client/api/http_api"
	httprequests "github.com/lidofinance/dc4bc/client/api/http_api/requests"
	api_responses "github.com/lidofinance/dc4bc/client/api/http_api/responses"
	"github.com/lidofinance/dc4bc/client/config"
	"github.com/lidofinance/dc4bc/client/modules/keystore"
	state2 "github.com/lidofinance/dc4bc/client/modules/state"
	oprepo "github.com/lidofinance/dc4bc/client/repositories/operation"
	sigrepo "github.com/lidofinance/dc4bc/client/repositories/signature"
	"github.com/lidofinance/dc4bc/client/services"
	"github.com/lidofinance/dc4bc/client/services/fsmservice"
	"github.com/lidofinance/dc4bc/client/services/node"
	"github.com/lidofinance/dc4bc/client/services/operation"
	"github.com/lidofinance/dc4bc/client/services/signature"
	"github.com/lidofinance/dc4bc/client/types"
	"github.com/lidofinance/dc4bc/fsm/fsm"
	"github.com/lidofinance/dc4bc/fsm/state_machines/dkg_proposal_fsm"
	signature_fsm "github.com/lidofinance/dc4bc/fsm/state_machines/signature_proposal_fsm"
	signing_fsm "github.com/lidofinance/dc4bc/fsm/state_machines/signing_proposal_fsm"
	"github.com/lidofinance/dc4bc/fsm/types/requests"
	fsm_responses "github.com/lidofinance/dc4bc/fsm/types/responses"
	"github.com/lidofinance/dc4bc/storage"
	"github.com/lidofinance/dc4bc/storage/file_storage"
)

var (
	errSendingNode            string
	commitConfirmationErrSent bool
	errEvent                  fsm.Event
	msgToIgnore               string
	roundToIgnore             string

	sigReconstructedRegexp = regexp.MustCompile(`(?m)\[node_\d] Successfully processed message with offset \d{0,3}, type signature_reconstructed`)

	sigReconstructionStartedRegexp = regexp.MustCompile(`(?m)\[node_\d] Collected enough partial signatures. Full signature reconstruction just started`)

	dkgAbortedRegexp = regexp.MustCompile(`(?m)\[node_\d] Participant node_\d got an error during DKG process: test error\. DKG aborted`)

	processedOperationPayloadMismatchRegexp = regexp.MustCompile(`(?m)\[node_\d] Failed to handle processed operation: node returned an error response: processed operation does not match stored operation: o1.Payload .+ != o2.Payload .+`)

	failedSignRecoverRegexp = regexp.MustCompile(`(?m)\[node_\d] Failed to process message with offset \d{0,3}: failed to reconstruct signatures: failed to reconstruct full signature for msg [\d\w-]+: .+`)

	partialSignReceivedNodeRegexp = regexp.MustCompile(`(?m)\[node_\d] message event_signing_partial_sign_received done successfully from (node_\d)`)

	partialSignReceivedOffsetRegexp = regexp.MustCompile(`(?m)\[node_\d] Handling message with offset (\d+), type event_signing_partial_sign_received`)
)

const (
	pollPauseDuration  = 500 * time.Millisecond
	dkgDuration        = 12 * time.Second
	signMsgDuration    = 8 * time.Second
	resetStateDuration = 8 * time.Second
	nodesStopDuration  = 5 * time.Second
)

type operationHandler func(operation *types.Operation, callback processedOperationCallback) error

type nodeInstance struct {
	ctx                 context.Context
	client              node.NodeService
	sigService          signature.SignatureService
	fsmService   fsmservice.FSMService
	clientCancel        context.CancelFunc
	clientLogger        *savingLogger
	storage             storage.Storage
	keyPair             *keystore.KeyPair
	air                 *airgapped.Machine
	listenAddr          string
	httpApi             *http_api.RESTApiProvider
	operationHandlersMu *sync.Mutex
	// operationHandlers allows to define special handlers for some operations
	operationHandlers     map[types.OperationType]operationHandler
	necessaryOperationsMu *sync.Mutex
	// necessaryOperations defines a list of operations needed to be picked out of the available
	// operations list and processed. If empty, all operations are necessary.
	necessaryOperations map[types.OperationType]struct{}
}

type OperationsResponse struct {
	ErrorMessage string                      `json:"error_message,omitempty"`
	Result       map[string]*types.Operation `json:"result"`
}

type processedOperationCallback func(n *nodeInstance, processedOperation *types.Operation)

type savingLogger struct {
	userName string
	logs     []string
}

func (l *savingLogger) Log(format string, args ...interface{}) {
	str := fmt.Sprintf("[%s] %s\n", l.userName, fmt.Sprintf(format, args...))
	l.logs = append(l.logs, str)
	log.Print(str)
}

func (l *savingLogger) checkLogsWithRegexp(re *regexp.Regexp, batchSize int) (matches int) {
	startPos := 0
	if len(l.logs)-batchSize > 0 {
		startPos = len(l.logs) - batchSize
	}
	logs := l.logs[startPos:]
	for _, str := range logs {
		if len(re.FindString(str)) > 0 {
			matches++
		}
	}

	return matches
}

// findNodePartialSignMsgOffset returns the offset of the earliest message with the node's partial sign.
func (l *savingLogger) findNodePartialSignMsgOffset(batchSize int) int {
	startPos := 0
	if len(l.logs)-batchSize > 0 {
		startPos = len(l.logs) - batchSize
	}
	logs := l.logs[startPos:]
	var offset int
	for _, str := range logs {
		if matches := partialSignReceivedOffsetRegexp.FindStringSubmatch(str); matches != nil {
			offset, _ = strconv.Atoi(matches[1])
		} else if matches := partialSignReceivedNodeRegexp.FindStringSubmatch(str); matches != nil {
			nodeName := matches[1]
			if nodeName == l.userName {
				return offset
			}
		}
	}
	return -1
}

func (l *savingLogger) resetLogs() {
	l.logs = make([]string, 0)
}

func initNodes(numNodes int, startingPort int, storagePath string, topic string, mnemonics []string) (nodes []*nodeInstance, err error) {
	nodes = make([]*nodeInstance, numNodes)
	for nodeID := 0; nodeID < numNodes; nodeID++ {
		var ctx, cancel = context.WithCancel(context.Background())
		var userName = fmt.Sprintf("node_%d", nodeID)
		var state, err = state2.NewLevelDBState(fmt.Sprintf("/tmp/dc4bc_node_%d_state", nodeID), topic)
		if err != nil {
			return nodes, fmt.Errorf("nodeInstance %d failed to init state: %v\n", nodeID, err)
		}

		stg, err := file_storage.NewFileStorage(storagePath)
		if err != nil {
			return nodes, fmt.Errorf("nodeInstance %d failed to init storage: %v\n", nodeID, err)
		}

		keyStore, err := keystore.NewLevelDBKeyStore(userName, fmt.Sprintf("/tmp/dc4bc_node_%d_key_store", nodeID))
		if err != nil {
			return nodes, fmt.Errorf("Failed to init key store: %v", err)
		}

		keyPair := keystore.NewKeyPair()
		if err := keyStore.PutKeys(userName, keyPair); err != nil {
			return nodes, fmt.Errorf("Failed to PutKeys: %v\n", err)
		}

		airgappedMachine, err := airgapped.NewMachine(fmt.Sprintf("/tmp/dc4bc_node_%d_airgapped_db", nodeID))
		if err != nil {
			return nodes, fmt.Errorf("failed to create airgapped machine: %v", err)
		}

		logger := &savingLogger{userName: userName}
		cfg := config.Config{
			Username:      userName,
			KeyStoreDBDSN: fmt.Sprintf("/tmp/dc4bc_node_%d_key_store", nodeID),
			HttpApiConfig: &config.HttpApiConfig{
				ListenAddr: fmt.Sprintf("localhost:%d", startingPort),
				Debug:      false,
			},
			KafkaStorageConfig: &config.KafkaStorageConfig{
				Topic: topic,
			},
		}

		sigRepo := sigrepo.NewSignatureRepo(state)
		opRepo, err := oprepo.NewOperationRepo(state, topic)
		if err != nil {
			return nodes, fmt.Errorf("failed to init operation repo: %v", err)
		}

		opService := operation.NewOperationService(opRepo)
		sigService := signature.NewSignatureService(sigRepo)

		fsmService := fsmservice.NewFSMService(state, stg, "")
		sp := services.ServiceProvider{}
		sp.SetLogger(logger)
		sp.SetState(state)
		sp.SetKeyStore(keyStore)
		sp.SetStorage(stg)
		sp.SetFSMService(fsmService)
		sp.SetOperationService(opService)
		sp.SetSignatureService(sigService)

		clt, err := node.NewNode(ctx, &cfg, &sp)
		if err != nil {
			return nodes, fmt.Errorf("nodeInstance %d failed to init nodeInstance: %v\n", nodeID, err)
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

		server := http_api.NewRESTApi(&cfg, clt, &sp)

		instance := &nodeInstance{
			ctx:                   ctx,
			client:                clt,
			clientCancel:          cancel,
			clientLogger:          logger,
			storage:               stg,
			keyPair:               keyPair,
			sigService:            sigService,
			fsmService:   fsmService,
			air:                   airgappedMachine,
			listenAddr:            fmt.Sprintf("localhost:%d", startingPort),
			httpApi:               server,
			operationHandlersMu:   &sync.Mutex{},
			operationHandlers:     make(map[types.OperationType]operationHandler),
			necessaryOperationsMu: &sync.Mutex{},
			necessaryOperations:   make(map[types.OperationType]struct{}),

		}
		instance.setOperationHandler(types.OperationType(signature_fsm.StateAwaitParticipantsConfirmations), instance.awaitParticipantConfirmationsHandler)
		nodes[nodeID] = instance
		startingPort++
	}

	return nodes, err
}

func getOperations(url string) (*OperationsResponse, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get operations for nodeInstance %w", err)
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
		return fmt.Errorf("request to node API failed: %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read body %v", err)
	}

	var response api_responses.BaseResponse
	if err = json.Unmarshal(responseBody, &response); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}
	if response.ErrorMessage != "" {
		return fmt.Errorf("node returned an error response: %s", response.ErrorMessage)
	}
	return nil
}

func startDkg(nodes []*nodeInstance, threshold int) ([]byte, error) {
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

	var response api_responses.BaseResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal HTTP response body: %w", err)
	}

	if len(response.ErrorMessage) != 0 {
		return nil, fmt.Errorf("failed to propose sign message: %s", response.ErrorMessage)
	}
	return messageDataBz, nil
}

func signBatchMessages(dkgID []byte, msg map[string][]byte, addr string) ([]byte, error) {

	messageDataBz, err := json.Marshal(map[string]interface{}{"data": msg,
		"dkgID": dkgID})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SignatureProposalParticipantsListRequest: %v\n", err)
	}

	resp, err := http.Post(fmt.Sprintf("http://%s/proposeSignBatchMessages", addr),
		"application/json", bytes.NewReader(messageDataBz))
	if err != nil {
		return nil, fmt.Errorf("failed to send HTTP request to sign message: %v\n", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read HTTP response body: %w", err)
	}

	var response api_responses.BaseResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal HTTP response body: %w", err)
	}

	if len(response.ErrorMessage) != 0 {
		return nil, fmt.Errorf("failed to propose sign message: %s", response.ErrorMessage)
	}
	return messageDataBz, nil
}

func (n *nodeInstance) setOperationHandler(operationType types.OperationType, handler operationHandler) {
	n.operationHandlersMu.Lock()
	defer n.operationHandlersMu.Unlock()
	n.operationHandlers[operationType] = handler
}

func (n *nodeInstance) getOperationHandler(operationType types.OperationType) operationHandler {
	n.operationHandlersMu.Lock()
	defer n.operationHandlersMu.Unlock()
	return n.operationHandlers[operationType]
}

func (n *nodeInstance) addNecessaryOperations(operations ...types.OperationType) {
	n.necessaryOperationsMu.Lock()
	defer n.necessaryOperationsMu.Unlock()
	for _, operationType := range operations {
		n.necessaryOperations[operationType] = struct{}{}
	}
}

func (n *nodeInstance) isNecessaryOperation(operation types.OperationType) bool {
	n.necessaryOperationsMu.Lock()
	defer n.necessaryOperationsMu.Unlock()
	if len(n.necessaryOperations) == 0 {
		return true
	}
	_, ex := n.necessaryOperations[operation]
	return ex
}

func (n *nodeInstance) run(callback processedOperationCallback, ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			n.client.GetLogger().Log("nodeInstance.run() stopped for %s", n.client.GetUsername())
			return
		default:
			operationsResponse, err := getOperations(fmt.Sprintf("http://%s/getOperations", n.listenAddr))
			if err != nil {
				panic(fmt.Sprintf("failed to get operations: %v", err))
			}

			operations := make(map[string]*types.Operation)
			for opID, op := range operationsResponse.Result {
				if n.isNecessaryOperation(op.Type) {
					operations[opID] = op
				}
			}
			if len(operations) == 0 {
				time.Sleep(pollPauseDuration)
				continue
			}

			n.client.GetLogger().Log("Got %d Operations from pool", len(operations))
			for _, operation := range operations {
				handler := n.getOperationHandler(operation.Type)
				if handler == nil {
					handler = n.defaultOperationHandler
				}
				if err := handler(operation, callback); err != nil {
					panic(err)
				}
			}
			time.Sleep(pollPauseDuration)
		}
	}
}

func (n *nodeInstance) awaitParticipantConfirmationsHandler(operation *types.Operation, _ processedOperationCallback) error {
	payloadBz, err := json.Marshal(map[string]string{"operationID": operation.ID})
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	resp, err := http.Post(fmt.Sprintf("http://%s/approveDKGParticipation", n.listenAddr), "application/json", bytes.NewReader(payloadBz))
	if err != nil {
		return fmt.Errorf("failed to make HTTP request to get operation: %w", err)
	}

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read body %w", err)
	}
	resp.Body.Close()

	var response api_responses.BaseResponse
	if err = json.Unmarshal(responseBody, &response); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}
	if response.ErrorMessage != "" {
		return fmt.Errorf("failed to approve participation: %s", response.ErrorMessage)
	}
	return nil
}

func (n *nodeInstance) defaultOperationHandler(operation *types.Operation, callback processedOperationCallback) error {
	n.client.GetLogger().Log("Handling operation %s in airgapped", operation.Type)
	processedOperation, err := n.air.GetOperationResult(*operation)
	if err != nil {
		n.client.GetLogger().Log("Failed to handle operation: %v", err)
	}
	n.client.GetLogger().Log("Operation %s handled in airgapped, result event is %s",
		operation.Type, processedOperation.Event)

	if err := handleFinishDKG(processedOperation); err != nil {
		return err
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
	return nil
}

// editAndSignMessageHandler is an operationHandler that edits the content of the message that has
// been proposed to be signed.
func (n *nodeInstance) editAndSignMessageHandler(operation *types.Operation, callback processedOperationCallback) error {
	var operationPayload fsm_responses.SigningPartialSignsParticipantInvitationsResponse
	err := json.Unmarshal(operation.Payload, &operationPayload)
	if err != nil {
		return fmt.Errorf("failed to unmarshal operation payload: %w", err)
	}

	var msgs []*requests.MessageToSign
	if err = json.Unmarshal(operationPayload.SrcPayload, &msgs); err != nil {
		return fmt.Errorf("failed to unmarshal messages to sign: %w", err)
	}

	msgs[0].Payload = append(msgs[0].Payload, []byte(" edited")...)
	if operationPayload.SrcPayload, err = json.Marshal(msgs); err != nil {
		return fmt.Errorf("failed to marshal edited messages to sign: %w", err)
	}

	if operation.Payload, err = json.Marshal(operationPayload); err != nil {
		return fmt.Errorf("failed to marshal edited operation payload: %w", err)
	}
	return n.defaultOperationHandler(operation, callback)
}

// junkSignMessageHandler is an operationHandler that spoofs the airgapped response signature.
func (n *nodeInstance) junkSignMessageHandler(operation *types.Operation, _ processedOperationCallback) error {
	n.client.GetLogger().Log("Handling operation %s in airgapped", operation.Type)
	processedOperation, err := n.air.GetOperationResult(*operation)
	if err != nil {
		n.client.GetLogger().Log("Failed to handle operation: %v", err)
	}
	n.client.GetLogger().Log("Operation %s handled in airgapped, result event is %s",
		operation.Type, processedOperation.Event)
	if err := spoofSignature(processedOperation); err != nil {
		return err
	}

	if err = handleProcessedOperation(fmt.Sprintf("http://%s/handleProcessedOperationJSON", n.listenAddr),
		processedOperation); err != nil {
		n.client.GetLogger().Log("Failed to handle processed operation: %v", err)
	} else {
		n.client.GetLogger().Log("Successfully handled processed operation %s", processedOperation.Event)
	}
	return nil
}

// messageHandlerWithReplacedDKG creates an operationHandler that replaces the operation DKG identifier.
func (n *nodeInstance) messageHandlerWithReplacedDKG(dkg string) operationHandler {
	return func(operation *types.Operation, callback processedOperationCallback) error {
		originalDKG := operation.DKGIdentifier

		operation.DKGIdentifier = dkg
		n.client.GetLogger().Log("Handling operation %s in airgapped", operation.Type)
		processedOperation, err := n.air.GetOperationResult(*operation)
		if err != nil {
			n.client.GetLogger().Log("Failed to handle operation: %v", err)
		}
		n.client.GetLogger().Log("Operation %s handled in airgapped, result event is %s",
			operation.Type, processedOperation.Event)

		processedOperation.DKGIdentifier = originalDKG
		for idx := range processedOperation.ResultMsgs {
			processedOperation.ResultMsgs[idx].DkgRoundID = originalDKG
		}

		if err = handleProcessedOperation(fmt.Sprintf("http://%s/handleProcessedOperationJSON", n.listenAddr),
			processedOperation); err != nil {
			n.client.GetLogger().Log("Failed to handle processed operation: %v", err)
		} else {
			n.client.GetLogger().Log("Successfully handled processed operation %s", processedOperation.Event)
		}
		return nil
	}
}

// handleFinishDKG is used for integration tests.
func handleFinishDKG(operation types.Operation) error {
	if operation.Event == dkg_proposal_fsm.EventDKGMasterKeyConfirmationReceived {
		msg := operation.ResultMsgs[0]
		var pubKeyReq requests.DKGProposalMasterKeyConfirmationRequest
		if err := json.Unmarshal(msg.Data, &pubKeyReq); err != nil {
			return fmt.Errorf("failed to unmarshal pubKey request: %w", err)
		}
		if err := ioutil.WriteFile(fmt.Sprintf("/tmp/participant_%d.pubkey",
			pubKeyReq.ParticipantId), []byte(hex.EncodeToString(pubKeyReq.MasterKey)), 0666); err != nil {
			return fmt.Errorf("failed to write pubkey to temp file: %w", err)
		}
	}
	return nil
}

// spoofSignature replaces the first operation message signature with a junk one.
func spoofSignature(operation types.Operation) error {
	var messageData requests.SigningProposalBatchPartialSignRequests
	if err := json.Unmarshal(operation.ResultMsgs[0].Data, &messageData); err != nil {
		return fmt.Errorf("failed to unmarshal message data: %w", err)
	}
	messageData.PartialSigns[0].Sign = []byte("junk signature")
	messageDataEncoded, err := json.Marshal(messageData)
	if err != nil {
		return fmt.Errorf("failed to marshal edited message data: %w", err)
	}
	operation.ResultMsgs[0].Data = messageDataEncoded
	return nil
}

func startServerRunAndPoll(nodes []*nodeInstance, callback processedOperationCallback) context.CancelFunc {
	runCtx, runCancel := context.WithCancel(context.Background())
	for nodeID, n := range nodes {
		go func(nodeID int, node *nodeInstance) {
			if err := node.httpApi.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				panic(fmt.Sprintf("failed to start HTTP server for nodeID #%d: %v\n", nodeID, err))
			}
			//if err := node.client.StartHTTPServer(node.listenAddr); err != nil && err != http.ErrServerClosed {
			//	panic(fmt.Sprintf("failed to start HTTP server for nodeID #%d: %v\n", nodeID, err))
			//}
		}(nodeID, n)
		time.Sleep(1 * time.Second)
		go nodes[nodeID].run(callback, runCtx)

		go func(nodeID int, node node.NodeService) {
			if err := node.Poll(); err != nil {
				panic(fmt.Sprintf("nodeInstance %d poller failed: %v\n", nodeID, err))
			}
		}(nodeID, n.client)

		log.Printf("nodeInstance %d started...\n", nodeID)
	}

	return runCancel
}

func verifySignatures(dkgID string, n *nodeInstance) error {
	allSignatures, err := n.sigService.GetSignatures(&dto.DkgIdDTO{DkgID: dkgID})
	if err != nil {
		return fmt.Errorf("failed to get signatures: %w", err)
	}

	//verifying on airgapped node
	for _, batchSignatures := range allSignatures {
		for _, participantReconstructedSignatures := range batchSignatures {
			for _, s := range participantReconstructedSignatures {
				err = n.air.VerifySign(s.SrcPayload, s.Signature, dkgID)
				if err != nil {
					return fmt.Errorf("failed to verify on airgapped: %w", err)
				}
			}
		}
	}

	//Verify with prysm compability
	keyrings, err := n.air.GetBLSKeyrings()
	if err != nil {
		return fmt.Errorf("failed to get a list of finished dkgs: %w", err)
	}

	pubkeyBz, err := keyrings[dkgID].PubPoly.Commit().MarshalBinary()
	if err != nil {
		return fmt.Errorf("failed to marshal pubkey: %w", err)
	}

	pubKeyBase64 := base64.StdEncoding.EncodeToString(pubkeyBz)

	for _, batchSignatures := range allSignatures {
		//prepare tmp data dir
		dir, err := ioutil.TempDir("/tmp", "dc4bc_messages_")
		if err != nil {
			return fmt.Errorf("failed to create tmp messages dir: %w", err)
		}

		defer os.RemoveAll(dir)
		for _, participantReconstructedSignatures := range batchSignatures {
			s := participantReconstructedSignatures[0]
			f, err := os.OpenFile(path.Join(dir, s.File), os.O_WRONLY|os.O_CREATE, 0600)
			if err != nil {
				return fmt.Errorf("filed to crete tmp file: %w", err)
			}
			defer f.Close()

			_, err = f.Write(s.SrcPayload)
			if err != nil {
				return fmt.Errorf("filed to write to tmp file: %w", err)
			}
		}

		prepared, err := utils.PrepareSignaturesToDump(batchSignatures)
		if err != nil {
			return fmt.Errorf("failed to convert signatures to \"export\" format: %w", err)
		}

		err = prysm.BatchVerification(*prepared, pubKeyBase64, dir)
		if err != nil {
			return fmt.Errorf("failed to make prysm verifification: %w", err)
		}

		fmt.Printf("%d signatures verified with prysm compability\n", len(allSignatures))
	}

	return nil
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

	// Each nodeInstance starts to Poll().
	runCancel := startServerRunAndPoll(nodes, nil)

	// Last nodeInstance tells other participants to start DKG.
	messageDataBz, err := startDkg(nodes, threshold)
	if err != nil {
		t.Fatal(err.Error())
	}

	waitForDKG()

	log.Println("Propose message to sign")

	dkgID := sha256.Sum256(messageDataBz)
	messageDataBz, err = signMessage(dkgID[:], "message to sign", nodes[len(nodes)-1].listenAddr)
	if err != nil {
		t.Fatal(err.Error())
	}
	waitForSignMsg()

	for _, n := range nodes {
		if matches := n.clientLogger.checkLogsWithRegexp(sigReconstructionStartedRegexp, 70); matches != 1 {
			t.Fatalf("not enough checks: %d", matches)
		} else {
			fmt.Println("reconstruction started")
		}
	}

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
	waitForSignMsg()

	for _, n := range nodes {
		if matches := n.clientLogger.checkLogsWithRegexp(sigReconstructedRegexp, 100); matches != 8 {
			t.Fatalf("not enough checks: %d", matches)
		} else {
			fmt.Println("messaged signed successfully")
		}
	}

	runCancel()
	for _, node := range nodes {
		node.httpApi.Stop(node.ctx)
		node.clientCancel()
	}
}

func TestStandardBatchFlow(t *testing.T) {
	_ = RemoveContents("/tmp", "dc4bc_*")
	defer func() { _ = RemoveContents("/tmp", "dc4bc_*") }()

	numNodes := 4
	threshold := 2
	startingPort := 8105
	topic := "test_topic"
	storagePath := "/tmp/dc4bc_storage"
	nodes, err := initNodes(numNodes, startingPort, storagePath, topic, nil)
	if err != nil {
		t.Fatalf("Failed to init nodes, err: %v", err)
	}

	// Each nodeInstance starts to Poll().
	runCancel := startServerRunAndPoll(nodes, nil)

	// Last nodeInstance tells other participants to start DKG.
	messageDataBz, err := startDkg(nodes, threshold)
	if err != nil {
		t.Fatal(err.Error())
	}

	waitForDKG()

	log.Println("Propose messages to sign")
	messagesToSign := map[string][]byte{
		"messageID1": []byte("message to sign1"),
		"messageID2": []byte("message to sign2"),
	}

	dkgID := sha256.Sum256(messageDataBz)
	messageDataBz, err = signBatchMessages(dkgID[:], messagesToSign, nodes[len(nodes)-1].listenAddr)
	if err != nil {
		t.Fatal(err.Error())
	}
	waitForSignMsg()

	for _, n := range nodes {
		if matches := n.clientLogger.checkLogsWithRegexp(sigReconstructedRegexp, 170); matches != 4 {
			t.Fatalf("not enough checks: %d", matches)
		} else {
			fmt.Println("messaged signed successfully")
		}
	}

	fmt.Println("Sign messages again")
	messageDataBz, err = json.Marshal(
		map[string]interface{}{
			"data": map[string][]byte{
				"messageID3": []byte("message to sign3"),
				"messageID4": []byte("message to sign4"),
			},
			"dkgID": dkgID})
	if err != nil {
		t.Fatal(err.Error())
	}

	if _, err := http.Post(fmt.Sprintf("http://%s/proposeSignBatchMessages", nodes[len(nodes)-1].listenAddr),
		"application/json", bytes.NewReader(messageDataBz)); err != nil {
		t.Fatalf("failed to send HTTP request to sign message: %v\n", err)
	}
	waitForSignMsg()

	for _, n := range nodes {
		if matches := n.clientLogger.checkLogsWithRegexp(sigReconstructedRegexp, 100); matches != 8 {
			t.Fatalf("not enough checks: %d", matches)
		} else {
			fmt.Println("messaged signed successfully")
		}
	}

	err = verifySignatures(hex.EncodeToString(dkgID[:]), nodes[0])
	if err != nil {
		t.Fatalf("failed to verify signatures: %v\n", err)
	}

	runCancel()
	for _, node := range nodes {
		node.httpApi.Stop(node.ctx)
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
	processedOperationCallback := func(n *nodeInstance, processedOperation *types.Operation) {
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

	// Each nodeInstance starts to Poll().
	runCancel := startServerRunAndPoll(nodes, processedOperationCallback)
	// Last nodeInstance tells other participants to start DKG.
	messageDataBz, err := startDkg(nodes, threshold)
	if err != nil {
		t.Fatal(err.Error())
	}
	dkgID := sha256.Sum256(messageDataBz)

	waitForDKG()

	for _, n := range nodes {
		if matches := n.clientLogger.checkLogsWithRegexp(dkgAbortedRegexp, 30); matches < 1 {
			t.Fatalf("not enough checks: %d", matches)
		}
	}

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

	log.Print("\n\n\nResetting nodes states\n\n\n")
	if err := resetNodesStates(nodes, []string{msgToIgnore}, false); err != nil {
		t.Fatalf("failed to reset nodes states: %v", err)
	}
	waitForResetState()
	log.Print("\n\n\nState recreated\n\n\n")

	log.Println("Propose message to sign")
	messageDataBz, err = signMessage(dkgID[:], "message to sign", nodes[len(nodes)-1].listenAddr)
	if err != nil {
		t.Fatal(err.Error())
	}
	waitForSignMsg()

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
	waitForSignMsg()

	for _, n := range nodes {
		if matches := n.clientLogger.checkLogsWithRegexp(sigReconstructedRegexp, 70); matches != 8 {
			t.Fatalf("not enough checks: %d", matches)
		} else {
			fmt.Println("messaged signed successfully")
		}
	}

	runCancel()
	for _, node := range nodes {
		node.httpApi.Stop(node.ctx)
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
		if fsm.Event(m.Event) == dkg_proposal_fsm.EventDKGMasterKeyConfirmationReceived {
			// removing PolyPub from old log
			var req requests.DKGProposalMasterKeyConfirmationRequest
			err := json.Unmarshal(m.Data, &req)
			if err != nil {
				panic("failed to unmarshal data")
			}
			req.PubPolyBz = nil
			m.Data, err = json.Marshal(req)
			if err != nil {
				panic("failed to marshal data")
			}
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
	if convertDKGTo10_1_4 {
		startingPort = 8100
	}
	topic := "test_topic"
	storagePath := "/tmp/dc4bc_storage"
	nodes, err := initNodes(numNodes, startingPort, storagePath, topic, mnemonics)
	if err != nil {
		t.Fatalf("Failed to init nodes, err: %v", err)
	}

	// Each nodeInstance starts to Poll().
	runCancel := startServerRunAndPoll(nodes, nil)

	// Last nodeInstance tells other participants to start DKG.
	messageDataBz, err := startDkg(nodes, threshold)
	if err != nil {
		t.Fatal(err.Error())
	}
	dkgID := sha256.Sum256(messageDataBz)

	waitForDKG()

	log.Println("Propose message to sign")
	messagesToSign := map[string][]byte{
		"messageID1": []byte("message to sign1"),
		"messageID2": []byte("message to sign2"),
	}

	messageDataBz, err = signBatchMessages(dkgID[:], messagesToSign, nodes[len(nodes)-1].listenAddr)
	if err != nil {
		t.Fatal(err.Error())
	}
	waitForSignMsg()

	for _, n := range nodes {
		if matches := n.clientLogger.checkLogsWithRegexp(sigReconstructedRegexp, 70); matches != 4 {
			t.Fatalf("not enough checks: %d", matches)
		} else {
			fmt.Println("messaged signed successfully")
		}
	}

	err = verifySignatures(hex.EncodeToString(dkgID[:]), nodes[0])
	if err != nil {
		t.Fatalf("failed to verify signatures: %v\n", err)
	}

	fmt.Println("Reinit DKG...")
	fmt.Println("-----------------------------------------------------------------------------------")
	runCancel()
	for _, node := range nodes {
		node.httpApi.Stop(node.ctx)
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

	waitForNodesStop()

	var newStoragePath = "/tmp/dc4bc_new_storage"
	newNodes, err := initNodes(numNodes, startingPort, newStoragePath, topic, mnemonics)
	if err != nil {
		t.Fatalf("Failed to init nodes, err: %v", err)
	}

	// Each nodeInstance starts to Poll().
	runCancel = startServerRunAndPoll(newNodes, nil)

	var newCommPubKeys = map[string][]byte{}
	reInitDKG, err := types.GenerateReDKGMessage(oldMessages, newCommPubKeys)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if convertDKGTo10_1_4 {
		// removing dkg_proposal_fsm.EventDKGDealConfirmationReceived self-confirm messages
		// to make reInitDKG message looks like in release 0.1.4
		newDKG := convertDKGMessageto0_1_4(*reInitDKG)

		// adding back self-confirm messages
		// this is our test target
		adaptedReDKG, err := node.GetAdaptedReDKG(&newDKG)
		if err != nil {
			t.Fatalf(err.Error())
		}
		reInitDKG = adaptedReDKG

		// skip messages signature verification, since we are unable to sign self-confirm messages by old priv key
		//for _, nodeInstance := range newNodes {
		//	nodeInstance.nodeInstance.SetSkipCommKeysVerification(true)
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

	waitForDKG()

	messageDataBz, err = signBatchMessages(dkgID[:], messagesToSign, nodes[len(nodes)-1].listenAddr)
	if err != nil {
		t.Fatal(err.Error())
	}
	waitForSignMsg()

	for _, n := range newNodes {
		if matches := n.clientLogger.checkLogsWithRegexp(sigReconstructedRegexp, 40); matches != 4 {
			t.Fatalf("not enough checks: %d", matches)
		} else {
			fmt.Println("message signed successfully")
		}
	}
	err = verifySignatures(hex.EncodeToString(dkgID[:]), nodes[0])
	if err != nil {
		t.Fatalf("failed to verify signatures: %v\n", err)
	}

	fmt.Println("Sign message again")
	messageDataBz, err = json.Marshal(
		map[string]interface{}{
			"data": map[string][]byte{
				"messageID3": []byte("message to sign3"),
				"messageID4": []byte("message to sign4"),
			},
			"dkgID": dkgID})
	if err != nil {
		t.Fatal(err.Error())
	}

	if _, err := http.Post(fmt.Sprintf("http://%s/proposeSignBatchMessages", nodes[len(nodes)-1].listenAddr),
		"application/json", bytes.NewReader(messageDataBz)); err != nil {
		t.Fatalf("failed to send HTTP request to sign message: %v\n", err)
	}

	waitForSignMsg()

	for _, n := range newNodes {
		if matches := n.clientLogger.checkLogsWithRegexp(sigReconstructedRegexp, 40); matches < 4 {
			t.Fatalf("not enough checks: %d", matches)
		} else {
			fmt.Println("messaged signed successfully")
		}
	}
	err = verifySignatures(hex.EncodeToString(dkgID[:]), nodes[0])
	if err != nil {
		t.Fatalf("failed to verify signatures: %v\n", err)
	}

	runCancel()
	for _, node := range newNodes {
		node.httpApi.Stop(node.ctx)
		node.clientCancel()
	}
}

func TestReinitDKGFlow(t *testing.T) {
	testReinitDKGFlow(t, false)
}

func TestReinitDKGFlowWithDump0_1_4(t *testing.T) {
	testReinitDKGFlow(t, true)
}

func TestModifiedMessageSigned(t *testing.T) {
	_ = RemoveContents("/tmp", "dc4bc_*")
	defer func() { _ = RemoveContents("/tmp", "dc4bc_*") }()

	numNodes := 2
	threshold := 2
	startingPort := 8085
	topic := "test_topic"
	storagePath := "/tmp/dc4bc_storage"
	nodes, err := initNodes(numNodes, startingPort, storagePath, topic, nil)
	if err != nil {
		t.Fatalf("Failed to init nodes, err: %v", err)
	}

	maliciousNodeIdx := 0
	maliciousNode := nodes[maliciousNodeIdx]
	maliciousNode.setOperationHandler(types.OperationType(signing_fsm.StateSigningAwaitPartialSigns), maliciousNode.editAndSignMessageHandler)
	soundNodeIdx := 1

	// Each nodeInstance starts to Poll().
	runCancel := startServerRunAndPoll(nodes, nil)

	// Last nodeInstance tells other participants to start DKG.
	messageDataBz, err := startDkg(nodes, threshold)
	if err != nil {
		t.Fatal(err.Error())
	}
	waitForDKG()

	log.Println("Propose message to sign")
	dkgID := sha256.Sum256(messageDataBz)
	messageDataBz, err = signMessage(dkgID[:], "message to sign", nodes[soundNodeIdx].listenAddr)
	if err != nil {
		t.Fatal(err.Error())
	}
	waitForSignMsg()

	for _, n := range nodes {
		if matches := n.clientLogger.checkLogsWithRegexp(sigReconstructionStartedRegexp, 50); matches != 0 {
			t.Fatalf("signature reconstruction should not have started")
		}
		if matches := n.clientLogger.checkLogsWithRegexp(sigReconstructedRegexp, 50); matches != 0 {
			t.Fatalf("signature should not have been reconstructed")
		}
	}
	if matches := maliciousNode.clientLogger.checkLogsWithRegexp(processedOperationPayloadMismatchRegexp, 70); matches == 0 {
		t.Fatalf("an operations' payloads mismatch error is expected for the malicious node")
	}

	maliciousNode.setOperationHandler(types.OperationType(signing_fsm.StateSigningAwaitPartialSigns), maliciousNode.defaultOperationHandler)
	fmt.Println("Sign message again without malware")
	waitForSignMsg()
	for _, n := range nodes {
		if matches := n.clientLogger.checkLogsWithRegexp(sigReconstructionStartedRegexp, 50); matches != 1 {
			t.Fatalf("signature reconstruction should have started for all nodes")
		}
		if matches := n.clientLogger.checkLogsWithRegexp(sigReconstructedRegexp, 50); matches != 2 {
			t.Fatalf("signature reconstruction should have succeeded for all nodes")
		}
	}

	runCancel()
	for _, node := range nodes {
		node.httpApi.Stop(node.ctx)
		node.clientCancel()
	}
}

func TestJunkPartialSignature(t *testing.T) {
	_ = RemoveContents("/tmp", "dc4bc_*")
	defer func() { _ = RemoveContents("/tmp", "dc4bc_*") }()

	numNodes := 2
	threshold := 2
	startingPort := 8085
	topic := "test_topic"
	storagePath := "/tmp/dc4bc_storage"
	nodes, err := initNodes(numNodes, startingPort, storagePath, topic, nil)
	if err != nil {
		t.Fatalf("Failed to init nodes, err: %v", err)
	}

	maliciousNodeIdx := 0
	maliciousNode := nodes[maliciousNodeIdx]
	maliciousNode.setOperationHandler(types.OperationType(signing_fsm.StateSigningAwaitPartialSigns), maliciousNode.junkSignMessageHandler)
	soundNodeIdx := 1
	soundNode := nodes[soundNodeIdx]

	// Each nodeInstance starts to Poll().
	runCancel := startServerRunAndPoll(nodes, nil)

	// Last nodeInstance tells other participants to start DKG.
	messageDataBz, err := startDkg(nodes, threshold)
	if err != nil {
		t.Fatal(err.Error())
	}
	waitForDKG()

	log.Println("Propose message to sign")
	dkgID := sha256.Sum256(messageDataBz)
	messageDataBz, err = signMessage(dkgID[:], "message to sign", soundNode.listenAddr)
	if err != nil {
		t.Fatal(err.Error())
	}
	waitForSignMsg()

	for _, n := range nodes {
		if matches := n.clientLogger.checkLogsWithRegexp(sigReconstructionStartedRegexp, 20); matches != 1 {
			t.Fatalf("signature reconstruction should have started")
		}
		if matches := n.clientLogger.checkLogsWithRegexp(sigReconstructedRegexp, 20); matches != 0 {
			t.Fatalf("signature should not have been reconstructed")
		}
		if matches := n.clientLogger.checkLogsWithRegexp(failedSignRecoverRegexp, 20); matches != 1 {
			t.Fatalf("signature reconstruction should have failed")
		}
	}

	spoiledMessageOffset := strconv.Itoa(maliciousNode.clientLogger.findNodePartialSignMsgOffset(10))
	fmt.Printf("\n\nReset nodes states ignoring msg with offset=%s and sign the message again without malware\n", spoiledMessageOffset)
	maliciousNode.setOperationHandler(types.OperationType(signing_fsm.StateSigningAwaitPartialSigns), maliciousNode.defaultOperationHandler)
	if err := resetNodesStates(nodes, []string{spoiledMessageOffset}, true); err != nil {
		t.Fatalf("failed to reset nodes states: %v", err)
	}
	for _, n := range nodes {
		n.addNecessaryOperations(types.OperationType(signing_fsm.StateSigningAwaitPartialSigns))
		n.clientLogger.resetLogs() // to perform next signing checks only with relative logs
	}

	waitForResetState()
	for _, n := range nodes {
		if matches := n.clientLogger.checkLogsWithRegexp(sigReconstructionStartedRegexp, 50); matches != 1 {
			t.Fatalf("signature reconstruction should have started for all nodes")
		}
		if matches := n.clientLogger.checkLogsWithRegexp(sigReconstructedRegexp, 50); matches != 2 {
			t.Fatalf("signature reconstruction should have succeeded for all nodes")
		}
	}

	runCancel()
	for _, node := range nodes {
		node.httpApi.Stop(node.ctx)
		node.clientCancel()
	}
}

func TestSignWithDifferentDKG(t *testing.T) {
	_ = RemoveContents("/tmp", "dc4bc_*")
	defer func() { _ = RemoveContents("/tmp", "dc4bc_*") }()

	numNodes := 3
	startingPort := 8085
	topic := "test_topic"
	storagePath := "/tmp/dc4bc_storage"
	nodes, err := initNodes(numNodes, startingPort, storagePath, topic, nil)
	if err != nil {
		t.Fatalf("Failed to init nodes, err: %v", err)
	}

	maliciousNodeIdx := 0
	maliciousNode := nodes[maliciousNodeIdx]
	soundNodeIdx := 1
	soundNode := nodes[soundNodeIdx]

	// Each nodeInstance starts to Poll().
	runCancel := startServerRunAndPoll(nodes, nil)

	fmt.Printf("\n\nStarting the first DKG round\n")
	// Last nodeInstance tells other participants to start DKG.
	messageDataBz, err := startDkg(nodes, numNodes)
	if err != nil {
		t.Fatal(err.Error())
	}
	waitForDKG()
	firstDkgID := sha256.Sum256(messageDataBz)

	fmt.Printf("\n\nStarting the second DKG round\n")
	// Last nodeInstance tells other participants to start another DKG.
	messageDataBz, err = startDkg(nodes, numNodes-1) // different threshold to create different keys
	if err != nil {
		t.Fatal(err.Error())
	}
	waitForDKG()
	secondDkgID := sha256.Sum256(messageDataBz)

	log.Println("Propose message to sign")
	maliciousNode.setOperationHandler(types.OperationType(signing_fsm.StateSigningAwaitPartialSigns), maliciousNode.messageHandlerWithReplacedDKG(fmt.Sprintf("%x", secondDkgID)))
	messageDataBz, err = signMessage(firstDkgID[:], "message to sign", soundNode.listenAddr)
	if err != nil {
		t.Fatal(err.Error())
	}
	waitForSignMsg()

	for _, n := range nodes {
		if matches := n.clientLogger.checkLogsWithRegexp(sigReconstructionStartedRegexp, 20); matches != 1 {
			t.Fatalf("signature reconstruction should have started")
		}
		if matches := n.clientLogger.checkLogsWithRegexp(sigReconstructedRegexp, 20); matches != 0 {
			t.Fatalf("signature should not have been reconstructed")
		}
		if matches := n.clientLogger.checkLogsWithRegexp(failedSignRecoverRegexp, 20); matches != 1 {
			t.Fatalf("signature reconstruction should have failed")
		}
	}

	spoiledMessageOffset := strconv.Itoa(maliciousNode.clientLogger.findNodePartialSignMsgOffset(10))
	fmt.Printf("\n\nReset nodes states ignoring msg with offset=%s and sign the message again without malware\n", spoiledMessageOffset)
	maliciousNode.setOperationHandler(types.OperationType(signing_fsm.StateSigningAwaitPartialSigns), maliciousNode.defaultOperationHandler)
	if err := resetNodesStates(nodes, []string{spoiledMessageOffset}, true); err != nil {
		t.Fatalf("failed to reset nodes states: %v", err)
	}
	for _, n := range nodes {
		n.addNecessaryOperations(types.OperationType(signing_fsm.StateSigningAwaitPartialSigns))
		n.clientLogger.resetLogs() // to perform next signing checks only with relative logs
	}

	waitForSignMsg()
	for _, n := range nodes {
		if matches := n.clientLogger.checkLogsWithRegexp(sigReconstructionStartedRegexp, 40); matches != 1 {
			t.Fatalf("signature reconstruction should have started for all nodes")
		}
		if matches := n.clientLogger.checkLogsWithRegexp(sigReconstructedRegexp, 40); matches != 3 {
			t.Fatalf("signature reconstruction should have succeeded for all nodes")
		}
	}

	runCancel()
	for _, node := range nodes {
		node.httpApi.Stop(node.ctx)
		node.clientCancel()
	}
}

func resetNodesStates(nodes []*nodeInstance, ignoreMsgs []string, offsets bool) error {
	for _, node := range nodes {
		resetReq := httprequests.ResetStateForm{
			UseOffset: offsets,
			Messages:  ignoreMsgs,
		}
		resetReqBz, err := json.Marshal(resetReq)
		if err != nil {
			return fmt.Errorf("failed to marshal ResetStateRequest: %w", err)
		}
		if _, err := http.Post(fmt.Sprintf("http://%s/resetState", node.listenAddr),
			"application/json", bytes.NewReader(resetReqBz)); err != nil {
			return fmt.Errorf("failed to send HTTP request to reset state: %w", err)
		}
	}
	return nil
}

func waitForDKG() {
	time.Sleep(dkgDuration)
}

func waitForSignMsg() {
	time.Sleep(signMsgDuration)
}

func waitForResetState() {
	time.Sleep(resetStateDuration)
}

func waitForNodesStop() {
	time.Sleep(nodesStopDuration)
}
