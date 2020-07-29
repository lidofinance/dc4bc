package client

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/syndtr/goleveldb/leveldb"
)

const (
	offsetKey     = "offset"
	operationsKey = "operations"
)

type State interface {
	SetOffset(int64) error
	GetOffset() (int, error)
	PutOperation(operation *Operation) error
	GetOperations() (map[string]*Operation, error)
	GetOperationByID(operationID string) (*Operation, error)
}

type LevelDBState struct {
	sync.Mutex
	stateDb *leveldb.DB
}

func NewLevelDBState(stateDbPath string) (State, error) {
	db, err := leveldb.OpenFile(stateDbPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open stateDB: %w", err)
	}

	return &LevelDBState{
		stateDb: db,
	}, nil
}

func (s *LevelDBState) SetOffset(offset int64) error {
	bz := make([]byte, 8)
	binary.LittleEndian.PutUint64(bz, uint64(offset))

	if err := s.stateDb.Put([]byte(offsetKey), bz, nil); err != nil {
		return fmt.Errorf("failed to set offset: %w", err)
	}

	return nil
}

func (s *LevelDBState) GetOffset() (int64, error) {
	bz, err := s.stateDb.Get([]byte(offsetKey), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to read offset: %w", err)
	}

	offset := int64(binary.LittleEndian.Uint64(bz))
	return offset, nil
}

func (s *LevelDBState) PutOperation(operation *Operation) error {
	s.Lock()
	defer s.Unlock()

	operations, err := s.getOperations()
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

	if err := s.stateDb.Put([]byte(operationsKey), operationsJSON, nil); err != nil {
		return fmt.Errorf("failed to put operations: %w", err)
	}

	return nil
}

func (s *LevelDBState) GetOperations() (map[string]*Operation, error) {
	s.Lock()
	defer s.Unlock()

	return s.getOperations()
}

func (s *LevelDBState) GetOperationByID(operationID string) (*Operation, error) {
	s.Lock()
	defer s.Unlock()

	operations, err := s.getOperations()
	if err != nil {
		return nil, fmt.Errorf("failed to getOperations: %w", err)
	}

	operation, ok := operations[operationID]
	if !ok {
		return nil, errors.New("operation not found")
	}

	return operation, nil
}

func (s *LevelDBState) getOperations() (map[string]*Operation, error) {
	bz, err := s.stateDb.Get([]byte(operationsKey), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get Operations (key: %s): %w", operationsKey, err)
	}

	var operations map[string]*Operation
	if err := json.Unmarshal(bz, &operations); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Operations: %w", err)
	}

	return operations, nil
}
