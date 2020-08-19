package requests

import (
	"errors"
	"fmt"

	"github.com/depools/dc4bc/fsm/config"
)

func (r *SignatureProposalParticipantsListRequest) Validate() error {
	if len(r.Participants) < config.ParticipantsMinCount {
		return errors.New(fmt.Sprintf("too few participants, minimum is {%d}", config.ParticipantsMinCount))
	}

	if r.SigningThreshold < 2 {
		return errors.New("{SigningThreshold} minimum count is {2}")
	}

	if r.SigningThreshold > len(r.Participants) {
		return errors.New("{SigningThreshold} cannot be higher than {ParticipantsCount}")
	}

	for _, participant := range r.Participants {
		if len(participant.Addr) < 3 {
			return errors.New("{Addr} minimum length is {3}")
		}

		if len(participant.Addr) > 150 {
			return errors.New("{Addr} maximum length is {150}")
		}

		if len(participant.PubKey) < 10 {
			return errors.New("{PubKey} too short")
		}

		if len(participant.DkgPubKey) < 10 {
			return errors.New("{DkgPubKey} too short")
		}
	}

	if r.CreatedAt.IsZero() {
		return errors.New("{CreatedAt} cannot be a nil")
	}

	return nil
}

func (r *SignatureProposalParticipantRequest) Validate() error {
	if r.ParticipantId < 0 {
		return errors.New("{ParticipantId} cannot be a negative number")
	}

	if r.CreatedAt.IsZero() {
		return errors.New("{CreatedAt} cannot be a nil")
	}
	return nil
}
