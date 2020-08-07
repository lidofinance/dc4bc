package dkg_proposal_fsm

import (
	"github.com/depools/dc4bc/fsm/fsm"
	"github.com/depools/dc4bc/fsm/fsm_pool"
	"github.com/depools/dc4bc/fsm/state_machines/signature_proposal_fsm"
)

const (
	fsmName = "dkg_proposal_fsm"

	StateDkgInitial = signature_proposal_fsm.StateValidationCompleted
	// Sending dkg pub keys
	StateDkgPubKeysSendingRequired = fsm.State("state_dkg_pub_keys_sending_required")

	StateDkgPubKeysSendingAwaitConfirmations = fsm.State("state_dkg_pub_keys_sending_await_confirmations")
	// Cancelled
	StateDkgPubKeysSendingCancelled          = fsm.State("state_dkg_pub_keys_sending_cancelled")
	StateDkgPubKeysSendingCancelledByTimeout = fsm.State("state_dkg_pub_keys_sending_cancelled_by_timeout")
	// Confirmed
	StateDkgPubKeysSendingConfirmed = fsm.State("state_dkg_pub_keys_sending_confirmed")

	// Sending dkg commits
	StateDkgCommitsSendingRequired           = fsm.State("state_dkg_commits_sending_required")
	StateDkgCommitsSendingAwaitConfirmations = fsm.State("state_dkg_commits_sending_await_confirmations")
	// Cancelled
	StateDkgCommitsSendingCancelled          = fsm.State("state_dkg_commits_sending_cancelled")
	StateDkgCommitsSendingCancelledByTimeout = fsm.State("state_dkg_commits_sending_cancelled_by_timeout")
	// Confirmed
	StateDkgCommitsSendingConfirmed = fsm.State("state_dkg_commits_sending_confirmed")

	// Sending dkg deals
	StateDkgDealsSendingRequired           = fsm.State("state_dkg_deals_sending_required")
	StateDkgDealsSendingAwaitConfirmations = fsm.State("state_dkg_deals_sending_await_confirmations")
	// Cancelled
	StateDkgDealsSendingCancelled          = fsm.State("state_dkg_deals_sending_cancelled")
	StateDkgDealsSendingCancelledByTimeout = fsm.State("state_dkg_deals_sending_cancelled_by_timeout")
	// Confirmed
	StateDkgDealsSendingConfirmed = fsm.State("state_dkg_deals_sending_confirmed")

	// Events
	EventDKGPubKeysSendingRequiredInternal = fsm.Event("event_dkg_pub_key_sending_required_internal")

	EventDKGPubKeysSent                                = fsm.Event("event_dkg_pub_keys_sent")
	EventDKGPubKeyConfirmationReceived                 = fsm.Event("event_dkg_pub_key_confirm_received")
	EventDKGPubKeyConfirmationError                    = fsm.Event("event_dkg_pub_key_confirm_canceled_by_error")
	EventDKGPubKeysConfirmationCancelByTimeoutInternal = fsm.Event("event_dkg_pub_keys_confirm_canceled_by_timeout_internal")
	EventDKGPubKeysConfirmedInternal                   = fsm.Event("event_dkg_pub_keys_confirmed_internal")

	EventDKGCommitsSendingRequiredInternal = fsm.Event("event_dkg_commits_sending_required_internal")

	EventDKGCommitsSent                                = fsm.Event("event_dkg_commits_sent")
	EventDKGCommitConfirmationReceived                 = fsm.Event("event_dkg_commit_confirm_received")
	EventDKGCommitConfirmationError                    = fsm.Event("event_dkg_commit_confirm_canceled_by_error")
	EventDKGCommitsConfirmationCancelByTimeoutInternal = fsm.Event("event_dkg_commits_confirm_canceled_by_timeout_internal")
	EventDKGCommitsConfirmedInternal                   = fsm.Event("event_dkg_commits_confirmed_internal")

	EventDKGDealsSendingRequiredInternal = fsm.Event("event_dkg_deals_sending_required_internal")

	EventDKGDealsSent                                = fsm.Event("event_dkg_deals_sent")
	EventDKGDealConfirmationReceived                 = fsm.Event("event_dkg_deal_confirm_received")
	EventDKGDealConfirmationError                    = fsm.Event("event_dkg_deal_confirm_canceled_by_error")
	EventDKGDealsConfirmationCancelByTimeoutInternal = fsm.Event("event_dkg_deals_confirm_canceled_by_timeout_internal")

	EventDKGMasterKeyRequiredInternal = fsm.Event("event_dkg_master_key_required_internal")
)

type DKGProposalFSM struct {
	*fsm.FSM
}

func New() fsm_pool.MachineProvider {
	machine := &DKGProposalFSM{}

	machine.FSM = fsm.MustNewFSM(
		fsmName,
		StateDkgInitial,
		[]fsm.EventDesc{

			// Init
			// Switch to pub keys required
			{Name: EventDKGPubKeysSendingRequiredInternal, SrcState: []fsm.State{StateDkgInitial}, DstState: StateDkgPubKeysSendingRequired, IsInternal: true},

			// Pub keys sending
			{Name: EventDKGPubKeysSent, SrcState: []fsm.State{StateDkgPubKeysSendingRequired}, DstState: StateDkgPubKeysSendingAwaitConfirmations},
			{Name: EventDKGPubKeyConfirmationReceived, SrcState: []fsm.State{StateDkgPubKeysSendingAwaitConfirmations}, DstState: StateDkgPubKeysSendingAwaitConfirmations},
			// Cancelled
			{Name: EventDKGPubKeyConfirmationError, SrcState: []fsm.State{StateDkgPubKeysSendingAwaitConfirmations}, DstState: StateDkgPubKeysSendingCancelled},
			{Name: EventDKGPubKeysConfirmationCancelByTimeoutInternal, SrcState: []fsm.State{StateDkgPubKeysSendingAwaitConfirmations}, DstState: StateDkgPubKeysSendingCancelledByTimeout, IsInternal: true},
			// Confirmed
			{Name: EventDKGPubKeysConfirmedInternal, SrcState: []fsm.State{StateDkgPubKeysSendingAwaitConfirmations}, DstState: StateDkgPubKeysSendingConfirmed, IsInternal: true},

			// Switch to commits required
			{Name: EventDKGCommitsSendingRequiredInternal, SrcState: []fsm.State{StateDkgPubKeysSendingConfirmed}, DstState: StateDkgCommitsSendingRequired, IsInternal: true},

			// Commits
			{Name: EventDKGCommitsSent, SrcState: []fsm.State{StateDkgCommitsSendingRequired}, DstState: StateDkgCommitsSendingAwaitConfirmations},
			{Name: EventDKGCommitConfirmationReceived, SrcState: []fsm.State{StateDkgCommitsSendingAwaitConfirmations}, DstState: StateDkgCommitsSendingAwaitConfirmations},
			// Cancelled
			{Name: EventDKGCommitConfirmationError, SrcState: []fsm.State{StateDkgCommitsSendingAwaitConfirmations}, DstState: StateDkgCommitsSendingCancelled},
			{Name: EventDKGCommitsConfirmationCancelByTimeoutInternal, SrcState: []fsm.State{StateDkgCommitsSendingAwaitConfirmations}, DstState: StateDkgCommitsSendingCancelledByTimeout, IsInternal: true},
			// Confirmed
			{Name: EventDKGCommitsConfirmedInternal, SrcState: []fsm.State{StateDkgCommitsSendingAwaitConfirmations}, DstState: StateDkgCommitsSendingConfirmed, IsInternal: true},

			// Switch to deals required
			{Name: EventDKGDealsSendingRequiredInternal, SrcState: []fsm.State{StateDkgDealsSendingConfirmed}, DstState: StateDkgDealsSendingRequired, IsInternal: true},

			// Deals
			{Name: EventDKGDealsSent, SrcState: []fsm.State{StateDkgDealsSendingRequired}, DstState: StateDkgDealsSendingAwaitConfirmations},
			{Name: EventDKGDealConfirmationReceived, SrcState: []fsm.State{StateDkgDealsSendingAwaitConfirmations}, DstState: StateDkgDealsSendingAwaitConfirmations},
			// Cancelled
			{Name: EventDKGDealConfirmationError, SrcState: []fsm.State{StateDkgCommitsSendingAwaitConfirmations}, DstState: StateDkgDealsSendingCancelled},
			{Name: EventDKGDealsConfirmationCancelByTimeoutInternal, SrcState: []fsm.State{StateDkgCommitsSendingAwaitConfirmations}, DstState: StateDkgDealsSendingCancelledByTimeout, IsInternal: true},

			// Done
			{Name: EventDKGMasterKeyRequiredInternal, SrcState: []fsm.State{StateDkgCommitsSendingAwaitConfirmations}, DstState: fsm.StateGlobalDone, IsInternal: true},
		},
		fsm.Callbacks{
			EventDKGPubKeysSent:                machine.actionDKGPubKeysSent,
			EventDKGPubKeyConfirmationReceived: machine.actionDKGPubKeyConfirmationReceived,
			EventDKGPubKeyConfirmationError:    machine.actionDKGPubKeyConfirmationError,

			EventDKGCommitsSent:                machine.actionDKGCommitsSent,
			EventDKGCommitConfirmationReceived: machine.actionDKGCommitConfirmationReceived,
			EventDKGCommitConfirmationError:    machine.actionDKGCommitConfirmationError,

			EventDKGDealsSent:                machine.actionDKGDealsSent,
			EventDKGDealConfirmationReceived: machine.actionDKGDealConfirmationReceived,
			EventDKGDealConfirmationError:    machine.actionDKGDealConfirmationError,
		},
	)
	return machine
}
