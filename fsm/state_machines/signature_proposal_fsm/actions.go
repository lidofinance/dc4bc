package signature_proposal_fsm

import (
	"errors"
	"fmt"

	"github.com/lidofinance/dc4bc/fsm/config"
	"github.com/lidofinance/dc4bc/fsm/fsm"
	"github.com/lidofinance/dc4bc/fsm/state_machines/internal"
	"github.com/lidofinance/dc4bc/fsm/types/requests"
	"github.com/lidofinance/dc4bc/fsm/types/responses"
)

// init -> awaitingConfirmations
// args: payload, signing id, participants list
func (m *SignatureProposalFSM) actionInitSignatureProposal(inEvent fsm.Event, args ...interface{}) (outEvent fsm.Event, response interface{}, err error) {
	m.payloadMu.Lock()
	defer m.payloadMu.Unlock()

	if len(args) != 1 {
		err = errors.New("{arg0} required {SignatureProposalParticipantsListRequest}")
		return
	}

	request, ok := args[0].(requests.SignatureProposalParticipantsListRequest)

	if !ok {
		err = errors.New("cannot cast {arg0} to type {SignatureProposalParticipantsListRequest}")
		return
	}

	if err = request.Validate(); err != nil {
		return
	}

	m.payload.SignatureProposalPayload = &internal.SignatureConfirmation{
		Quorum:    make(internal.SignatureProposalQuorum),
		CreatedAt: request.CreatedAt,
		ExpiresAt: request.CreatedAt.Add(config.SignatureProposalConfirmationDeadline),
	}

	for index, participant := range request.Participants {
		m.payload.SignatureProposalPayload.Quorum[index] = &internal.SignatureProposalParticipant{
			Username:  participant.Username,
			PubKey:    participant.PubKey,
			DkgPubKey: participant.DkgPubKey,
			Status:    internal.SigConfirmationAwaitConfirmation,
			Threshold: request.SigningThreshold,
			UpdatedAt: request.CreatedAt,
		}

		m.payload.SetPubKeyUsername(participant.Username, participant.PubKey)
		m.payload.SetIDUsername(participant.Username, index)
	}

	// Checking fo quorum length
	if m.payload.SigQuorumCount() != len(request.Participants) {
		err = errors.New("error with creating {SignatureProposalQuorum}")
		return
	}
	m.payload.Threshold = request.SigningThreshold

	// Make response

	responseData := make(responses.SignatureProposalParticipantInvitationsResponse, 0)

	for participantId, participant := range m.payload.SignatureProposalPayload.Quorum {
		responseEntry := &responses.SignatureProposalParticipantInvitationEntry{
			ParticipantId: participantId,
			Username:      participant.Username,
			Threshold:     participant.Threshold,
			DkgPubKey:     participant.DkgPubKey,
			PubKey:        participant.PubKey,
		}
		responseData = append(responseData, responseEntry)
	}

	return inEvent, responseData, nil
}

// TODO: Add timeout checking
func (m *SignatureProposalFSM) actionProposalResponseByParticipant(inEvent fsm.Event, args ...interface{}) (outEvent fsm.Event, response interface{}, err error) {
	m.payloadMu.Lock()
	defer m.payloadMu.Unlock()

	if len(args) != 1 {
		err = errors.New("{arg0} required {SignatureProposalParticipantRequest}")
		return
	}

	request, ok := args[0].(requests.SignatureProposalParticipantRequest)

	if !ok {
		err = errors.New("cannot cast {arg0} to type {SignatureProposalParticipantRequest}")
		return
	}

	if err = request.Validate(); err != nil {
		return
	}

	if !m.payload.SigQuorumExists(request.ParticipantId) {
		err = errors.New("{ParticipantId} not exist in quorum")
		return
	}

	signatureProposalParticipant := m.payload.SigQuorumGet(request.ParticipantId)
	if signatureProposalParticipant.UpdatedAt.Add(config.SignatureProposalConfirmationDeadline).Before(request.CreatedAt) {
		outEvent = eventSetValidationCanceledByTimeout
		return
	}

	if signatureProposalParticipant.Status != internal.SigConfirmationAwaitConfirmation {
		err = fmt.Errorf("cannot apply reply participant with {Status} = {\"%s\"}", signatureProposalParticipant.Status)
		return
	}

	switch inEvent {
	case EventConfirmSignatureProposal:
		signatureProposalParticipant.Status = internal.SigConfirmationConfirmed
	case EventDeclineProposal:
		signatureProposalParticipant.Status = internal.SigConfirmationDeclined
	default:
		err = fmt.Errorf("unsupported event for action {inEvent} = {\"%s\"}", inEvent)
		return
	}

	signatureProposalParticipant.UpdatedAt = request.CreatedAt
	m.payload.SignatureProposalPayload.UpdatedAt = request.CreatedAt

	m.payload.SigQuorumUpdate(request.ParticipantId, signatureProposalParticipant)

	return
}

func (m *SignatureProposalFSM) actionValidateSignatureProposal(fsm.Event, ...interface{}) (outEvent fsm.Event, response interface{}, err error) {
	var (
		isContainsDecline bool
	)

	m.payloadMu.Lock()
	defer m.payloadMu.Unlock()

	if m.payload.SignatureProposalPayload.IsExpired() {
		outEvent = eventSetValidationCanceledByTimeout
		return
	}

	unconfirmedParticipants := m.payload.SigQuorumCount()

	for _, participant := range m.payload.SignatureProposalPayload.Quorum {
		if participant.Status == internal.SigConfirmationConfirmed {
			unconfirmedParticipants--
		} else if participant.Status == internal.SigConfirmationDeclined {
			isContainsDecline = true
		}
	}

	if isContainsDecline {
		outEvent = eventSetValidationCanceledByParticipant
		return
	}

	// The are no declined and timed out participants, check for all confirmations
	if unconfirmedParticipants > 0 {
		return
	}

	responseData := make(responses.SignatureProposalParticipantStatusResponse, 0)

	for participantId, participant := range m.payload.SignatureProposalPayload.Quorum {
		responseEntry := &responses.SignatureProposalParticipantStatusEntry{
			ParticipantId: participantId,
			Username:      participant.Username,
			Status:        uint8(participant.Status),
		}
		responseData = append(responseData, responseEntry)
	}

	return eventSetProposalValidatedInternal, responseData, nil
}
