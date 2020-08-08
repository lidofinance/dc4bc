package signature_proposal_fsm

import (
	"crypto/sha1"
	"encoding/base64"
	"math/rand"

	"github.com/depools/dc4bc/fsm/state_machines/internal"
	"github.com/depools/dc4bc/fsm/types/responses"
)

// Request and response mutators

func ProposalParticipantsQuorumToResponse(list *internal.SignatureProposalQuorum) responses.SignatureProposalParticipantInvitationsResponse {
	var response responses.SignatureProposalParticipantInvitationsResponse
	for quorumId, parcipant := range *list {
		response = append(response, &responses.SignatureProposalParticipantInvitationEntry{
			Title:             parcipant.Title,
			PubKeyFingerprint: quorumId,
			// TODO: Add encryption
			EncryptedInvitation: parcipant.InvitationSecret,
		})
	}
	return response
}

// Common functions

func createFingerprint(data *[]byte) string {
	hash := sha1.Sum(*data)
	return base64.StdEncoding.EncodeToString(hash[:])
}

// https://blog.questionable.services/article/generating-secure-random-numbers-crypto-rand/

// GenerateRandomBytes returns securely generated random bytes.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func generateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, err
	}

	return b, nil
}

// GenerateRandomString returns a URL-safe, base64 encoded
// securely generated random string.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func generateRandomString(s int) (string, error) {
	b, err := generateRandomBytes(s)
	return base64.URLEncoding.EncodeToString(b), err
}

func encryptWithPubKey(key []byte, value string) (string, error) {
	return value, nil
}
