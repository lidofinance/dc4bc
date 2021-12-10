package signature

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/lidofinance/dc4bc/client/modules/state"
	"github.com/lidofinance/dc4bc/fsm/types"
)

const (
	SignaturesKeyPrefix = "signatures"
)

type SignatureRepo interface {
	SaveSignatures(signature []types.ReconstructedSignature) error
	GetSignatureByID(dkgID, signatureID string) ([]types.ReconstructedSignature, error)
	GetSignatures(dkgID string) (map[string][]types.ReconstructedSignature, error)
}

type BaseSignatureRepo struct {
	state state.State
}

func NewSignatureRepo(state state.State) *BaseSignatureRepo {
	return &BaseSignatureRepo{state}
}

func (r *BaseSignatureRepo) GetSignatures(dkgID string) (signatures map[string][]types.ReconstructedSignature, err error) {
	key := state.MakeCompositeKeyString(SignaturesKeyPrefix, dkgID)

	bz, err := r.state.Get(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get signatures for dkgID %s: %w", dkgID, err)
	}

	if bz == nil {
		return nil, nil
	}

	if err := json.Unmarshal(bz, &signatures); err != nil {
		return nil, fmt.Errorf("failed to unmarshal signature: %w", err)
	}

	return signatures, nil
}

func (r *BaseSignatureRepo) GetSignatureByID(dkgID, signatureID string) ([]types.ReconstructedSignature, error) {
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

func (r *BaseSignatureRepo) SaveSignatures(signaturesToSave []types.ReconstructedSignature) error {
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

	key := state.MakeCompositeKeyString(SignaturesKeyPrefix, signaturesToSave[0].DKGRoundID)

	if err := r.state.Set(key, signaturesJSON); err != nil {
		return fmt.Errorf("failed to save signatures: %w", err)
	}

	return nil
}
