package main

import (
	"context"
	"fmt"
	_ "image/jpeg"
	"log"
	"sync"

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

func main() {
	var transport = &Transport{}
	var numNodes = 4
	for nodeID := 0; nodeID < numNodes; nodeID++ {
		var ctx = context.Background()
		var userName = fmt.Sprintf("node_%d", nodeID)
		var state, err = client.NewLevelDBState(fmt.Sprintf("/tmp/dc4bc_node_%d_state", nodeID))
		if err != nil {
			log.Fatalf("node %d failed to init state: %v\n", nodeID, err)
		}

		stg, err := storage.NewFileStorage("/tmp/dc4bc_storage")
		if err != nil {
			log.Fatalf("node %d failed to init storage: %v\n", nodeID, err)
		}

		keyStore, err := client.NewLevelDBKeyStore(userName, fmt.Sprintf("/tmp/dc4bc_node_%d_key_store", nodeID))
		if err != nil {
			log.Fatalf("Failed to init key store: %v", err)
		}

		clt, err := client.NewClient(
			ctx,
			userName,
			nil,
			state,
			stg,
			keyStore,
			qr.NewCameraProcessor(),
		)
		if err != nil {
			log.Fatalf("node %d failed to init client: %v\n", nodeID, err)
		}
		transport.nodes = append(transport.nodes, clt)
	}

	for nodeID, node := range transport.nodes {
		go func(nodeID int, node *client.Client) {
			if err := node.StartHTTPServer(fmt.Sprintf("localhost:808%d", nodeID)); err != nil {
				log.Fatalf("client %d http server failed: %v\n", nodeID, err)
			}
		}(nodeID, node)

		go func(nodeID int, node *client.Client) {
			if err := node.Poll(); err != nil {
				log.Fatalf("client %d poller failed: %v\n", nodeID, err)
			}
		}(nodeID, node)

		log.Printf("client %d started...\n", nodeID)
	}

	var wg = sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}

//	// Participants broadcast PKs.
//	runStep(transport, func(participantID string, participant *dkglib.DKG, wg *sync.WaitGroup) {
//		transport.BroadcastPK(participantID, participant.GetPubKey())
//		wg.Done()
//	})
//
//	// Participants init their DKGInstances.
//	runStep(transport, func(participantID string, participant *dkglib.DKG, wg *sync.WaitGroup) {
//		if err := participant.InitDKGInstance(threshold); err != nil {
//			log.Fatalf("Failed to InitDKGInstance: %v", err)
//		}
//		wg.Done()
//	})
//
//	// Participants broadcast their Commits.
//	runStep(transport, func(participantID string, participant *dkglib.DKG, wg *sync.WaitGroup) {
//		commits := participant.GetCommits()
//		transport.BroadcastCommits(participantID, commits)
//		wg.Done()
//	})
//
//	// Participants broadcast their deal.
//	runStep(transport, func(participantID string, participant *dkglib.DKG, wg *sync.WaitGroup) {
//		deals, err := participant.GetDeals()
//		if err != nil {
//			log.Fatalf("failed to getDeals for participant %s: %v", participantID, err)
//		}
//		transport.BroadcastDeals(participantID, deals)
//		wg.Done()
//	})
//
//	// Participants broadcast their responses.
//	runStep(transport, func(participantID string, participant *dkglib.DKG, wg *sync.WaitGroup) {
//		responses, err := participant.ProcessDeals()
//		if err != nil {
//			log.Fatalf("failed to ProcessDeals for participant %s: %v", participantID, err)
//		}
//		transport.BroadcastResponses(participantID, responses)
//		wg.Done()
//	})
//
//	// Participants process their responses.
//	runStep(transport, func(participantID string, participant *dkglib.DKG, wg *sync.WaitGroup) {
//		if err := participant.ProcessResponses(); err != nil {
//			log.Fatalf("failed to ProcessResponses for participant %s: %v", participantID, err)
//		}
//		wg.Done()
//	})
//
//	for idx, node := range transport.nodes {
//		if err := node.Reconstruct(); err != nil {
//			fmt.Println("Node", idx, "is not ready:", err)
//		} else {
//			fmt.Println("Node", idx, "is ready")
//		}
//	}
//}
//
//func runStep(transport *Transport, cb func(participantID string, participant *dkglib.DKG, wg *sync.WaitGroup)) {
//	var wg = &sync.WaitGroup{}
//	for idx, node := range transport.nodes {
//		wg.Add(1)
//		n := node
//		go cb(fmt.Sprintf("participant_%d", idx), n, wg)
//	}
//	wg.Wait()
//}
