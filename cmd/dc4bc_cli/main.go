package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/depools/dc4bc/client"
	"github.com/depools/dc4bc/client/types"
	"github.com/depools/dc4bc/fsm/types/requests"
	"github.com/depools/dc4bc/qr"
	"github.com/spf13/cobra"
)

const (
	flagListenAddr = "listen_addr"
)

func init() {
	rootCmd.PersistentFlags().String(flagListenAddr, "localhost:8080", "Listen Address")
}

var rootCmd = &cobra.Command{
	Use:   "dc4bc_cli",
	Short: "dc4bc client cli utilities implementation",
}

func main() {
	rootCmd.AddCommand(
		getOperationsCommand(),
		getOperationQRPathCommand(),
		readOperationFromCameraCommand(),
		startDKGCommand(),
		proposeSignMessageCommand(),
		getAddressCommand(),
		getPubKeyCommand(),
	)
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Failed to execute root command: %v", err)
	}
}

type OperationsResponse struct {
	ErrorMessage string                      `json:"error_message,omitempty"`
	Result       map[string]*types.Operation `json:"result"`
}

func getOperationsRequest(host string) (*OperationsResponse, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s/getOperations", host))
	if err != nil {
		return nil, fmt.Errorf("failed to get operations: %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
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
			operations, err := getOperationsRequest(listenAddr)
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

type OperationQRPathsResponse struct {
	ErrorMessage string   `json:"error_message,omitempty"`
	Result       []string `json:"result"`
}

func getOperationsQRPathsRequest(host string, operationID string) (*OperationQRPathsResponse, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s/getOperationQRPath?operationID=%s", host, operationID))
	if err != nil {
		return nil, fmt.Errorf("failed to get operation: %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	var response OperationQRPathsResponse
	if err = json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}
	return &response, nil
}

func getOperationQRPathCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get_operation_qr [operationID]",
		Args:  cobra.ExactArgs(1),
		Short: "returns path to QR codes which contains the operation",
		RunE: func(cmd *cobra.Command, args []string) error {
			listenAddr, err := cmd.Flags().GetString(flagListenAddr)
			if err != nil {
				return fmt.Errorf("failed to read configuration: %v", err)
			}
			operationID := args[0]
			operations, err := getOperationsQRPathsRequest(listenAddr, operationID)
			if err != nil {
				return fmt.Errorf("failed to get operations: %w", err)
			}
			if operations.ErrorMessage != "" {
				return fmt.Errorf("failed to get operations: %s", operations.ErrorMessage)
			}
			fmt.Printf("List of paths to QR codes for operation %s:\n", operationID)
			for idx, path := range operations.Result {
				fmt.Printf("%d) QR code: %s\n", idx, path)
			}
			return nil
		},
	}
}

func rawGetRequest(url string) (*client.Response, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get operations for node %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body %w", err)
	}

	var response client.Response
	if err = json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}
	return &response, nil
}

func getPubKeyCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get_pubkey",
		Short: "returns client's pubkey",
		RunE: func(cmd *cobra.Command, args []string) error {
			listenAddr, err := cmd.Flags().GetString(flagListenAddr)
			if err != nil {
				return fmt.Errorf("failed to read configuration: %v", err)
			}

			resp, err := rawGetRequest(fmt.Sprintf("http://%s//getPubKey", listenAddr))
			if err != nil {
				return fmt.Errorf("failed to get client's pubkey: %w", err)
			}
			if resp.ErrorMessage != "" {
				return fmt.Errorf("failed to get client's pubkey: %w", resp.ErrorMessage)
			}
			fmt.Println(resp.Result.(string))
			return nil
		},
	}
}

func getAddressCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get_address",
		Short: "returns client's address",
		RunE: func(cmd *cobra.Command, args []string) error {
			listenAddr, err := cmd.Flags().GetString(flagListenAddr)
			if err != nil {
				return fmt.Errorf("failed to read configuration: %v", err)
			}

			resp, err := rawGetRequest(fmt.Sprintf("http://%s//getAddress", listenAddr))
			if err != nil {
				return fmt.Errorf("failed to get client's address: %w", err)
			}
			if resp.ErrorMessage != "" {
				return fmt.Errorf("failed to get client's address: %w", resp.ErrorMessage)
			}
			fmt.Println(resp.Result.(string))
			return nil
		},
	}
}

func rawPostRequest(url string, contentType string, data []byte) (*client.Response, error) {
	resp, err := http.Post(url,
		contentType, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body %w", err)
	}

	var response client.Response
	if err = json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}
	return &response, nil
}

func readOperationFromCameraCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "read_from_camera",
		Short: "opens the camera and reads QR codes which should contain a processed operation",
		RunE: func(cmd *cobra.Command, args []string) error {
			listenAddr, err := cmd.Flags().GetString(flagListenAddr)
			if err != nil {
				return fmt.Errorf("failed to read configuration: %v", err)
			}

			processor := qr.NewCameraProcessor()
			data, err := qr.ReadDataFromQRChunks(processor)
			if err != nil {
				return fmt.Errorf("failed to read data from QR: %w", err)
			}
			resp, err := rawPostRequest(fmt.Sprintf("http://%s/handleProcessedOperationJSON", listenAddr),
				"application/json", data)
			if err != nil {
				return fmt.Errorf("failed to handle processed operation: %w", err)
			}
			if resp.ErrorMessage != "" {
				return fmt.Errorf("failed to handle processed operation: %w", resp.ErrorMessage)
			}
			return nil
		},
	}
}

func startDKGCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "start_dkg [participants count] [threshold]",
		Args:  cobra.ExactArgs(2),
		Short: "sends a propose message to start a DKG process",
		RunE: func(cmd *cobra.Command, args []string) error {
			listenAddr, err := cmd.Flags().GetString(flagListenAddr)
			if err != nil {
				return fmt.Errorf("failed to read configuration: %v", err)
			}

			participantsCount, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("failed to get participants count: %w", err)
			}
			if participantsCount < 0 {
				return fmt.Errorf("invalid number of participants: %d", participantsCount)
			}

			threshold, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("failed to get threshold: %w", err)
			}
			if participantsCount < 0 || threshold > participantsCount {
				return fmt.Errorf("invalid threshold: %d", threshold)
			}

			reader := bufio.NewReader(os.Stdin)
			var participants []*requests.SignatureProposalParticipantsEntry
			for i := 0; i < participantsCount; i++ {
				p := &requests.SignatureProposalParticipantsEntry{}
				fmt.Printf("Enter a necessary data for participant %d:\n", i)
				fmt.Printf("Enter address: ")
				addr, _, err := reader.ReadLine()
				if err != nil {
					return fmt.Errorf("failed to read addr: %w", err)
				}
				p.Addr = string(addr)

				fmt.Printf("Enter pubkey (base64): ")
				pubKey, _, err := reader.ReadLine()
				if err != nil {
					return fmt.Errorf("failed to read pubKey: %w", err)
				}
				p.PubKey, err = base64.StdEncoding.DecodeString(string(pubKey))
				if err != nil {
					return fmt.Errorf("failed to decode pubKey: %w", err)
				}

				fmt.Printf("Enter DKGPubKey (base64): ")
				DKGPubKey, _, err := reader.ReadLine()
				if err != nil {
					return fmt.Errorf("failed to read DKGPubKey: %w", err)
				}
				p.DkgPubKey, err = base64.StdEncoding.DecodeString(string(DKGPubKey))
				if err != nil {
					return fmt.Errorf("failed to decode DKGPubKey: %w", err)
				}
				participants = append(participants, p)
			}

			messageData := requests.SignatureProposalParticipantsListRequest{
				Participants:     participants,
				SigningThreshold: threshold,
				CreatedAt:        time.Now(),
			}
			messageDataBz, err := json.Marshal(messageData)
			if err != nil {
				return fmt.Errorf("failed to marshal SignatureProposalParticipantsListRequest: %v\n", err)
			}
			resp, err := rawPostRequest(fmt.Sprintf("http://%s/startDKG", listenAddr),
				"application/json", messageDataBz)
			if err != nil {
				return fmt.Errorf("failed to make HTTP request to start DKG: %w", err)
			}
			if resp.ErrorMessage != "" {
				return fmt.Errorf("failed to make HTTP request to start DKG: %w", resp.ErrorMessage)
			}
			return nil
		},
	}
}

func proposeSignMessageCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "sign_data [dkg_id] [data]",
		Args:  cobra.ExactArgs(2),
		Short: "sends a propose message to sign the data",
		RunE: func(cmd *cobra.Command, args []string) error {
			listenAddr, err := cmd.Flags().GetString(flagListenAddr)
			if err != nil {
				return fmt.Errorf("failed to read configuration: %v", err)
			}

			dkgID, err := hex.DecodeString(args[0])
			if err != nil {
				return fmt.Errorf("failed to decode dkgID: %w", err)
			}

			data, err := base64.StdEncoding.DecodeString(args[1])
			if err != nil {
				return fmt.Errorf("failed to decode data")
			}

			messageDataSign := requests.SigningProposalStartRequest{
				ParticipantId: 0, //TODO: determine participantID
				SrcPayload:    data,
				CreatedAt:     time.Now(),
			}
			messageDataSignBz, err := json.Marshal(messageDataSign)
			if err != nil {
				return fmt.Errorf("failed to marshal SigningProposalStartRequest: %v\n", err)
			}

			messageDataBz, err := json.Marshal(map[string][]byte{"data": messageDataSignBz,
				"dkgID": dkgID})
			if err != nil {
				return fmt.Errorf("failed to marshal SigningProposalStartRequest: %v\n", err)
			}

			resp, err := rawPostRequest(fmt.Sprintf("http://%s/proposeSignMessage", listenAddr),
				"application/json", messageDataBz)
			if err != nil {
				return fmt.Errorf("failed to make HTTP request to propose message to sign: %w", err)
			}
			if resp.ErrorMessage != "" {
				return fmt.Errorf("failed to make HTTP request to propose message to sign: %w", resp.ErrorMessage)
			}
			return nil
		},
	}
}
