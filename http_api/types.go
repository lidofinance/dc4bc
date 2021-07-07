package http_api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Response struct {
	ErrorMessage string      `json:"error_message,omitempty"`
	Result       interface{} `json:"result"`
}

type ResetStateRequest struct {
	NewStateDBDSN      string   `json:"new_state_dbdsn,omitempty"`
	UseOffset          bool     `json:"use_offset"`
	KafkaConsumerGroup string   `json:"kafka_consumer_group"`
	Messages           []string `json:"messages,omitempty"`
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
