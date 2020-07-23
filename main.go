package main

import (
	"fmt"
	_ "image/jpeg"
	"log"
	"sync"

	"go.dedis.ch/kyber/v3"

	dkglib "p2p.org/dc4bc/dkg"

	_ "image/gif"
	_ "image/png"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

type Transport struct {
	nodes []*dkglib.DKG
}

func (t *Transport) BroadcastPK(participant string, pk kyber.Point) {
	for idx, node := range t.nodes {
		if ok := node.StorePubKey(participant, pk); !ok {
			log.Fatalf("Failed to store PK for participant %d", idx)
		}
	}
}

func (t *Transport) BroadcastCommits(participant string, commits []kyber.Point) {
	for _, node := range t.nodes {
		node.StoreCommits(participant, commits)
	}
}

func main() {
	var threshold = 3
	var transport = &Transport{}
	var numNodes = 4
	for i := 0; i < numNodes; i++ {
		transport.nodes = append(transport.nodes, dkglib.Init())
	}

	// Participants broadcast PKs.
	runStep(transport, func(participantID string, participant *dkglib.DKG, wg *sync.WaitGroup) {
		transport.BroadcastPK(participantID, participant.GetPubKey())
		wg.Done()
	})

	// Participants init their DKGInstances.
	runStep(transport, func(participantID string, participant *dkglib.DKG, wg *sync.WaitGroup) {
		if err := participant.InitDKGInstance(threshold); err != nil {
			log.Fatalf("Failed to InitDKGInstance: %v", err)
		}
		wg.Done()
	})

	// Participants broadcast their Commits.
	runStep(transport, func(participantID string, participant *dkglib.DKG, wg *sync.WaitGroup) {
		commits := participant.GetCommits()
		transport.BroadcastCommits(participantID, commits)
		wg.Done()
	})

	for idx, node := range transport.nodes {
		if err := node.Reconstruct(); err != nil {
			fmt.Println("Node", idx, "is not ready")
		} else {
			fmt.Println("Node", idx, "is ready")
		}
	}
}

func runStep(transport *Transport, cb func(participantID string, participant *dkglib.DKG, wg *sync.WaitGroup)) {
	var wg = &sync.WaitGroup{}
	for idx, node := range transport.nodes {
		wg.Add(1)
		go cb(fmt.Sprintf("participant_%d", idx), node, wg)
	}
	wg.Wait()
}

//func runQRTest() {
//	clearTerminal()
//	var data = "Hello, world!"
//
//	log.Println("A QR code will be shown on your screen.")
//	log.Println("Please take a photo of the QR code with your smartphone.")
//	log.Println("When you close the image, you will have 5 seconds to" +
//		"scan the QR code with your laptop's camera.")
//	err := qr.ShowQR(data)
//	if err != nil {
//		log.Fatalf("Failed to show QR code: %v", err)
//	}
//
//	var scannedData string
//	for {
//		clearTerminal()
//		if err != nil {
//			log.Printf("Failed to scan QR code: %v\n", err)
//		}
//
//		log.Println("Please center the photo of the QR-code in front" +
//			"of your web-camera...")
//
//		scannedData, err = qr.ReadQRFromCamera()
//		if err == nil {
//			break
//		}
//	}
//
//	clearTerminal()
//	log.Printf("QR code successfully scanned; the data is: %s\n", scannedData)
//}
//
//func clearTerminal() {
//	c := exec.Command("clear")
//	c.Stdout = os.Stdout
//	_ = c.Run()
//}
