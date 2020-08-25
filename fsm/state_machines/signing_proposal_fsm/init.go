package signing_proposal_fsm

import (
	"github.com/depools/dc4bc/fsm/fsm"
	dkp "github.com/depools/dc4bc/fsm/state_machines/dkg_proposal_fsm"
	"github.com/depools/dc4bc/fsm/state_machines/internal"
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

	StateSigningAwaitPartialKeys = fsm.State("state_signing_await_partial_keys")
	// Cancelled
	StateSigningPartialKeysAwaitCancelledByTimeout     = fsm.State("state_signing_partial_signatures_await_cancelled_by_timeout")
	StateSigningPartialKeysAwaitCancelledByParticipant = fsm.State("state_signing_partial_signatures_await_cancelled_by_participant")

	StateSigningPartialKeysCollected = fsm.State("state_signing_partial_signatures_collected")

	// Events

	EventSigningInit                                    = fsm.Event("event_signing_init")
	EventSigningStart                                   = fsm.Event("event_signing_start")
	EventConfirmSigningConfirmation                     = fsm.Event("event_signing_proposal_confirm_by_participant")
	EventDeclineSigningConfirmation                     = fsm.Event("event_signing_proposal_decline_by_participant")
	eventSetSigningConfirmCanceledByParticipantInternal = fsm.Event("event_signing_proposal_canceled_by_participant")
	eventSetSigningConfirmCanceledByTimeoutInternal     = fsm.Event("event_signing_proposal_canceled_by_timeout")

	eventAutoValidateProposalInternal = fsm.Event("event_signing_proposal_validate")
	eventSetProposalValidatedInternal = fsm.Event("event_signing_proposal_set_validated")

	EventSigningPartialKeyReceived                = fsm.Event("event_signing_partial_key_received")
	EventSigningPartialKeyError                   = fsm.Event("event_signing_partial_key_error_received")
	eventSigningPartialKeyCancelByTimeoutInternal = fsm.Event("event_signing_partial_key_canceled_by_timeout_internal")
	eventSigningPartialKeyCancelByErrorInternal   = fsm.Event("event_signing_partial_key_canceled_by_error_internal")
	eventSigningPartialKeysConfirmedInternal      = fsm.Event("event_signing_partial_keys_confirmed_internal")
	EventSigningRestart                           = fsm.Event("event_signing_restart")
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
			{Name: eventAutoValidateProposalInternal, SrcState: []fsm.State{StateSigningAwaitConfirmations}, DstState: StateSigningAwaitConfirmations, IsInternal: true, IsAuto: true},

			{Name: eventSetProposalValidatedInternal, SrcState: []fsm.State{StateSigningAwaitConfirmations}, DstState: StateSigningAwaitPartialKeys, IsInternal: true},

			// Canceled
			{Name: EventSigningPartialKeyReceived, SrcState: []fsm.State{StateSigningAwaitPartialKeys}, DstState: StateSigningAwaitPartialKeys},
			{Name: EventSigningPartialKeyError, SrcState: []fsm.State{StateSigningAwaitPartialKeys}, DstState: StateSigningPartialKeysAwaitCancelledByParticipant},
			{Name: eventSigningPartialKeyCancelByTimeoutInternal, SrcState: []fsm.State{StateSigningAwaitPartialKeys}, DstState: StateSigningPartialKeysAwaitCancelledByTimeout, IsInternal: true},
			{Name: eventSigningPartialKeyCancelByErrorInternal, SrcState: []fsm.State{StateSigningAwaitPartialKeys}, DstState: StateSigningPartialKeysAwaitCancelledByParticipant, IsInternal: true, IsAuto: true},

			{Name: eventSigningPartialKeysConfirmedInternal, SrcState: []fsm.State{StateSigningAwaitPartialKeys}, DstState: StateSigningPartialKeysCollected, IsInternal: true},

			{Name: EventSigningRestart, SrcState: []fsm.State{StateSigningPartialKeysCollected}, DstState: StateSigningIdle, IsInternal: true},
		},
		fsm.Callbacks{
			EventSigningInit:                  machine.actionInitSigningProposal,
			EventSigningStart:                 machine.actionStartSigningProposal,
			EventConfirmSigningConfirmation:   machine.actionProposalResponseByParticipant,
			EventDeclineSigningConfirmation:   machine.actionProposalResponseByParticipant,
			eventAutoValidateProposalInternal: machine.actionValidateSigningProposalConfirmations,
			EventSigningPartialKeyReceived:    machine.actionPartialKeyConfirmationReceived,
			EventSigningPartialKeyError:       machine.actionValidateSigningPartialKeyAwaitConfirmations,
			// actionConfirmationError
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
