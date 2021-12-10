package signature

import (
	"github.com/lidofinance/dc4bc/client/api/dto"
	"github.com/lidofinance/dc4bc/client/repositories/signature"
	"github.com/lidofinance/dc4bc/fsm/types"
)

type SignatureService interface {
	GetSignatures(dto *dto.DkgIdDTO) (map[string][]types.ReconstructedSignature, error)
	GetSignatureByID(dto *dto.SignatureByIdDTO) ([]types.ReconstructedSignature, error)
	SaveSignatures(signature []types.ReconstructedSignature) error
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

func (s *BaseSignatureService) SaveSignatures(signature []types.ReconstructedSignature) error {
	return s.signatureRepo.SaveSignatures(signature)
}
