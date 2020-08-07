package dkg_proposal_fsm

import (
	"errors"
	"fmt"
	"github.com/depools/dc4bc/fsm/fsm"
	"github.com/depools/dc4bc/fsm/state_machines/internal"
	"github.com/depools/dc4bc/fsm/types/requests"
)

// Pub keys

func (m *DKGProposalFSM) actionPubKeyPrepareConfirmations(event fsm.Event, args ...interface{}) (response interface{}, err error) {
	return
}

func (m *DKGProposalFSM) actionPubKeyConfirmationReceived(event fsm.Event, args ...interface{}) (response interface{}, err error) {
	m.payloadMu.Lock()
	defer m.payloadMu.Unlock()

	if len(args) != 1 {
		err = errors.New("{arg0} required {DKGProposalPubKeyConfirmationRequest}")
		return
	}

	request, ok := args[0].(requests.DKGProposalPubKeyConfirmationRequest)

	if !ok {
		err = errors.New("cannot cast {arg0} to type {DKGProposalPubKeyConfirmationRequest}")
		return
	}

	if err = request.Validate(); err != nil {
		return
	}

	dkgProposalParticipant, ok := m.payload.DKGProposalPayload[request.ParticipantId]

	if !ok {
		err = errors.New("{ParticipantId} not exist in quorum")
		return
	}

	dkgProposalParticipant.PublicKey = request.PubKey
	dkgProposalParticipant.UpdatedAt = request.CreatedAt
	dkgProposalParticipant.Status = internal.PubKeyConfirmed

	m.payload.DKGProposalPayload[request.ParticipantId] = dkgProposalParticipant

	return
}

// Commits

func (m *DKGProposalFSM) actionCommitConfirmationReceived(event fsm.Event, args ...interface{}) (response interface{}, err error) {
	m.payloadMu.Lock()
	defer m.payloadMu.Unlock()

	if len(args) != 1 {
		err = errors.New("{arg0} required {DKGProposalCommitConfirmationRequest}")
		return
	}

	request, ok := args[0].(requests.DKGProposalCommitConfirmationRequest)

	if !ok {
		err = errors.New("cannot cast {arg0} to type {DKGProposalCommitConfirmationRequest}")
		return
	}

	if err = request.Validate(); err != nil {
		return
	}

	dkgProposalParticipant, ok := m.payload.DKGProposalPayload[request.ParticipantId]

	if !ok {
		err = errors.New("{ParticipantId} not exist in quorum")
		return
	}

	dkgProposalParticipant.Commit = request.Commit
	dkgProposalParticipant.UpdatedAt = request.CreatedAt
	dkgProposalParticipant.Status = internal.CommitConfirmed

	m.payload.DKGProposalPayload[request.ParticipantId] = dkgProposalParticipant

	return
}

// Deals

func (m *DKGProposalFSM) actionDealConfirmationReceived(event fsm.Event, args ...interface{}) (response interface{}, err error) {
	m.payloadMu.Lock()
	defer m.payloadMu.Unlock()

	if len(args) != 1 {
		err = errors.New("{arg0} required {DKGProposalDealConfirmationRequest}")
		return
	}

	request, ok := args[0].(requests.DKGProposalDealConfirmationRequest)

	if !ok {
		err = errors.New("cannot cast {arg0} to type {DKGProposalDealConfirmationRequest}")
		return
	}

	if err = request.Validate(); err != nil {
		return
	}

	dkgProposalParticipant, ok := m.payload.DKGProposalPayload[request.ParticipantId]

	if !ok {
		err = errors.New("{ParticipantId} not exist in quorum")
		return
	}

	dkgProposalParticipant.Deal = request.Deal
	dkgProposalParticipant.UpdatedAt = request.CreatedAt
	dkgProposalParticipant.Status = internal.DealConfirmed

	m.payload.DKGProposalPayload[request.ParticipantId] = dkgProposalParticipant

	return
}

// Responses

func (m *DKGProposalFSM) actionResponseConfirmationReceived(event fsm.Event, args ...interface{}) (response interface{}, err error) {
	m.payloadMu.Lock()
	defer m.payloadMu.Unlock()

	if len(args) != 1 {
		err = errors.New("{arg0} required {DKGProposalResponseConfirmationRequest}")
		return
	}

	request, ok := args[0].(requests.DKGProposalResponseConfirmationRequest)

	if !ok {
		err = errors.New("cannot cast {arg0} to type {DKGProposalResponseConfirmationRequest}")
		return
	}

	if err = request.Validate(); err != nil {
		return
	}

	dkgProposalParticipant, ok := m.payload.DKGProposalPayload[request.ParticipantId]

	if !ok {
		err = errors.New("{ParticipantId} not exist in quorum")
		return
	}

	dkgProposalParticipant.Response = request.Response
	dkgProposalParticipant.UpdatedAt = request.CreatedAt
	dkgProposalParticipant.Status = internal.ResponseConfirmed

	m.payload.DKGProposalPayload[request.ParticipantId] = dkgProposalParticipant

	return
}

// Errors
func (m *DKGProposalFSM) actionConfirmationError(event fsm.Event, args ...interface{}) (response interface{}, err error) {
	m.payloadMu.Lock()
	defer m.payloadMu.Unlock()

	if len(args) != 1 {
		err = errors.New("{arg0} required {DKGProposalConfirmationErrorRequest}")
		return
	}

	request, ok := args[0].(requests.DKGProposalConfirmationErrorRequest)

	if !ok {
		err = errors.New("cannot cast {arg0} to type {DKGProposalConfirmationErrorRequest}")
		return
	}

	if err = request.Validate(); err != nil {
		return
	}

	dkgProposalParticipant, ok := m.payload.DKGProposalPayload[request.ParticipantId]

	if !ok {
		err = errors.New("{ParticipantId} not exist in quorum")
		return
	}

	// TODO: Move to methods
	switch event {
	case EventDKGPubKeyConfirmationError:
		switch dkgProposalParticipant.Status {
		case internal.PubKeyConAwaitConfirmation:
			dkgProposalParticipant.Status = internal.PubKeyConfirmationError
		case internal.PubKeyConfirmed:
			err = errors.New("{Status} already confirmed")
		case internal.PubKeyConfirmationError:
			err = errors.New(fmt.Sprintf("{Status} already has {\"%s\"}", internal.PubKeyConfirmationError))
		default:
			err = errors.New(fmt.Sprintf(
				"{Status} now is \"%s\" and cannot set to {\"%s\"}",
				dkgProposalParticipant.Status,
				internal.PubKeyConfirmationError,
			))
		}
	case EventDKGCommitConfirmationError:
		switch dkgProposalParticipant.Status {
		case internal.CommitAwaitConfirmation:
			dkgProposalParticipant.Status = internal.CommitConfirmationError
		case internal.CommitConfirmed:
			err = errors.New("{Status} already confirmed")
		case internal.CommitConfirmationError:
			err = errors.New(fmt.Sprintf("{Status} already has {\"%s\"}", internal.CommitConfirmationError))
		default:
			err = errors.New(fmt.Sprintf(
				"{Status} now is \"%s\" and cannot set to {\"%s\"}",
				dkgProposalParticipant.Status,
				internal.PubKeyConfirmationError,
			))
		}
	case EventDKGDealConfirmationError:
		switch dkgProposalParticipant.Status {
		case internal.DealAwaitConfirmation:
			dkgProposalParticipant.Status = internal.PubKeyConfirmationError
		case internal.DealConfirmed:
			err = errors.New("{Status} already confirmed")
		case internal.DealConfirmationError:
			err = errors.New(fmt.Sprintf("{Status} already has {\"%s\"}", internal.DealConfirmationError))
		default:
			err = errors.New(fmt.Sprintf(
				"{Status} now is \"%s\" and cannot set to {\"%s\"}",
				dkgProposalParticipant.Status,
				internal.PubKeyConfirmationError,
			))
		}
	case EventDKGResponseConfirmationError:
		switch dkgProposalParticipant.Status {
		case internal.ResponseAwaitConfirmation:
			dkgProposalParticipant.Status = internal.PubKeyConfirmationError
		case internal.ResponseConfirmed:
			err = errors.New("{Status} already confirmed")
		case internal.ResponseConfirmationError:
			err = errors.New(fmt.Sprintf("{Status} already has {\"%s\"}", internal.ResponseConfirmationError))
		default:
			err = errors.New(fmt.Sprintf(
				"{Status} now is \"%s\" and cannot set to {\"%s\"}",
				dkgProposalParticipant.Status,
				internal.PubKeyConfirmationError,
			))
		}
	default:
		err = errors.New(fmt.Sprintf("{%s} event cannot be used for action {actionConfirmationError}", event))
	}

	if err != nil {
		return
	}

	dkgProposalParticipant.UpdatedAt = request.CreatedAt

	m.payload.DKGProposalPayload[request.ParticipantId] = dkgProposalParticipant

	return
}
