package client

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/depools/dc4bc/client/types"
	"github.com/depools/dc4bc/fsm/fsm"
	spf "github.com/depools/dc4bc/fsm/state_machines/signature_proposal_fsm"
	sif "github.com/depools/dc4bc/fsm/state_machines/signing_proposal_fsm"
	"github.com/depools/dc4bc/fsm/types/requests"
	"github.com/google/uuid"

	"github.com/depools/dc4bc/qr"
	"github.com/depools/dc4bc/storage"
)

type Response struct {
	ErrorMessage string      `json:"error_message,omitempty"`
	Result       interface{} `json:"result"`
}

func rawResponse(w http.ResponseWriter, response []byte) {
	if _, err := w.Write(response); err != nil {
		panic(fmt.Sprintf("failed to write response: %v", err))
	}
}

func errorResponse(w http.ResponseWriter, statusCode int, error string) {
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "application/json")
	resp := Response{ErrorMessage: error}
	respBz, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Failed to marshal response: %v\n", err)
		return
	}
	if _, err := w.Write(respBz); err != nil {
		panic(fmt.Sprintf("failed to write response: %v", err))
	}
}

func successResponse(w http.ResponseWriter, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	resp := Response{Result: response}
	respBz, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Failed to marshal response: %v\n", err)
		return
	}
	if _, err := w.Write(respBz); err != nil {
		panic(fmt.Sprintf("failed to write response: %v", err))
	}
}

func (c *BaseClient) StartHTTPServer(listenAddr string) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/getUsername", c.getUsernameHandler)
	mux.HandleFunc("/getPubKey", c.getPubkeyHandler)

	mux.HandleFunc("/sendMessage", c.sendMessageHandler)
	mux.HandleFunc("/getOperations", c.getOperationsHandler)
	mux.HandleFunc("/getOperationQRPath", c.getOperationQRPathHandler)

	mux.HandleFunc("/getSignatures", c.getSignaturesHandler)
	mux.HandleFunc("/getSignatureByDataHash", c.getSignatureByDataHashHandler)

	mux.HandleFunc("/getOperationQR", c.getOperationQRToBodyHandler)
	mux.HandleFunc("/handleProcessedOperationJSON", c.handleJSONOperationHandler)
	mux.HandleFunc("/getOperation", c.getOperationHandler)

	mux.HandleFunc("/startDKG", c.startDKGHandler)
	mux.HandleFunc("/proposeSignMessage", c.proposeSignDataHandler)

	mux.HandleFunc("/saveOffset", c.saveOffsetHandler)
	mux.HandleFunc("/getOffset", c.getOffsetHandler)

	mux.HandleFunc("/getFSMDump", c.getFSMDumpHandler)
	mux.HandleFunc("/getFSMList", c.getFSMList)

	c.Logger.Log("Starting HTTP server on address: %s", listenAddr)
	return http.ListenAndServe(listenAddr, mux)
}

func (c *BaseClient) getFSMDumpHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errorResponse(w, http.StatusBadRequest, "Wrong HTTP method")
		return
	}
	dump, err := c.GetFSMDump(r.URL.Query().Get("dkgID"))
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	successResponse(w, dump)
}

func (c *BaseClient) getFSMList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errorResponse(w, http.StatusBadRequest, "Wrong HTTP method")
		return
	}
	fsmInstances, err := c.state.GetAllFSM()
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to get all FSM instances: %v", err))
		return
	}
	fsmInstancesStates := make(map[string]string, len(fsmInstances))
	for k, v := range fsmInstances {
		state, err := v.State()
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to get FSM state: %v", err))
			return
		}
		fsmInstancesStates[k] = state.String()
	}
	successResponse(w, fsmInstancesStates)
}

func (c *BaseClient) getUsernameHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errorResponse(w, http.StatusBadRequest, "Wrong HTTP method")
		return
	}
	successResponse(w, c.GetUsername())
}

func (c *BaseClient) getPubkeyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errorResponse(w, http.StatusBadRequest, "Wrong HTTP method")
		return
	}
	successResponse(w, c.GetPubKey())
}

func (c *BaseClient) getOffsetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errorResponse(w, http.StatusBadRequest, "Wrong HTTP method")
		return
	}
	offset, err := c.state.LoadOffset()
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to load offset: %v", err))
		return
	}
	successResponse(w, offset)
}

func (c *BaseClient) saveOffsetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorResponse(w, http.StatusBadRequest, "Wrong HTTP method")
		return
	}
	reqBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, fmt.Sprintf("failed to read request body: %v", err))
		return
	}
	defer r.Body.Close()

	var req map[string]uint64
	if err = json.Unmarshal(reqBytes, &req); err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to unmarshal request: %v", err))
		return
	}
	if _, ok := req["offset"]; !ok {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("offset cannot be null: %v", err))
		return
	}
	if err = c.state.SaveOffset(req["offset"]); err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to save offset: %v", err))
		return
	}
	successResponse(w, "ok")
}

func (c *BaseClient) sendMessageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorResponse(w, http.StatusBadRequest, "Wrong HTTP method")
		return
	}
	reqBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, fmt.Sprintf("failed to read request body: %v", err))
		return
	}
	defer r.Body.Close()

	var msg storage.Message
	if err = json.Unmarshal(reqBytes, &msg); err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to unmarshal message: %v", err))
		return
	}

	if err = c.SendMessage(msg); err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to send message to the storage: %v", err))
		return
	}

	successResponse(w, "ok")
}

func (c *BaseClient) getOperationsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errorResponse(w, http.StatusBadRequest, "Wrong HTTP method")
		return
	}

	operations, err := c.GetOperations()
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to get operations: %v", err))
		return
	}

	successResponse(w, operations)
}

func (c *BaseClient) getSignaturesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errorResponse(w, http.StatusBadRequest, "Wrong HTTP method")
		return
	}

	signatures, err := c.GetSignatures(r.URL.Query().Get("dkgID"))
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to get signatures: %v", err))
		return
	}

	successResponse(w, signatures)
}

func (c *BaseClient) getSignatureByDataHashHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errorResponse(w, http.StatusBadRequest, "Wrong HTTP method")
		return
	}

	signature, err := c.GetSignatureByDataHash(r.URL.Query().Get("dkgID"), r.URL.Query().Get("hash"))
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to get signature: %v", err))
		return
	}

	successResponse(w, signature)
}

func (c *BaseClient) getOperationQRPathHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errorResponse(w, http.StatusBadRequest, "Wrong HTTP method")
		return
	}
	operationID := r.URL.Query().Get("operationID")

	qrPaths, err := c.GetOperationQRPath(operationID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to get operation QR path: %v", err))
		return
	}

	successResponse(w, qrPaths)
}

func (c *BaseClient) getOperationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errorResponse(w, http.StatusBadRequest, "Wrong HTTP method")
		return
	}
	operationID := r.URL.Query().Get("operationID")

	operation, err := c.getOperationJSON(operationID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to get operation: %v", err))
		return
	}

	successResponse(w, operation)
}

func (c *BaseClient) getOperationQRToBodyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errorResponse(w, http.StatusBadRequest, "Wrong HTTP method")
		return
	}
	operationID := r.URL.Query().Get("operationID")

	operationJSON, err := c.getOperationJSON(operationID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to get operation in JSON: %v", err))
		return
	}

	encodedData, err := qr.EncodeQR(operationJSON)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to encode operation: %v", err))
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(encodedData)))
	rawResponse(w, encodedData)
}

func (c *BaseClient) startDKGHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorResponse(w, http.StatusBadRequest, "Wrong HTTP method")
		return
	}
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to read body: %v", err))
		return
	}
	defer r.Body.Close()

	dkgRoundID := md5.Sum(reqBody)
	message, err := c.buildMessage(hex.EncodeToString(dkgRoundID[:]), spf.EventInitProposal, reqBody)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to build message: %v", err))
		return
	}
	if err = c.SendMessage(*message); err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to send message: %v", err))
		return
	}
	successResponse(w, "ok")
}

func (c *BaseClient) proposeSignDataHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorResponse(w, http.StatusBadRequest, "Wrong HTTP method")
		return
	}
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to read body: %v", err))
		return
	}
	defer r.Body.Close()

	var req map[string][]byte
	if err = json.Unmarshal(reqBody, &req); err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to umarshal request: %v", err))
		return
	}

	fsmInstance, err := c.getFSMInstance(hex.EncodeToString(req["dkgID"]))
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to get FSM instance: %v", err))
		return
	}
	participantID, err := fsmInstance.GetIDByAddr(c.GetUsername())
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to get participantID: %v", err))
		return
	}

	messageDataSign := requests.SigningProposalStartRequest{
		ParticipantId: participantID,
		SrcPayload:    req["data"],
		CreatedAt:     time.Now(),
	}
	messageDataSignBz, err := json.Marshal(messageDataSign)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to marshal SigningProposalStartRequest: %v", err))
		return
	}

	message, err := c.buildMessage(hex.EncodeToString(req["dkgID"]), sif.EventSigningStart, messageDataSignBz)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to build message: %v", err))
		return
	}
	if err = c.SendMessage(*message); err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to send message: %v", err))
		return
	}
	successResponse(w, "ok")
}

func (c *BaseClient) handleJSONOperationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorResponse(w, http.StatusBadRequest, "Wrong HTTP method")
		return
	}
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to read body: %v", err))
		return
	}
	defer r.Body.Close()

	var req types.Operation
	if err = json.Unmarshal(reqBody, &req); err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to umarshal request: %v", err))
		return
	}

	if err = c.handleProcessedOperation(req); err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to handle processed operation: %v", err))
		return
	}

	successResponse(w, "ok")
}

func (c *BaseClient) buildMessage(dkgRoundID string, event fsm.Event, data []byte) (*storage.Message, error) {
	message := storage.Message{
		ID:         uuid.New().String(),
		DkgRoundID: dkgRoundID,
		Event:      string(event),
		Data:       data,
		SenderAddr: c.GetUsername(),
	}
	signature, err := c.signMessage(message.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to sign message: %w", err)
	}
	message.Signature = signature
	return &message, nil
}
