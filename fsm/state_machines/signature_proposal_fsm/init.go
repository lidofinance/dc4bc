package signature_proposal_fsm

import (
	"github.com/depools/dc4bc/fsm/fsm"
	"github.com/depools/dc4bc/fsm/fsm_pool"
)

const (
	fsmName      = "signature_proposal_fsm"
	signingIdLen = 32

	StateAwaitParticipantsConfirmations = fsm.State("state_validation_await_participants_confirmations") // waiting participants

	StateValidationCanceledByParticipant = fsm.State("state_validation_canceled_by_participant")
	StateValidationCanceledByTimeout     = fsm.State("state_validation_canceled_by_timeout")

	StateValidationCompleted = fsm.State("state_validation_completed")

	EventInitProposal         = fsm.Event("event_proposal_init")
	EventConfirmProposal      = fsm.Event("event_proposal_confirm_by_participant")
	EventDeclineProposal      = fsm.Event("event_proposal_decline_by_participant")
	EventValidateProposal     = fsm.Event("event_proposal_validate")
	EventSetProposalValidated = fsm.Event("event_proposal_set_validated")

	eventSetValidationCanceledByTimeout = fsm.Event("proposal_canceled_timeout")

	// Switch to next fsm

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
			{Name: EventInitProposal, SrcState: []fsm.State{fsm.StateGlobalIdle}, DstState: StateAwaitParticipantsConfirmations},

			// Validate by participants
			{Name: EventConfirmProposal, SrcState: []fsm.State{StateAwaitParticipantsConfirmations}, DstState: StateAwaitParticipantsConfirmations},
			// Is decline event should auto change state to default, or it process will initiated by client (external emit)?
			// Now set for external emitting.
			{Name: EventDeclineProposal, SrcState: []fsm.State{StateAwaitParticipantsConfirmations}, DstState: StateValidationCanceledByParticipant},

			{Name: EventValidateProposal, SrcState: []fsm.State{StateAwaitParticipantsConfirmations}, DstState: StateAwaitParticipantsConfirmations},

			// eventProposalValidate internal or from client?
			// yay
			// Exit point
			{Name: EventSetProposalValidated, SrcState: []fsm.State{StateAwaitParticipantsConfirmations}, DstState: fsm.State("state_dkg_pub_keys_sending_required"), IsInternal: true},
			// nan
			{Name: eventSetValidationCanceledByTimeout, SrcState: []fsm.State{StateAwaitParticipantsConfirmations}, DstState: StateValidationCanceledByTimeout, IsInternal: true},
		},
		fsm.Callbacks{
			EventInitProposal:     machine.actionInitProposal,
			EventConfirmProposal:  machine.actionConfirmProposalByParticipant,
			EventDeclineProposal:  machine.actionDeclineProposalByParticipant,
			EventValidateProposal: machine.actionValidateProposal,
		},
	)
	return machine
}
