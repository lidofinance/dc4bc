package dkg_proposal_fsm

import (
	"github.com/lidofinance/dc4bc/fsm/fsm"
	"github.com/lidofinance/dc4bc/fsm/state_machines/internal"
	spf "github.com/lidofinance/dc4bc/fsm/state_machines/signature_proposal_fsm"
	"sync"
)

const (
	FsmName = "dkg_proposal_fsm"

	StateDkgInitial = spf.StateSignatureProposalCollected

	// Sending dkg commits
	StateDkgCommitsAwaitConfirmations = fsm.State("state_dkg_commits_await_confirmations")
	// Canceled
	StateDkgCommitsAwaitCanceledByError   = fsm.State("state_dkg_commits_await_canceled_by_error")
	StateDkgCommitsAwaitCanceledByTimeout = fsm.State("state_dkg_commits_await_canceled_by_timeout")
	// Confirmed
	StateDkgCommitsCollected = fsm.State("state_dkg_commits_collected")

	// Sending dkg deals
	StateDkgDealsAwaitConfirmations = fsm.State("state_dkg_deals_await_confirmations")
	// Canceled
	StateDkgDealsAwaitCanceledByError   = fsm.State("state_dkg_deals_await_canceled_by_error")
	StateDkgDealsAwaitCanceledByTimeout = fsm.State("state_dkg_deals_await_canceled_by_timeout")
	// Confirmed
	//StateDkgDealsCollected = fsm.State("state_dkg_deals_collected")

	StateDkgResponsesAwaitConfirmations = fsm.State("state_dkg_responses_await_confirmations")
	// Canceled
	StateDkgResponsesAwaitCanceledByError   = fsm.State("state_dkg_responses_await_canceled_by_error")
	StateDkgResponsesAwaitCanceledByTimeout = fsm.State("state_dkg_responses_sending_canceled_by_timeout")
	// Confirmed
	StateDkgResponsesCollected = fsm.State("state_dkg_responses_collected")

	StateDkgMasterKeyAwaitConfirmations     = fsm.State("state_dkg_master_key_await_confirmations")
	StateDkgMasterKeyAwaitCanceledByError   = fsm.State("state_dkg_master_key_await_canceled_by_error")
	StateDkgMasterKeyAwaitCanceledByTimeout = fsm.State("state_dkg_master_key_await_canceled_by_timeout")

	StateDkgMasterKeyCollected = fsm.State("state_dkg_master_key_collected")

	// Events
	EventDKGInitProcess = fsm.Event("event_dkg_init_process")

	EventDKGCommitConfirmationReceived                 = fsm.Event("event_dkg_commit_confirm_received")
	EventDKGCommitConfirmationError                    = fsm.Event("event_dkg_commit_confirm_canceled_by_error")
	eventDKGCommitsConfirmationCancelByTimeoutInternal = fsm.Event("event_dkg_commits_confirm_canceled_by_timeout_internal")
	eventDKGCommitsConfirmationCancelByErrorInternal   = fsm.Event("event_dkg_commits_confirm_canceled_by_error_internal")
	eventDKGCommitsConfirmedInternal                   = fsm.Event("event_dkg_commits_confirmed_internal")
	eventAutoDKGValidateConfirmationCommitsInternal    = fsm.Event("event_dkg_commits_validate_internal")

	EventDKGDealConfirmationReceived                 = fsm.Event("event_dkg_deal_confirm_received")
	EventDKGDealConfirmationError                    = fsm.Event("event_dkg_deal_confirm_canceled_by_error")
	eventDKGDealsConfirmationCancelByTimeoutInternal = fsm.Event("event_dkg_deals_confirm_canceled_by_timeout_internal")
	eventDKGDealsConfirmationCancelByErrorInternal   = fsm.Event("event_dkg_deals_confirm_canceled_by_error_internal")
	eventDKGDealsConfirmedInternal                   = fsm.Event("event_dkg_deals_confirmed_internal")
	eventAutoDKGValidateConfirmationDealsInternal    = fsm.Event("event_dkg_deals_validate_internal")

	EventDKGResponseConfirmationReceived                = fsm.Event("event_dkg_response_confirm_received")
	EventDKGResponseConfirmationError                   = fsm.Event("event_dkg_response_confirm_canceled_by_error")
	eventDKGResponseConfirmationCancelByTimeoutInternal = fsm.Event("event_dkg_response_confirm_canceled_by_timeout_internal")
	eventDKGResponseConfirmationCancelByErrorInternal   = fsm.Event("event_dkg_response_confirm_canceled_by_error_internal")
	eventDKGResponsesConfirmedInternal                  = fsm.Event("event_dkg_responses_confirmed_internal")
	eventAutoDKGValidateResponsesConfirmationInternal   = fsm.Event("event_dkg_responses_validate_internal")

	EventDKGMasterKeyConfirmationReceived                = fsm.Event("event_dkg_master_key_confirm_received")
	EventDKGMasterKeyConfirmationError                   = fsm.Event("event_dkg_master_key_confirm_canceled_by_error")
	eventDKGMasterKeyConfirmationCancelByTimeoutInternal = fsm.Event("event_dkg_master_key_confirm_canceled_by_timeout_internal")
	eventDKGMasterKeyConfirmationCancelByErrorInternal   = fsm.Event("event_dkg_master_key_confirm_canceled_by_error_internal")
	eventDKGMasterKeyConfirmedInternal                   = fsm.Event("event_dkg_master_key_confirmed_internal")
	eventAutoDKGValidateMasterKeyConfirmationInternal    = fsm.Event("event_dkg_master_key_validate_internal")

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

			// {Name: EventDKGInitProcess, SrcState: []fsm.State{StateDkgPubKeysAwaitConfirmations}, DstState: StateDkgPubKeysAwaitConfirmations, IsInternal: true, IsAuto: true, AutoRunMode: fsm.EventRunBefore},

			// StateDkgCommitsCollected = fsm.State("state_dkg_commits_collected")

			// {Name: eventAutoDKGInitInternal, SrcState: []fsm.State{StateDkgInitial}, DstState: StateDkgCommitsAwaitConfirmations, IsInternal: true, IsAuto: true, AutoRunMode: fsm.EventRunBefore},

			{Name: EventDKGInitProcess, SrcState: []fsm.State{StateDkgInitial}, DstState: StateDkgCommitsAwaitConfirmations},

			// Commits
			{Name: EventDKGCommitConfirmationReceived, SrcState: []fsm.State{StateDkgCommitsAwaitConfirmations}, DstState: StateDkgCommitsAwaitConfirmations},
			// Canceled
			{Name: EventDKGCommitConfirmationError, SrcState: []fsm.State{StateDkgCommitsAwaitConfirmations}, DstState: StateDkgCommitsAwaitCanceledByError},
			{Name: eventDKGCommitsConfirmationCancelByTimeoutInternal, SrcState: []fsm.State{StateDkgCommitsAwaitConfirmations}, DstState: StateDkgCommitsAwaitCanceledByTimeout, IsInternal: true},

			{Name: eventAutoDKGValidateConfirmationCommitsInternal, SrcState: []fsm.State{StateDkgCommitsAwaitConfirmations}, DstState: StateDkgCommitsAwaitConfirmations, IsInternal: true, IsAuto: true},

			// Confirmed
			{Name: eventDKGCommitsConfirmedInternal, SrcState: []fsm.State{StateDkgCommitsAwaitConfirmations}, DstState: StateDkgDealsAwaitConfirmations, IsInternal: true},

			// Deals
			{Name: EventDKGDealConfirmationReceived, SrcState: []fsm.State{StateDkgDealsAwaitConfirmations}, DstState: StateDkgDealsAwaitConfirmations},
			// Canceled
			{Name: EventDKGDealConfirmationError, SrcState: []fsm.State{StateDkgDealsAwaitConfirmations}, DstState: StateDkgDealsAwaitCanceledByError},
			{Name: eventDKGDealsConfirmationCancelByTimeoutInternal, SrcState: []fsm.State{StateDkgDealsAwaitConfirmations}, DstState: StateDkgDealsAwaitCanceledByTimeout, IsInternal: true},
			{Name: eventAutoDKGValidateConfirmationDealsInternal, SrcState: []fsm.State{StateDkgDealsAwaitConfirmations}, DstState: StateDkgDealsAwaitConfirmations, IsInternal: true, IsAuto: true},

			{Name: eventDKGDealsConfirmedInternal, SrcState: []fsm.State{StateDkgDealsAwaitConfirmations}, DstState: StateDkgResponsesAwaitConfirmations, IsInternal: true},

			// Responses
			{Name: EventDKGResponseConfirmationReceived, SrcState: []fsm.State{StateDkgResponsesAwaitConfirmations}, DstState: StateDkgResponsesAwaitConfirmations},
			// Canceled
			{Name: EventDKGResponseConfirmationError, SrcState: []fsm.State{StateDkgResponsesAwaitConfirmations}, DstState: StateDkgResponsesAwaitCanceledByError},
			{Name: eventDKGResponseConfirmationCancelByTimeoutInternal, SrcState: []fsm.State{StateDkgResponsesAwaitConfirmations}, DstState: StateDkgResponsesAwaitCanceledByTimeout, IsInternal: true},

			{Name: eventAutoDKGValidateResponsesConfirmationInternal, SrcState: []fsm.State{StateDkgResponsesAwaitConfirmations}, DstState: StateDkgResponsesAwaitConfirmations, IsInternal: true, IsAuto: true},

			{Name: eventDKGResponsesConfirmedInternal, SrcState: []fsm.State{StateDkgResponsesAwaitConfirmations}, DstState: StateDkgMasterKeyAwaitConfirmations, IsInternal: true},

			// Master key

			{Name: EventDKGMasterKeyConfirmationReceived, SrcState: []fsm.State{StateDkgMasterKeyAwaitConfirmations}, DstState: StateDkgMasterKeyAwaitConfirmations},
			{Name: EventDKGMasterKeyConfirmationError, SrcState: []fsm.State{StateDkgMasterKeyAwaitConfirmations}, DstState: StateDkgMasterKeyAwaitCanceledByError},
			{Name: eventDKGMasterKeyConfirmationCancelByErrorInternal, SrcState: []fsm.State{StateDkgMasterKeyAwaitConfirmations}, DstState: StateDkgMasterKeyAwaitCanceledByError, IsInternal: true},
			{Name: eventDKGMasterKeyConfirmationCancelByTimeoutInternal, SrcState: []fsm.State{StateDkgMasterKeyAwaitConfirmations}, DstState: StateDkgMasterKeyAwaitCanceledByTimeout, IsInternal: true},

			{Name: eventAutoDKGValidateMasterKeyConfirmationInternal, SrcState: []fsm.State{StateDkgMasterKeyAwaitConfirmations}, DstState: StateDkgMasterKeyAwaitConfirmations, IsInternal: true, IsAuto: true},

			// Done
			{Name: eventDKGMasterKeyConfirmedInternal, SrcState: []fsm.State{StateDkgMasterKeyAwaitConfirmations}, DstState: StateDkgMasterKeyCollected, IsInternal: true},
		},
		fsm.Callbacks{
			EventDKGInitProcess: machine.actionInitDKGProposal,

			EventDKGCommitConfirmationReceived:              machine.actionCommitConfirmationReceived,
			EventDKGCommitConfirmationError:                 machine.actionConfirmationError,
			eventAutoDKGValidateConfirmationCommitsInternal: machine.actionValidateDkgProposalAwaitCommits,

			EventDKGDealConfirmationReceived:              machine.actionDealConfirmationReceived,
			EventDKGDealConfirmationError:                 machine.actionConfirmationError,
			eventAutoDKGValidateConfirmationDealsInternal: machine.actionValidateDkgProposalAwaitDeals,

			EventDKGResponseConfirmationReceived:              machine.actionResponseConfirmationReceived,
			EventDKGResponseConfirmationError:                 machine.actionConfirmationError,
			eventAutoDKGValidateResponsesConfirmationInternal: machine.actionValidateDkgProposalAwaitResponses,

			EventDKGMasterKeyConfirmationReceived:             machine.actionMasterKeyConfirmationReceived,
			EventDKGMasterKeyConfirmationError:                machine.actionConfirmationError,
			eventAutoDKGValidateMasterKeyConfirmationInternal: machine.actionValidateDkgProposalAwaitMasterKey,
		},
	)
	return machine
}

func (m *DKGProposalFSM) WithSetup(state fsm.State, payload *internal.DumpedMachineStatePayload) internal.DumpedMachineProvider {
	m.payloadMu.Lock()
	defer m.payloadMu.Unlock()

	m.payload = payload
	m.FSM = m.FSM.MustCopyWithState(state)
	return m
}
