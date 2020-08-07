package signature_proposal_fsm

import (
	"errors"
	"log"

	"github.com/depools/dc4bc/fsm/fsm"
	"github.com/depools/dc4bc/fsm/state_machines/internal"
	"github.com/depools/dc4bc/fsm/types/requests"
	"github.com/depools/dc4bc/fsm/types/responses"
)

// init -> awaitingConfirmations
// args: payload, signing id, participants list
func (m *SignatureProposalFSM) actionInitProposal(event fsm.Event, args ...interface{}) (response interface{}, err error) {
	m.payloadMu.Lock()
	defer m.payloadMu.Unlock()

	if len(args) != 1 {
		err = errors.New("participants list required")
		return
	}

	request, ok := args[0].(requests.SignatureProposalParticipantsListRequest)

	if !ok {
		err = errors.New("cannot cast participants list")
		return
	}

	if err = request.Validate(); err != nil {
		return
	}

	m.payload.ConfirmationProposalPayload = make(internal.SignatureProposalQuorum)

	for participantIntId, participant := range request {
		participantId := createFingerprint(&participant.PublicKey)
		secret, err := generateRandomString(32)
		if err != nil {
			return nil, errors.New("cannot generateRandomString")
		}
		m.payload.ConfirmationProposalPayload[participantId] = internal.SignatureProposalParticipant{
			ParticipantId:    participantIntId,
			Title:            participant.Title,
			PublicKey:        participant.PublicKey,
			InvitationSecret: secret,
			UpdatedAt:        nil,
		}
	}

	// Make response

	responseData := make(responses.SignatureProposalParticipantInvitationsResponse, 0)

	for pubKeyFingerprint, proposal := range m.payload.ConfirmationProposalPayload {
		encryptedInvitationSecret, err := encryptWithPubKey(proposal.PublicKey, proposal.InvitationSecret)
		if err != nil {
			return nil, errors.New("cannot encryptWithPubKey")
		}
		responseEntry := &responses.SignatureProposalParticipantInvitationEntry{
			Title:               proposal.Title,
			PubKeyFingerprint:   pubKeyFingerprint,
			EncryptedInvitation: encryptedInvitationSecret,
		}
		responseData = append(responseData, responseEntry)
	}

	// Change state

	return responseData, nil
}

//
func (m *SignatureProposalFSM) actionConfirmProposalByParticipant(event fsm.Event, args ...interface{}) (response interface{}, err error) {
	log.Println("I'm actionConfirmProposalByParticipant")
	return
}

func (m *SignatureProposalFSM) actionDeclineProposalByParticipant(event fsm.Event, args ...interface{}) (response interface{}, err error) {
	log.Println("I'm  actionDeclineProposalByParticipant")
	return
}

func (m *SignatureProposalFSM) actionValidateProposal(event fsm.Event, args ...interface{}) (response interface{}, err error) {
	log.Println("I'm  actionValidateProposal")
	return
}
