package client

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/depools/dc4bc/fsm/state_machines"

	"github.com/syndtr/goleveldb/leveldb"
)

const (
	offsetKey     = "offset"
	operationsKey = "operations"
	fsmStateKey   = "fsm_state"
)

type State interface {
	SaveOffset(uint64) error
	LoadOffset() (uint64, error)

	SaveFSM(dkgRoundID string, dump []byte) error
	LoadFSM(dkgRoundID string) (*state_machines.FSMInstance, bool, error)

	PutOperation(operation *Operation) error
	DeleteOperation(operationID string) error
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

	state := &LevelDBState{
		stateDb: db,
	}

	// Init state key for operations JSON.
	if _, err := state.stateDb.Get([]byte(operationsKey), nil); err != nil {
		if err := state.initJsonKey(operationsKey, map[string]*Operation{}); err != nil {
			return nil, fmt.Errorf("failed to init %s storage: %w", operationsKey, err)
		}
	}

	// Init state key for offset bytes.
	if _, err := state.stateDb.Get([]byte(offsetKey), nil); err != nil {
		bz := make([]byte, 8)
		binary.LittleEndian.PutUint64(bz, 0)
		if err := db.Put([]byte(offsetKey), bz, nil); err != nil {
			return nil, fmt.Errorf("failed to init %s storage: %w", offsetKey, err)
		}
	}

	// Init state key for FSMs state.
	if _, err := state.stateDb.Get([]byte(fsmStateKey), nil); err != nil {
		if err := state.initJsonKey(fsmStateKey, map[string]*state_machines.FSMInstance{}); err != nil {
			return nil, fmt.Errorf("failed to init %s storage: %w", fsmStateKey, err)
		}
	}

	return state, nil
}

func (s *LevelDBState) initJsonKey(key string, data interface{}) error {
	if _, err := s.stateDb.Get([]byte(key), nil); err != nil {
		dataBz, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal storage structure: %w", err)
		}
		err = s.stateDb.Put([]byte(key), dataBz, nil)
		if err != nil {
			return fmt.Errorf("failed to init state: %w", err)
		}
	}

	return nil
}

func (s *LevelDBState) SaveOffset(offset uint64) error {
	bz := make([]byte, 8)
	binary.LittleEndian.PutUint64(bz, offset)

	if err := s.stateDb.Put([]byte(offsetKey), bz, nil); err != nil {
		return fmt.Errorf("failed to set offset: %w", err)
	}

	return nil
}

func (s *LevelDBState) LoadOffset() (uint64, error) {
	bz, err := s.stateDb.Get([]byte(offsetKey), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to read offset: %w", err)
	}

	offset := binary.LittleEndian.Uint64(bz)
	return offset, nil
}

func (s *LevelDBState) SaveFSM(dkgRoundID string, dump []byte) error {
	bz, err := s.stateDb.Get([]byte(fsmStateKey), nil)
	if err != nil {
		return fmt.Errorf("failed to get FSM instances: %w", err)
	}

	var fsmInstances = map[string][]byte{}
	if err := json.Unmarshal(bz, &fsmInstances); err != nil {
		return fmt.Errorf("failed to unmarshal FSM instances: %w", err)
	}

	fsmInstances[dkgRoundID] = dump

	if err := s.stateDb.Put([]byte(fsmStateKey), dump, nil); err != nil {
		return fmt.Errorf("failed to save fsm state: %w", err)
	}

	return nil
}

func (s *LevelDBState) LoadFSM(dkgRoundID string) (*state_machines.FSMInstance, bool, error) {
	bz, err := s.stateDb.Get([]byte(fsmStateKey), nil)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get FSM instances: %w", err)
	}

	var fsmInstances = map[string][]byte{}
	if err := json.Unmarshal(bz, &fsmInstances); err != nil {
		return nil, false, fmt.Errorf("failed to unmarshal FSM instances: %w", err)
	}

	fsmInstanceBz, ok := fsmInstances[dkgRoundID]
	if !ok {
		return nil, false, nil
	}

	fsmInstance, err := state_machines.FromDump(fsmInstanceBz)
	if err != nil {
		return nil, false, fmt.Errorf("failed to restore FSM instance from dump: %w", err)
	}

	return fsmInstance, ok, nil
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

func (s *LevelDBState) DeleteOperation(operationID string) error {
	s.Lock()
	defer s.Unlock()

	operations, err := s.getOperations()
	if err != nil {
		return fmt.Errorf("failed to getOperations: %w", err)
	}

	delete(operations, operationID)

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
