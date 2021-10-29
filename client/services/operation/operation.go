package operation_service

import (
	"encoding/json"
	"fmt"

	"github.com/lidofinance/dc4bc/client/api/dto"
	operation_repo "github.com/lidofinance/dc4bc/client/repositories/operation"
	"github.com/lidofinance/dc4bc/client/types"
)

type OperationService interface {
	GetOperations() (map[string]*types.Operation, error)
	GetOperation(dto *dto.OperationIdDTO) ([]byte, error)
}

type BaseOperationService struct {
	operationRepo operation_repo.OperationRepo
}

func NewOperationService(operationRepo operation_repo.OperationRepo) *BaseOperationService {
	return &BaseOperationService{operationRepo}
}

// GetOperations returns available operations for current state
func (s *BaseOperationService) GetOperations() (map[string]*types.Operation, error) {
	return s.operationRepo.GetOperations()
}

func (s *BaseOperationService) GetOperation(dto *dto.OperationIdDTO) ([]byte, error) {
	operation, err := s.operationRepo.GetOperationByID(dto.OperationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get operation: %w", err)
	}

	operationJSON, err := json.Marshal(operation)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal operation: %w", err)
	}

	return operationJSON, nil
}
