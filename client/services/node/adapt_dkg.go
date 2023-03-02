package node

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/lidofinance/dc4bc/client/types"
	"github.com/lidofinance/dc4bc/fsm/fsm"
	"github.com/lidofinance/dc4bc/fsm/state_machines/dkg_proposal_fsm"
	"github.com/lidofinance/dc4bc/fsm/types/requests"
	"github.com/lidofinance/dc4bc/storage"
)

func createMessage(origMesage storage.Message) (storage.Message, error) {
	fsmReq, err := types.FSMRequestFromMessage(origMesage)
	if err != nil {
		return storage.Message{}, fmt.Errorf("failed to get FSMRequestFromMessage:  %w", err)
	}
	request, ok := fsmReq.(requests.DKGProposalDealConfirmationRequest)
	if !ok {
		return storage.Message{}, errors.New("cannot cast message request  to type {DKGProposalDealConfirmationRequest}")
	}
	req := requests.DKGProposalDealConfirmationRequest{
		ParticipantId: request.ParticipantId,
		Deal:          []byte("self-confirm"),
		CreatedAt:     request.CreatedAt,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return storage.Message{}, fmt.Errorf("failed to encode FSMRequest:  %w", err)
	}
	newMsg := storage.Message{
		ID:            uuid.New().String(),
		DkgRoundID:    origMesage.DkgRoundID,
		Event:         string(dkg_proposal_fsm.EventDKGDealConfirmationReceived),
		Data:          data,
		SenderAddr:    origMesage.SenderAddr,
		RecipientAddr: origMesage.SenderAddr,
	}

	return newMsg, nil
}

func GetAdaptedReDKG(originalDKG *types.ReDKG) (*types.ReDKG, error) {
	adaptedReDKG := &types.ReDKG{}

	adaptedReDKG.DKGID = originalDKG.DKGID
	adaptedReDKG.Participants = originalDKG.Participants
	adaptedReDKG.Threshold = originalDKG.Threshold
	adaptedReDKG.Messages = []storage.Message{}
	var newOffset uint64
	fixedSenders := map[string]struct{}{}
	for _, m := range originalDKG.Messages {
		if _, found := fixedSenders[m.SenderAddr]; !found && fsm.Event(m.Event) == dkg_proposal_fsm.EventDKGDealConfirmationReceived {
			fixedSenders[m.SenderAddr] = struct{}{}
			workAroundMessage, err := createMessage(m)
			if err != nil {
				return nil, fmt.Errorf("failed to construct new message for adapted reinit DKG message:  %w", err)
			}
			workAroundMessage.Offset = newOffset
			newOffset++
			adaptedReDKG.Messages = append(adaptedReDKG.Messages, workAroundMessage)
		}
		m.Offset = newOffset
		newOffset++
		adaptedReDKG.Messages = append(adaptedReDKG.Messages, m)
	}
	return adaptedReDKG, nil
}
