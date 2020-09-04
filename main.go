package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	_ "image/jpeg"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/depools/dc4bc/airgapped"
	"github.com/depools/dc4bc/client"
	"github.com/depools/dc4bc/fsm/types/requests"
	"github.com/depools/dc4bc/qr"
	"github.com/depools/dc4bc/storage"
)

type node struct {
	client  *client.Client
	keyPair *client.KeyPair
	air     *airgapped.AirgappedMachine
}

func (n *node) run() {
	for {
		operations, err := n.client.GetOperations()
		if err != nil {
			n.client.Logger.Log("Failed to get operations: %v", err)
		}

		if len(operations) == 0 {
			time.Sleep(1 * time.Second)
			continue
		}

		n.client.Logger.Log("Got %d Operations from pool", len(operations))
		for _, operation := range operations {
			n.client.Logger.Log("Handling operation %s in airgapped", operation.Type)
			processedOperation, err := n.air.HandleOperation(*operation)
			if err != nil {
				n.client.Logger.Log("Failed to handle operation: %v", err)
			}

			n.client.Logger.Log("Got %d Processed Operations from Airgapped", len(operations))
			n.client.Logger.Log("Operation %s handled in airgapped, result event is %s",
				operation.Event, processedOperation.Event)
			if err = n.client.HandleProcessedOperation(processedOperation); err != nil {
				n.client.Logger.Log("Failed to handle processed operation: %v", err)
			} else {
				n.client.Logger.Log("Successfully handled processed operation %s", processedOperation.Event)
			}
		}
	}
}

func main() {
	var numNodes = 4
	var threshold = 3
	var storagePath = "/tmp/dc4bc_storage"
	var nodes = make([]*node, numNodes)
	startingPort := 8080
	for nodeID := 0; nodeID < numNodes; nodeID++ {
		var ctx = context.Background()
		var userName = fmt.Sprintf("node_%d", nodeID)
		var state, err = client.NewLevelDBState(fmt.Sprintf("/tmp/dc4bc_node_%d_state", nodeID))
		if err != nil {
			log.Fatalf("node %d failed to init state: %v\n", nodeID, err)
		}

		stg, err := storage.NewFileStorage(storagePath)
		if err != nil {
			log.Fatalf("node %d failed to init storage: %v\n", nodeID, err)
		}

		keyStore, err := client.NewLevelDBKeyStore(userName, fmt.Sprintf("/tmp/dc4bc_node_%d_key_store", nodeID))
		if err != nil {
			log.Fatalf("Failed to init key store: %v", err)
		}

		keyPair := client.NewKeyPair()
		if err := keyStore.PutKeys(userName, keyPair); err != nil {
			log.Fatalf("Failed to PutKeys: %v\n", err)
		}

		airgappedMachine, err := airgapped.NewAirgappedMachine(fmt.Sprintf("/tmp/dc4bc_node_%d_airgapped_db", nodeID))
		if err != nil {
			log.Fatalf("Failed to create airgapped machine: %v", err)
		}

		clt, err := client.NewClient(
			ctx,
			userName,
			state,
			stg,
			keyStore,
			qr.NewCameraProcessor(),
		)
		if err != nil {
			log.Fatalf("node %d failed to init client: %v\n", nodeID, err)
		}
		airgappedMachine.SetAddress(clt.GetAddr())

		nodes[nodeID] = &node{
			client:  clt,
			keyPair: keyPair,
			air:     airgappedMachine,
		}
	}

	// Each node starts to Poll().
	for nodeID, n := range nodes {
		go func(nodeID int, node *node, port int) {
			if err := node.client.StartHTTPServer(fmt.Sprintf(":%d", port)); err != nil {
				log.Fatalf("failed to start HTTP server for nodeID #%d: %v\n", nodeID, err)
			}
		}(nodeID, n, startingPort)
		startingPort++
		go nodes[nodeID].run()

		go func(nodeID int, node *client.Client) {
			if err := node.Poll(); err != nil {
				log.Fatalf("client %d poller failed: %v\n", nodeID, err)
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
		log.Fatalf("failed to marshal SignatureProposalParticipantsListRequest: %v\n", err)
	}

	if _, err := http.Post(fmt.Sprintf("http://localhost:%d/startDKG", startingPort-1),
		"application/json", bytes.NewReader(messageDataBz)); err != nil {
		log.Fatalf("failed to send HTTP request to start DKG: %v\n", err)
	}

	// i haven't a better idea to test signing without big changes in the client code
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
		log.Fatalf("failed to marshal SigningProposalStartRequest: %v\n", err)
	}

	messageDataBz, err = json.Marshal(map[string][]byte{"data": messageDataSignBz,
		"dkgID": dkgRoundID[:]})
	if err != nil {
		log.Fatalf("failed to marshal SignatureProposalParticipantsListRequest: %v\n", err)
	}

	if _, err := http.Post(fmt.Sprintf("http://localhost:%d/proposeSignMessage", startingPort-1),
		"application/json", bytes.NewReader(messageDataBz)); err != nil {
		log.Fatalf("failed to send HTTP request to sign message: %v\n", err)
	}

	var wg = sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}
