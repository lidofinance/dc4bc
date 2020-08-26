package signing_proposal_fsm

import (
	"errors"
	"fmt"
	"github.com/depools/dc4bc/fsm/config"
	"github.com/depools/dc4bc/fsm/fsm"
	"github.com/depools/dc4bc/fsm/state_machines/internal"
	"github.com/depools/dc4bc/fsm/types/requests"
	"github.com/depools/dc4bc/fsm/types/responses"
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

	m.payload.SigningProposalPayload.SigningId, err = generateSigningId()

	if err != nil {
		err = errors.New("cannot generate {SigningId}")
		return
	}

	m.payload.SigningProposalPayload.InitiatorId = request.ParticipantId
	m.payload.SigningProposalPayload.SrcPayload = request.SrcPayload

	m.payload.SigningProposalPayload.Quorum = make(internal.SigningProposalQuorum)

	// Initialize new quorum
	for id, dkgEntry := range m.payload.DKGProposalPayload.Quorum {
		m.payload.SigningProposalPayload.Quorum[id] = &internal.SigningProposalParticipant{
			Addr:      dkgEntry.Addr,
			Status:    internal.SigningAwaitConfirmation,
			UpdatedAt: request.CreatedAt,
		}
	}

	m.payload.SigningProposalPayload.Quorum[request.ParticipantId].Status = internal.SigningConfirmed
	m.payload.SigningProposalPayload.CreatedAt = request.CreatedAt

	// Make response

	responseData := responses.SigningProposalParticipantInvitationsResponse{
		SigningId:    m.payload.SigningProposalPayload.SigningId,
		InitiatorId:  m.payload.SigningProposalPayload.InitiatorId,
		SrcPayload:   m.payload.SigningProposalPayload.SrcPayload,
		Participants: make([]*responses.SigningProposalParticipantInvitationEntry, 0),
	}

	for participantId, participant := range m.payload.SigningProposalPayload.Quorum {
		responseEntry := &responses.SigningProposalParticipantInvitationEntry{
			ParticipantId: participantId,
			Addr:          participant.Addr,
			Status:        uint8(participant.Status),
		}
		responseData.Participants = append(responseData.Participants, responseEntry)
	}

	return inEvent, responseData, nil
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
		err = errors.New(fmt.Sprintf("cannot confirm participant with {Status} = {\"%s\"}", signingProposalParticipant.Status))
		return
	}

	switch inEvent {
	case EventConfirmSigningConfirmation:
		signingProposalParticipant.Status = internal.SigningConfirmed
	case EventDeclineSigningConfirmation:
		signingProposalParticipant.Status = internal.SigningDeclined
	default:
		err = errors.New(fmt.Sprintf("unsupported event for action {inEvent} = {\"%s\"}", inEvent))
		return
	}

	signingProposalParticipant.UpdatedAt = request.CreatedAt
	m.payload.SigningProposalPayload.UpdatedAt = request.CreatedAt

	m.payload.SigningQuorumUpdate(request.ParticipantId, signingProposalParticipant)

	return
}

func (m *SigningProposalFSM) actionValidateSigningProposalConfirmations(inEvent fsm.Event, args ...interface{}) (outEvent fsm.Event, response interface{}, err error) {
	var (
		isContainsDecline bool
	)

	m.payloadMu.Lock()
	defer m.payloadMu.Unlock()

	if m.payload.SigningProposalPayload.IsExpired() {
		outEvent = eventSetSigningConfirmCanceledByTimeoutInternal
		return
	}

	unconfirmedParticipants := m.payload.SigningQuorumCount()
	for _, participant := range m.payload.SigningProposalPayload.Quorum {
		if participant.Status == internal.SigningDeclined {
			isContainsDecline = true
		} else if participant.Status == internal.SigningConfirmed {
			unconfirmedParticipants--
		}
	}

	if isContainsDecline {
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

func (m *SigningProposalFSM) actionPartialKeyConfirmationReceived(inEvent fsm.Event, args ...interface{}) (outEvent fsm.Event, response interface{}, err error) {
	m.payloadMu.Lock()
	defer m.payloadMu.Unlock()

	if len(args) != 1 {
		err = errors.New("{arg0} required {SigningProposalPartialKeyRequest}")
		return
	}

	request, ok := args[0].(requests.SigningProposalPartialKeyRequest)

	if !ok {
		err = errors.New("cannot cast {arg0} to type {SigningProposalPartialKeyRequest}")
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

	if signingProposalParticipant.Status != internal.SigningAwaitPartialKeys {
		err = errors.New(fmt.Sprintf("cannot confirm response with {Status} = {\"%s\"}", signingProposalParticipant.Status))
		return
	}

	signingProposalParticipant.PartialKey = make([]byte, len(request.PartialKey))
	copy(signingProposalParticipant.PartialKey, request.PartialKey)
	signingProposalParticipant.Status = internal.SigningPartialKeysConfirmed

	signingProposalParticipant.UpdatedAt = request.CreatedAt
	m.payload.SignatureProposalPayload.UpdatedAt = request.CreatedAt

	m.payload.SigningQuorumUpdate(request.ParticipantId, signingProposalParticipant)

	return
}

func (m *SigningProposalFSM) actionValidateSigningPartialKeyAwaitConfirmations(inEvent fsm.Event, args ...interface{}) (outEvent fsm.Event, response interface{}, err error) {
	var (
		isContainsError bool
	)

	m.payloadMu.Lock()
	defer m.payloadMu.Unlock()

	if m.payload.SigningProposalPayload.IsExpired() {
		outEvent = eventSigningPartialKeyCancelByTimeoutInternal
		return
	}

	unconfirmedParticipants := m.payload.SigningQuorumCount()
	for _, participant := range m.payload.SigningProposalPayload.Quorum {
		if participant.Status == internal.SigningError {
			isContainsError = true
		} else if participant.Status == internal.SigningPartialKeysConfirmed {
			unconfirmedParticipants--
		}
	}

	if isContainsError {
		outEvent = eventSigningPartialKeyCancelByErrorInternal
		return
	}

	// The are no declined and timed out participants, check for all confirmations
	if unconfirmedParticipants > 0 {
		return
	}

	outEvent = eventSigningPartialKeysConfirmedInternal

	for _, participant := range m.payload.SigningProposalPayload.Quorum {
		participant.Status = internal.SigningProcess
	}

	return
}

// Errors
func (m *SigningProposalFSM) actionConfirmationError(inEvent fsm.Event, args ...interface{}) (outEvent fsm.Event, response interface{}, err error) {
	m.payloadMu.Lock()
	defer m.payloadMu.Unlock()

	if len(args) != 1 {
		err = errors.New("{arg0} required {SignatureProposalConfirmationErrorRequest}")
		return
	}

	request, ok := args[0].(requests.SignatureProposalConfirmationErrorRequest)

	if !ok {
		err = errors.New("cannot cast {arg0} to type {SignatureProposalConfirmationErrorRequest}")
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

	// TODO: Move to methods
	switch inEvent {
	case EventSigningPartialKeyError:
		switch signingProposalParticipant.Status {
		case internal.SigningAwaitPartialKeys:
			signingProposalParticipant.Status = internal.SigningError
		case internal.SigningPartialKeysConfirmed:
			err = errors.New("{Status} already confirmed")
		case internal.SigningError:
			err = errors.New(fmt.Sprintf("{Status} already has {\"%s\"}", internal.SigningError))
		default:
			err = errors.New(fmt.Sprintf(
				"{Status} now is \"%s\" and cannot set to {\"%s\"}",
				signingProposalParticipant.Status,
				internal.SigningError,
			))
		}
	default:
		err = errors.New(fmt.Sprintf("{%s} event cannot be used for action {actionConfirmationError}", inEvent))
	}

	if err != nil {
		return
	}

	signingProposalParticipant.Error = request.Error

	signingProposalParticipant.UpdatedAt = request.CreatedAt
	m.payload.SignatureProposalPayload.UpdatedAt = request.CreatedAt

	m.payload.SigningQuorumUpdate(request.ParticipantId, signingProposalParticipant)

	return
}
