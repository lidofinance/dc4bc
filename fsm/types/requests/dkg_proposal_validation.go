package requests

import "errors"

func (r *DKGProposalPubKeyConfirmationRequest) Validate() error {
	if r.ParticipantId < 0 {
		return errors.New("{ParticipantId} cannot be a negative number")
	}

	if len(r.PubKey) == 0 {
		return errors.New("{PubKey} cannot zero length")
	}

	if r.CreatedAt.IsZero() {
		return errors.New("{CreatedAt} is not set")
	}

	return nil
}

func (r *DKGProposalCommitConfirmationRequest) Validate() error {
	if r.ParticipantId < 0 {
		return errors.New("{ParticipantId} cannot be a negative number")
	}

	if len(r.Commit) == 0 {
		return errors.New("{Commit} cannot zero length")
	}

	if r.CreatedAt.IsZero() {
		return errors.New("{CreatedAt} is not set")
	}

	return nil
}

func (r *DKGProposalDealConfirmationRequest) Validate() error {
	if r.ParticipantId < 0 {
		return errors.New("{ParticipantId} cannot be a negative number")
	}

	if len(r.Deal) == 0 {
		return errors.New("{Deal} cannot zero length")
	}

	if r.CreatedAt.IsZero() {
		return errors.New("{CreatedAt} is not set")
	}

	return nil
}

func (r *DKGProposalResponseConfirmationRequest) Validate() error {
	if r.ParticipantId < 0 {
		return errors.New("{ParticipantId} cannot be a negative number")
	}

	if len(r.Response) == 0 {
		return errors.New("{Response} cannot zero length")
	}

	if r.CreatedAt.IsZero() {
		return errors.New("{CreatedAt} is not set")
	}

	return nil
}

func (r *DKGProposalMasterKeyConfirmationRequest) Validate() error {
	if r.ParticipantId < 0 {
		return errors.New("{ParticipantId} cannot be a negative number")
	}

	if len(r.MasterKey) == 0 {
		return errors.New("{MasterKey} cannot zero length")
	}

	if r.CreatedAt.IsZero() {
		return errors.New("{CreatedAt} is not set")
	}

	return nil
}

func (r *DKGProposalConfirmationErrorRequest) Validate() error {
	if r.ParticipantId < 0 {
		return errors.New("{ParticipantId} cannot be a negative number")
	}

	if r.Error == nil {
		return errors.New("{Error} cannot be a nil")
	}

	if r.CreatedAt.IsZero() {
		return errors.New("{CreatedAt} is not set")
	}

	return nil
}
