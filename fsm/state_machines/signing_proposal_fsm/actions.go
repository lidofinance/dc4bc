package signing_proposal_fsm

import (
	"errors"
	"fmt"
	"github.com/depools/dc4bc/fsm/config"
	"github.com/depools/dc4bc/fsm/fsm"
	"github.com/depools/dc4bc/fsm/state_machines/internal"
	"github.com/depools/dc4bc/fsm/types/requests"
	"time"
)

func (m *SigningProposalFSM) actionInitSigningProposal(inEvent fsm.Event, args ...interface{}) (outEvent fsm.Event, response interface{}, err error) {
	m.payloadMu.Lock()
	defer m.payloadMu.Unlock()

	if len(args) != 1 {
		err = errors.New("{arg0} required {DefaultRequest}")
		return
	}

	request, ok := args[0].(requests.DefaultRequest)

	if !ok {
		err = errors.New("cannot cast {arg0} to type {DefaultRequest}")
		return
	}

	if err = request.Validate(); err != nil {
		return
	}

	m.payload.SigningProposalPayload = &internal.SigningConfirmation{
		Quorum:    make(internal.SigningProposalQuorum),
		CreatedAt: request.CreatedAt,
		ExpiresAt: request.CreatedAt.Add(config.SigningConfirmationDeadline),
	}

	for participantId, participant := range m.payload.SignatureProposalPayload.Quorum {
		m.payload.SigningProposalPayload.Quorum[participantId] = &internal.SigningProposalParticipant{
			Addr:      participant.Addr,
			Status:    internal.SigningIdle,
			UpdatedAt: participant.UpdatedAt,
		}
	}

	return
}

func (m *SigningProposalFSM) actionStartSigningProposal(inEvent fsm.Event, args ...interface{}) (outEvent fsm.Event, response interface{}, err error) {
	m.payloadMu.Lock()
	defer m.payloadMu.Unlock()

	if len(args) != 1 {
		err = errors.New("{arg0} required {SigningProposalStartRequest}")
		return
	}

	request, ok := args[0].(requests.SigningProposalStartRequest)

	if !ok {
		err = errors.New("cannot cast {arg0} to type {SigningProposalStartRequest}")
		return
	}

	if err = request.Validate(); err != nil {
		return
	}

	m.payload.SigningProposalPayload.CreatedAt = request.CreatedAt

	return
}

func (m *SigningProposalFSM) actionProposalResponseByParticipant(inEvent fsm.Event, args ...interface{}) (outEvent fsm.Event, response interface{}, err error) {
	m.payloadMu.Lock()
	defer m.payloadMu.Unlock()

	if len(args) != 1 {
		err = errors.New("{arg0} required {SigningProposalParticipantRequest}")
		return
	}

	request, ok := args[0].(requests.SigningProposalParticipantRequest)

	if !ok {
		err = errors.New("cannot cast {arg0} to type {SigningProposalParticipantRequest}")
		return
	}

	if err = request.Validate(); err != nil {
		return
	}

	if !m.payload.SigningQuorumExists(request.ParticipantId) {
		err = errors.New("{ParticipantId} not exist in quorum")
		return
	}

	signingProposalParticipant := m.payload.SigningQuorumGet(request.ParticipantId)

	if signingProposalParticipant.Status != internal.SigningAwaitConfirmation {
		err = errors.New(fmt.Sprintf("cannot confirm commit with {Status} = {\"%s\"}", signingProposalParticipant.Status))
		return
	}

	// copy(signingProposalParticipant.Commit, request.Commit)
	signingProposalParticipant.UpdatedAt = request.CreatedAt
	signingProposalParticipant.Status = internal.SigningConfirmed

	m.payload.SigningQuorumUpdate(request.ParticipantId, signingProposalParticipant)

	return
}

func (m *SigningProposalFSM) actionValidateSigningProposalConfirmations(inEvent fsm.Event, args ...interface{}) (outEvent fsm.Event, response interface{}, err error) {
	var (
		isContainsError, isContainsExpired bool
	)

	m.payloadMu.Lock()
	defer m.payloadMu.Unlock()

	tm := time.Now()

	unconfirmedParticipants := m.payload.SigningQuorumCount()
	for _, participant := range m.payload.SigningProposalPayload.Quorum {
		if participant.Status == internal.SigningAwaitConfirmation {
			if participant.UpdatedAt.Add(config.SigningConfirmationDeadline).Before(tm) {
				isContainsExpired = true
			}
		} else {
			if participant.Status == internal.SigningDeclined {
				isContainsError = true
			} else if participant.Status == internal.SigningConfirmed {
				unconfirmedParticipants--
			}
		}
	}

	if isContainsError {
		outEvent = eventSetSigningConfirmCanceledByTimeoutInternal
		return
	}

	if isContainsExpired {
		outEvent = eventSetSigningConfirmCanceledByParticipantInternal
		return
	}

	// The are no declined and timed out participants, check for all confirmations
	if unconfirmedParticipants > 0 {
		return
	}

	outEvent = eventSetProposalValidatedInternal

	for _, participant := range m.payload.SigningProposalPayload.Quorum {
		participant.Status = internal.SigningAwaitPartialKeys
	}

	return
}
