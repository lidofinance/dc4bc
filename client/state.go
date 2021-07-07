package client

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	ops "github.com/lidofinance/dc4bc/client/operations"
	"github.com/lidofinance/dc4bc/fsm/state_machines"
	"github.com/syndtr/goleveldb/leveldb"
)

const (
	offsetKey            = "offset"
	operationsKey        = "operations"
	deletedOperationsKey = "deleted_operations"
	fsmStateKey          = "fsm_state"
	signaturesKeyPrefix  = "signatures"
)

func makeCompositeKey(prefix, key string) []byte {
	return []byte(fmt.Sprintf("%s_%s", prefix, key))
}

// State is the client's State (it keeps the offset, the FSM State and
// the Operation pool.
type State interface {
	NewStateFromOld(stateDbPath string) (State, string, error)

	SaveOffset(uint64) error
	LoadOffset() (uint64, error)

	SaveFSM(dkgRoundID string, dump []byte) error
	LoadFSM(dkgRoundID string) (*state_machines.FSMInstance, bool, error)
	GetAllFSM() (map[string]*state_machines.FSMInstance, error)

	PutOperation(operation *ops.Operation) error
	DeleteOperation(operation *ops.Operation) error
	GetOperations() (map[string]*ops.Operation, error)
	GetOperationByID(operationID string) (*ops.Operation, error)

	SaveSignature(signature ops.ReconstructedSignature) error
	GetSignatureByID(dkgID, signatureID string) ([]ops.ReconstructedSignature, error)
	GetSignatures(dkgID string) (map[string][]ops.ReconstructedSignature, error)
}

type LevelDBState struct {
	sync.Mutex
	stateDb     *leveldb.DB
	topic       string
	stateDbPath string
}

func NewLevelDBState(stateDbPath string, topic string) (State, error) {
	db, err := leveldb.OpenFile(stateDbPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open stateDB: %w", err)
	}

	state := &LevelDBState{
		stateDb:     db,
		topic:       topic,
		stateDbPath: stateDbPath,
	}

	// Init State key for operations JSON.
	operationsCompositeKey := makeCompositeKey(topic, operationsKey)
	if _, err := state.stateDb.Get(operationsCompositeKey, nil); err != nil {
		if err := state.initJsonKey(operationsCompositeKey, map[string]*ops.Operation{}); err != nil {
			return nil, fmt.Errorf("failed to init %s Storage: %w", string(operationsCompositeKey), err)
		}
	}

	// Init State key for operations JSON.
	deleteOperationsCompositeKey := makeCompositeKey(topic, deletedOperationsKey)
	if _, err := state.stateDb.Get(deleteOperationsCompositeKey, nil); err != nil {
		if err := state.initJsonKey(deleteOperationsCompositeKey, map[string]*ops.Operation{}); err != nil {
			return nil, fmt.Errorf("failed to init %s Storage: %w", string(deleteOperationsCompositeKey), err)
		}
	}

	// Init State key for offset bytes.
	offsetCompositeKey := makeCompositeKey(topic, offsetKey)
	if _, err := state.stateDb.Get(offsetCompositeKey, nil); err != nil {
		bz := make([]byte, 8)
		binary.LittleEndian.PutUint64(bz, 0)
		if err := db.Put(offsetCompositeKey, bz, nil); err != nil {
			return nil, fmt.Errorf("failed to init %s Storage: %w", string(offsetCompositeKey), err)
		}
	}

	fsmStateCompositeKey := makeCompositeKey(topic, fsmStateKey)
	if _, err := state.stateDb.Get(fsmStateCompositeKey, nil); err != nil {
		if err := db.Put(fsmStateCompositeKey, []byte{}, nil); err != nil {
			return nil, fmt.Errorf("failed to init %s Storage: %w", string(fsmStateCompositeKey), err)
		}
	}

	return state, nil
}

func (s *LevelDBState) NewStateFromOld(stateDbPath string) (State, string, error) {
	if len(stateDbPath) < 1 {
		stateDbPath = fmt.Sprintf("%s_%d", s.stateDbPath, time.Now().Unix())
	}

	state, err := NewLevelDBState(stateDbPath, s.topic)

	return state, stateDbPath, err
}

func (s *LevelDBState) initJsonKey(key []byte, data interface{}) error {
	if _, err := s.stateDb.Get(key, nil); err != nil {
		operationsBz, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal Storage structure: %w", err)
		}
		err = s.stateDb.Put(key, operationsBz, nil)
		if err != nil {
			return fmt.Errorf("failed to init State: %w", err)
		}
	}

	return nil
}

func (s *LevelDBState) SaveOffset(offset uint64) error {
	bz := make([]byte, 8)
	binary.LittleEndian.PutUint64(bz, offset)

	if err := s.stateDb.Put(makeCompositeKey(s.topic, offsetKey), bz, nil); err != nil {
		return fmt.Errorf("failed to set offset: %w", err)
	}

	return nil
}

func (s *LevelDBState) LoadOffset() (uint64, error) {
	bz, err := s.stateDb.Get(makeCompositeKey(s.topic, offsetKey), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to read offset: %w", err)
	}

	offset := binary.LittleEndian.Uint64(bz)

	return offset, nil
}

func (s *LevelDBState) SaveFSM(dkgRoundID string, dump []byte) error {
	bz, err := s.stateDb.Get(makeCompositeKey(s.topic, fsmStateKey), nil)
	if err != nil {
		return fmt.Errorf("failed to get FSM instances: %w", err)
	}

	var fsmInstances = map[string][]byte{}
	if len(bz) > 0 {
		if err := json.Unmarshal(bz, &fsmInstances); err != nil {
			return fmt.Errorf("failed to unmarshal FSM instances: %w", err)
		}
	}

	fsmInstances[dkgRoundID] = dump

	fsmInstancesBz, err := json.Marshal(fsmInstances)
	if err != nil {
		return fmt.Errorf("failed to marshal FSM instances: %w", err)
	}

	if err := s.stateDb.Put(makeCompositeKey(s.topic, fsmStateKey), fsmInstancesBz, nil); err != nil {
		return fmt.Errorf("failed to save fsm State: %w", err)
	}

	return nil
}

func (s *LevelDBState) GetAllFSM() (map[string]*state_machines.FSMInstance, error) {
	bz, err := s.stateDb.Get(makeCompositeKey(s.topic, fsmStateKey), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get FSM instances: %w", err)
	}
	var fsmInstancesBz = map[string][]byte{}
	if len(bz) > 0 {
		if err := json.Unmarshal(bz, &fsmInstancesBz); err != nil {
			return nil, fmt.Errorf("failed to unmarshal FSM instances: %w", err)
		}
	}
	fsmInstances := make(map[string]*state_machines.FSMInstance, len(fsmInstancesBz))
	for k, v := range fsmInstancesBz {
		fsmInstances[k], err = state_machines.FromDump(v)
		if err != nil {
			return nil, fmt.Errorf("failed to restore FSM instance from dump: %w", err)
		}
	}
	return fsmInstances, nil
}

func (s *LevelDBState) LoadFSM(dkgRoundID string) (*state_machines.FSMInstance, bool, error) {
	bz, err := s.stateDb.Get(makeCompositeKey(s.topic, fsmStateKey), nil)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get FSM instances: %w", err)
	}

	var fsmInstances = map[string][]byte{}
	if len(bz) > 0 {
		if err := json.Unmarshal(bz, &fsmInstances); err != nil {
			return nil, false, fmt.Errorf("failed to unmarshal FSM instances: %w", err)
		}
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

func (s *LevelDBState) PutOperation(operation *ops.Operation) error {
	s.Lock()
	defer s.Unlock()

	deletedOperations, err := s.getDeletedOperations()
	if err != nil {
		return fmt.Errorf("failed to getDeletedOperations: %w", err)
	}

	if _, ok := deletedOperations[operation.ID]; ok {
		return fmt.Errorf("operation %s was deleted", operation.ID)
	}

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

	if err := s.stateDb.Put(makeCompositeKey(s.topic, operationsKey), operationsJSON, nil); err != nil {
		return fmt.Errorf("failed to put operations: %w", err)
	}

	return nil
}

// DeleteOperation deletes operation from an operation pool
func (s *LevelDBState) DeleteOperation(operation *ops.Operation) error {
	s.Lock()
	defer s.Unlock()

	deletedOperations, err := s.getDeletedOperations()
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

	if err := s.stateDb.Put(makeCompositeKey(s.topic, deletedOperationsKey), deletedOperationsJSON, nil); err != nil {
		return fmt.Errorf("failed to put deleted operations: %w", err)
	}

	operations, err := s.getOperations()
	if err != nil {
		return fmt.Errorf("failed to getOperations: %w", err)
	}

	delete(operations, operation.ID)

	operationsJSON, err := json.Marshal(operations)
	if err != nil {
		return fmt.Errorf("failed to marshal operations: %w", err)
	}

	if err := s.stateDb.Put(makeCompositeKey(s.topic, operationsKey), operationsJSON, nil); err != nil {
		return fmt.Errorf("failed to put operations: %w", err)
	}

	return nil
}

// GetOperations returns all operations from an operation pool
func (s *LevelDBState) GetOperations() (map[string]*ops.Operation, error) {
	s.Lock()
	defer s.Unlock()

	return s.getOperations()
}

func (s *LevelDBState) GetOperationByID(operationID string) (*ops.Operation, error) {
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

func (s *LevelDBState) getOperations() (map[string]*ops.Operation, error) {
	deletedOperations, err := s.getDeletedOperations()
	if err != nil {
		return nil, fmt.Errorf("failed to getDeletedOperations: %w", err)
	}

	operationsCompositeKey := makeCompositeKey(s.topic, operationsKey)
	bz, err := s.stateDb.Get(operationsCompositeKey, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get Operations (key: %s): %w", string(operationsCompositeKey), err)
	}

	var operations map[string]*ops.Operation
	if err := json.Unmarshal(bz, &operations); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Operations: %w", err)
	}

	result := make(map[string]*ops.Operation)
	for id, operation := range operations {
		if _, ok := deletedOperations[id]; !ok {
			result[id] = operation
		}
	}

	return result, nil
}

func (s *LevelDBState) getDeletedOperations() (map[string]*ops.Operation, error) {
	deletedOperationsCompositeKey := makeCompositeKey(s.topic, deletedOperationsKey)
	bz, err := s.stateDb.Get(deletedOperationsCompositeKey, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get deleted Operations (key: %s): %w", string(deletedOperationsCompositeKey), err)
	}

	var operations map[string]*ops.Operation
	if err := json.Unmarshal(bz, &operations); err != nil {
		return nil, fmt.Errorf("failed to unmarshal deleted Operations: %w", err)
	}

	return operations, nil
}

func (s *LevelDBState) getSignatures(dkgID string) (map[string][]ops.ReconstructedSignature, error) {
	bz, err := s.stateDb.Get(makeCompositeKey(signaturesKeyPrefix, dkgID), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get signatures for dkgID %s: %w", dkgID, err)
	}

	var signatures map[string][]ops.ReconstructedSignature
	if err := json.Unmarshal(bz, &signatures); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Operations: %w", err)
	}

	return signatures, nil
}

func (s *LevelDBState) GetSignatures(dkgID string) (map[string][]ops.ReconstructedSignature, error) {
	s.Lock()
	defer s.Unlock()

	return s.getSignatures(dkgID)
}

func (s *LevelDBState) GetSignatureByID(dkgID, signatureID string) ([]ops.ReconstructedSignature, error) {
	s.Lock()
	defer s.Unlock()

	signatures, err := s.getSignatures(dkgID)
	if err != nil {
		return nil, fmt.Errorf("failed to getSignatures: %w", err)
	}

	signature, ok := signatures[signatureID]
	if !ok {
		return nil, errors.New("signature not found")
	}

	return signature, nil
}

func (s *LevelDBState) SaveSignature(signature ops.ReconstructedSignature) error {
	s.Lock()
	defer s.Unlock()

	signatures, err := s.getSignatures(signature.DKGRoundID)
	if err != nil {
		return fmt.Errorf("failed to getSignatures: %w", err)
	}
	if signatures == nil {
		signatures = make(map[string][]ops.ReconstructedSignature)
	}

	sig := signatures[signature.SigningID]
	sig = append(sig, signature)
	signatures[signature.SigningID] = sig

	signaturesJSON, err := json.Marshal(signatures)
	if err != nil {
		return fmt.Errorf("failed to marshal signatures: %w", err)
	}

	if err := s.stateDb.Put(makeCompositeKey(signaturesKeyPrefix, signature.DKGRoundID), signaturesJSON, nil); err != nil {
		return fmt.Errorf("failed to save signatures: %w", err)
	}

	return nil
}
