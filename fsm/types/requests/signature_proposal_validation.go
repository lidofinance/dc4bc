package requests

import (
	"errors"
	"github.com/depools/dc4bc/fsm/config"
)

func (r *SignatureProposalParticipantsListRequest) Validate() error {
	if len(r.Participants) < config.ParticipantsMinCount {
		return errors.New("too few participants")
	}

	for _, participant := range r.Participants {
		if len(participant.Title) < 3 {
			return errors.New("{Title} too short")
		}

		if len(participant.Title) > 150 {
			return errors.New("{Title} too long")
		}

		if len(participant.PublicKey) < 10 {
			return errors.New("{PublicKey} too short")
		}
	}

	if r.CreatedAt == nil {
		return errors.New("{CreatedAt} cannot be a nil")
	}

	return nil
}

func (r *SignatureProposalParticipantRequest) Validate() error {
	if len(r.PubKeyFingerprint) == 0 {
		return errors.New("{PubKeyFingerprint} cannot zero length")
	}

	if len(r.EncryptedInvitation) == 0 {
		return errors.New("{EncryptedInvitation} cannot zero length")
	}

	if r.CreatedAt == nil {
		return errors.New("{CreatedAt} cannot be a nil")
	}
	return nil
}
