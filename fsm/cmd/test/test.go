package main

import (
	"github.com/depools/dc4bc/fsm/fsm"
	"github.com/depools/dc4bc/fsm/types/responses"
	"log"

	"github.com/depools/dc4bc/fsm/state_machines"
	"github.com/depools/dc4bc/fsm/types/requests"
)

func main() {
	fsmMachine, err := state_machines.New([]byte{})
	log.Println(fsmMachine, err)
	resp, dump, err := fsmMachine.Do(
		"event_proposal_init",
		"d8a928b2043db77e340b523547bf16cb4aa483f0645fe0a290ed1f20aab76257",
		requests.SignatureProposalParticipantsListRequest{
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
	log.Println("Err", err)
	log.Println("Dump", string(dump))

	processResponse(resp)

}

func processResponse(resp *fsm.Response) {
	switch resp.State {
	// Await proposals
	case fsm.State("validate_proposal"):
		data, ok := resp.Data.(responses.SignatureProposalParticipantInvitationsResponse)
		if !ok {
			log.Printf("undefined response type for state \"%s\"\n", resp.State)
			return
		}
		sendInvitations(data)

	case fsm.State("validation_canceled_by_participant"):
		updateDashboardWithCanceled("Participant")
	case fsm.State("validation_canceled_by_timeout"):
		updateDashboardWithCanceled("Timeout")
	default:
		log.Printf("undefined response type for state \"%s\"\n", resp.State)
	}
}

func sendInvitations(invitations responses.SignatureProposalParticipantInvitationsResponse) {
	for _, invitation := range invitations {
		log.Printf(
			"Dear %s, please encrypt value \"%s\" with your key, fingerprint: %s\n",
			invitation.Title,
			invitation.EncryptedInvitation,
			invitation.PubKeyFingerprint,
		)
	}
}

func updateDashboardWithCanceled(msg string) {
	log.Printf("Breaking news! Proposal canceled with reason: %s\n", msg)
}
