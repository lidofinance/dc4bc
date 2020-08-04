package signature_proposal_fsm

import (
	"github.com/depools/dc4bc/fsm/fsm"
	"github.com/depools/dc4bc/fsm/fsm_pool"
)

const (
	fsmName      = "signature_proposal_fsm"
	signingIdLen = 32

	stateAwaitProposalConfirmation = fsm.State("validate_proposal") // waiting participants

	stateValidationCanceledByParticipant = fsm.State("validation_canceled_by_participant")
	stateValidationCanceledByTimeout     = fsm.State("validation_canceled_by_timeout")

	stateProposed = "proposed"

	eventInitProposal         = fsm.Event("proposal_init")
	eventConfirmProposal      = fsm.Event("proposal_confirm_by_participant")
	eventDeclineProposal      = fsm.Event("proposal_decline_by_participant")
	eventValidateProposal     = fsm.Event("proposal_validate")
	eventSetProposalValidated = fsm.Event("proposal_set_validated")

	eventSetValidationCanceledByTimeout = fsm.Event("proposal_canceled_timeout")
	eventSwitchProposedToSigning        = fsm.Event("switch_state_to_signing")
)

type SignatureProposalFSM struct {
	*fsm.FSM
}

func New() fsm_pool.MachineProvider {
	machine := &SignatureProposalFSM{}

	machine.FSM = fsm.MustNewFSM(
		fsmName,
		fsm.StateGlobalIdle,
		[]fsm.EventDesc{
			// {Name: "", SrcState: []string{""}, DstState: ""},

			// Init
			{Name: eventInitProposal, SrcState: []fsm.State{fsm.StateGlobalIdle}, DstState: stateAwaitProposalConfirmation},

			// Validate by participants
			{Name: eventConfirmProposal, SrcState: []fsm.State{stateAwaitProposalConfirmation}, DstState: stateAwaitProposalConfirmation},
			// Is decline event should auto change state to default, or it process will initiated by client (external emit)?
			// Now set for external emitting.
			{Name: eventDeclineProposal, SrcState: []fsm.State{stateAwaitProposalConfirmation}, DstState: stateValidationCanceledByParticipant},

			{Name: eventValidateProposal, SrcState: []fsm.State{stateAwaitProposalConfirmation}, DstState: stateAwaitProposalConfirmation},

			// eventProposalValidate internal or from client?
			// yay
			// Exit point
			{Name: eventSetProposalValidated, SrcState: []fsm.State{stateAwaitProposalConfirmation}, DstState: "process_sig", IsInternal: true},
			// nan
			{Name: eventSetValidationCanceledByTimeout, SrcState: []fsm.State{stateAwaitProposalConfirmation}, DstState: stateValidationCanceledByTimeout, IsInternal: true},
		},
		fsm.Callbacks{
			eventInitProposal:     machine.actionInitProposal,
			eventConfirmProposal:  machine.actionConfirmProposalByParticipant,
			eventDeclineProposal:  machine.actionDeclineProposalByParticipant,
			eventValidateProposal: machine.actionValidateProposal,
		},
	)
	return machine
}
