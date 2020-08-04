package signature_proposal_fsm

import (
	"github.com/depools/dc4bc/fsm/fsm"
	"github.com/depools/dc4bc/fsm/fsm_pool"
)

const (
	fsmName      = "signature_proposal_fsm"
	signingIdLen = 32

	stateAwaitProposalConfirmation = "validate_proposal" // waiting participants

	stateValidationCanceledByParticipant = "validation_canceled_by_participant"
	stateValidationCanceledByTimeout     = "validation_canceled_by_timeout"

	stateProposed = "proposed"

	eventInitProposal         = "proposal_init"
	eventConfirmProposal      = "proposal_confirm_by_participant"
	eventDeclineProposal      = "proposal_decline_by_participant"
	eventValidateProposal     = "proposal_validate"
	eventSetProposalValidated = "proposal_set_validated"

	eventSetValidationCanceledByTimeout = "proposal_canceled_timeout"
	eventSwitchProposedToSigning        = "switch_state_to_signing"
)

type SignatureProposalFSM struct {
	*fsm.FSM
}

func New() fsm_pool.IStateMachine {
	machine := &SignatureProposalFSM{}

	machine.FSM = fsm.MustNewFSM(
		fsmName,
		fsm.StateGlobalIdle,
		[]fsm.Event{
			// {Name: "", SrcState: []string{""}, DstState: ""},

			// Init
			{Name: eventInitProposal, SrcState: []string{fsm.StateGlobalIdle}, DstState: stateAwaitProposalConfirmation},

			// Validate by participants
			{Name: eventConfirmProposal, SrcState: []string{stateAwaitProposalConfirmation}, DstState: stateAwaitProposalConfirmation},
			// Is decline event should auto change state to default, or it process will initiated by client (external emit)?
			// Now set for external emitting.
			{Name: eventDeclineProposal, SrcState: []string{stateAwaitProposalConfirmation}, DstState: stateValidationCanceledByParticipant},

			{Name: eventValidateProposal, SrcState: []string{stateAwaitProposalConfirmation}, DstState: stateAwaitProposalConfirmation},

			// eventProposalValidate internal or from client?
			// yay
			// Exit point
			{Name: eventSetProposalValidated, SrcState: []string{stateAwaitProposalConfirmation}, DstState: "process_sig", IsInternal: true},
			// nan
			{Name: eventSetValidationCanceledByTimeout, SrcState: []string{stateAwaitProposalConfirmation}, DstState: stateValidationCanceledByTimeout, IsInternal: true},
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
