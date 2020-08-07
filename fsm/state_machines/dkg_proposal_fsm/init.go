package dkg_proposal_fsm

import (
	"github.com/depools/dc4bc/fsm/fsm"
	"github.com/depools/dc4bc/fsm/state_machines/internal"
	"github.com/depools/dc4bc/fsm/state_machines/signature_proposal_fsm"
	"sync"
)

const (
	fsmName = "dkg_proposal_fsm"

	StateDkgInitial = signature_proposal_fsm.StateValidationCompleted

	StateDkgPubKeysAwaitConfirmations = fsm.State("state_dkg_pub_keys_await_confirmations")
	// Cancelled
	StateDkgPubKeysAwaitCancelled          = fsm.State("state_dkg_pub_keys_await_cancelled")
	StateDkgPubKeysAwaitCancelledByTimeout = fsm.State("state_dkg_pub_keys_await_cancelled_by_timeout")
	// Confirmed
	StateDkgPubKeysAwaitConfirmed = fsm.State("state_dkg_pub_keys_await_confirmed")

	// Sending dkg commits
	StateDkgCommitsAwaitConfirmations = fsm.State("state_dkg_commits_sending_await_confirmations")
	// Cancelled
	StateDkgCommitsAwaitCancelled          = fsm.State("state_dkg_commits_await_cancelled")
	StateDkgCommitsAwaitCancelledByTimeout = fsm.State("state_dkg_commits_await_cancelled_by_timeout")
	// Confirmed
	StateDkgCommitsAwaitConfirmed = fsm.State("state_dkg_commits_await_confirmed")

	// Sending dkg deals
	StateDkgDealsAwaitConfirmations = fsm.State("state_dkg_deals_await_confirmations")
	// Cancelled
	StateDkgDealsAwaitCancelled          = fsm.State("state_dkg_deals_await_cancelled")
	StateDkgDealsAwaitCancelledByTimeout = fsm.State("state_dkg_deals_sending_cancelled_by_timeout")
	// Confirmed
	StateDkgDealsAwaitConfirmed = fsm.State("state_dkg_deals_await_confirmed")

	StateDkgResponsesAwaitConfirmations = fsm.State("state_dkg_responses_await_confirmations")
	// Cancelled
	StateDkgResponsesAwaitCancelled          = fsm.State("state_dkg_responses_await_cancelled")
	StateDkgResponsesAwaitCancelledByTimeout = fsm.State("state_dkg_responses_sending_cancelled_by_timeout")
	// Confirmed
	StateDkgResponsesAwaitConfirmed = fsm.State("state_dkg_responses_await_confirmed")

	// Events
	EventDKGPubKeysSendingRequiredInternal = fsm.Event("event_dkg_pub_key_sending_required_internal")

	EventDKGPubKeyConfirmationReceived                 = fsm.Event("event_dkg_pub_key_confirm_received")
	EventDKGPubKeyConfirmationError                    = fsm.Event("event_dkg_pub_key_confirm_canceled_by_error")
	EventDKGPubKeysConfirmationCancelByTimeoutInternal = fsm.Event("event_dkg_pub_keys_confirm_canceled_by_timeout_internal")
	EventDKGPubKeysConfirmedInternal                   = fsm.Event("event_dkg_pub_keys_confirmed_internal")

	EventDKGCommitsSendingRequiredInternal = fsm.Event("event_dkg_commits_sending_required_internal")

	EventDKGCommitConfirmationReceived                 = fsm.Event("event_dkg_commit_confirm_received")
	EventDKGCommitConfirmationError                    = fsm.Event("event_dkg_commit_confirm_canceled_by_error")
	EventDKGCommitsConfirmationCancelByTimeoutInternal = fsm.Event("event_dkg_commits_confirm_canceled_by_timeout_internal")
	EventDKGCommitsConfirmedInternal                   = fsm.Event("event_dkg_commits_confirmed_internal")

	EventDKGDealsSendingRequiredInternal = fsm.Event("event_dkg_deals_sending_required_internal")

	EventDKGDealConfirmationReceived                 = fsm.Event("event_dkg_deal_confirm_received")
	EventDKGDealConfirmationError                    = fsm.Event("event_dkg_deal_confirm_canceled_by_error")
	EventDKGDealsConfirmationCancelByTimeoutInternal = fsm.Event("event_dkg_deals_confirm_canceled_by_timeout_internal")

	EventDKGResponsesSendingRequiredInternal = fsm.Event("event_dkg_responses_sending_required_internal")

	EventDKGResponseConfirmationReceived                = fsm.Event("event_dkg_response_confirm_received")
	EventDKGResponseConfirmationError                   = fsm.Event("event_dkg_response_confirm_canceled_by_error")
	EventDKGResponseConfirmationCancelByTimeoutInternal = fsm.Event("event_dkg_response_confirm_canceled_by_timeout_internal")

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
		fsmName,
		StateDkgInitial,
		[]fsm.EventDesc{

			// Init
			// Switch to pub keys required
			{Name: EventDKGPubKeysSendingRequiredInternal, SrcState: []fsm.State{StateDkgInitial}, DstState: StateDkgPubKeysAwaitConfirmations, IsInternal: true},

			// Pub keys sending
			{Name: EventDKGPubKeyConfirmationReceived, SrcState: []fsm.State{StateDkgPubKeysAwaitConfirmations}, DstState: StateDkgPubKeysAwaitConfirmations},
			// Cancelled
			{Name: EventDKGPubKeyConfirmationError, SrcState: []fsm.State{StateDkgPubKeysAwaitConfirmations}, DstState: StateDkgPubKeysAwaitCancelled},
			{Name: EventDKGPubKeysConfirmationCancelByTimeoutInternal, SrcState: []fsm.State{StateDkgPubKeysAwaitConfirmations}, DstState: StateDkgPubKeysAwaitCancelledByTimeout, IsInternal: true},
			// Confirmed
			{Name: EventDKGPubKeysConfirmedInternal, SrcState: []fsm.State{StateDkgPubKeysAwaitConfirmations}, DstState: StateDkgPubKeysAwaitConfirmed, IsInternal: true},

			// Switch to commits required
			{Name: EventDKGCommitsSendingRequiredInternal, SrcState: []fsm.State{StateDkgPubKeysAwaitConfirmed}, DstState: StateDkgCommitsAwaitConfirmations, IsInternal: true},

			// Commits
			{Name: EventDKGCommitConfirmationReceived, SrcState: []fsm.State{StateDkgCommitsAwaitConfirmations}, DstState: StateDkgCommitsAwaitConfirmations},
			// Cancelled
			{Name: EventDKGCommitConfirmationError, SrcState: []fsm.State{StateDkgCommitsAwaitConfirmations}, DstState: StateDkgCommitsAwaitCancelled},
			{Name: EventDKGCommitsConfirmationCancelByTimeoutInternal, SrcState: []fsm.State{StateDkgCommitsAwaitConfirmations}, DstState: StateDkgCommitsAwaitCancelledByTimeout, IsInternal: true},
			// Confirmed
			{Name: EventDKGCommitsConfirmedInternal, SrcState: []fsm.State{StateDkgCommitsAwaitConfirmations}, DstState: StateDkgCommitsAwaitConfirmed, IsInternal: true},

			// Switch to deals required
			{Name: EventDKGDealsSendingRequiredInternal, SrcState: []fsm.State{StateDkgDealsAwaitConfirmed}, DstState: StateDkgDealsAwaitConfirmations, IsInternal: true},

			// Deals
			{Name: EventDKGDealConfirmationReceived, SrcState: []fsm.State{StateDkgDealsAwaitConfirmations}, DstState: StateDkgDealsAwaitConfirmations},
			// Cancelled
			{Name: EventDKGDealConfirmationError, SrcState: []fsm.State{StateDkgCommitsAwaitConfirmations}, DstState: StateDkgDealsAwaitCancelled},
			{Name: EventDKGDealsConfirmationCancelByTimeoutInternal, SrcState: []fsm.State{StateDkgCommitsAwaitConfirmations}, DstState: StateDkgDealsAwaitCancelledByTimeout, IsInternal: true},

			// Switch to responses required
			{Name: EventDKGResponsesSendingRequiredInternal, SrcState: []fsm.State{StateDkgResponsesAwaitConfirmed}, DstState: StateDkgResponsesAwaitConfirmations, IsInternal: true},

			// Deals
			{Name: EventDKGResponseConfirmationReceived, SrcState: []fsm.State{StateDkgResponsesAwaitConfirmations}, DstState: StateDkgResponsesAwaitConfirmations},
			// Cancelled
			{Name: EventDKGResponseConfirmationError, SrcState: []fsm.State{StateDkgResponsesAwaitConfirmations}, DstState: StateDkgResponsesAwaitCancelled},
			{Name: EventDKGResponseConfirmationCancelByTimeoutInternal, SrcState: []fsm.State{StateDkgResponsesAwaitConfirmations}, DstState: StateDkgResponsesAwaitCancelledByTimeout, IsInternal: true},

			// Done
			{Name: EventDKGMasterKeyRequiredInternal, SrcState: []fsm.State{StateDkgResponsesAwaitConfirmations}, DstState: fsm.StateGlobalDone, IsInternal: true},
		},
		fsm.Callbacks{
			EventDKGPubKeyConfirmationReceived: machine.actionPubKeyConfirmationReceived,
			EventDKGPubKeyConfirmationError:    machine.actionConfirmationError,

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
