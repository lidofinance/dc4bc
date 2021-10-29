package operation_repo

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/lidofinance/dc4bc/client/modules/state"
	"github.com/lidofinance/dc4bc/client/types"
)

const (
	OperationsKey        = "operations"
	DeletedOperationsKey = "deleted_operations"
	FSMStateKey          = "fsm_state"
)

type OperationRepo interface {
	PutOperation(operation *types.Operation) error
	DeleteOperation(operation *types.Operation) error
	GetOperations() (map[string]*types.Operation, error)
	GetOperationByID(operationID string) (*types.Operation, error)
}

type BaseOperationRepo struct {
	state                        state.State
	operationsCompositeKey       string
	deleteOperationsCompositeKey string
}

func NewOperationRepo(state state.State, topic string) (*BaseOperationRepo, error) {
	operationsCompositeKey := types.MakeCompositeKeyString(topic, OperationsKey)
	deleteOperationsCompositeKey := types.MakeCompositeKeyString(topic, DeletedOperationsKey)

	repo := &BaseOperationRepo{
		state:                        state,
		operationsCompositeKey:       operationsCompositeKey,
		deleteOperationsCompositeKey: deleteOperationsCompositeKey,
	}

	if err := repo.initJsonKey(operationsCompositeKey, map[string]*types.Operation{}); err != nil {
		return nil, fmt.Errorf("failed to init %s storage: %w", string(operationsCompositeKey), err)
	}

	if err := repo.initJsonKey(deleteOperationsCompositeKey, map[string]*types.Operation{}); err != nil {
		return nil, fmt.Errorf("failed to init %s storage: %w", string(deleteOperationsCompositeKey), err)
	}

	return repo, nil
}

func (r *BaseOperationRepo) PutOperation(operation *types.Operation) error {
	operations, err := r.GetOperations()
	if err != nil {
		return fmt.Errorf("failed to getOperations: %w", err)
	}

	if _, ok := operations[operation.ID]; ok {
		return fmt.Errorf("operation %s already exists", operation.ID)
	}

	operations[operation.ID] = operation
	operationsJSON, err := json.Marshal(operations)
	if err != nil {
		return fmt.Errorf("failed to marshal operations: %w", err)
	}

	if err := r.state.Set(r.operationsCompositeKey, operationsJSON); err != nil {
		return fmt.Errorf("failed to put operations: %w", err)
	}

	return nil
}

// DeleteOperation deletes operation from an operation pool
func (r *BaseOperationRepo) DeleteOperation(operation *types.Operation) error {
	deletedOperations, err := r.getDeletedOperations()
	if err != nil {
		return fmt.Errorf("failed to getDeletedOperations: %w", err)
	}

	if _, ok := deletedOperations[operation.ID]; ok {
		return fmt.Errorf("operation %s was already deleted", operation.ID)
	}

	deletedOperations[operation.ID] = operation
	deletedOperationsJSON, err := json.Marshal(deletedOperations)
	if err != nil {
		return fmt.Errorf("failed to marshal deleted operations: %w", err)
	}

	if err := r.state.Set(r.deleteOperationsCompositeKey, deletedOperationsJSON); err != nil {
		return fmt.Errorf("failed to put deleted operations: %w", err)
	}

	operations, err := r.GetOperations()
	if err != nil {
		return fmt.Errorf("failed to getOperations: %w", err)
	}

	delete(operations, operation.ID)

	operationsJSON, err := json.Marshal(operations)
	if err != nil {
		return fmt.Errorf("failed to marshal operations: %w", err)
	}

	if err := r.state.Set(r.operationsCompositeKey, operationsJSON); err != nil {
		return fmt.Errorf("failed to put operations: %w", err)
	}

	return nil
}

func (r *BaseOperationRepo) GetOperationByID(operationID string) (*types.Operation, error) {
	operations, err := r.GetOperations()
	if err != nil {
		return nil, fmt.Errorf("failed to getOperations: %w", err)
	}

	operation, ok := operations[operationID]
	if !ok {
		return nil, errors.New("operation not found")
	}

	return operation, nil
}

// GetOperations returns all operations from an operation pool
func (r *BaseOperationRepo) GetOperations() (map[string]*types.Operation, error) {
	deletedOperations, err := r.getDeletedOperations()
	if err != nil {
		return nil, fmt.Errorf("failed to getDeletedOperations: %w", err)
	}

	bz, err := r.state.Get(r.operationsCompositeKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get Operations (key: %s): %w", r.deleteOperationsCompositeKey, err)
	}

	var operations map[string]*types.Operation
	if err := json.Unmarshal(bz, &operations); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Operations: %w", err)
	}

	result := make(map[string]*types.Operation)
	for id, operation := range operations {
		if _, ok := deletedOperations[id]; !ok {
			result[id] = operation
		}
	}

	return result, nil
}

func (r *BaseOperationRepo) getDeletedOperations() (map[string]*types.Operation, error) {
	bz, err := r.state.Get(r.deleteOperationsCompositeKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get deleted Operations (key: %s): %w", r.deleteOperationsCompositeKey, err)
	}

	if bz == nil {
		return nil, nil
	}

	var operations map[string]*types.Operation
	if err := json.Unmarshal(bz, &operations); err != nil {
		return nil, fmt.Errorf("failed to unmarshal deleted Operations: %w", err)
	}

	return operations, nil
}

func (r *BaseOperationRepo) initJsonKey(key string, data interface{}) error {
	operationsBz, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal storage structure: %w", err)
	}

	err = r.state.Set(key, operationsBz)
	if err != nil {
		return fmt.Errorf("failed to init state: %w", err)
	}

	return nil
}