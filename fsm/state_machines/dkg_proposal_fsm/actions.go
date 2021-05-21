package dkg_proposal_fsm

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/lidofinance/dc4bc/fsm/config"
	"github.com/lidofinance/dc4bc/fsm/fsm"
	"github.com/lidofinance/dc4bc/fsm/state_machines/internal"
	"github.com/lidofinance/dc4bc/fsm/types/requests"
	"github.com/lidofinance/dc4bc/fsm/types/responses"
)

// Init

func (m *DKGProposalFSM) actionInitDKGProposal(inEvent fsm.Event, args ...interface{}) (outEvent fsm.Event, response interface{}, err error) {
	m.payloadMu.Lock()
	defer m.payloadMu.Unlock()

	if m.payload.DKGProposalPayload != nil {
		return
	}

	if len(args) != 1 {
		err = errors.New("{arg0} required {DefaultRequest}")
		return
	}

	request, ok := args[0].(requests.DefaultRequest)

	if !ok {
		err = errors.New("cannot cast {arg0} to type {DefaultRequest}")
		return
	}

	m.payload.DKGProposalPayload = &internal.DKGConfirmation{
		Quorum:    make(internal.DKGProposalQuorum),
		CreatedAt: request.CreatedAt,
		ExpiresAt: request.CreatedAt.Add(config.DkgConfirmationDeadline),
	}

	for participantId, participant := range m.payload.SignatureProposalPayload.Quorum {
		m.payload.DKGProposalPayload.Quorum[participantId] = &internal.DKGProposalParticipant{
			Username:  participant.Username,
			DkgPubKey: make([]byte, len(participant.DkgPubKey)),
			Status:    internal.CommitAwaitConfirmation,
			UpdatedAt: participant.UpdatedAt,
		}
		copy(m.payload.DKGProposalPayload.Quorum[participantId].DkgPubKey, participant.DkgPubKey)
	}

	// Remove m.payload.SignatureProposalPayload?

	// Make response

	responseData := make(responses.DKGProposalPubKeysParticipantResponse, 0)

	for _, participant := range m.payload.DKGProposalPayload.Quorum.GetOrderedParticipants() {
		responseEntry := &responses.DKGProposalPubKeysParticipantEntry{
			ParticipantId: participant.ParticipantID,
			Username:      participant.Username,
			DkgPubKey:     participant.DkgPubKey,
		}
		responseData = append(responseData, responseEntry)
	}

	return inEvent, responseData, nil
}

// Commits
func (m *DKGProposalFSM) actionCommitConfirmationReceived(inEvent fsm.Event, args ...interface{}) (outEvent fsm.Event, response interface{}, err error) {
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

	if !m.payload.DKGQuorumExists(request.ParticipantId) {
		err = errors.New("{ParticipantId} not exist in quorum")
		return
	}

	dkgProposalParticipant := m.payload.DKGQuorumGet(request.ParticipantId)

	if dkgProposalParticipant.Status != internal.CommitAwaitConfirmation {
		err = fmt.Errorf("cannot confirm commit with {Status} = {\"%s\"}", dkgProposalParticipant.Status)
		return
	}

	dkgProposalParticipant.DkgCommit = make([]byte, len(request.Commit))
	copy(dkgProposalParticipant.DkgCommit, request.Commit)
	dkgProposalParticipant.Status = internal.CommitConfirmed

	dkgProposalParticipant.UpdatedAt = request.CreatedAt
	m.payload.DKGProposalPayload.UpdatedAt = request.CreatedAt

	m.payload.DKGQuorumUpdate(request.ParticipantId, dkgProposalParticipant)

	return
}

func (m *DKGProposalFSM) actionValidateDkgProposalAwaitCommits(inEvent fsm.Event, args ...interface{}) (outEvent fsm.Event, response interface{}, err error) {
	var (
		isContainsError bool
	)

	m.payloadMu.Lock()
	defer m.payloadMu.Unlock()

	if m.payload.DKGProposalPayload.IsExpired() {
		outEvent = eventDKGCommitsConfirmationCancelByTimeoutInternal
		return
	}

	unconfirmedParticipants := m.payload.DKGQuorumCount()
	for _, participant := range m.payload.DKGProposalPayload.Quorum {
		if participant.Status == internal.CommitConfirmationError {
			isContainsError = true
		} else if participant.Status == internal.CommitConfirmed {
			unconfirmedParticipants--
		}
	}

	if isContainsError {
		outEvent = eventDKGCommitsConfirmationCancelByErrorInternal
		return
	}

	// The are no declined and timed out participants, check for all confirmations
	if unconfirmedParticipants > 0 {
		return
	}

	outEvent = eventDKGCommitsConfirmedInternal

	for _, participant := range m.payload.DKGProposalPayload.Quorum {
		participant.Status = internal.DealAwaitConfirmation
	}

	// Make response

	responseData := make(responses.DKGProposalCommitParticipantResponse, 0)

	for _, participant := range m.payload.DKGProposalPayload.Quorum.GetOrderedParticipants() {
		responseEntry := &responses.DKGProposalCommitParticipantEntry{
			ParticipantId: participant.ParticipantID,
			Username:      participant.Username,
			DkgCommit:     participant.DkgCommit,
		}
		responseData = append(responseData, responseEntry)
	}

	response = responseData

	return
}

// Deals

func (m *DKGProposalFSM) actionDealConfirmationReceived(inEvent fsm.Event, args ...interface{}) (outEvent fsm.Event, response interface{}, err error) {
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

	if !m.payload.DKGQuorumExists(request.ParticipantId) {
		err = errors.New("{ParticipantId} not exist in quorum")
		return
	}

	dkgProposalParticipant := m.payload.DKGQuorumGet(request.ParticipantId)

	if dkgProposalParticipant.Status != internal.DealAwaitConfirmation {
		err = fmt.Errorf("cannot confirm deal with {Status} = {\"%s\"}", dkgProposalParticipant.Status)
		return
	}

	dkgProposalParticipant.DkgDeal = make([]byte, len(request.Deal))
	copy(dkgProposalParticipant.DkgDeal, request.Deal)
	dkgProposalParticipant.Status = internal.DealConfirmed

	dkgProposalParticipant.UpdatedAt = request.CreatedAt
	m.payload.DKGProposalPayload.UpdatedAt = request.CreatedAt

	m.payload.DKGQuorumUpdate(request.ParticipantId, dkgProposalParticipant)

	return
}

func (m *DKGProposalFSM) actionValidateDkgProposalAwaitDeals(inEvent fsm.Event, args ...interface{}) (outEvent fsm.Event, response interface{}, err error) {
	var (
		isContainsError bool
	)

	m.payloadMu.Lock()
	defer m.payloadMu.Unlock()

	if m.payload.DKGProposalPayload.IsExpired() {
		outEvent = eventDKGDealsConfirmationCancelByTimeoutInternal
		return
	}

	unconfirmedDealsParticipants := m.payload.DKGQuorumCount()
	for _, participant := range m.payload.DKGProposalPayload.Quorum {
		if participant.Status == internal.DealConfirmationError {
			isContainsError = true
		} else if participant.Status == internal.DealConfirmed {
			unconfirmedDealsParticipants--
		}
	}

	if isContainsError {
		outEvent = eventDKGDealsConfirmationCancelByErrorInternal
		return
	}

	if unconfirmedDealsParticipants > 0 {
		return
	}

	outEvent = eventDKGDealsConfirmedInternal

	for _, participant := range m.payload.DKGProposalPayload.Quorum {
		participant.Status = internal.ResponseAwaitConfirmation
	}

	// Make response

	responseData := make(responses.DKGProposalDealParticipantResponse, 0)

	for _, participant := range m.payload.DKGProposalPayload.Quorum.GetOrderedParticipants() {
		if len(participant.DkgDeal) == 0 {
			continue
		}
		responseEntry := &responses.DKGProposalDealParticipantEntry{
			ParticipantId: participant.ParticipantID,
			Username:      participant.Username,
			DkgDeal:       participant.DkgDeal,
		}
		responseData = append(responseData, responseEntry)
	}

	response = responseData

	return
}

// Responses

func (m *DKGProposalFSM) actionResponseConfirmationReceived(inEvent fsm.Event, args ...interface{}) (outEvent fsm.Event, response interface{}, err error) {
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

	if !m.payload.DKGQuorumExists(request.ParticipantId) {
		err = errors.New("{ParticipantId} not exist in quorum")
		return
	}

	dkgProposalParticipant := m.payload.DKGQuorumGet(request.ParticipantId)

	if dkgProposalParticipant.Status != internal.ResponseAwaitConfirmation {
		err = fmt.Errorf("cannot confirm response with {Status} = {\"%s\"}", dkgProposalParticipant.Status)
		return
	}

	dkgProposalParticipant.DkgResponse = make([]byte, len(request.Response))
	copy(dkgProposalParticipant.DkgResponse, request.Response)
	dkgProposalParticipant.Status = internal.ResponseConfirmed

	dkgProposalParticipant.UpdatedAt = request.CreatedAt
	m.payload.DKGProposalPayload.UpdatedAt = request.CreatedAt

	m.payload.DKGQuorumUpdate(request.ParticipantId, dkgProposalParticipant)

	return
}

func (m *DKGProposalFSM) actionValidateDkgProposalAwaitResponses(inEvent fsm.Event, args ...interface{}) (outEvent fsm.Event, response interface{}, err error) {
	var (
		isContainsError bool
	)

	m.payloadMu.Lock()
	defer m.payloadMu.Unlock()

	if m.payload.DKGProposalPayload.IsExpired() {
		outEvent = eventDKGResponseConfirmationCancelByTimeoutInternal
		return
	}

	unconfirmedParticipants := m.payload.DKGQuorumCount()
	for _, participant := range m.payload.DKGProposalPayload.Quorum {
		if participant.Status == internal.ResponseConfirmationError {
			isContainsError = true
		} else if participant.Status == internal.ResponseConfirmed {
			unconfirmedParticipants--
		}
	}

	if isContainsError {
		outEvent = eventDKGResponseConfirmationCancelByErrorInternal
		return
	}

	// The are no declined and timed out participants, check for all confirmations
	if unconfirmedParticipants > 0 {
		return
	}

	outEvent = eventDKGResponsesConfirmedInternal

	for _, participant := range m.payload.DKGProposalPayload.Quorum {
		participant.Status = internal.MasterKeyAwaitConfirmation
	}

	// Make response

	responseData := make(responses.DKGProposalResponseParticipantResponse, 0)

	for _, participant := range m.payload.DKGProposalPayload.Quorum.GetOrderedParticipants() {
		responseEntry := &responses.DKGProposalResponseParticipantEntry{
			ParticipantId: participant.ParticipantID,
			Username:      participant.Username,
			DkgResponse:   participant.DkgResponse,
		}
		responseData = append(responseData, responseEntry)
	}

	response = responseData

	return
}

// Master key

func (m *DKGProposalFSM) actionMasterKeyConfirmationReceived(inEvent fsm.Event, args ...interface{}) (outEvent fsm.Event, response interface{}, err error) {
	m.payloadMu.Lock()
	defer m.payloadMu.Unlock()

	if len(args) != 1 {
		err = errors.New("{arg0} required {DKGProposalMasterKeyConfirmationRequest}")
		return
	}

	request, ok := args[0].(requests.DKGProposalMasterKeyConfirmationRequest)

	if !ok {
		err = errors.New("cannot cast {arg0} to type {DKGProposalMasterKeyConfirmationRequest}")
		return
	}

	if err = request.Validate(); err != nil {
		return
	}

	if !m.payload.DKGQuorumExists(request.ParticipantId) {
		err = errors.New("{ParticipantId} not exist in quorum")
		return
	}

	dkgProposalParticipant := m.payload.DKGQuorumGet(request.ParticipantId)

	if dkgProposalParticipant.Status != internal.MasterKeyAwaitConfirmation {
		err = fmt.Errorf("cannot confirm response with {Status} = {\"%s\"}", dkgProposalParticipant.Status)
		return
	}

	dkgProposalParticipant.DkgMasterKey = make([]byte, len(request.MasterKey))
	copy(dkgProposalParticipant.DkgMasterKey, request.MasterKey)
	dkgProposalParticipant.Status = internal.MasterKeyConfirmed

	dkgProposalParticipant.UpdatedAt = request.CreatedAt
	m.payload.DKGProposalPayload.UpdatedAt = request.CreatedAt

	m.payload.DKGQuorumUpdate(request.ParticipantId, dkgProposalParticipant)

	return
}

func (m *DKGProposalFSM) actionValidateDkgProposalAwaitMasterKey(inEvent fsm.Event, args ...interface{}) (outEvent fsm.Event, response interface{}, err error) {
	var (
		isContainsError bool
		masterKeys      [][]byte
	)

	m.payloadMu.Lock()
	defer m.payloadMu.Unlock()

	if m.payload.DKGProposalPayload.IsExpired() {
		outEvent = eventDKGMasterKeyConfirmationCancelByTimeoutInternal
		return
	}

	unconfirmedParticipants := m.payload.DKGQuorumCount()

	for _, participant := range m.payload.DKGProposalPayload.Quorum {
		if participant.Status == internal.MasterKeyConfirmationError {
			isContainsError = true
		} else if participant.Status == internal.MasterKeyConfirmed {
			masterKeys = append(masterKeys, participant.DkgMasterKey)
			unconfirmedParticipants--
		}
	}

	if isContainsError {
		outEvent = eventDKGMasterKeyConfirmationCancelByErrorInternal
		return
	}

	// Temporary simplest match master keys
	if len(masterKeys) > 1 {
		for _, masterKey := range masterKeys {
			if !reflect.DeepEqual(masterKey, masterKeys[0]) {
				for _, participant := range m.payload.DKGProposalPayload.Quorum {
					participant.Status = internal.MasterKeyConfirmationError
					participant.Error = requests.NewFSMError(errors.New("master key is mismatched"))
				}

				outEvent = eventDKGMasterKeyConfirmationCancelByErrorInternal
				return
			}
		}
	}

	// The are no declined and timed out participants, check for all confirmations
	if unconfirmedParticipants > 0 {
		return
	}

	outEvent = eventDKGMasterKeyConfirmedInternal

	for _, participant := range m.payload.DKGProposalPayload.Quorum {
		participant.Status = internal.MasterKeyConfirmed
	}

	return
}

// Errors
func (m *DKGProposalFSM) actionConfirmationError(inEvent fsm.Event, args ...interface{}) (outEvent fsm.Event, response interface{}, err error) {
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

	if !m.payload.DKGQuorumExists(request.ParticipantId) {
		err = errors.New("{ParticipantId} not exist in quorum")
		return
	}

	dkgProposalParticipant := m.payload.DKGQuorumGet(request.ParticipantId)

	// TODO: Move to methods
	switch inEvent {
	case EventDKGCommitConfirmationError:
		switch dkgProposalParticipant.Status {
		case internal.CommitAwaitConfirmation:
			dkgProposalParticipant.Status = internal.CommitConfirmationError
		case internal.CommitConfirmed:
			err = errors.New("{Status} already confirmed")
		case internal.CommitConfirmationError:
			err = fmt.Errorf("{Status} already has {\"%s\"}", internal.CommitConfirmationError)
		default:
			err = fmt.Errorf(
				"{Status} now is \"%s\" and cannot set to {\"%s\"}",
				dkgProposalParticipant.Status,
				internal.CommitConfirmationError,
			)
		}
	case EventDKGDealConfirmationError:
		switch dkgProposalParticipant.Status {
		case internal.DealAwaitConfirmation:
			dkgProposalParticipant.Status = internal.DealConfirmationError
		case internal.DealConfirmed:
			err = errors.New("{Status} already confirmed")
		case internal.DealConfirmationError:
			err = fmt.Errorf("{Status} already has {\"%s\"}", internal.DealConfirmationError)
		default:
			err = fmt.Errorf(
				"{Status} now is \"%s\" and cannot set to {\"%s\"}",
				dkgProposalParticipant.Status,
				internal.DealConfirmationError,
			)
		}
	case EventDKGResponseConfirmationError:
		switch dkgProposalParticipant.Status {
		case internal.ResponseAwaitConfirmation:
			dkgProposalParticipant.Status = internal.ResponseConfirmationError
		case internal.ResponseConfirmed:
			err = errors.New("{Status} already confirmed")
		case internal.ResponseConfirmationError:
			err = fmt.Errorf("{Status} already has {\"%s\"}", internal.ResponseConfirmationError)
		default:
			err = fmt.Errorf(
				"{Status} now is \"%s\" and cannot set to {\"%s\"}",
				dkgProposalParticipant.Status,
				internal.ResponseConfirmationError,
			)
		}
	case EventDKGMasterKeyConfirmationError:
		switch dkgProposalParticipant.Status {
		case internal.MasterKeyAwaitConfirmation:
			dkgProposalParticipant.Status = internal.MasterKeyConfirmationError
		case internal.MasterKeyConfirmed:
			err = errors.New("{Status} already confirmed")
		case internal.MasterKeyConfirmationError:
			err = fmt.Errorf("{Status} already has {\"%s\"}", internal.MasterKeyConfirmationError)
		default:
			err = fmt.Errorf(
				"{Status} now is \"%s\" and cannot set to {\"%s\"}",
				dkgProposalParticipant.Status,
				internal.MasterKeyConfirmationError,
			)
		}
	default:
		err = fmt.Errorf("{%s} event cannot be used for action {actionConfirmationError}", inEvent)
	}

	if err != nil {
		return
	}

	dkgProposalParticipant.Error = request.Error

	dkgProposalParticipant.UpdatedAt = request.CreatedAt
	m.payload.DKGProposalPayload.UpdatedAt = request.CreatedAt

	m.payload.DKGQuorumUpdate(request.ParticipantId, dkgProposalParticipant)

	// TODO: Add outEvent

	return
}
