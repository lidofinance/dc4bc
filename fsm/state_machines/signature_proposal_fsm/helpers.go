package signature_proposal_fsm

import (
	"crypto/sha256"
	"encoding/base64"
	"github.com/depools/dc4bc/fsm/state_machines/internal"
	"github.com/depools/dc4bc/fsm/types/responses"
	"math/rand"
)

// Request and response mutators

func ProposalParticipantsQuorumToResponse(list *internal.ProposalConfirmationPrivateQuorum) responses.ProposalParticipantInvitationsResponse {
	var response responses.ProposalParticipantInvitationsResponse
	for quorumId, parcipant := range *list {
		response = append(response, &responses.ProposalParticipantInvitationEntryResponse{
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
	hash := sha256.Sum256(*data)
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
