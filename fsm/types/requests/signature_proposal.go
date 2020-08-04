package requests

import (
	"errors"
	"github.com/depools/dc4bc/fsm/config"
)

// Requests

type ProposalParticipantsListRequest []ProposalParticipantsEntryRequest

type ProposalParticipantsEntryRequest struct {
	// Public title for address, such as name, nickname, organization
	Title     string
	PublicKey []byte
}

func (r *ProposalParticipantsListRequest) Validate() error {
	if len(*r) < config.ParticipantsMinCount {
		return errors.New("too few participants")
	}

	for _, participant := range *r {
		if len(participant.Title) < 3 {
			return errors.New("title too short")
		}

		if len(participant.Title) > 150 {
			return errors.New("title too long")
		}

		if len(participant.PublicKey) < 10 {
			return errors.New("pub key too short")
		}
	}

	return nil
}

type ProposalParticipantConfirmationRequest struct {
	// Public title for address, such as name, nickname, organization
	ParticipantId       string
	EncryptedInvitation string
}

func (r *ProposalParticipantConfirmationRequest) Validate() error {
	return nil
}
