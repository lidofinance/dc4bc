package client

import (
	"encoding/json"
	"fmt"
	"github.com/depool/dc4bc/storage"
	"io/ioutil"
	"log"
	"net/http"
)

func errorResponse(w http.ResponseWriter, statusCode int, err string) {
	log.Println(err)
	w.WriteHeader(statusCode)
	if _, err := w.Write([]byte(err)); err != nil {
		panic(fmt.Sprintf("failed to write response: %v", err))
	}
}

func successResponse(w http.ResponseWriter, response []byte) {
	if _, err := w.Write(response); err != nil {
		panic(fmt.Sprintf("failed to write response: %v", err))
	}
}

func (c *Client) StartHTTPServer(listenAddr string) error {
	http.HandleFunc("/sendMessage", c.sendMessageHandler)
	http.HandleFunc("/getOperations", c.getOperationsHandler)
	http.HandleFunc("/getOperationQRPath", c.getOperationQRPathHandler)
	http.HandleFunc("/readProcessedOperationFromCamera", c.readProcessedOperationFromCameraHandler)
	return http.ListenAndServe(listenAddr, nil)
}

func (c *Client) sendMessageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorResponse(w, http.StatusBadRequest, "Wrong HTTP method")
		return
	}
	reqBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, fmt.Sprintf("failed to read request body: %v", err))
		return
	}

	var msg storage.Message
	if err = json.Unmarshal(reqBytes, &msg); err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to unmarshal message: %v", err))
		return
	}

	if err = c.SendMessage(msg); err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to send message to the storage: %v", err))
		return
	}

	successResponse(w, []byte("ok"))
}

func (c *Client) getOperationsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errorResponse(w, http.StatusBadRequest, "Wrong HTTP method")
		return
	}

	operations, err := c.GetOperations()
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to get operations: %v", err))
		return
	}

	response, err := json.Marshal(operations)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to marshal operations: %v", err))
		return
	}

	successResponse(w, response)
}

func (c *Client) getOperationQRPathHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errorResponse(w, http.StatusBadRequest, "Wrong HTTP method")
		return
	}
	operationID := r.URL.Query().Get("operationID")

	qrPath, err := c.GetOperationQRPath(operationID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to get operation QR path: %v", err))
		return
	}

	successResponse(w, []byte(qrPath))
}

func (c *Client) readProcessedOperationFromCameraHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errorResponse(w, http.StatusBadRequest, "Wrong HTTP method")
		return
	}

	if err := c.ReadProcessedOperation(); err != nil {
		errorResponse(w, http.StatusInternalServerError,
			fmt.Sprintf("failed to read processed operation from camera path: %v", err))
		return
	}

	successResponse(w, []byte("ok"))
}
