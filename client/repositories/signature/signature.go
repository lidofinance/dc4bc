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

type SignaturesStorage map[string]map[string][]types.ReconstructedSignature

type SignatureRepo interface {
	SaveSignatures(signature []types.ReconstructedSignature) error
	GetSignatureByID(dkgID, signatureID string) ([]types.ReconstructedSignature, error)
	GetSignaturesByBatchID(dkgID, batchID string) (map[string][]types.ReconstructedSignature, error)
	GetSignatures(dkgID string) (SignaturesStorage, error)
	GetBatches(dkgID string) ([]string, error)
}

type BaseSignatureRepo struct {
	state state.State
}

func NewSignatureRepo(state state.State) *BaseSignatureRepo {
	return &BaseSignatureRepo{state}
}

func (r *BaseSignatureRepo) GetSignatures(dkgID string) (signatures SignaturesStorage, err error) {
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

func (r *BaseSignatureRepo) GetSignaturesByBatchID(dkgID, batchID string) (map[string][]types.ReconstructedSignature, error) {
	signatures, err := r.GetSignatures(dkgID)
	if err != nil {
		return nil, fmt.Errorf("failed to getSignatures: %w", err)
	}

	return signatures[batchID], nil
}

func (r *BaseSignatureRepo) GetSignatureByID(dkgID, signatureID string) ([]types.ReconstructedSignature, error) {
	allSignatures, err := r.GetSignatures(dkgID)
	if err != nil {
		return nil, fmt.Errorf("failed to getSignatures: %w", err)
	}

	for _, batchSignatures := range allSignatures {
		signature, ok := batchSignatures[signatureID]
		if ok {
			return signature, nil
		}
	}
	return nil, errors.New("signature not found")
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
		signatures = make(SignaturesStorage)
	}

	for _, signatureToSave := range signaturesToSave {
		batchSignatures, found := signatures[signatureToSave.BatchID]
		var signs []types.ReconstructedSignature
		if found {
			signs = batchSignatures[signatureToSave.MessageID]
		} else {
			batchSignatures = make(map[string][]types.ReconstructedSignature)
		}
		usernameFound := false
		for i, s := range signs {
			if s.Username == signatureToSave.Username {
				signs[i] = signatureToSave
				usernameFound = true
				break
			}
		}
		if !usernameFound {
			signs = append(signs, signatureToSave)
		}
		batchSignatures[signatureToSave.MessageID] = signs
		signatures[signatureToSave.BatchID] = batchSignatures
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

func (r *BaseSignatureRepo) GetBatches(dkgID string) ([]string, error) {
	allSignatures := make(SignaturesStorage)
	key := state.MakeCompositeKeyString(SignaturesKeyPrefix, dkgID)

	allSignaturesbz, err := r.state.Get(key)
	if err != nil {
		return nil, fmt.Errorf("failed to read batch state from storage: %w", err)
	} else if allSignaturesbz != nil {
		err = json.Unmarshal(allSignaturesbz, &allSignatures)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal batch state: %w", err)
		}
	}
	batchIDS := make([]string, 0, len(allSignatures))
	for batchID := range allSignatures {
		batchIDS = append(batchIDS, batchID)
	}
	return batchIDS, nil
}
