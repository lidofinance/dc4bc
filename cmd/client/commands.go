package main

import (
	"encoding/json"
	"fmt"
	"github.com/depools/dc4bc/client/types"
	"github.com/spf13/cobra"
	"io/ioutil"
	"net/http"
)

type OperationsResponse struct {
	ErrorMessage string                      `json:"error_message,omitempty"`
	Result       map[string]*types.Operation `json:"result"`
}

func getOperations(host string) (*OperationsResponse, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s/getOperations", host))
	if err != nil {
		return nil, fmt.Errorf("failed to get operations for node %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body %w", err)
	}

	var response OperationsResponse
	if err = json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}
	return &response, nil
}

func getOperationsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get_operations",
		Short: "returns all operations that should be processed on the airgapped machine",
		RunE: func(cmd *cobra.Command, args []string) error {
			listenAddr, err := cmd.Flags().GetString(flagListenAddr)
			if err != nil {
				return fmt.Errorf("failed to read configuration: %v", err)
			}
			operations, err := getOperations(listenAddr)
			if err != nil {
				return fmt.Errorf("failed to get operations: %w", err)
			}
			if operations.ErrorMessage != "" {
				return fmt.Errorf("failed to get operations: %s", operations.ErrorMessage)
			}
			for _, operation := range operations.Result {
				fmt.Printf("Operation ID: %s\n", operation.ID)
				operationBz, err := json.Marshal(operation)
				if err != nil {
					return fmt.Errorf("failed to marshal operation: %w", err)
				}
				fmt.Printf("Operation: %s\n", string(operationBz))
				fmt.Println("-----------------------------------------------------")
			}
			return nil
		},
	}
}
