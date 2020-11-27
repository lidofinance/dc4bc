package config

import "time"

const (
	ParticipantsMinCount                  = 2
	SignatureProposalConfirmationDeadline = time.Hour * 24 * 7
	DkgConfirmationDeadline               = time.Hour * 24 * 7
	SigningConfirmationDeadline           = time.Hour * 24 * 7
)
