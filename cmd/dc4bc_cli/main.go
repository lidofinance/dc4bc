package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/lidofinance/dc4bc/client/types"

	"github.com/lidofinance/dc4bc/fsm/state_machines"

	"github.com/lidofinance/dc4bc/fsm/fsm"
	"github.com/lidofinance/dc4bc/fsm/types/responses"

	httprequests "github.com/lidofinance/dc4bc/client/api/http_api/requests"
	httpresponses "github.com/lidofinance/dc4bc/client/api/http_api/responses"

	"github.com/fatih/color"
	spf "github.com/lidofinance/dc4bc/fsm/state_machines/signature_proposal_fsm"
	"github.com/lidofinance/dc4bc/fsm/types/requests"
	"github.com/spf13/cobra"
)

const (
	flagListenAddr         = "listen_addr"
	flagJSONFilesFolder    = "json_files_folder"
	flagNewStateDBDSN      = "new_state_dbdsn"
	flagUseOffsetInsteadId = "use_offset_instead_id"
	flagKafkaConsumerGroup = "kafka_consumer_group"
)

var (
	useOffset          bool
	newStateDBDSN      string
	kafkaConsumerGroup string

	rootCmd = &cobra.Command{
		Use:   "dc4bc_cli",
		Short: "dc4bc node cli utilities implementation",
	}

	refreshStateCmd = &cobra.Command{
		Use:   "refresh_state [--use_offset_instead_id | -o] [--new_state_dbsn | -s] [--kafka_consumer_group | -g] [messageId...]",
		Short: "drops current state and replays it from storage ignoring messages with provided ids or offsets",
	}
)

func init() {
	rootCmd.PersistentFlags().String(flagListenAddr, "localhost:8080", "Listen Address")
	rootCmd.PersistentFlags().String(flagJSONFilesFolder, "/tmp", "Folder to save JSON files")

	refreshStateCmd.Flags().BoolVarP(&useOffset, flagUseOffsetInsteadId, "o", false,
		"Ignore messages by offset instead of ids")
	refreshStateCmd.Flags().StringVarP(&newStateDBDSN, flagNewStateDBDSN, "s", "",
		"State DBDSN")
	refreshStateCmd.Flags().StringVarP(&kafkaConsumerGroup, flagKafkaConsumerGroup, "g", "",
		"Kafka consumer group")
}

func main() {
	rootCmd.AddCommand(
		getOperationsCommand(),
		reinitDKGPathCommand(),
		readOperationResultCommand(),
		approveDKGParticipationCommand(),
		startDKGCommand(),
		proposeSignMessageCommand(),
		proposeSignBatchMessagesCommand(),
		getUsernameCommand(),
		getPubKeyCommand(),
		getHashOfStartDKGCommand(),
		getHashOfReinitDKGMessageCommand(),
		getSignaturesCommand(),
		getSignatureCommand(),
		saveOffsetCommand(),
		getOffsetCommand(),
		getFSMStatusCommand(),
		getFSMListCommand(),
		getSignatureDataCommand(),
		refreshState(),
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

			if len(operations.Result) == 0 {
				color.New(color.Bold).Println("The are no available operations yet")
				return nil
			}

			colorTitle := color.New(color.Bold)
			colorDKG := color.New(color.FgCyan)
			colorOperationId := color.New(color.FgGreen)
			colorTitle.Println("Please, select operation:")
			fmt.Println("-----------------------------------------------------")

			actionsMap := map[string]string{}
			actionId := 1
			for operationId, operation := range operations.Result {
				actionsMap[strconv.Itoa(actionId)] = operationId
				fmt.Printf(" %s)\t\t", color.YellowString("%d", actionId))

				colorTitle.Print("DKG round ID:")
				colorDKG.Printf(" %s\n", operation.DKGIdentifier)

				colorTitle.Print("\t\tOperation ID:")
				colorOperationId.Printf(" %s\n", operation.ID)

				colorTitle.Print("\t\tDescription:")
				fmt.Printf(" %s\n", getShortOperationDescription(operation.Type))

				if strings.HasPrefix(string(operation.Type), "state_signing_") {
					var payload responses.SigningPartialSignsParticipantInvitationsResponse
					if err := json.Unmarshal(operation.Payload, &payload); err != nil {
						return fmt.Errorf("failed to unmarshal operation payload")
					}
					msgHash := sha256.Sum256(payload.SrcPayload)
					fmt.Printf("\t\tHash of the data to sign - %s\n", hex.EncodeToString(msgHash[:]))
					fmt.Printf("\t\tSigning ID: %s\n", payload.BatchID)
				}
				if fsm.State(operation.Type) == types.ReinitDKG {
					fmt.Printf("\t\tHash of the reinit DKG message - %s\n", hex.EncodeToString(operation.ExtraData))
				}
				fmt.Println("-----------------------------------------------------")
				actionId++
			}

			colorTitle.Println("Select operation and press Enter. Ctrl+C for cancel")

			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				if operationId, ok := actionsMap[scanner.Text()]; ok {
					colorTitle.Print("Processing operation")
					colorOperationId.Printf(" %s\n", operationId)

					opCmd := &cobra.Command{}

					switch fsm.State(operations.Result[operationId].Type) {
					case spf.StateAwaitParticipantsConfirmations:
						opCmd = approveDKGParticipationCommand()
					default:
						opCmd = getOperationPathCommand()
					}

					opCmd.SetArgs([]string{operationId})
					opCmd.Flags().AddFlagSet(cmd.Flags())
					opCmd.Execute()

					return nil
				}

				color.New(color.FgRed).Println("Unknown operation action")
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
			dkgID := args[0]
			signatures, err := getSignaturesRequest(listenAddr, dkgID)
			if err != nil {
				return fmt.Errorf("failed to get signatures: %w", err)
			}
			if signatures.ErrorMessage != "" {
				return fmt.Errorf("failed to get signatures: %s", signatures.ErrorMessage)
			}
			if len(signatures.Result) == 0 {
				fmt.Printf("No signatures found for dkgID %s", dkgID)
				return nil
			}
			for sigID, signature := range signatures.Result {
				fmt.Printf("Signing ID: %s\n", sigID)
				for _, participantSig := range signature {
					fmt.Printf("\tDKG round ID: %s\n", participantSig.DKGRoundID)
					fmt.Printf("\tParticipant: %s\n", participantSig.Username)
					fmt.Printf("\tReconstructed signature for the data: %s\n", base64.StdEncoding.EncodeToString(participantSig.Signature))
					fmt.Println()
				}
			}
			return nil
		},
	}
}

func getSignatureRequest(host string, dkgID, dataHash string) (*SignatureResponse, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s/getSignatureByID?dkgID=%s&id=%s", host, dkgID, dataHash))
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
		Use:   "get_signature [dkgID] [signing_id]",
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
				fmt.Printf("\tParticipant: %s\n", participantSig.Username)
				fmt.Printf("\tReconstructed signature for the data: %s\n", base64.StdEncoding.EncodeToString(participantSig.Signature))
				fmt.Println()
			}
			return nil
		},
	}
}

func getSignatureDataCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get_signature_data [dkgID] [signing_id]",
		Args:  cobra.ExactArgs(2),
		Short: "returns a data which was signed",
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
			if len(signatures.Result) > 0 {
				fmt.Println(string(signatures.Result[0].SrcPayload))
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

func getOperationPathCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get_operation [operationID]",
		Args:  cobra.ExactArgs(1),
		Short: "returns path to json files which contains the operation",
		RunE: func(cmd *cobra.Command, args []string) error {
			listenAddr, err := cmd.Flags().GetString(flagListenAddr)
			if err != nil {
				return fmt.Errorf("failed to read configuration: %v", err)
			}

			folder, err := cmd.Flags().GetString(flagJSONFilesFolder)
			if err != nil {
				return fmt.Errorf("failed to read configuration: %w", err)
			}

			operationID := args[0]
			operationResponse, err := getOperationRequest(listenAddr, operationID)
			if err != nil {
				return fmt.Errorf("failed to get operations: %w", err)
			}
			if operationResponse.ErrorMessage != "" {
				return fmt.Errorf("failed to get operations: %s", operationResponse.ErrorMessage)
			}

			operationPath := filepath.Join(folder, operationResponse.Result.Filename()+"_request.json")

			f, err := os.OpenFile(operationPath, os.O_WRONLY|os.O_CREATE, 0600)
			if err != nil {
				return fmt.Errorf("failed to open file: %w", err)
			}

			defer f.Close()

			operationBytes, err := json.Marshal(operationResponse.Result)
			if err != nil {
				return fmt.Errorf("failed to get operations: %w", err)
			}

			_, err = f.Write(operationBytes)
			if err != nil {
				return fmt.Errorf("failed to write file: %w", err)
			}

			fmt.Printf("json file was saved to: %s\n", operationPath)

			return nil
		},
	}
}

func reinitDKGPathCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "reinit_dkg [reDKG JSON file path]",
		Args:  cobra.ExactArgs(1),
		Short: "send reinitDKG message to a storage",
		RunE: func(cmd *cobra.Command, args []string) error {
			listenAddr, err := cmd.Flags().GetString(flagListenAddr)
			if err != nil {
				return fmt.Errorf("failed to read configuration: %v", err)
			}

			reDKGFile := args[0]

			reDKGDData, err := ioutil.ReadFile(reDKGFile)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %w", reDKGFile, err)
			}

			if _, err := rawPostRequest(fmt.Sprintf("http://%s/reinitDKG", listenAddr),
				"application/json", reDKGDData); err != nil {
				return fmt.Errorf("failed to reinit DKG: %w", err)
			}
			return nil
		},
	}
}

func rawGetRequest(url string) (*httpresponses.BaseResponse, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get operations for node %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body %w", err)
	}

	var response httpresponses.BaseResponse
	if err = json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}
	return &response, nil
}

func getPubKeyCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get_pubkey",
		Short: "returns node's pubkey",
		RunE: func(cmd *cobra.Command, args []string) error {
			listenAddr, err := cmd.Flags().GetString(flagListenAddr)
			if err != nil {
				return fmt.Errorf("failed to read configuration: %v", err)
			}

			resp, err := rawGetRequest(fmt.Sprintf("http://%s/getPubKey", listenAddr))
			if err != nil {
				return fmt.Errorf("failed to get node's pubkey: %w", err)
			}
			if resp.ErrorMessage != "" {
				return fmt.Errorf("failed to get node's pubkey: %v", resp.ErrorMessage)
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

			resp, err := rawGetRequest(fmt.Sprintf("http://%s/getOffset", listenAddr))
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

func getUsername(listenAddr string) (string, error) {
	resp, err := rawGetRequest(fmt.Sprintf("http://%s/getUsername", listenAddr))
	if err != nil {
		return "", fmt.Errorf("failed to do HTTP request: %w", err)
	}
	if resp.ErrorMessage != "" {
		return "", fmt.Errorf("failed to get node's username: %v", resp.ErrorMessage)
	}
	return resp.Result.(string), nil
}

func getUsernameCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get_username",
		Short: "returns node's username",
		RunE: func(cmd *cobra.Command, args []string) error {
			listenAddr, err := cmd.Flags().GetString(flagListenAddr)
			if err != nil {
				return fmt.Errorf("failed to read configuration: %v", err)
			}

			username, err := getUsername(listenAddr)
			if err != nil {
				return fmt.Errorf("failed to get node's username: %w", err)
			}
			fmt.Println(username)
			return nil
		},
	}
}

func rawPostRequest(url string, contentType string, data []byte) (*httpresponses.BaseResponse, error) {
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

	var response httpresponses.BaseResponse
	if err = json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}
	return &response, nil
}

func readOperationResultCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "read_operation_result",
		Short: "given the path to Operation JSON file, decodes and processes it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			listenAddr, err := cmd.Flags().GetString(flagListenAddr)
			if err != nil {
				return fmt.Errorf("failed to read configuration: %v", err)
			}

			operationBz, err := ioutil.ReadFile(strings.Trim(args[0], " \n"))
			if err != nil {
				return fmt.Errorf("failed to read Operation file: %w", err)
			}

			resp, err := rawPostRequest(fmt.Sprintf("http://%s/handleProcessedOperationJSON", listenAddr),
				"application/json", operationBz)
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

func approveDKGParticipationCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "approve_participation [operationID]",
		Args:  cobra.ExactArgs(1),
		Short: "approve participation in a DKG process",
		RunE: func(cmd *cobra.Command, args []string) error {
			listenAddr, err := cmd.Flags().GetString(flagListenAddr)
			if err != nil {
				return fmt.Errorf("failed to read configuration: %v", err)
			}

			operationID := args[0]

			payloadBz, err := json.Marshal(map[string]string{"operationID": operationID})
			if err != nil {
				return fmt.Errorf("failed to marshal payload: %v", err)
			}
			resp, err := rawPostRequest(fmt.Sprintf("http://%s/approveDKGParticipation", listenAddr), "application/json", payloadBz)
			if err != nil {
				return fmt.Errorf("failed to approve participation: %w", err)
			}
			if resp.ErrorMessage != "" {
				return fmt.Errorf("failed to approve participation: %v", resp.ErrorMessage)
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
				if _, err := hashPayload.Write([]byte(p.Username)); err != nil {
					return err
				}
			}
			hash := md5.Sum(hashPayload.Bytes())
			fmt.Println(hex.EncodeToString(hash[:]))
			return nil
		},
	}
}

func getHashOfReinitDKGMessageCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get_reinit_dkg_file_hash [reinit_file]",
		Args:  cobra.ExactArgs(1),
		Short: "returns hash of reinit message for DKG reinit to verify correctness",
		RunE: func(cmd *cobra.Command, args []string) error {

			dkgReinitFileData, err := ioutil.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}
			hash, err := types.CalcStartReInitDKGMessageHash(dkgReinitFileData)
			if err != nil {
				return err
			}
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

func proposeSignBatchMessagesCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "sign_batch_data [dkg_id] [dir_path]",
		Args:  cobra.ExactArgs(2),
		Short: "sends a propose batch messages to sign the data in the dir",
		RunE: func(cmd *cobra.Command, args []string) error {
			listenAddr, err := cmd.Flags().GetString(flagListenAddr)
			if err != nil {
				return fmt.Errorf("failed to read configuration: %v", err)
			}

			dkgID, err := hex.DecodeString(args[0])
			if err != nil {
				return fmt.Errorf("failed to decode dkgID: %w", err)
			}

			req := httprequests.ProposeSignBatchMessagesForm{
				DkgID: dkgID,
				Data:  make(map[string][]byte),
			}

			files, err := ioutil.ReadDir(args[1])
			if err != nil {
				return fmt.Errorf("failde to read dir {%s}: %w", args[1], err)
			}

			for _, f := range files {
				if f.IsDir() {
					//skipping dirs
					continue
				}
				data, err := ioutil.ReadFile(path.Join(args[1], f.Name()))
				if err != nil {
					return fmt.Errorf("failed to read the file")
				}
				req.Data[uuid.New().String()] = data
			}

			messageDataBz, err := json.Marshal(&req)
			if err != nil {
				return fmt.Errorf("failed to marshal SigningBatchProposalStartRequest: %w", err)
			}

			resp, err := rawPostRequest(fmt.Sprintf("http://%s/proposeSignBatchMessages", listenAddr),
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
			if strings.HasPrefix(string(dump.State), "state_sig_") {
				for k, v := range dump.Payload.SignatureProposalPayload.Quorum {
					quorum[k] = v
				}
			}

			waiting := make([]string, 0)
			confirmed := make([]string, 0)
			failed := make([]string, 0)

			username, err := getUsername(listenAddr)
			if err != nil {
				return fmt.Errorf("failed to get node's username: %w", err)
			}

			for _, p := range quorum {
				if strings.Contains(p.GetStatus().String(), "Await") {
					// deals are private messages, so we don't need to wait messages from ourself
					if p.GetStatus().String() == "DealAwaitConfirmation" && p.GetUsername() == username {
						continue
					}
					waiting = append(waiting, p.GetUsername())
				}
				if strings.Contains(p.GetStatus().String(), "Error") {
					failed = append(failed, p.GetUsername())
				}
				if strings.Contains(p.GetStatus().String(), "Confirmed") {
					confirmed = append(confirmed, p.GetUsername())
				}
			}

			if len(waiting) > 0 {
				fmt.Printf("Waiting for data from: %s\n", strings.Join(waiting, ", "))
			}
			if len(confirmed) > 0 {
				fmt.Printf("Received data from: %s\n", strings.Join(confirmed, ", "))
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
		Short: "returns a list of all FSMs served by the node",
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

func refreshState() *cobra.Command {
	runFunc := func(cmd *cobra.Command, args []string) error {
		listenAddr, err := cmd.Flags().GetString(flagListenAddr)
		if err != nil {
			return fmt.Errorf("failed to read listen address: %v", err)
		}

		if len(kafkaConsumerGroup) < 1 {
			username, err := getUsername(listenAddr)
			if err != nil {
				return fmt.Errorf("failed to get node's username: %w", err)
			}

			kafkaConsumerGroup = fmt.Sprintf("%s_%d", username, time.Now().Unix())
		}

		req := httprequests.ResetStateForm{
			NewStateDBDSN:      newStateDBDSN,
			UseOffset:          useOffset,
			KafkaConsumerGroup: kafkaConsumerGroup,
		}
		reqBytes, err := json.Marshal(req)
		if err != nil {
			return fmt.Errorf("failed to marshal reset state request: %w", err)
		}

		resp, err := rawPostRequest(fmt.Sprintf("http://%s/resetState", listenAddr),
			"application/json", reqBytes)
		if err != nil {
			return fmt.Errorf("failed to make HTTP request to reset state: %w", err)
		}
		if resp.ErrorMessage != "" {
			return fmt.Errorf("failed to make HTTP request to reset state: %v", resp.ErrorMessage)
		}

		dir := resp.Result.(string)
		fmt.Printf("New state was saved to %s directory", dir)

		return nil
	}

	refreshStateCmd.RunE = runFunc

	return refreshStateCmd
}
