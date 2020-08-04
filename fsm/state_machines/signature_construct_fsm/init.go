package signature_construct_fsm

import (
	"github.com/depools/dc4bc/fsm/fsm"
	"github.com/depools/dc4bc/fsm/fsm_pool"
)

const (
	fsmName = "signature_construct_fsm"

	stateConstructorEntryPoint = "process_sig"
	awaitConstructor           = "validate_process_sig" // waiting participants

	eventInitSignatureConstructor = "process_sig_init"
	eventInitSignatureFinishTmp   = "process_sig_fin"
)

type SignatureConstructFSM struct {
	*fsm.FSM
}

func New() fsm_pool.IStateMachine {
	machine := &SignatureConstructFSM{}

	machine.FSM = fsm.MustNewFSM(
		fsmName,
		stateConstructorEntryPoint,
		[]fsm.Event{
			// {Name: "", SrcState: []string{""}, DstState: ""},

			// Init
			{Name: eventInitSignatureConstructor, SrcState: []string{stateConstructorEntryPoint}, DstState: awaitConstructor},
			{Name: eventInitSignatureFinishTmp, SrcState: []string{awaitConstructor}, DstState: "dkg_proposal_fsm"},
		},
		fsm.Callbacks{},
	)

	return machine
}
