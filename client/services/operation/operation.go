package operation

import (
	"fmt"

	"github.com/lidofinance/dc4bc/client/repositories/operation"
	"github.com/lidofinance/dc4bc/client/types"
)

type OperationService interface {
	GetOperations() (map[string]*types.Operation, error)
	GetOperationByID(operationID string) (*types.Operation, error)
	PutOperation(operation *types.Operation) error
	DeleteOperation(operation *types.Operation) error
}

type BaseOperationService struct {
	operationRepo operation.OperationRepo
}

func NewOperationService(operationRepo operation.OperationRepo) *BaseOperationService {
	return &BaseOperationService{operationRepo}
}

// GetOperations returns available operations for current state
func (s *BaseOperationService) GetOperations() (map[string]*types.Operation, error) {
	return s.operationRepo.GetOperations()
}

func (s *BaseOperationService) GetOperationByID(operationID string) (*types.Operation, error) {
	operation, err := s.operationRepo.GetOperationByID(operationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get operation: %w", err)
	}

	return operation, nil
}

func (s *BaseOperationService) PutOperation(operation *types.Operation) error {
	if err := s.operationRepo.PutOperation(operation); err != nil {
		return fmt.Errorf("failed to put operation: %w", err)
	}

	return nil
}

func (s *BaseOperationService) DeleteOperation(operation *types.Operation) error {
	if err := s.operationRepo.DeleteOperation(operation); err != nil {
		return fmt.Errorf("failed to delete operation: %w", err)
	}

	return nil
}
