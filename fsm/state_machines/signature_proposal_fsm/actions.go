package signature_proposal_fsm

import (
	"crypto/x509"
	"errors"
	"github.com/depools/dc4bc/fsm/fsm"
	"github.com/depools/dc4bc/fsm/state_machines/internal"
	"github.com/depools/dc4bc/fsm/types/requests"
	"github.com/depools/dc4bc/fsm/types/responses"
)

// init -> awaitingConfirmations
// args: payload, signing id, participants list
func (m *SignatureProposalFSM) actionInitProposal(inEvent fsm.Event, args ...interface{}) (outEvent fsm.Event, response interface{}, err error) {
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

	m.payload.ConfirmationProposalPayload = make(internal.SignatureProposalQuorum)

	for index, participant := range request.Participants {
		participantId := createFingerprint(&participant.PubKey)
		secret, err := generateRandomString(32)
		if err != nil {
			return inEvent, nil, errors.New("cannot generate source for {InvitationSecret}")
		}

		parsedPubKey, err := x509.ParsePKCS1PublicKey(participant.PubKey)

		if err != nil {
			return inEvent, nil, errors.New("cannot parse {PubKey}")
		}

		m.payload.ConfirmationProposalPayload[participantId] = internal.SignatureProposalParticipant{
			ParticipantId:    index,
			Title:            participant.Title,
			PublicKey:        parsedPubKey,
			InvitationSecret: secret,
			Status:           internal.SignatureAwaitConfirmation,
			UpdatedAt:        request.CreatedAt,
		}
	}

	// Checking fo quorum length
	if len(m.payload.ConfirmationProposalPayload) != len(request.Participants) {
		err = errors.New("error with creating {SignatureProposalQuorum}")
		return
	}

	// Make response

	responseData := make(responses.SignatureProposalParticipantInvitationsResponse, 0)

	for pubKeyFingerprint, proposal := range m.payload.ConfirmationProposalPayload {
		encryptedInvitationSecret, err := encryptWithPubKey(proposal.PublicKey, proposal.InvitationSecret)
		if err != nil {
			return inEvent, nil, errors.New("cannot encryptWithPubKey")
		}
		responseEntry := &responses.SignatureProposalParticipantInvitationEntry{
			ParticipantId:       proposal.ParticipantId,
			Title:               proposal.Title,
			PubKeyFingerprint:   pubKeyFingerprint,
			EncryptedInvitation: encryptedInvitationSecret,
		}
		responseData = append(responseData, responseEntry)
	}

	// Change state
	return inEvent, responseData, nil
}

// TODO: Add timeout checking
func (m *SignatureProposalFSM) actionProposalResponseByParticipant(inEvent fsm.Event, args ...interface{}) (outEvent fsm.Event, response interface{}, err error) {
	// m.payloadMu.Lock()
	// defer m.payloadMu.Unlock()

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

	signatureProposalParticipant, ok := m.payload.ConfirmationProposalPayload[request.PubKeyFingerprint]

	if !ok {
		err = errors.New("{PubKeyFingerprint} not exist in quorum")
		return
	}

	if signatureProposalParticipant.InvitationSecret != request.DecryptedInvitation {
		err = errors.New("{InvitationSecret} not match {DecryptedInvitation}")
		return
	}

	signatureProposalParticipant.Status = internal.SignatureConfirmed
	signatureProposalParticipant.UpdatedAt = request.CreatedAt

	m.payload.ConfirmationProposalPayload[request.PubKeyFingerprint] = signatureProposalParticipant

	outEvent, response, err = m.actionValidateProposal(eventValidateProposalInternal)

	return
}

func (m *SignatureProposalFSM) actionValidateProposal(inEvent fsm.Event, args ...interface{}) (outEvent fsm.Event, response interface{}, err error) {
	// m.payloadMu.Lock()
	// defer m.payloadMu.Unlock()

	unconfirmedParticipants := len(m.payload.ConfirmationProposalPayload)
	for _, participant := range m.payload.ConfirmationProposalPayload {
		if participant.Status == internal.SignatureConfirmed {
			unconfirmedParticipants--
		}
	}

	if unconfirmedParticipants > 0 {
		return
	}
	outEvent = eventSetProposalValidatedInternal
	return
}
