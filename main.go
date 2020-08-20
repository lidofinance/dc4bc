package main

import (
	"context"
	"crypto/ed25519"
	"crypto/md5"
	"encoding/json"
	"fmt"
	_ "image/jpeg"
	"log"
	"sync"
	"time"

	spf "github.com/depools/dc4bc/fsm/state_machines/signature_proposal_fsm"
	"github.com/depools/dc4bc/fsm/types/requests"
	"github.com/google/uuid"

	"github.com/depools/dc4bc/qr"
	"github.com/depools/dc4bc/storage"

	"github.com/depools/dc4bc/client"

	_ "image/gif"
	_ "image/png"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

type Transport struct {
	nodes []*client.Client
}

type node struct {
	client  *client.Client
	keyPair *client.KeyPair
}

func main() {
	var numNodes = 4
	var threshold = 3
	var storagePath = "/tmp/dc4bc_storage"
	var nodes = make([]*node, 4)
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

		nodes[nodeID] = &node{
			client:  clt,
			keyPair: keyPair,
		}
	}

	// Each node starts to Poll().
	for nodeID, node := range nodes {
		go func(nodeID int, node *client.Client) {
			if err := node.Poll(); err != nil {
				log.Fatalf("client %d poller failed: %v\n", nodeID, err)
			}
		}(nodeID, node.client)

		log.Printf("client %d started...\n", nodeID)
	}

	stg, err := storage.NewFileStorage(storagePath)
	if err != nil {
		log.Fatalf("main namespace failed to init storage: %v\n", err)
	}

	// Node1 tells other participants to start DKG.
	var participants []*requests.SignatureProposalParticipantsEntry
	for _, node := range nodes {
		participants = append(participants, &requests.SignatureProposalParticipantsEntry{
			Addr:      node.client.GetAddr(),
			PubKey:    node.client.GetPubKey(),
			DkgPubKey: make([]byte, 128), // TODO: Use a real one.
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

	dkgRoundID := md5.Sum(messageDataBz)
	message := storage.Message{
		ID:         uuid.New().String(),
		DkgRoundID: string(dkgRoundID[:]),
		Event:      string(spf.EventInitProposal),
		Data:       messageDataBz,
		SenderAddr: nodes[0].client.GetAddr(),
	}

	message.Signature = ed25519.Sign(nodes[0].keyPair.Priv, message.Bytes())
	if _, err := stg.Send(message); err != nil {
		log.Fatalf("Failed to send %+v to storage: %v\n", message, err)
	}

	var wg = sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}
