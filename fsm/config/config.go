package config

import "time"

const (
	// TODO: Move to machine level configs?
	ParticipantsMinCount                  = 3
	SignatureProposalConfirmationDeadline = time.Hour * 24
	DkgConfirmationDeadline               = time.Hour * 24
	SigningConfirmationDeadline           = time.Hour * 24
)
