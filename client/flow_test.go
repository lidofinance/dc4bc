package client

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/depools/dc4bc/airgapped"
	"github.com/depools/dc4bc/client/types"
	"github.com/depools/dc4bc/fsm/state_machines/dkg_proposal_fsm"
	"github.com/depools/dc4bc/fsm/types/requests"
	"github.com/depools/dc4bc/qr"
	"github.com/depools/dc4bc/storage"
	bls12381 "github.com/depools/kyber-bls12381"
)

type node struct {
	client     Poller
	keyPair    *KeyPair
	air        *airgapped.AirgappedMachine
	listenAddr string
}

type OperationsResponse struct {
	Result map[string]*types.Operation `json:"result"`
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

func (n *node) run(t *testing.T) {
	for {
		operationsResponse, err := getOperations(fmt.Sprintf("http://%s/getOperations", n.listenAddr))
		if err != nil {
			t.Fatalf("failed to get operations: %v", err)
		}

		operations := operationsResponse.Result
		if len(operations) == 0 {
			time.Sleep(1 * time.Second)
			continue
		}

		n.client.GetLogger().Log("Got %d Operations from pool", len(operations))
		for _, operation := range operations {
			n.client.GetLogger().Log("Handling operation %s in airgapped", operation.Type)
			processedOperation, err := n.air.HandleOperation(*operation)
			if err != nil {
				n.client.GetLogger().Log("Failed to handle operation: %v", err)
			}

			n.client.GetLogger().Log("Got %d Processed Operations from Airgapped", len(operations))
			n.client.GetLogger().Log("Operation %s handled in airgapped, result event is %s",
				operation.Event, processedOperation.Event)

			// for integration tests
			if processedOperation.Event == dkg_proposal_fsm.EventDKGMasterKeyConfirmationReceived {
				msg := processedOperation.ResultMsgs[0]
				var pubKeyReq requests.DKGProposalMasterKeyConfirmationRequest
				if err = json.Unmarshal(msg.Data, &pubKeyReq); err != nil {
					t.Fatalf("failed to unmarshal pubKey request: %v", err)
				}
				pubKey := bls12381.NewBLS12381Suite().Point()
				if err = pubKey.UnmarshalBinary(pubKeyReq.MasterKey); err != nil {
					t.Fatalf("failed to unmarshal pubkey: %v", err)
				}
				if err = ioutil.WriteFile(fmt.Sprintf("/tmp/dc4bc_participant_%d.pubkey", pubKeyReq.ParticipantId), []byte(pubKey.String()), 666); err != nil {
					t.Fatalf("failed to write pubkey to temp file: %v", err)
				}
			}

			if err = handleProcessedOperation(fmt.Sprintf("http://%s/handleProcessedOperationJSON", n.listenAddr),
				processedOperation); err != nil {
				n.client.GetLogger().Log("Failed to handle processed operation: %v", err)
			} else {
				n.client.GetLogger().Log("Successfully handled processed operation %s", processedOperation.Event)
			}

		}
	}
}

func TestFullFlow(t *testing.T) {
	files, _ := filepath.Glob("/tmp/dc4bc_*")
	for _, f := range files {
		_ = os.Remove(f)
	}

	var numNodes = 4
	var threshold = 3
	var storagePath = "/tmp/dc4bc_storage"
	var nodes = make([]*node, numNodes)
	startingPort := 8080
	for nodeID := 0; nodeID < numNodes; nodeID++ {
		var ctx = context.Background()
		var userName = fmt.Sprintf("node_%d", nodeID)
		var state, err = NewLevelDBState(fmt.Sprintf("/tmp/dc4bc_node_%d_state", nodeID))
		if err != nil {
			t.Fatalf("node %d failed to init state: %v\n", nodeID, err)
		}

		stg, err := storage.NewFileStorage(storagePath)
		if err != nil {
			t.Fatalf("node %d failed to init storage: %v\n", nodeID, err)
		}

		keyStore, err := NewLevelDBKeyStore(userName, fmt.Sprintf("/tmp/dc4bc_node_%d_key_store", nodeID))
		if err != nil {
			t.Fatalf("Failed to init key store: %v", err)
		}

		keyPair := NewKeyPair()
		if err := keyStore.PutKeys(userName, keyPair); err != nil {
			t.Fatalf("Failed to PutKeys: %v\n", err)
		}

		airgappedMachine, err := airgapped.NewAirgappedMachine(fmt.Sprintf("/tmp/dc4bc_node_%d_airgapped_db", nodeID))
		if err != nil {
			t.Fatalf("Failed to create airgapped machine: %v", err)
		}

		clt, err := NewClient(
			ctx,
			userName,
			state,
			stg,
			keyStore,
			qr.NewCameraProcessor(),
		)
		if err != nil {
			t.Fatalf("node %d failed to init client: %v\n", nodeID, err)
		}
		airgappedMachine.SetAddress(clt.GetAddr())

		nodes[nodeID] = &node{
			client:     clt,
			keyPair:    keyPair,
			air:        airgappedMachine,
			listenAddr: fmt.Sprintf("localhost:%d", startingPort),
		}
		startingPort++
	}

	// Each node starts to Poll().
	for nodeID, n := range nodes {
		go func(nodeID int, node *node) {
			if err := node.client.StartHTTPServer(node.listenAddr); err != nil {
				t.Fatalf("failed to start HTTP server for nodeID #%d: %v\n", nodeID, err)
			}
		}(nodeID, n)
		time.Sleep(1 * time.Second)
		go nodes[nodeID].run(t)

		go func(nodeID int, node Poller) {
			if err := node.Poll(); err != nil {
				t.Fatalf("client %d poller failed: %v\n", nodeID, err)
			}
		}(nodeID, n.client)

		log.Printf("client %d started...\n", nodeID)
	}

	// Node1 tells other participants to start DKG.
	var participants []*requests.SignatureProposalParticipantsEntry
	for _, node := range nodes {
		dkgPubKey, err := node.air.GetPubKey().MarshalBinary()
		if err != nil {
			log.Fatalln("failed to get DKG pubKey:", err.Error())
		}
		participants = append(participants, &requests.SignatureProposalParticipantsEntry{
			Addr:      node.client.GetAddr(),
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
		t.Fatalf("failed to marshal SignatureProposalParticipantsListRequest: %v\n", err)
	}

	if _, err := http.Post(fmt.Sprintf("http://localhost:%d/startDKG", startingPort-1),
		"application/json", bytes.NewReader(messageDataBz)); err != nil {
		t.Fatalf("failed to send HTTP request to start DKG: %v\n", err)
	}

	time.Sleep(10 * time.Second)
	log.Println("Propose message to sign")

	dkgRoundID := md5.Sum(messageDataBz)
	messageDataSign := requests.SigningProposalStartRequest{
		ParticipantId: len(nodes) - 1,
		SrcPayload:    []byte("message to sign"),
		CreatedAt:     time.Now(),
	}
	messageDataSignBz, err := json.Marshal(messageDataSign)
	if err != nil {
		t.Fatalf("failed to marshal SigningProposalStartRequest: %v\n", err)
	}

	messageDataBz, err = json.Marshal(map[string][]byte{"data": messageDataSignBz,
		"dkgID": dkgRoundID[:]})
	if err != nil {
		t.Fatalf("failed to marshal SignatureProposalParticipantsListRequest: %v\n", err)
	}

	if _, err := http.Post(fmt.Sprintf("http://localhost:%d/proposeSignMessage", startingPort-1),
		"application/json", bytes.NewReader(messageDataBz)); err != nil {
		t.Fatalf("failed to send HTTP request to sign message: %v\n", err)
	}
	time.Sleep(5 * time.Second)

}
