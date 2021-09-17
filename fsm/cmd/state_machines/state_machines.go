package main

import (
	"fmt"
	"log"

	"github.com/lidofinance/dc4bc/fsm/fsm"
	"github.com/lidofinance/dc4bc/fsm/state_machines/dkg_proposal_fsm"
	"github.com/lidofinance/dc4bc/fsm/state_machines/signature_proposal_fsm"
	"github.com/lidofinance/dc4bc/fsm/state_machines/signing_proposal_fsm"
)

func main() {
	dkgFSM, ok := dkg_proposal_fsm.New().(*dkg_proposal_fsm.DKGProposalFSM)
	if !ok {
		log.Fatal("invalid type")
	}
	fmt.Println(fsm.Visualize(dkgFSM.FSM))

	sigFSM, ok := signature_proposal_fsm.New().(*signature_proposal_fsm.SignatureProposalFSM)
	if !ok {
		log.Fatal("invalid type")
	}
	fmt.Println(fsm.Visualize(sigFSM.FSM))

	signingFSM, ok := signing_proposal_fsm.New().(*signing_proposal_fsm.SigningProposalFSM)
	if !ok {
		log.Fatal("invalid type")
	}
	fmt.Println(fsm.Visualize(signingFSM.FSM))
}
