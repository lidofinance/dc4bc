package requests

import "errors"

func (r *SigningProposalStartRequest) Validate() error {
	if r.ParticipantId < 0 {
		return errors.New("{ParticipantId} cannot be a negative number")
	}

	if len(r.SrcPayload) == 0 {
		return errors.New("{SrcPayload} cannot zero length")
	}

	if r.CreatedAt.IsZero() {
		return errors.New("{CreatedAt} is not set")
	}

	return nil
}

func (r *SigningProposalParticipantRequest) Validate() error {
	if r.ParticipantId < 0 {
		return errors.New("{ParticipantId} cannot be a negative number")
	}

	if r.CreatedAt.IsZero() {
		return errors.New("{CreatedAt} is not set")
	}

	return nil
}

func (r *SigningProposalPartialKeyRequest) Validate() error {
	if r.ParticipantId < 0 {
		return errors.New("{ParticipantId} cannot be a negative number")
	}

	if len(r.PartialKey) == 0 {
		return errors.New("{PartialKey} cannot zero length")
	}

	if r.CreatedAt.IsZero() {
		return errors.New("{CreatedAt} is not set")
	}

	return nil
}
