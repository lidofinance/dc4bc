package signing_proposal_fsm

import (
	"sync"

	"github.com/lidofinance/dc4bc/fsm/fsm"
	dkp "github.com/lidofinance/dc4bc/fsm/state_machines/dkg_proposal_fsm"
	"github.com/lidofinance/dc4bc/fsm/state_machines/internal"
)

const (
	FsmName = "signing_proposal_fsm"

	StateSigningInitial = dkp.StateDkgMasterKeyCollected

	StateSigningIdle = fsm.State("stage_signing_idle")

	StateSigningAwaitPartialSigns = fsm.State("state_signing_await_partial_signs")

	StateSigningPartialSignsAwaitCancelledByTimeout = fsm.State("state_signing_partial_signs_await_cancelled_by_timeout")
	StateSigningPartialSignsAwaitCancelledByError   = fsm.State("state_signing_partial_signs_await_cancelled_by_error")

	StateSigningPartialSignsCollected = fsm.State("state_signing_partial_signs_collected")

	EventSigningInit = fsm.Event("event_signing_init")

	EventSigningStart = fsm.Event("event_signing_start")

	EventSigningPartialSignReceived = fsm.Event("event_signing_partial_sign_received")

	EventSigningPartialSignError                         = fsm.Event("event_signing_partial_sign_error_received")
	eventSigningPartialSignsAwaitCancelByTimeoutInternal = fsm.Event("event_signing_partial_signs_await_cancel_by_timeout_internal")
	eventSigningPartialSignsAwaitCancelByErrorInternal   = fsm.Event("event_signing_partial_signs_await_sign_cancel_by_error_internal")

	eventAutoSigningValidatePartialSignInternal = fsm.Event("event_signing_partial_signs_await_validate")

	eventSigningPartialSignsConfirmedInternal = fsm.Event("event_signing_partial_signs_confirmed_internal")

	EventSigningRestart = fsm.Event("event_signing_restart")
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
			{Name: EventSigningInit, SrcState: []fsm.State{StateSigningInitial}, DstState: StateSigningIdle},

			{Name: EventSigningStart, SrcState: []fsm.State{StateSigningIdle}, DstState: StateSigningAwaitPartialSigns},

			{Name: EventSigningPartialSignReceived, SrcState: []fsm.State{StateSigningAwaitPartialSigns}, DstState: StateSigningAwaitPartialSigns},

			{Name: EventSigningPartialSignError, SrcState: []fsm.State{StateSigningAwaitPartialSigns}, DstState: StateSigningAwaitPartialSigns},
			{Name: eventSigningPartialSignsAwaitCancelByTimeoutInternal, SrcState: []fsm.State{StateSigningAwaitPartialSigns}, DstState: StateSigningPartialSignsAwaitCancelledByTimeout, IsInternal: true},
			{Name: eventSigningPartialSignsAwaitCancelByErrorInternal, SrcState: []fsm.State{StateSigningAwaitPartialSigns}, DstState: StateSigningPartialSignsAwaitCancelledByError, IsInternal: true},

			{Name: eventAutoSigningValidatePartialSignInternal, SrcState: []fsm.State{StateSigningAwaitPartialSigns}, DstState: StateSigningAwaitPartialSigns, IsInternal: true, IsAuto: true},

			{Name: eventSigningPartialSignsConfirmedInternal, SrcState: []fsm.State{StateSigningAwaitPartialSigns}, DstState: StateSigningPartialSignsCollected, IsInternal: true},

			{Name: EventSigningRestart, SrcState: []fsm.State{StateSigningPartialSignsCollected, StateSigningPartialSignsAwaitCancelledByTimeout, StateSigningPartialSignsAwaitCancelledByError}, DstState: StateSigningIdle},
		},
		fsm.Callbacks{
			EventSigningInit:                            machine.actionInitSigningProposal,
			EventSigningStart:                           machine.actionStartSigningProposal,
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
