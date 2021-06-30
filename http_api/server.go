package http_api

import (
	"context"
	"crypto/ed25519"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/lidofinance/dc4bc/client/operations"

	"github.com/lidofinance/dc4bc/common"

	"github.com/google/uuid"
	"github.com/lidofinance/dc4bc/client"
	"github.com/lidofinance/dc4bc/client/types"
	"github.com/lidofinance/dc4bc/fsm/fsm"
	spf "github.com/lidofinance/dc4bc/fsm/state_machines/signature_proposal_fsm"
	sif "github.com/lidofinance/dc4bc/fsm/state_machines/signing_proposal_fsm"
	"github.com/lidofinance/dc4bc/fsm/types/requests"
	"github.com/lidofinance/dc4bc/fsm/types/responses"
	"github.com/lidofinance/dc4bc/qr"
	"github.com/lidofinance/dc4bc/storage"
)

type API interface {
	Start(listenAddr string) error
	Stop() error
}

type NaiveHttpAPI struct {
	server      *http.Server
	Logger      types.Logger
	Client      client.Client
	QrProcessor qr.Processor
}

func NewNaiveHttpAPI(client client.Client, qrProcessor qr.Processor) API {
	return &NaiveHttpAPI{
		Logger:      common.NewLogger(client.GetUsername()),
		Client:      client,
		QrProcessor: qrProcessor,
	}
}

func (api *NaiveHttpAPI) Start(listenAddr string) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/getUsername", api.getUsernameHandler)
	mux.HandleFunc("/getPubKey", api.getPubkeyHandler)

	mux.HandleFunc("/sendMessage", api.sendMessageHandler)
	mux.HandleFunc("/getOperations", api.getOperationsHandler)
	mux.HandleFunc("/getOperationQRPath", api.getOperationQRPathHandler)

	mux.HandleFunc("/getSignatures", api.getSignaturesHandler)
	mux.HandleFunc("/getSignatureByID", api.getSignatureByIDHandler)

	mux.HandleFunc("/getOperationQR", api.getOperationQRToBodyHandler)
	mux.HandleFunc("/handleProcessedOperationJSON", api.handleJSONOperationHandler)
	mux.HandleFunc("/getOperation", api.getOperationHandler)

	mux.HandleFunc("/startDKG", api.startDKGHandler)
	mux.HandleFunc("/proposeSignMessage", api.proposeSignDataHandler)
	mux.HandleFunc("/approveDKGParticipation", api.approveParticipationHandler)
	mux.HandleFunc("/reinitDKG", api.reinitDKGHandler)

	mux.HandleFunc("/saveOffset", api.saveOffsetHandler)
	mux.HandleFunc("/getOffset", api.getOffsetHandler)

	mux.HandleFunc("/getFSMDump", api.getFSMDumpHandler)
	mux.HandleFunc("/getFSMList", api.getFSMList)

	mux.HandleFunc("/resetState", api.resetStateHandler)

	api.server = &http.Server{Addr: listenAddr, Handler: mux}
	api.Logger.Log("HTTP server started on address: %s", listenAddr)

	return api.server.ListenAndServe()
}

func (api *NaiveHttpAPI) Stop() error {
	return api.server.Shutdown(context.Background())
}

func (api *NaiveHttpAPI) getFSMDumpHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errorResponse(w, http.StatusBadRequest, "Wrong HTTP method")
		return
	}
	fsmInstance, err := api.Client.GetFSMInstance(r.URL.Query().Get("dkgID"))
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	fsmDump, _ := fsmInstance.Dump()

	successResponse(w, fsmDump)
}

func (api *NaiveHttpAPI) getFSMList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errorResponse(w, http.StatusBadRequest, "Wrong HTTP method")
		return
	}
	fsmInstances, err := api.Client.GetState().GetAllFSM()
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

func (api *NaiveHttpAPI) getUsernameHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errorResponse(w, http.StatusBadRequest, "Wrong HTTP method")
		return
	}
	successResponse(w, api.Client.GetUsername())
}

func (api *NaiveHttpAPI) getPubkeyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errorResponse(w, http.StatusBadRequest, "Wrong HTTP method")
		return
	}
	successResponse(w, api.Client.GetPubKey())
}

func (api *NaiveHttpAPI) getOffsetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errorResponse(w, http.StatusBadRequest, "Wrong HTTP method")
		return
	}
	offset, err := api.Client.GetState().LoadOffset()
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to load offset: %v", err))
		return
	}
	successResponse(w, offset)
}

func (api *NaiveHttpAPI) saveOffsetHandler(w http.ResponseWriter, r *http.Request) {
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
	if err = api.Client.GetState().SaveOffset(req["offset"]); err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to save offset: %v", err))
		return
	}
	successResponse(w, "ok")
}

func (api *NaiveHttpAPI) sendMessageHandler(w http.ResponseWriter, r *http.Request) {
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

	if err = api.Client.SendMessage(msg); err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to send message to the storage: %v", err))
		return
	}

	successResponse(w, "ok")
}

func (api *NaiveHttpAPI) getOperationsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errorResponse(w, http.StatusBadRequest, "Wrong HTTP method")
		return
	}

	operations, err := api.Client.GetState().GetOperations()
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to get operations: %v", err))
		return
	}

	successResponse(w, operations)
}

func (api *NaiveHttpAPI) getSignaturesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errorResponse(w, http.StatusBadRequest, "Wrong HTTP method")
		return
	}

	signatures, err := api.Client.GetState().GetSignatures(r.URL.Query().Get("dkgID"))
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to get signatures: %v", err))
		return
	}

	successResponse(w, signatures)
}

func (api *NaiveHttpAPI) getSignatureByIDHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errorResponse(w, http.StatusBadRequest, "Wrong HTTP method")
		return
	}

	signature, err := api.Client.GetState().GetSignatureByID(r.URL.Query().Get("dkgID"), r.URL.Query().Get("id"))
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to get signature: %v", err))
		return
	}

	successResponse(w, signature)
}

func (api *NaiveHttpAPI) getOperationQRPathHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errorResponse(w, http.StatusBadRequest, "Wrong HTTP method")
		return
	}

	operationID := r.URL.Query().Get("operationID")
	operation, err := api.Client.GetState().GetOperationByID(operationID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to get operation: %v", err))
		return
	}

	qrPath := qr.GetDefaultGIFPath(operation)
	operationJSON, err := operation.ToJson()
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to get operation JSON: %v", err))
		return
	}

	if err := api.QrProcessor.WriteQR(qrPath, operationJSON); err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to get operation QR path: %v", err))
		return
	}

	successResponse(w, qrPath)
}

func (api *NaiveHttpAPI) getOperationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errorResponse(w, http.StatusBadRequest, "Wrong HTTP method")
		return
	}
	operationID := r.URL.Query().Get("operationID")

	operation, err := api.Client.GetState().GetOperationByID(operationID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to get operation: %v", err))
		return
	}

	successResponse(w, operation)
}

func (api *NaiveHttpAPI) getOperationQRToBodyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errorResponse(w, http.StatusBadRequest, "Wrong HTTP method")
		return
	}
	operationID := r.URL.Query().Get("operationID")

	operation, err := api.Client.GetState().GetOperationByID(operationID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to get operation: %v", err))
		return
	}

	operationJson, err := operation.ToJson()
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to get operation JSON: %v", err))
	}

	encodedData, err := qr.EncodeQR(operationJson)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to encode operation: %v", err))
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(encodedData)))
	rawResponse(w, encodedData)
}

func (api *NaiveHttpAPI) startDKGHandler(w http.ResponseWriter, r *http.Request) {
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
	message, err := api.Client.SignMessage(hex.EncodeToString(dkgRoundID[:]), spf.EventInitProposal, reqBody)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to build message: %v", err))
		return
	}
	if err = api.Client.SendMessage(*message); err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to send message: %v", err))
		return
	}
	successResponse(w, "ok")
}

func (api *NaiveHttpAPI) approveParticipationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorResponse(w, http.StatusBadRequest, "Wrong HTTP method")
		return
	}
	decoder := json.NewDecoder(r.Body)

	var req map[string]string
	err := decoder.Decode(&req)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to umarshal request: %v", err))
		return
	}

	operationID, ok := req["operationID"]
	if !ok {
		errorResponse(w, http.StatusBadRequest, fmt.Sprintf("operationID is required: %v", err))
		return
	}

	operations, err := api.Client.GetState().GetOperations()
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to get operations: %v", err))
		return
	}

	operation, ok := operations[operationID]
	if !ok {
		errorResponse(w, http.StatusNotFound, fmt.Sprintf("operation %s not found", operationID))
		return
	}
	if fsm.State(operation.Type) != spf.StateAwaitParticipantsConfirmations {
		errorResponse(w, http.StatusBadRequest, fmt.Sprintf("cannot approve participation with operationID %s", operationID))
		return
	}

	var payload responses.SignatureProposalParticipantInvitationsResponse
	if err = json.Unmarshal(operation.Payload, &payload); err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to unmarshal payload: %v", err))
		return
	}

	pid := -1
	for _, p := range payload {
		if api.Client.GetPubKey().Equal(ed25519.PublicKey(p.PubKey)) {
			pid = p.ParticipantId
			break
		}
	}
	if pid < 0 {
		errorResponse(w, http.StatusInternalServerError, "failed to determine participant id")
		return
	}

	fsmRequest := requests.SignatureProposalParticipantRequest{
		ParticipantId: pid,
		CreatedAt:     operation.CreatedAt,
	}
	reqBz, err := json.Marshal(fsmRequest)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to generate FSM request: %v", err))
		return
	}

	operation.Event = spf.EventConfirmSignatureProposal
	operation.ResultMsgs = append(operation.ResultMsgs, storage.Message{
		Event:         string(operation.Event),
		Data:          reqBz,
		DkgRoundID:    operation.DKGIdentifier,
		RecipientAddr: operation.To,
	})

	if err = api.Client.HandleProcessedOperation(*operation); err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to handle processed operation: %v", err))
		return
	}

	successResponse(w, "ok")
}

func (api *NaiveHttpAPI) proposeSignDataHandler(w http.ResponseWriter, r *http.Request) {
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

	fsmInstance, err := api.Client.GetFSMInstance(hex.EncodeToString(req["dkgID"]))
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to get FSM instance: %v", err))
		return
	}
	participantID, err := fsmInstance.GetIDByUsername(api.Client.GetUsername())
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to get participantID: %v", err))
		return
	}

	messageDataSign := requests.SigningProposalStartRequest{
		SigningID:     uuid.New().String(),
		ParticipantId: participantID,
		SrcPayload:    req["data"],
		CreatedAt:     time.Now(),
	}
	messageDataSignBz, err := json.Marshal(messageDataSign)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to marshal SigningProposalStartRequest: %v", err))
		return
	}

	message, err := api.Client.SignMessage(hex.EncodeToString(req["dkgID"]), sif.EventSigningStart, messageDataSignBz)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to build message: %v", err))
		return
	}
	if err = api.Client.SendMessage(*message); err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to send message: %v", err))
		return
	}
	successResponse(w, "ok")
}

func (api *NaiveHttpAPI) handleJSONOperationHandler(w http.ResponseWriter, r *http.Request) {
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

	var req operations.Operation
	if err = json.Unmarshal(reqBody, &req); err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to umarshal request: %v", err))
		return
	}

	if err = api.Client.HandleProcessedOperation(req); err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to handle processed operation: %v", err))
		return
	}

	successResponse(w, "ok")
}

func (api *NaiveHttpAPI) resetStateHandler(w http.ResponseWriter, r *http.Request) {
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

	var req ResetStateRequest
	if err = json.Unmarshal(reqBody, &req); err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to umarshal request: %v", err))
		return
	}

	newStateDbPath, err := api.Client.ResetState(req.NewStateDBDSN, req.KafkaConsumerGroup, req.Messages, req.UseOffset)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to reset state: %v", err))
		return
	}

	successResponse(w, newStateDbPath)
}

func (api *NaiveHttpAPI) reinitDKGHandler(w http.ResponseWriter, r *http.Request) {
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

	var req operations.ReDKG
	if err = json.Unmarshal(reqBody, &req); err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to umarshal request: %v", err))
		return
	}

	message, err := api.Client.SignMessage(req.DKGID, fsm.Event(operations.ReinitDKG), reqBody)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to build message: %v", err))
		return
	}

	if err = api.Client.SendMessage(*message); err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to send message: %v", err))
		return
	}
	successResponse(w, "ok")
}
