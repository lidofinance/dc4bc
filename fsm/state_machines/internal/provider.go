package internal

import "github.com/depools/dc4bc/fsm/fsm_pool"

type DumpedMachineStatePayload struct {
	TransactionId               string
	ConfirmationProposalPayload SignatureProposalQuorum
	DKGProposalPayload          DKGProposalQuorum
}

type DumpedMachineProvider interface {
	fsm_pool.MachineProvider
	SetUpPayload(payload *DumpedMachineStatePayload)
}
