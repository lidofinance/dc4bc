package main

import (
	"log"

	"github.com/depools/dc4bc/fsm/state_machines"
	"github.com/depools/dc4bc/fsm/types/requests"
)

func main() {
	fsmMachine, err := state_machines.New([]byte{})
	log.Println(fsmMachine, err)
	resp, dump, err := fsmMachine.Do(
		"proposal_init",
		"d8a928b2043db77e340b523547bf16cb4aa483f0645fe0a290ed1f20aab76257",
		requests.ProposalParticipantsListRequest{
			{
				"John Doe",
				[]byte("pubkey123123"),
			},
			{
				"Crypto Billy",
				[]byte("pubkey456456"),
			},
			{
				"Matt",
				[]byte("pubkey789789"),
			},
		},
	)
	log.Println("Response", resp)
	log.Println("Err", err)
	log.Println("Dump", string(dump))

}
