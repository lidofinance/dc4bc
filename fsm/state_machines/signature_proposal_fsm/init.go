package signature_proposal_fsm

import (
	"github.com/depools/dc4bc/fsm/fsm"
	"github.com/depools/dc4bc/fsm/state_machines/dkg_proposal_fsm"
	"github.com/depools/dc4bc/fsm/state_machines/internal"
	"sync"
)

const (
	FsmName      = "signature_proposal_fsm"
	signingIdLen = 32

	StateParticipantsConfirmationsInit = fsm.StateGlobalIdle

	StateAwaitParticipantsConfirmations = fsm.State("state_sig_proposal_await_participants_confirmations") // waiting participants

	StateValidationCanceledByParticipant = fsm.State("state_sig_proposal_canceled_by_participant")
	StateValidationCanceledByTimeout     = fsm.State("state_sig_proposal_canceled_by_timeout")

	// Out state

	EventInitProposal                 = fsm.Event("event_sig_proposal_init")
	EventConfirmProposal              = fsm.Event("event_sig_proposal_confirm_by_participant")
	EventDeclineProposal              = fsm.Event("event_sig_proposal_decline_by_participant")
	eventAutoValidateProposalInternal = fsm.Event("event_sig_proposal_validate")
	eventSetProposalValidatedInternal = fsm.Event("event_sig_proposal_set_validated")

	eventSetValidationCanceledByTimeout     = fsm.Event("event_sig_proposal_canceled_timeout")
	eventSetValidationCanceledByParticipant = fsm.Event("event_sig_proposal_declined_timeout")

	// Switch to next fsm

)

type SignatureProposalFSM struct {
	*fsm.FSM
	payload   *internal.DumpedMachineStatePayload
	payloadMu sync.RWMutex
}

func New() internal.DumpedMachineProvider {
	machine := &SignatureProposalFSM{}

	machine.FSM = fsm.MustNewFSM(
		FsmName,
		fsm.StateGlobalIdle,
		[]fsm.EventDesc{
			// {Name: "", SrcState: []string{""}, DstState: ""},

			// Init
			{Name: EventInitProposal, SrcState: []fsm.State{StateParticipantsConfirmationsInit}, DstState: StateAwaitParticipantsConfirmations},

			// Validate by participants
			{Name: EventConfirmProposal, SrcState: []fsm.State{StateAwaitParticipantsConfirmations}, DstState: StateAwaitParticipantsConfirmations},
			// Is decline event should auto change state to default, or it process will initiated by client (external emit)?
			// Now set for external emitting.
			{Name: EventDeclineProposal, SrcState: []fsm.State{StateAwaitParticipantsConfirmations}, DstState: StateAwaitParticipantsConfirmations},
			{Name: eventSetValidationCanceledByParticipant, SrcState: []fsm.State{StateAwaitParticipantsConfirmations}, DstState: StateValidationCanceledByParticipant, IsInternal: true},

			{Name: eventAutoValidateProposalInternal, SrcState: []fsm.State{StateAwaitParticipantsConfirmations}, DstState: StateAwaitParticipantsConfirmations, IsInternal: true, IsAuto: true},

			// eventProposalValidate internal or from client?
			// yay
			// Exit point
			{Name: eventSetProposalValidatedInternal, SrcState: []fsm.State{StateAwaitParticipantsConfirmations}, DstState: dkg_proposal_fsm.StateDkgPubKeysAwaitConfirmations, IsInternal: true},
			// nan
			{Name: eventSetValidationCanceledByTimeout, SrcState: []fsm.State{StateAwaitParticipantsConfirmations}, DstState: StateValidationCanceledByTimeout, IsInternal: true},
		},
		fsm.Callbacks{
			EventInitProposal:                 machine.actionInitProposal,
			EventConfirmProposal:              machine.actionProposalResponseByParticipant,
			EventDeclineProposal:              machine.actionProposalResponseByParticipant,
			eventAutoValidateProposalInternal: machine.actionValidateSignatureProposal,
			eventSetProposalValidatedInternal: machine.actionSetValidatedSignatureProposal,
		},
	)
	return machine
}

func (m *SignatureProposalFSM) SetUpPayload(payload *internal.DumpedMachineStatePayload) {
	m.payloadMu.Lock()
	defer m.payloadMu.Unlock()

	m.payload = payload
}
