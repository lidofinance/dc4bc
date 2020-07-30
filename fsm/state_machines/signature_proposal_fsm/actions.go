package signature_proposal_fsm

import (
	"errors"
	"github.com/p2p-org/dc4bc/fsm/state_machines/internal"
	"github.com/p2p-org/dc4bc/fsm/types/requests"
	"github.com/p2p-org/dc4bc/fsm/types/responses"
	"log"
)

// init -> awaitingConfirmations
// args: payload, signing id, participants list
func (s *SignatureProposalFSM) actionInitProposal(event string, args ...interface{}) (response interface{}, err error) {
	var payload internal.MachineStatePayload
	// Init proposal
	log.Println("I'm actionInitProposal")

	if len(args) < 3 {
		err = errors.New("payload and signing id required and participants list required")
		return
	}

	if len(args) > 3 {
		err = errors.New("too many arguments")
		return
	}

	payload, ok := args[0].(internal.MachineStatePayload)

	if !ok {
		err = errors.New("cannot cast payload")
		return
	}

	signingId, ok := args[1].(string)
	if !ok {
		err = errors.New("cannot cast signing id, awaiting string value")
		return
	}

	if len(signingId) < signingIdLen {
		err = errors.New("signing id to short ")
		return
	}

	request, ok := args[2].(requests.ProposalParticipantsListRequest)

	if !ok {
		err = errors.New("cannot cast participants list")
		return
	}

	if err = request.Validate(); err != nil {
		return
	}

	payload.ProposalPayload = make(internal.ProposalConfirmationPrivateQuorum)

	for _, participant := range request {
		participantId := createFingerprint(&participant.PublicKey)
		secret, err := generateRandomString(32)
		if err != nil {
			return nil, errors.New("cannot generateRandomString")
		}
		payload.ProposalPayload[participantId] = internal.ProposalParticipantPrivate{
			Title:            participant.Title,
			PublicKey:        participant.PublicKey,
			InvitationSecret: secret,
			ConfirmedAt:      nil,
		}
	}

	/*s.state = &fsm_pool.FSMachine{
		Id:    signingId,
		State: stateAwaitProposalConfirmation,
	}
	s.state.Payload.ProposalPayload = &privateParticipantsList*/
	return internal.MachineCombinedResponse{
		Response: responses.ProposalParticipantInvitationsResponse{},
		Payload:  &payload,
	}, nil
}

//
func (s *SignatureProposalFSM) actionConfirmProposalByParticipant(event string, args ...interface{}) (response interface{}, err error) {
	log.Println("I'm actionConfirmProposalByParticipant")
	return
}

func (s *SignatureProposalFSM) actionDeclineProposalByParticipant(event string, args ...interface{}) (response interface{}, err error) {
	log.Println("I'm  actionDeclineProposalByParticipant")
	return
}

func (s *SignatureProposalFSM) actionValidateProposal(event string, args ...interface{}) (response interface{}, err error) {
	log.Println("I'm  actionValidateProposal")
	return
}
