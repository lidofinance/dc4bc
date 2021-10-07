package signing_proposal_fsm

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/lidofinance/dc4bc/fsm/config"
	"github.com/lidofinance/dc4bc/fsm/fsm"
	"github.com/lidofinance/dc4bc/fsm/state_machines/internal"
	"github.com/lidofinance/dc4bc/fsm/types/requests"
	"github.com/lidofinance/dc4bc/fsm/types/responses"
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
		err = errors.New("{arg0} required {SigningBatchProposalStartRequest}")
		return
	}

	request, ok := args[0].(requests.SigningBatchProposalStartRequest)

	if !ok {
		err = errors.New("cannot cast {arg0} to type {SigningBatchProposalStartRequest}")
		return
	}

	if err = request.Validate(); err != nil {
		return
	}

	payload, err := json.Marshal(request.MessagesToSign)
	if err != nil {
		err = fmt.Errorf("failed to marshal messages to sign: %w", err)
		return
	}

	m.payload.SigningProposalPayload.BatchID = request.BatchID

	m.payload.SigningProposalPayload.InitiatorId = request.ParticipantId
	m.payload.SigningProposalPayload.SrcPayload = payload

	m.payload.SigningProposalPayload.Quorum = make(internal.SigningProposalQuorum)

	// Initialize new quorum
	for _, dkgEntry := range m.payload.DKGProposalPayload.Quorum.GetOrderedParticipants() {
		m.payload.SigningProposalPayload.Quorum[dkgEntry.ParticipantID] = &internal.SigningProposalParticipant{
			Username:  dkgEntry.Username,
			Status:    internal.SigningAwaitConfirmation,
			UpdatedAt: request.CreatedAt,
		}
	}

	m.payload.SigningProposalPayload.Quorum[request.ParticipantId].Status = internal.SigningConfirmed
	m.payload.SigningProposalPayload.CreatedAt = request.CreatedAt

	// Make response
	responseData := responses.SigningProposalParticipantInvitationsResponse{
		BatchID:      m.payload.SigningProposalPayload.BatchID,
		InitiatorId:  m.payload.SigningProposalPayload.InitiatorId,
		SrcPayload:   m.payload.SigningProposalPayload.SrcPayload,
		Participants: make([]*responses.SigningProposalParticipantInvitationEntry, 0),
	}

	for _, participant := range m.payload.SigningProposalPayload.Quorum.GetOrderedParticipants() {
		responseEntry := &responses.SigningProposalParticipantInvitationEntry{
			ParticipantId: participant.ParticipantID,
			Username:      participant.Username,
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
		err = fmt.Errorf("cannot confirm participant with {Status} = {\"%s\"}", signingProposalParticipant.Status)
		return
	}

	switch inEvent {
	case EventConfirmSigningConfirmation:
		signingProposalParticipant.Status = internal.SigningConfirmed
	case EventDeclineSigningConfirmation:
		signingProposalParticipant.Status = internal.SigningDeclined
	default:
		err = fmt.Errorf("unsupported event for action {inEvent} = {\"%s\"}", inEvent)
		return
	}

	signingProposalParticipant.UpdatedAt = request.CreatedAt
	m.payload.SigningProposalPayload.UpdatedAt = request.CreatedAt

	m.payload.SigningQuorumUpdate(request.ParticipantId, signingProposalParticipant)

	return
}

func (m *SigningProposalFSM) actionValidateSigningProposalConfirmations(inEvent fsm.Event, args ...interface{}) (outEvent fsm.Event, response interface{}, err error) {
	m.payloadMu.Lock()
	defer m.payloadMu.Unlock()

	if m.payload.SigningProposalPayload.IsExpired() {
		outEvent = eventSetSigningConfirmCanceledByTimeoutInternal
		return
	}

	declinesCount := 0
	unconfirmedParticipants := m.payload.SigningQuorumCount()
	for _, participant := range m.payload.SigningProposalPayload.Quorum {
		if participant.Status == internal.SigningDeclined {
			declinesCount++
		} else if participant.Status == internal.SigningConfirmed {
			unconfirmedParticipants--
		}
	}

	if declinesCount > m.payload.SigningQuorumCount()-m.payload.GetThreshold() {
		outEvent = eventSetSigningConfirmCanceledByParticipantInternal
		return
	}

	// The are no declined and timed out participants, check for all confirmations
	if unconfirmedParticipants > m.payload.SigningQuorumCount()-m.payload.GetThreshold() {
		return
	}

	outEvent = eventSetProposalValidatedInternal

	for _, participant := range m.payload.SigningProposalPayload.Quorum {
		participant.Status = internal.SigningAwaitPartialSigns
	}

	// Make response
	responseData := responses.SigningPartialSignsParticipantInvitationsResponse{
		BatchID:     m.payload.SigningProposalPayload.BatchID,
		InitiatorId: m.payload.SigningProposalPayload.InitiatorId,
		SrcPayload:  m.payload.SigningProposalPayload.SrcPayload,
	}

	response = responseData

	return
}

func (m *SigningProposalFSM) actionPartialSignConfirmationReceived(inEvent fsm.Event, args ...interface{}) (outEvent fsm.Event, response interface{}, err error) {
	m.payloadMu.Lock()
	defer m.payloadMu.Unlock()

	if len(args) != 1 {
		err = errors.New("{arg0} required {SigningProposalPartialSignRequest}")
		return
	}

	request, ok := args[0].(requests.SigningProposalBatchPartialSignRequests)

	if !ok {
		err = errors.New("cannot cast {arg0} to type {SigningProposalBatchPartialSignRequests}")
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

	if signingProposalParticipant.Status != internal.SigningAwaitPartialSigns {
		err = fmt.Errorf("cannot confirm response with {Status} = {\"%s\"}", signingProposalParticipant.Status)
		return
	}
	for _, partialSign := range request.PartialSigns {
		signingProposalParticipant.PartialSigns[partialSign.MessageID] = make([]byte, len(partialSign.Sign))
		copy(signingProposalParticipant.PartialSigns[partialSign.MessageID], partialSign.Sign)
	}

	signingProposalParticipant.Status = internal.SigningPartialSignsConfirmed

	signingProposalParticipant.UpdatedAt = request.CreatedAt
	m.payload.SignatureProposalPayload.UpdatedAt = request.CreatedAt

	m.payload.SigningQuorumUpdate(request.ParticipantId, signingProposalParticipant)

	return
}

func (m *SigningProposalFSM) actionValidateSigningPartialSignsAwaitConfirmations(inEvent fsm.Event, args ...interface{}) (outEvent fsm.Event, response interface{}, err error) {
	m.payloadMu.Lock()
	defer m.payloadMu.Unlock()

	if m.payload.SigningProposalPayload.IsExpired() {
		outEvent = eventSigningPartialSignsAwaitCancelByTimeoutInternal
		return
	}

	failedParticipantsCount := 0
	unconfirmedParticipants := m.payload.SigningQuorumCount()
	for _, participant := range m.payload.SigningProposalPayload.Quorum {
		if participant.Status == internal.SigningError {
			failedParticipantsCount++
		} else if participant.Status == internal.SigningPartialSignsConfirmed {
			unconfirmedParticipants--
		}
	}

	if failedParticipantsCount > m.payload.SigningQuorumCount()-m.payload.GetThreshold() {
		outEvent = eventSigningPartialSignsAwaitCancelByErrorInternal
		return
	}

	// The are no declined and timed out participants, check for all confirmations
	if unconfirmedParticipants > m.payload.SigningQuorumCount()-m.payload.GetThreshold() {
		return
	}

	outEvent = eventSigningPartialSignsConfirmedInternal

	for _, participant := range m.payload.SigningProposalPayload.Quorum {
		participant.Status = internal.SigningProcess
	}

	// Response
	responseData := responses.SigningProcessParticipantResponse{
		BatchID:      m.payload.SigningProposalPayload.BatchID,
		SrcPayload:   m.payload.SigningProposalPayload.SrcPayload,
		Participants: make([]*responses.SigningProcessParticipantEntry, 0),
	}

	for _, participant := range m.payload.SigningProposalPayload.Quorum.GetOrderedParticipants() {
		// don't return participants who didn't broadcast partial signature
		if len(participant.PartialSigns) == 0 {
			continue
		}
		responseEntry := &responses.SigningProcessParticipantEntry{
			ParticipantId: participant.ParticipantID,
			Username:      participant.Username,
			PartialSigns:  participant.PartialSigns,
		}
		responseData.Participants = append(responseData.Participants, responseEntry)
	}

	response = responseData

	return
}

func (m *SigningProposalFSM) actionSigningRestart(inEvent fsm.Event, args ...interface{}) (outEvent fsm.Event, response interface{}, err error) {

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
	case EventSigningPartialSignError:
		switch signingProposalParticipant.Status {
		case internal.SigningAwaitPartialSigns:
			signingProposalParticipant.Status = internal.SigningError
		case internal.SigningPartialSignsConfirmed:
			err = errors.New("{Status} already confirmed")
		case internal.SigningError:
			err = fmt.Errorf("{Status} already has {\"%s\"}", internal.SigningError)
		default:
			err = fmt.Errorf(
				"{Status} now is \"%s\" and cannot set to {\"%s\"}",
				signingProposalParticipant.Status,
				internal.SigningError,
			)
		}
	default:
		err = fmt.Errorf("{%s} event cannot be used for action {actionConfirmationError}", inEvent)
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
