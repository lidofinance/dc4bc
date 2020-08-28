package signing_proposal_fsm

import (
	"crypto/rand"
	"encoding/base64"
)

const (
	dkgTransactionIdLength = 512
)

func generateSigningId() (string, error) {
	b := make([]byte, dkgTransactionIdLength)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(b), err
}
