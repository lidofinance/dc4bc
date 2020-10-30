package main

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/depools/dc4bc/fsm/state_machines"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/depools/dc4bc/fsm/fsm"
	"github.com/depools/dc4bc/fsm/state_machines/signature_proposal_fsm"
	"github.com/depools/dc4bc/fsm/state_machines/signing_proposal_fsm"
	"github.com/depools/dc4bc/fsm/types/responses"

	"github.com/depools/dc4bc/client"
	"github.com/depools/dc4bc/fsm/types/requests"
	"github.com/depools/dc4bc/qr"
	"github.com/spf13/cobra"
)

const (
	flagListenAddr    = "listen_addr"
	flagFramesDelay   = "frames_delay"
	flagChunkSize     = "chunk_size"
	flagQRCodesFolder = "qr_codes_folder"
)

func init() {
	rootCmd.PersistentFlags().String(flagListenAddr, "localhost:8080", "Listen Address")
	rootCmd.PersistentFlags().Int(flagFramesDelay, 10, "Delay times between frames in 100ths of a second")
	rootCmd.PersistentFlags().Int(flagChunkSize, 256, "QR-code's chunk size")
	rootCmd.PersistentFlags().String(flagQRCodesFolder, "/tmp", "Folder to save QR codes")
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
		getUsernameCommand(),
		getPubKeyCommand(),
		getHashOfStartDKGCommand(),
		getSignaturesCommand(),
		getSignatureCommand(),
		saveOffsetCommand(),
		getOffsetCommand(),
		getFSMStatusCommand(),
		getFSMListCommand(),
	)
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Failed to execute root command: %v", err)
	}
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
				fmt.Printf("DKG round ID: %s\n", operation.DKGIdentifier)
				fmt.Printf("Operation ID: %s\n", operation.ID)
				fmt.Printf("Description: %s\n", getShortOperationDescription(operation.Type))
				if fsm.State(operation.Type) == signature_proposal_fsm.StateAwaitParticipantsConfirmations {
					payloadHash, err := calcStartDKGMessageHash(operation.Payload)
					if err != nil {
						return fmt.Errorf("failed to get hash of start DKG message: %w", err)
					}
					fmt.Printf("Hash of the proposing DKG message - %s\n", hex.EncodeToString(payloadHash))
				}
				if fsm.State(operation.Type) == signing_proposal_fsm.StateSigningAwaitConfirmations {
					var payload responses.SigningProposalParticipantInvitationsResponse
					if err := json.Unmarshal(operation.Payload, &payload); err != nil {
						return fmt.Errorf("failed to unmarshal operation payload")
					}
					msgHash := md5.Sum(payload.SrcPayload)
					fmt.Printf("Hash of the message to sign - %s\n", hex.EncodeToString(msgHash[:]))
				}
				fmt.Println("-----------------------------------------------------")
			}
			return nil
		},
	}
}

func getSignaturesRequest(host string, dkgID string) (*SignaturesResponse, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s/getSignatures?dkgID=%s", host, dkgID))
	if err != nil {
		return nil, fmt.Errorf("failed to get signatures: %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	var response SignaturesResponse
	if err = json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}
	return &response, nil
}

func getSignaturesCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get_signatures [dkgID]",
		Args:  cobra.ExactArgs(1),
		Short: "returns all signatures for the given DKG round that were reconstructed on the airgapped machine",
		RunE: func(cmd *cobra.Command, args []string) error {
			listenAddr, err := cmd.Flags().GetString(flagListenAddr)
			if err != nil {
				return fmt.Errorf("failed to read configuration: %v", err)
			}
			signatures, err := getSignaturesRequest(listenAddr, args[0])
			if err != nil {
				return fmt.Errorf("failed to get signatures: %w", err)
			}
			if signatures.ErrorMessage != "" {
				return fmt.Errorf("failed to get signatures: %s", signatures.ErrorMessage)
			}
			for dataHash, signature := range signatures.Result {
				fmt.Printf("Hash of the signing data: %s\n", dataHash)
				for _, participantSig := range signature {
					fmt.Printf("\tDKG round ID: %s\n", participantSig.DKGRoundID)
					fmt.Printf("\tParticipant: %s\n", participantSig.Participant)
					fmt.Printf("\tReconstructed signature for the data: %s\n", base64.StdEncoding.EncodeToString(participantSig.Signature))
					fmt.Println()
				}
			}
			return nil
		},
	}
}

func getSignatureRequest(host string, dkgID, dataHash string) (*SignatureResponse, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s/getSignatureByDataHash?dkgID=%s&hash=%s", host, dkgID, dataHash))
	if err != nil {
		return nil, fmt.Errorf("failed to get signatures: %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	var response SignatureResponse
	if err = json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}
	return &response, nil
}

func getSignatureCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get_signature [dkgID] [hash_of_the_signed_data]",
		Args:  cobra.ExactArgs(2),
		Short: "returns a list of reconstructed signatures of the signed data broadcasted by users",
		RunE: func(cmd *cobra.Command, args []string) error {
			listenAddr, err := cmd.Flags().GetString(flagListenAddr)
			if err != nil {
				return fmt.Errorf("failed to read configuration: %v", err)
			}
			signatures, err := getSignatureRequest(listenAddr, args[0], args[1])
			if err != nil {
				return fmt.Errorf("failed to get signatures: %w", err)
			}
			if signatures.ErrorMessage != "" {
				return fmt.Errorf("failed to get signatures: %s", signatures.ErrorMessage)
			}
			for _, participantSig := range signatures.Result {
				fmt.Printf("\tParticipant: %s\n", participantSig.Participant)
				fmt.Printf("\tReconstructed signature for the data: %s\n", base64.StdEncoding.EncodeToString(participantSig.Signature))
				fmt.Println()
			}
			return nil
		},
	}
}

func getOperationRequest(host string, operationID string) (*OperationResponse, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s/getOperation?operationID=%s", host, operationID))
	if err != nil {
		return nil, fmt.Errorf("failed to get operation: %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	var response OperationResponse
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
			framesDelay, err := cmd.Flags().GetInt(flagFramesDelay)
			if err != nil {
				return fmt.Errorf("failed to read configuration: %w", err)
			}
			chunkSize, err := cmd.Flags().GetInt(flagChunkSize)
			if err != nil {
				return fmt.Errorf("failed to read configuration: %w", err)
			}
			qrCodeFolder, err := cmd.Flags().GetString(flagQRCodesFolder)
			if err != nil {
				return fmt.Errorf("failed to read configuration: %w", err)
			}

			operationID := args[0]
			operation, err := getOperationRequest(listenAddr, operationID)
			if err != nil {
				return fmt.Errorf("failed to get operations: %w", err)
			}
			if operation.ErrorMessage != "" {
				return fmt.Errorf("failed to get operations: %s", operation.ErrorMessage)
			}

			operationQRPath := filepath.Join(qrCodeFolder, fmt.Sprintf("dc4bc_qr_%s-request", operationID))

			qrPath := fmt.Sprintf("%s.gif", operationQRPath)

			processor := qr.NewCameraProcessor()
			processor.SetChunkSize(chunkSize)
			processor.SetDelay(framesDelay)

			if err = processor.WriteQR(qrPath, operation.Result); err != nil {
				return fmt.Errorf("failed to save QR gif: %w", err)
			}

			fmt.Printf("QR code was saved to: %s\n", qrPath)
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
				return fmt.Errorf("failed to get client's pubkey: %v", resp.ErrorMessage)
			}
			fmt.Println(resp.Result.(string))
			return nil
		},
	}
}

func saveOffsetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "save_offset [offset]",
		Short: "saves a new offset for a storage",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			listenAddr, err := cmd.Flags().GetString(flagListenAddr)
			if err != nil {
				return fmt.Errorf("failed to read configuration: %v", err)
			}

			offset, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse uint: %w", err)
			}
			req := map[string]uint64{"offset": offset}
			data, err := json.Marshal(req)
			if err != nil {
				return fmt.Errorf("failed to create request: %w", err)
			}
			resp, err := rawPostRequest(fmt.Sprintf("http://%s/saveOffset", listenAddr), "application/json", data)
			if err != nil {
				return fmt.Errorf("failed to save offset: %w", err)
			}
			if resp.ErrorMessage != "" {
				return fmt.Errorf("failed to save offset: %v", resp.ErrorMessage)
			}
			fmt.Println(resp.Result.(string))
			return nil
		},
	}
}

func getOffsetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get_offset",
		Short: "returns a current offset for the storage",
		RunE: func(cmd *cobra.Command, args []string) error {
			listenAddr, err := cmd.Flags().GetString(flagListenAddr)
			if err != nil {
				return fmt.Errorf("failed to read configuration: %v", err)
			}

			resp, err := rawGetRequest(fmt.Sprintf("http://%s//getOffset", listenAddr))
			if err != nil {
				return fmt.Errorf("failed to get offset: %w", err)
			}
			if resp.ErrorMessage != "" {
				return fmt.Errorf("failed to get offset: %v", resp.ErrorMessage)
			}
			fmt.Println(uint64(resp.Result.(float64)))
			return nil
		},
	}
}

func getUsernameCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get_username",
		Short: "returns client's username",
		RunE: func(cmd *cobra.Command, args []string) error {
			listenAddr, err := cmd.Flags().GetString(flagListenAddr)
			if err != nil {
				return fmt.Errorf("failed to read configuration: %v", err)
			}

			resp, err := rawGetRequest(fmt.Sprintf("http://%s//getUsername", listenAddr))
			if err != nil {
				return fmt.Errorf("failed to get client's username: %w", err)
			}
			if resp.ErrorMessage != "" {
				return fmt.Errorf("failed to get client's username: %v", resp.ErrorMessage)
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
			data, err := processor.ReadQR()
			if err != nil {
				return fmt.Errorf("failed to read data from QR: %w", err)
			}
			resp, err := rawPostRequest(fmt.Sprintf("http://%s/handleProcessedOperationJSON", listenAddr),
				"application/json", data)
			if err != nil {
				return fmt.Errorf("failed to handle processed operation: %w", err)
			}
			if resp.ErrorMessage != "" {
				return fmt.Errorf("failed to handle processed operation: %v", resp.ErrorMessage)
			}
			return nil
		},
	}
}

func startDKGCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "start_dkg [proposing_file]",
		Args:  cobra.ExactArgs(1),
		Short: "sends a propose message to start a DKG process",
		RunE: func(cmd *cobra.Command, args []string) error {
			listenAddr, err := cmd.Flags().GetString(flagListenAddr)
			if err != nil {
				return fmt.Errorf("failed to read configuration: %v", err)
			}

			dkgProposeFileData, err := ioutil.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}
			var req requests.SignatureProposalParticipantsListRequest
			if err = json.Unmarshal(dkgProposeFileData, &req); err != nil {
				return fmt.Errorf("failed to unmarshal dkg proposing file: %w", err)
			}

			if len(req.Participants) == 0 || req.SigningThreshold > len(req.Participants) {
				return fmt.Errorf("invalid threshold: %d", req.SigningThreshold)
			}
			req.CreatedAt = time.Now()

			messageData := req
			messageDataBz, err := json.Marshal(messageData)
			if err != nil {
				return fmt.Errorf("failed to marshal SignatureProposalParticipantsListRequest: %v", err)
			}
			resp, err := rawPostRequest(fmt.Sprintf("http://%s/startDKG", listenAddr),
				"application/json", messageDataBz)
			if err != nil {
				return fmt.Errorf("failed to make HTTP request to start DKG: %w", err)
			}
			if resp.ErrorMessage != "" {
				return fmt.Errorf("failed to make HTTP request to start DKG: %v", resp.ErrorMessage)
			}
			return nil
		},
	}
}

func getHashOfStartDKGCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get_start_dkg_file_hash [proposing_file]",
		Args:  cobra.ExactArgs(1),
		Short: "returns hash of proposing message for DKG start to verify correctness",
		RunE: func(cmd *cobra.Command, args []string) error {

			dkgProposeFileData, err := ioutil.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}
			var req requests.SignatureProposalParticipantsListRequest
			if err = json.Unmarshal(dkgProposeFileData, &req); err != nil {
				return fmt.Errorf("failed to unmarshal dkg proposing file: %w", err)
			}

			participants := DKGParticipants(req.Participants)
			sort.Sort(participants)

			hashPayload := bytes.NewBuffer(nil)
			if _, err := hashPayload.Write([]byte(fmt.Sprintf("%d", req.SigningThreshold))); err != nil {
				return err
			}
			for _, p := range participants {
				if _, err := hashPayload.Write(p.PubKey); err != nil {
					return err
				}
				if _, err := hashPayload.Write(p.DkgPubKey); err != nil {
					return err
				}
				if _, err := hashPayload.Write([]byte(p.Addr)); err != nil {
					return err
				}
			}
			hash := md5.Sum(hashPayload.Bytes())
			fmt.Println(hex.EncodeToString(hash[:]))
			return nil
		},
	}
}

func proposeSignMessageCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "sign_data [dkg_id] [file_path]",
		Args:  cobra.ExactArgs(2),
		Short: "sends a propose message to sign the data in the file",
		RunE: func(cmd *cobra.Command, args []string) error {
			listenAddr, err := cmd.Flags().GetString(flagListenAddr)
			if err != nil {
				return fmt.Errorf("failed to read configuration: %v", err)
			}

			dkgID, err := hex.DecodeString(args[0])
			if err != nil {
				return fmt.Errorf("failed to decode dkgID: %w", err)
			}

			data, err := ioutil.ReadFile(args[1])
			if err != nil {
				return fmt.Errorf("failed to read the file")
			}

			messageDataBz, err := json.Marshal(map[string][]byte{"data": data,
				"dkgID": dkgID})
			if err != nil {
				return fmt.Errorf("failed to marshal SigningProposalStartRequest: %v", err)
			}

			resp, err := rawPostRequest(fmt.Sprintf("http://%s/proposeSignMessage", listenAddr),
				"application/json", messageDataBz)
			if err != nil {
				return fmt.Errorf("failed to make HTTP request to propose message to sign: %w", err)
			}
			if resp.ErrorMessage != "" {
				return fmt.Errorf("failed to make HTTP request to propose message to sign: %v", resp.ErrorMessage)
			}
			return nil
		},
	}
}

func getFSMDumpRequest(host string, dkgID string) (*FSMDumpResponse, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s/getFSMDump?dkgID=%s", host, dkgID))
	if err != nil {
		return nil, fmt.Errorf("failed to get FSM dump: %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	var response FSMDumpResponse
	if err = json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}
	return &response, nil
}

func getFSMStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show_fsm_status [dkg_id]",
		Args:  cobra.ExactArgs(1),
		Short: "shows the current status of FSM",
		RunE: func(cmd *cobra.Command, args []string) error {
			listenAddr, err := cmd.Flags().GetString(flagListenAddr)
			if err != nil {
				return fmt.Errorf("failed to read configuration: %v", err)
			}

			fsmDumpResponse, err := getFSMDumpRequest(listenAddr, args[0])
			if err != nil {
				return fmt.Errorf("failed to get FSM dump: %w", err)
			}
			if fsmDumpResponse.ErrorMessage != "" {
				return fmt.Errorf("failed to get FSM dump: %v", fsmDumpResponse.ErrorMessage)
			}
			dump := fsmDumpResponse.Result

			fmt.Printf("FSM current status is %s\n", dump.State)

			quorum := make(map[int]state_machines.Participant)
			if strings.HasPrefix(string(dump.State), "state_signing") {
				for k, v := range dump.Payload.SigningProposalPayload.Quorum {
					quorum[k] = v
				}
			}
			if strings.HasPrefix(string(dump.State), "state_dkg") {
				for k, v := range dump.Payload.DKGProposalPayload.Quorum {
					quorum[k] = v
				}
			}
			if strings.HasPrefix(string(dump.State), "state_sig") {
				for k, v := range dump.Payload.SignatureProposalPayload.Quorum {
					quorum[k] = v
				}
			}

			waiting := make([]string, 0)
			confirmed := make([]string, 0)
			failed := make([]string, 0)

			for _, p := range quorum {
				if strings.Contains(p.GetStatus().String(), "Await") {
					waiting = append(waiting, p.GetAddr())
				}
				if strings.Contains(p.GetStatus().String(), "Error") {
					failed = append(failed, p.GetAddr())
				}
				if strings.Contains(p.GetStatus().String(), "Confirmed") {
					confirmed = append(confirmed, p.GetAddr())
				}
			}

			if len(waiting) > 0 {
				fmt.Printf("Waiting for a data from: %s\n", strings.Join(waiting, ", "))
			}
			if len(confirmed) > 0 {
				fmt.Printf("Received a data from: %s\n", strings.Join(confirmed, ", "))
			}
			if len(failed) > 0 {
				fmt.Printf("Participants who got some error during a process: %s\n", strings.Join(waiting, ", "))
			}

			return nil
		},
	}
}

func getFSMListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get_fsm_list",
		Short: "returns a list of all FSMs served by the client",
		RunE: func(cmd *cobra.Command, args []string) error {
			listenAddr, err := cmd.Flags().GetString(flagListenAddr)
			if err != nil {
				return fmt.Errorf("failed to read configuration: %v", err)
			}

			resp, err := rawGetRequest(fmt.Sprintf("http://%s/getFSMList", listenAddr))
			if err != nil {
				return fmt.Errorf("failed to make HTTP request to get FSM list: %w", err)
			}
			if resp.ErrorMessage != "" {
				return fmt.Errorf("failed to make HTTP request to get FSM list: %v", resp.ErrorMessage)
			}
			fsms := resp.Result.(map[string]interface{})
			for dkgID, state := range fsms {
				fmt.Printf("DKG ID: %s - FSM state: %s\n", dkgID, state.(string))
			}
			return nil
		},
	}
}
