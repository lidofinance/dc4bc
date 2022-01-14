package signature

import (
	"bytes"
	"fmt"
	"github.com/corestario/kyber/pairing"
	"github.com/corestario/kyber/pairing/bls12381"
	"github.com/corestario/kyber/sign/bls"
	"github.com/lidofinance/dc4bc/client/api/dto"
	"github.com/lidofinance/dc4bc/client/repositories/signature"
	"github.com/lidofinance/dc4bc/dkg"
	"github.com/lidofinance/dc4bc/fsm/state_machines"
	"github.com/lidofinance/dc4bc/fsm/types"
)

type SignatureService interface {
	GetSignatures(dto *dto.DkgIdDTO) (map[string][]types.ReconstructedSignature, error)
	GetSignatureByID(dto *dto.SignatureByIdDTO) ([]types.ReconstructedSignature, error)
	GetSignaturesByBatchID(dto *dto.SignaturesByBatchIdDTO) (map[string][]types.ReconstructedSignature, error)
	GetBatches(dto *dto.DkgIdDTO) (map[string][]string, error)
	SaveSignatures(batchID string, signature []types.ReconstructedSignature) error
	VerifySign(signingFSM *state_machines.FSMInstance, dto *dto.SignatureByIdDTO) error
}

type BaseSignatureService struct {
	signatureRepo signature.SignatureRepo
}

func NewSignatureService(signatureRepo signature.SignatureRepo) *BaseSignatureService {
	return &BaseSignatureService{signatureRepo}
}

// GetSignatures returns all signatures for the given DKG round that were reconstructed on the airgapped machine and
// broadcasted by users
func (s *BaseSignatureService) GetSignatures(dto *dto.DkgIdDTO) (map[string][]types.ReconstructedSignature, error) {
	return s.signatureRepo.GetSignatures(dto.DkgID)
}

func (s *BaseSignatureService) GetSignatureByID(dto *dto.SignatureByIdDTO) ([]types.ReconstructedSignature, error) {
	return s.signatureRepo.GetSignatureByID(dto.DkgID, dto.ID)
}

func (s *BaseSignatureService) GetSignaturesByBatchID(dto *dto.SignaturesByBatchIdDTO) (map[string][]types.ReconstructedSignature, error) {
	return s.signatureRepo.GetSignaturesByBatchID(dto.DkgID, dto.BatchID)
}

func (s *BaseSignatureService) GetBatches(dto *dto.DkgIdDTO) (map[string][]string, error) {
	return s.signatureRepo.GetBatches(dto.DkgID)
}

func (s *BaseSignatureService) SaveSignatures(batchID string, signature []types.ReconstructedSignature) error {
	return s.signatureRepo.SaveSignatures(batchID, signature)
}

// VerifySign verifies a signature of a message
func (s *BaseSignatureService) VerifySign(signingFSM *state_machines.FSMInstance, dto *dto.SignatureByIdDTO) error {
	signatures, err := s.signatureRepo.GetSignatureByID(dto.DkgID, dto.ID)
	if err != nil {
		return fmt.Errorf("failed to verify signature: %w", err)
	}
	suite := bls12381.NewBLS12381Suite(nil)
	blsKeyring, err := dkg.LoadPubPolyBLSKeyringFromBytes(suite, signingFSM.FSMDump().Payload.DKGProposalPayload.PubPolyBz)
	if err != nil {
		return fmt.Errorf("failed to unmarshal BLSKeyring's PubPoly")
	}

	//ensure all signatures are equal
	for _, s := range signatures {
		if !bytes.Equal(s.Signature, signatures[0].Signature) {
			return fmt.Errorf("reconstructed signatures from users %s and %s are not equal", s.Username, signatures[0].Username)
		}
	}

	return bls.Verify(suite.(pairing.Suite), blsKeyring.PubPoly.Commit(), signatures[0].SrcPayload, signatures[0].Signature)
}
