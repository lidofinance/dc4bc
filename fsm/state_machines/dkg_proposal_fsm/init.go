package dkg_proposal_fsm

import (
	"github.com/depools/dc4bc/fsm/fsm"
	"github.com/depools/dc4bc/fsm/state_machines/internal"
	"sync"
)

const (
	FsmName = "dkg_proposal_fsm"

	StateDkgInitial = StateDkgPubKeysAwaitConfirmations

	StateDkgPubKeysAwaitConfirmations = fsm.State("state_dkg_pub_keys_await_confirmations")
	// Canceled
	StateDkgPubKeysAwaitCanceled          = fsm.State("state_dkg_pub_keys_await_canceled")
	StateDkgPubKeysAwaitCanceledByTimeout = fsm.State("state_dkg_pub_keys_await_canceled_by_timeout")
	// Confirmed
	// StateDkgPubKeysAwaitConfirmed = fsm.State("state_dkg_pub_keys_await_confirmed")

	// Sending dkg commits
	StateDkgCommitsAwaitConfirmations = fsm.State("state_dkg_commits_await_confirmations")
	// Canceled
	StateDkgCommitsAwaitCanceled          = fsm.State("state_dkg_commits_await_canceled")
	StateDkgCommitsAwaitCanceledByTimeout = fsm.State("state_dkg_commits_await_canceled_by_timeout")
	// Confirmed
	StateDkgCommitsAwaitConfirmed = fsm.State("state_dkg_commits_await_confirmed")

	// Sending dkg deals
	StateDkgDealsAwaitConfirmations = fsm.State("state_dkg_deals_await_confirmations")
	// Canceled
	StateDkgDealsAwaitCanceled          = fsm.State("state_dkg_deals_await_canceled")
	StateDkgDealsAwaitCanceledByTimeout = fsm.State("state_dkg_deals_sending_canceled_by_timeout")
	// Confirmed
	//StateDkgDealsAwaitConfirmed = fsm.State("state_dkg_deals_await_confirmed")

	StateDkgResponsesAwaitConfirmations = fsm.State("state_dkg_responses_await_confirmations")
	// Canceled
	StateDkgResponsesAwaitCanceled          = fsm.State("state_dkg_responses_await_canceled")
	StateDkgResponsesAwaitCanceledByTimeout = fsm.State("state_dkg_responses_sending_canceled_by_timeout")
	// Confirmed
	StateDkgResponsesAwaitConfirmed = fsm.State("state_dkg_responses_await_confirmed")

	// Events

	eventAutoDKGInitialInternal = fsm.Event("event_dkg_init_internal")

	EventDKGPubKeyConfirmationReceived = fsm.Event("event_dkg_pub_key_confirm_received")
	EventDKGPubKeyConfirmationError    = fsm.Event("event_dkg_pub_key_confirm_canceled_by_error")

	eventAutoValidatePubKeysInternal = fsm.Event("event_dkg_pub_keys_validate_internal")

	eventDKGSetPubKeysConfirmationCanceledByTimeoutInternal = fsm.Event("event_dkg_pub_keys_confirm_canceled_by_timeout_internal")
	eventDKGSetPubKeysConfirmationCanceledByErrorInternal   = fsm.Event("event_dkg_pub_keys_confirm_canceled_by_error_internal")
	eventDKGSetPubKeysConfirmedInternal                     = fsm.Event("event_dkg_pub_keys_confirmed_internal")

	EventDKGCommitConfirmationReceived                 = fsm.Event("event_dkg_commit_confirm_received")
	EventDKGCommitConfirmationError                    = fsm.Event("event_dkg_commit_confirm_canceled_by_error")
	eventDKGCommitsConfirmationCancelByTimeoutInternal = fsm.Event("event_dkg_commits_confirm_canceled_by_timeout_internal")
	eventDKGCommitsConfirmationCancelByErrorInternal   = fsm.Event("event_dkg_commits_confirm_canceled_by_error_internal")
	eventDKGCommitsConfirmedInternal                   = fsm.Event("event_dkg_commits_confirmed_internal")

	// EventDKGDealsSendingRequiredInternal = fsm.Event("event_dkg_deals_sending_required_internal")

	EventDKGDealConfirmationReceived                 = fsm.Event("event_dkg_deal_confirm_received")
	EventDKGDealConfirmationError                    = fsm.Event("event_dkg_deal_confirm_canceled_by_error")
	eventDKGDealsConfirmationCancelByTimeoutInternal = fsm.Event("event_dkg_deals_confirm_canceled_by_timeout_internal")
	eventDKGDealsConfirmationCancelByErrorInternal   = fsm.Event("event_dkg_deals_confirm_canceled_by_error_internal")
	eventDKGDealsConfirmedInternal                   = fsm.Event("event_dkg_deals_confirmed_internal")

	EventDKGResponseConfirmationReceived                = fsm.Event("event_dkg_response_confirm_received")
	EventDKGResponseConfirmationError                   = fsm.Event("event_dkg_response_confirm_canceled_by_error")
	eventDKGResponseConfirmationCancelByTimeoutInternal = fsm.Event("event_dkg_response_confirm_canceled_by_timeout_internal")
	eventDKGResponseConfirmationCancelByErrorInternal   = fsm.Event("event_dkg_response_confirm_canceled_by_error_internal")
	eventDKGResponsesConfirmedInternal                  = fsm.Event("event_dkg_responses_confirmed_internal")

	EventDKGMasterKeyRequiredInternal = fsm.Event("event_dkg_master_key_required_internal")
)

type DKGProposalFSM struct {
	*fsm.FSM
	payload   *internal.DumpedMachineStatePayload
	payloadMu sync.RWMutex
}

func New() internal.DumpedMachineProvider {
	machine := &DKGProposalFSM{}

	machine.FSM = fsm.MustNewFSM(
		FsmName,
		StateDkgInitial,
		[]fsm.EventDesc{

			// Init
			// Switch to pub keys required
			// 	{Name: eventDKGPubKeysSendingRequiredAuto, SrcState: []fsm.State{StateDkgInitial}, DstState: StateDkgPubKeysAwaitConfirmations, IsInternal: true, IsAuto: true, AutoRunMode: fsm.EventRunAfter},

			// {Name: eventAutoDKGInitialInternal, SrcState: []fsm.State{StateDkgPubKeysAwaitConfirmations}, DstState: StateDkgPubKeysAwaitConfirmations, IsInternal: true, IsAuto: true, AutoRunMode: fsm.EventRunBefore},

			// Pub keys sending
			{Name: EventDKGPubKeyConfirmationReceived, SrcState: []fsm.State{StateDkgPubKeysAwaitConfirmations}, DstState: StateDkgPubKeysAwaitConfirmations},
			// Canceled
			{Name: EventDKGPubKeyConfirmationError, SrcState: []fsm.State{StateDkgPubKeysAwaitConfirmations}, DstState: StateDkgPubKeysAwaitCanceled},

			{Name: eventAutoValidatePubKeysInternal, SrcState: []fsm.State{StateDkgPubKeysAwaitConfirmations}, DstState: StateDkgPubKeysAwaitConfirmations, IsInternal: true, IsAuto: true},

			{Name: eventDKGSetPubKeysConfirmationCanceledByTimeoutInternal, SrcState: []fsm.State{StateDkgPubKeysAwaitConfirmations}, DstState: StateDkgPubKeysAwaitCanceledByTimeout, IsInternal: true},
			// Confirmed
			{Name: eventDKGSetPubKeysConfirmedInternal, SrcState: []fsm.State{StateDkgPubKeysAwaitConfirmations}, DstState: StateDkgCommitsAwaitConfirmations, IsInternal: true},

			// Switch to commits required
			//{Name: EventDKGCommitsSendingRequiredInternal, SrcState: []fsm.State{StateDkgPubKeysAwaitConfirmed}, DstState: StateDkgCommitsAwaitConfirmations, IsInternal: true},

			// Commits
			{Name: EventDKGCommitConfirmationReceived, SrcState: []fsm.State{StateDkgCommitsAwaitConfirmations}, DstState: StateDkgCommitsAwaitConfirmations},
			// Canceled
			{Name: EventDKGCommitConfirmationError, SrcState: []fsm.State{StateDkgCommitsAwaitConfirmations}, DstState: StateDkgCommitsAwaitCanceled},
			{Name: eventDKGCommitsConfirmationCancelByTimeoutInternal, SrcState: []fsm.State{StateDkgCommitsAwaitConfirmations}, DstState: StateDkgCommitsAwaitCanceledByTimeout, IsInternal: true},
			// Confirmed
			{Name: eventDKGCommitsConfirmedInternal, SrcState: []fsm.State{StateDkgCommitsAwaitConfirmations}, DstState: StateDkgDealsAwaitConfirmations, IsInternal: true},

			// Switch to deals required
			//	{Name: EventDKGDealsSendingRequiredInternal, SrcState: []fsm.State{StateDkgDealsAwaitConfirmed}, DstState: StateDkgDealsAwaitConfirmations, IsInternal: true},

			// Deals
			{Name: EventDKGDealConfirmationReceived, SrcState: []fsm.State{StateDkgDealsAwaitConfirmations}, DstState: StateDkgDealsAwaitConfirmations},
			// Canceled
			{Name: EventDKGDealConfirmationError, SrcState: []fsm.State{StateDkgCommitsAwaitConfirmations}, DstState: StateDkgDealsAwaitCanceled},
			{Name: eventDKGDealsConfirmationCancelByTimeoutInternal, SrcState: []fsm.State{StateDkgCommitsAwaitConfirmations}, DstState: StateDkgDealsAwaitCanceledByTimeout, IsInternal: true},

			// Switch to responses required
			// 	{Name: eventDKGResponsesSendingRequiredInternal, SrcState: []fsm.State{StateDkgResponsesAwaitConfirmed}, DstState: StateDkgResponsesAwaitConfirmations, IsInternal: true},

			// Deals
			{Name: EventDKGResponseConfirmationReceived, SrcState: []fsm.State{StateDkgResponsesAwaitConfirmations}, DstState: StateDkgResponsesAwaitConfirmations},
			// Canceled
			{Name: EventDKGResponseConfirmationError, SrcState: []fsm.State{StateDkgResponsesAwaitConfirmations}, DstState: StateDkgResponsesAwaitCanceled},
			{Name: eventDKGResponseConfirmationCancelByTimeoutInternal, SrcState: []fsm.State{StateDkgResponsesAwaitConfirmations}, DstState: StateDkgResponsesAwaitCanceledByTimeout, IsInternal: true},

			// Done
			{Name: EventDKGMasterKeyRequiredInternal, SrcState: []fsm.State{StateDkgResponsesAwaitConfirmations}, DstState: fsm.StateGlobalDone, IsInternal: true},
		},
		fsm.Callbacks{

			EventDKGPubKeyConfirmationReceived: machine.actionPubKeyConfirmationReceived,
			EventDKGPubKeyConfirmationError:    machine.actionConfirmationError,
			// actionValidateDkgProposalPubKeys
			eventAutoValidatePubKeysInternal: machine.actionValidateDkgProposalPubKeys,

			EventDKGCommitConfirmationReceived: machine.actionCommitConfirmationReceived,
			EventDKGCommitConfirmationError:    machine.actionConfirmationError,

			EventDKGDealConfirmationReceived: machine.actionDealConfirmationReceived,
			EventDKGDealConfirmationError:    machine.actionConfirmationError,

			EventDKGResponseConfirmationReceived: machine.actionResponseConfirmationReceived,
			EventDKGResponseConfirmationError:    machine.actionConfirmationError,
		},
	)
	return machine
}

func (m *DKGProposalFSM) SetUpPayload(payload *internal.DumpedMachineStatePayload) {
	m.payloadMu.Lock()
	defer m.payloadMu.Unlock()

	m.payload = payload
}
