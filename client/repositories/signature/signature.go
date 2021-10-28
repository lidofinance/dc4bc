package signature_repo

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/lidofinance/dc4bc/client/types"
)

const (
	SignaturesKeyPrefix = "signatures"
)

type state interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
	Delete(key string) error
	Reset(stateDbPath string) (string, error)
}

type SignatureRepo struct {
	stateDb state
}

func NewSignatureRepo(stateDb state) *SignatureRepo {
	return &SignatureRepo{stateDb}
}

func (r *SignatureRepo) GetSignatures(dkgID string) (signatures map[string][]types.ReconstructedSignature, err error) {
	key := types.MakeCompositeKey(SignaturesKeyPrefix, dkgID)

	bz, err := r.stateDb.Get(string(key))
	if err != nil {
		return nil, fmt.Errorf("failed to get signatures for dkgID %s: %w", dkgID, err)
	}

	if bz == nil {
		return nil, nil
	}

	if err := json.Unmarshal(bz, &signatures); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Operations: %w", err)
	}

	return signatures, nil
}

func (r *SignatureRepo) GetSignatureByID(dkgID, signatureID string) ([]types.ReconstructedSignature, error) {
	signatures, err := r.GetSignatures(dkgID)
	if err != nil {
		return nil, fmt.Errorf("failed to getSignatures: %w", err)
	}

	signature, ok := signatures[signatureID]
	if !ok {
		return nil, errors.New("signature not found")
	}

	return signature, nil
}

func (r *SignatureRepo) SaveSignatures(signaturesToSave []types.ReconstructedSignature) error {
	if len(signaturesToSave) == 0 {
		return errors.New("nothing to save")
	}

	signatures, err := r.GetSignatures(signaturesToSave[0].DKGRoundID)
	if err != nil {
		return fmt.Errorf("failed to getSignatures: %w", err)
	}
	if signatures == nil {
		signatures = make(map[string][]types.ReconstructedSignature)
	}

	for _, signature := range signaturesToSave {
		signs := signatures[signature.MessageID]
		usernameFound := false
		for i, s := range signs {
			if s.Username == signature.Username {
				signs[i] = signature
				usernameFound = true
				break
			}
		}
		if !usernameFound {
			signs = append(signs, signature)
		}
		signatures[signature.MessageID] = signs
	}

	signaturesJSON, err := json.Marshal(signatures)
	if err != nil {
		return fmt.Errorf("failed to marshal signatures: %w", err)
	}

	key := types.MakeCompositeKey(SignaturesKeyPrefix, signaturesToSave[0].DKGRoundID)

	if err := r.stateDb.Set(string(key), signaturesJSON); err != nil {
		return fmt.Errorf("failed to save signatures: %w", err)
	}

	return nil
}
