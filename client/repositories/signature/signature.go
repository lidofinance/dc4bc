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
	BatchKeyPrefix      = "batch"
)

type SignatureRepo interface {
	SaveSignatures(batchID string, signature []types.ReconstructedSignature) error
	GetSignatureByID(dkgID, signatureID string) ([]types.ReconstructedSignature, error)
	GetSignaturesByBatchID(dkgID, batchID string) (map[string][]types.ReconstructedSignature, error)
	GetSignatures(dkgID string) (map[string][]types.ReconstructedSignature, error)
	GetBatches(dkgID string) (map[string][]string, error)
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

func (r *BaseSignatureRepo) GetSignaturesByBatchID(dkgID, batchID string) (map[string][]types.ReconstructedSignature, error) {
	batchState := make(map[string][]string)
	signatures := make(map[string][]types.ReconstructedSignature)

	key := state.MakeCompositeKeyString(BatchKeyPrefix, dkgID)
	bz, err := r.state.Get(key)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch batch's signatures ids: %w", err)
	}

	err = json.Unmarshal(bz, &batchState)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal batch's signatures ids: %w", err)
	}

	allSignatures, err := r.GetSignatures(dkgID)
	if err != nil {
		return nil, fmt.Errorf("failed to getSignatures: %w", err)
	}

	var ok bool
	for _, mID := range batchState[batchID] {
		signatures[mID], ok = allSignatures[mID]
		if !ok {
			return nil, fmt.Errorf("signature with ID \"%s\" not found", mID)
		}
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

func (r *BaseSignatureRepo) SaveSignatures(batchID string, signaturesToSave []types.ReconstructedSignature) error {
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

	messageIDS := make([]string, 0, len(signaturesToSave))

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
		messageIDS = append(messageIDS, signature.MessageID)
	}

	signaturesJSON, err := json.Marshal(signatures)
	if err != nil {
		return fmt.Errorf("failed to marshal signatures: %w", err)
	}

	key := state.MakeCompositeKeyString(SignaturesKeyPrefix, signaturesToSave[0].DKGRoundID)

	// WARN: ideally we have to make both save operations in single transaction
	if err := r.state.Set(key, signaturesJSON); err != nil {
		return fmt.Errorf("failed to save signatures: %w", err)
	}
	if len(batchID) > 0 {
		if err := r.updateBatchState(signaturesToSave[0].DKGRoundID, batchID, messageIDS); err != nil {
			return fmt.Errorf("failed to save signatures's ids: %w", err)
		}
	}

	return nil
}

func (r *BaseSignatureRepo) GetBatches(dkgID string) (map[string][]string, error) {
	batchState := make(map[string][]string)
	key := state.MakeCompositeKeyString(BatchKeyPrefix, dkgID)

	batchStatebz, err := r.state.Get(key)
	if err != nil {
		return nil, fmt.Errorf("failed to read batch state from storage: %w", err)
	} else if batchStatebz != nil {
		err = json.Unmarshal(batchStatebz, &batchState)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal batch state: %w", err)
		}
	}
	return batchState, nil
}

func (r *BaseSignatureRepo) updateBatchState(dkgID, batchID string, MessageIDS []string) error {
	batchState, err := r.GetBatches(dkgID)
	if err != nil {
		return fmt.Errorf("failed to get batch state:%w", err)
	}
	batchState[batchID] = MessageIDS
	key := state.MakeCompositeKeyString(BatchKeyPrefix, dkgID)

	batchStatebz, err := json.Marshal(batchState)
	if err != nil {
		return fmt.Errorf("failed to marshal batch state: %w", err)
	}
	if err := r.state.Set(key, batchStatebz); err != nil {
		return fmt.Errorf("failed to save batch state: %w", err)
	}
	return nil
}
