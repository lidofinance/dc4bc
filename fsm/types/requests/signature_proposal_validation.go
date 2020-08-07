package requests

import (
	"errors"
	"github.com/depools/dc4bc/fsm/config"
)

func (r *SignatureProposalParticipantsListRequest) Validate() error {
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

func (r *SignatureProposalParticipantRequest) Validate() error {
	return nil
}
