package requests

import (
	"errors"
	"fmt"
)

func (r *SigningProposalParticipantRequest) Validate() error {
	if r.BatchID == "" {
		return errors.New("{BatchID} cannot be empty")
	}

	if r.ParticipantId < 0 {
		return errors.New("{ParticipantId} cannot be a negative number")
	}

	if r.CreatedAt.IsZero() {
		return errors.New("{CreatedAt} is not set")
	}

	return nil
}

func (m *MessageToSign) Validate() error {
	if m.MessageID == "" {
		return errors.New("{MessageID} cannot be empty")
	}
	if len(m.Payload) == 0 {
		return errors.New("{Payload} cannot zero length")
	}
	return nil
}

func (r *SigningBatchProposalStartRequest) Validate() error {
	if len(r.BatchID) == 0 {
		return fmt.Errorf("{BatchID} can not be empty")
	}
	if len(r.MessagesToSign) == 0 {
		return fmt.Errorf("{MessagesToSign} can not be empty")
	}
	if r.ParticipantId < 0 {
		return errors.New("{ParticipantId} cannot be a negative number")
	}
	if r.CreatedAt.IsZero() {
		return errors.New("{CreatedAt} is not set")
	}
	for _, m := range r.MessagesToSign {
		err := m.Validate()
		if err != nil {
			return fmt.Errorf("failed to validate message to sign %+v: %w", m, err)
		}
	}
	return nil
}

func (s *PartialSign) Validate() error {
	if len(s.MessageID) == 0 {
		return fmt.Errorf("{MessageID} can not be empty")
	}
	if len(s.Sign) == 0 {
		return fmt.Errorf("{Sign} can not be empty")
	}
	return nil
}

func (r *SigningProposalBatchPartialSignRequests) Validate() error {
	if len(r.BatchID) == 0 {
		return fmt.Errorf("{BatchID} can not be empty")
	}
	if r.CreatedAt.IsZero() {
		return errors.New("{CreatedAt} is not set")
	}
	if r.ParticipantId < 0 {
		return errors.New("{ParticipantId} cannot be a negative number")
	}
	if len(r.PartialSigns) == 0 {
		return fmt.Errorf("{MessagesToSign} can not be empty")
	}
	for _, s := range r.PartialSigns {
		err := s.Validate()
		if err != nil {
			return fmt.Errorf("failed to validate partial sign %+v: %w", s, err)
		}
	}
	return nil
}
