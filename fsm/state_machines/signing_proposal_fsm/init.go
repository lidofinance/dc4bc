package signing_proposal_fsm

import (
	"github.com/lidofinance/dc4bc/fsm/fsm"
	dkp "github.com/lidofinance/dc4bc/fsm/state_machines/dkg_proposal_fsm"
	"github.com/lidofinance/dc4bc/fsm/state_machines/internal"
	"sync"
)

const (
	FsmName = "signing_proposal_fsm"

	StateSigningInitial = dkp.StateDkgMasterKeyCollected

	StateSigningIdle = fsm.State("stage_signing_idle")

	// Starting

	StateSigningAwaitConfirmations = fsm.State("state_signing_await_confirmations")
	// Cancelled
	StateSigningConfirmationsAwaitCancelledByTimeout     = fsm.State("state_signing_confirmations_await_cancelled_by_timeout")
	StateSigningConfirmationsAwaitCancelledByParticipant = fsm.State("state_signing_confirmations_await_cancelled_by_participant")

	StateSigningAwaitPartialSigns = fsm.State("state_signing_await_partial_signs")
	// Cancelled
	StateSigningPartialSignsAwaitCancelledByTimeout = fsm.State("state_signing_partial_signs_await_cancelled_by_timeout")
	StateSigningPartialSignsAwaitCancelledByError   = fsm.State("state_signing_partial_signs_await_cancelled_by_error")

	StateSigningPartialSignsCollected = fsm.State("state_signing_partial_signs_collected")

	// Events

	EventSigningInit                                    = fsm.Event("event_signing_init")
	EventSigningStart                                   = fsm.Event("event_signing_start")
	EventConfirmSigningConfirmation                     = fsm.Event("event_signing_proposal_confirm_by_participant")
	EventDeclineSigningConfirmation                     = fsm.Event("event_signing_proposal_decline_by_participant")
	eventSetSigningConfirmCanceledByParticipantInternal = fsm.Event("event_signing_proposal_canceled_by_participant")
	eventSetSigningConfirmCanceledByTimeoutInternal     = fsm.Event("event_signing_proposal_canceled_by_timeout")

	eventAutoSigningValidateProposalInternal = fsm.Event("event_signing_proposal_await_validate")
	eventSetProposalValidatedInternal        = fsm.Event("event_signing_proposal_set_validated")

	EventSigningPartialSignReceived                      = fsm.Event("event_signing_partial_sign_received")
	EventSigningPartialSignError                         = fsm.Event("event_signing_partial_sign_error_received")
	eventSigningPartialSignsAwaitCancelByTimeoutInternal = fsm.Event("event_signing_partial_signs_await_cancel_by_timeout_internal")
	eventSigningPartialSignsAwaitCancelByErrorInternal   = fsm.Event("event_signing_partial_signs_await_sign_cancel_by_error_internal")

	eventAutoSigningValidatePartialSignInternal = fsm.Event("event_signing_partial_signs_await_validate")

	eventSigningPartialSignsConfirmedInternal = fsm.Event("event_signing_partial_signs_confirmed_internal")
	EventSigningRestart                       = fsm.Event("event_signing_restart")
)

type SigningProposalFSM struct {
	*fsm.FSM
	payload   *internal.DumpedMachineStatePayload
	payloadMu sync.RWMutex
}

func New() internal.DumpedMachineProvider {
	machine := &SigningProposalFSM{}

	machine.FSM = fsm.MustNewFSM(
		FsmName,
		StateSigningInitial,
		[]fsm.EventDesc{
			// Init
			{Name: EventSigningInit, SrcState: []fsm.State{StateSigningInitial}, DstState: StateSigningIdle},

			// Start
			{Name: EventSigningStart, SrcState: []fsm.State{StateSigningIdle}, DstState: StateSigningAwaitConfirmations},

			// Validate by participants
			{Name: EventConfirmSigningConfirmation, SrcState: []fsm.State{StateSigningAwaitConfirmations}, DstState: StateSigningAwaitConfirmations},
			{Name: EventDeclineSigningConfirmation, SrcState: []fsm.State{StateSigningAwaitConfirmations}, DstState: StateSigningAwaitConfirmations},

			// Canceled
			{Name: eventSetSigningConfirmCanceledByParticipantInternal, SrcState: []fsm.State{StateSigningAwaitConfirmations}, DstState: StateSigningConfirmationsAwaitCancelledByParticipant, IsInternal: true},
			{Name: eventSetSigningConfirmCanceledByTimeoutInternal, SrcState: []fsm.State{StateSigningAwaitConfirmations}, DstState: StateSigningConfirmationsAwaitCancelledByTimeout, IsInternal: true},

			// Validate
			{Name: eventAutoSigningValidateProposalInternal, SrcState: []fsm.State{StateSigningAwaitConfirmations}, DstState: StateSigningAwaitConfirmations, IsInternal: true, IsAuto: true},

			{Name: eventSetProposalValidatedInternal, SrcState: []fsm.State{StateSigningAwaitConfirmations}, DstState: StateSigningAwaitPartialSigns, IsInternal: true},

			// Canceled
			{Name: EventSigningPartialSignReceived, SrcState: []fsm.State{StateSigningAwaitPartialSigns}, DstState: StateSigningAwaitPartialSigns},
			{Name: EventSigningPartialSignError, SrcState: []fsm.State{StateSigningAwaitPartialSigns, StateSigningPartialSignsAwaitCancelledByError}, DstState: StateSigningPartialSignsAwaitCancelledByError},
			{Name: eventSigningPartialSignsAwaitCancelByTimeoutInternal, SrcState: []fsm.State{StateSigningAwaitPartialSigns}, DstState: StateSigningPartialSignsAwaitCancelledByTimeout, IsInternal: true},
			{Name: eventSigningPartialSignsAwaitCancelByErrorInternal, SrcState: []fsm.State{StateSigningAwaitPartialSigns, StateSigningPartialSignsAwaitCancelledByError}, DstState: StateSigningPartialSignsAwaitCancelledByError, IsInternal: true},

			// Validate
			{Name: eventAutoSigningValidatePartialSignInternal, SrcState: []fsm.State{StateSigningAwaitPartialSigns}, DstState: StateSigningAwaitPartialSigns, IsInternal: true, IsAuto: true},

			{Name: eventSigningPartialSignsConfirmedInternal, SrcState: []fsm.State{StateSigningAwaitPartialSigns}, DstState: StateSigningPartialSignsCollected, IsInternal: true},

			{Name: EventSigningRestart, SrcState: []fsm.State{StateSigningPartialSignsCollected, StateSigningPartialSignsAwaitCancelledByError, StateSigningPartialSignsAwaitCancelledByTimeout, StateSigningConfirmationsAwaitCancelledByTimeout}, DstState: StateSigningIdle},
		},
		fsm.Callbacks{
			EventSigningInit:                            machine.actionInitSigningProposal,
			EventSigningStart:                           machine.actionStartSigningProposal,
			EventConfirmSigningConfirmation:             machine.actionProposalResponseByParticipant,
			EventDeclineSigningConfirmation:             machine.actionProposalResponseByParticipant,
			eventAutoSigningValidateProposalInternal:    machine.actionValidateSigningProposalConfirmations,
			EventSigningPartialSignReceived:             machine.actionPartialSignConfirmationReceived,
			eventAutoSigningValidatePartialSignInternal: machine.actionValidateSigningPartialSignsAwaitConfirmations,
			EventSigningPartialSignError:                machine.actionConfirmationError,
			EventSigningRestart:                         machine.actionSigningRestart,
		},
	)

	return machine
}

func (m *SigningProposalFSM) WithSetup(state fsm.State, payload *internal.DumpedMachineStatePayload) internal.DumpedMachineProvider {
	m.payloadMu.Lock()
	defer m.payloadMu.Unlock()

	m.payload = payload
	m.FSM = m.FSM.MustCopyWithState(state)
	return m
}
