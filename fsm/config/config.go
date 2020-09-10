package config

import "time"

const (
	ParticipantsMinCount                  = 2
	SignatureProposalConfirmationDeadline = time.Hour * 24
	DkgConfirmationDeadline               = time.Hour * 24
	SigningConfirmationDeadline           = time.Hour * 24
)
