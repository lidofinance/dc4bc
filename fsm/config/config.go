package config

import "time"

const (
	// Signature proposal
	UsernameMinLength                         = 3
	UsernameMaxLength                         = 150
	ParticipantPubKeyMinLength                = 10
	DkgPubKeyMinLength                        = 10
	SignatureProposalSigningThresholdMinCount = 2
	ParticipantsMinCount                      = 2
	SignatureProposalConfirmationDeadline     = time.Hour * 24 * 7

	// DKG
	DkgConfirmationDeadline = time.Hour * 24 * 7

	// Signing
	SigningConfirmationDeadline = time.Hour * 24 * 7
)
