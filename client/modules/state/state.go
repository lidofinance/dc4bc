package state

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/lidofinance/dc4bc/client/types"

	"github.com/syndtr/goleveldb/leveldb"
)

const (
	OffsetKey            = "offset"
	OperationsKey        = "operations"
	DeletedOperationsKey = "deleted_operations"
	FSMStateKey          = "fsm_state"
	SignaturesKeyPrefix  = "signatures"
)

func makeCompositeKey(prefix, key string) []byte {
	return []byte(fmt.Sprintf("%s_%s", prefix, key))
}

// State is the node's state (it keeps the offset, the signatures and
// the Operation pool.
type State interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
	Delete(key string) error
	Reset(stateDbPath string) (string,error)

	SaveOffset(uint64) error
	LoadOffset() (uint64, error)

	PutOperation(operation *types.Operation) error
	DeleteOperation(operation *types.Operation) error
	GetOperations() (map[string]*types.Operation, error)
	GetOperationByID(operationID string) (*types.Operation, error)

	SaveSignatures(signature []types.ReconstructedSignature) error
	GetSignatureByID(dkgID, signatureID string) ([]types.ReconstructedSignature, error)
	GetSignatures(dkgID string) (map[string][]types.ReconstructedSignature, error)
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

	// TODO remove storage preinitialization after "service" methods moved out from state interface

	// Init state key for operations JSON.
	operationsCompositeKey := makeCompositeKey(topic, OperationsKey)
	if _, err := state.stateDb.Get(operationsCompositeKey, nil); err != nil {
		if err := state.initJsonKey(operationsCompositeKey, map[string]*types.Operation{}); err != nil {
			return nil, fmt.Errorf("failed to init %s storage: %w", string(operationsCompositeKey), err)
		}
	}

	// Init state key for operations JSON.
	deleteOperationsCompositeKey := makeCompositeKey(topic, DeletedOperationsKey)
	if _, err := state.stateDb.Get(deleteOperationsCompositeKey, nil); err != nil {
		if err := state.initJsonKey(deleteOperationsCompositeKey, map[string]*types.Operation{}); err != nil {
			return nil, fmt.Errorf("failed to init %s storage: %w", string(deleteOperationsCompositeKey), err)
		}
	}

	// Init state key for offset bytes.
	offsetCompositeKey := makeCompositeKey(topic, OffsetKey)
	if _, err := state.stateDb.Get(offsetCompositeKey, nil); err != nil {
		bz := make([]byte, 8)
		binary.LittleEndian.PutUint64(bz, 0)
		if err := db.Put(offsetCompositeKey, bz, nil); err != nil {
			return nil, fmt.Errorf("failed to init %s storage: %w", string(offsetCompositeKey), err)
		}
	}

	fsmStateCompositeKey := makeCompositeKey(topic, FSMStateKey)
	if _, err := state.stateDb.Get(fsmStateCompositeKey, nil); err != nil {
		if err := db.Put(fsmStateCompositeKey, []byte{}, nil); err != nil {
			return nil, fmt.Errorf("failed to init %s storage: %w", string(fsmStateCompositeKey), err)
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
			return fmt.Errorf("failed to marshal storage structure: %w", err)
		}
		err = s.stateDb.Put([]byte(key), operationsBz, nil)
		if err != nil {
			return fmt.Errorf("failed to init state: %w", err)
		}
	}

	return nil
}

// Reset creates new underlying leveldb storage
func (s *LevelDBState) Reset(stateDbPath string) (string, error) {
	s.Lock()
	defer s.Unlock()

	if len(stateDbPath) < 1 {
		stateDbPath = fmt.Sprintf("%s_%d", s.stateDbPath, time.Now().Unix())
	}

	newstate, err := NewLevelDBState(stateDbPath, s.topic)
	if err != nil {
		return stateDbPath, fmt.Errorf("failed to open stateDB: %w", err)
	}
	s.stateDb = newstate.(*LevelDBState).stateDb
	s.stateDbPath = stateDbPath

	return stateDbPath, err
}

func (s *LevelDBState) Get(key string) ([]byte, error) {
	s.Lock()
	defer s.Unlock()
	var (
		value []byte
		err   error
	)
	if value, err = s.stateDb.Get([]byte(key), nil); err != nil && !errors.Is(leveldb.ErrNotFound, err) {
		return nil, fmt.Errorf("failed to get value with key {%s} from leveldb storage: %w", key, err)
	}
	return value, nil
}

func (s *LevelDBState) Set(key string, value []byte) error {
	s.Lock()
	defer s.Unlock()
	if err := s.stateDb.Put([]byte(key), value, nil); err != nil {
		return fmt.Errorf("failed to save value with key %s: %w", key, err)
	}
	return nil
}

func (s *LevelDBState) Delete(key string) error {
	s.Lock()
	defer s.Unlock()

	err := s.stateDb.Delete([]byte(key), nil)
	if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
		return fmt.Errorf("failed to delete value with key {%s}: %w", key, err)
	}
	return nil
}

func (s *LevelDBState) SaveOffset(offset uint64) error {
	bz := make([]byte, 8)
	binary.LittleEndian.PutUint64(bz, offset)

	if err := s.stateDb.Put(makeCompositeKey(s.topic, OffsetKey), bz, nil); err != nil {
		return fmt.Errorf("failed to set offset: %w", err)
	}

	return nil
}

func (s *LevelDBState) LoadOffset() (uint64, error) {
	bz, err := s.stateDb.Get(makeCompositeKey(s.topic, OffsetKey), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to read offset: %w", err)
	}

	offset := binary.LittleEndian.Uint64(bz)

	return offset, nil
}

func (s *LevelDBState) PutOperation(operation *types.Operation) error {
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

	if err := s.stateDb.Put(makeCompositeKey(s.topic, OperationsKey), operationsJSON, nil); err != nil {
		return fmt.Errorf("failed to put operations: %w", err)
	}

	return nil
}

// DeleteOperation deletes operation from an operation pool
func (s *LevelDBState) DeleteOperation(operation *types.Operation) error {
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

	if err := s.stateDb.Put(makeCompositeKey(s.topic, DeletedOperationsKey), deletedOperationsJSON, nil); err != nil {
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

	if err := s.stateDb.Put(makeCompositeKey(s.topic, OperationsKey), operationsJSON, nil); err != nil {
		return fmt.Errorf("failed to put operations: %w", err)
	}

	return nil
}

// GetOperations returns all operations from an operation pool
func (s *LevelDBState) GetOperations() (map[string]*types.Operation, error) {
	s.Lock()
	defer s.Unlock()

	return s.getOperations()
}

func (s *LevelDBState) GetOperationByID(operationID string) (*types.Operation, error) {
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

func (s *LevelDBState) getOperations() (map[string]*types.Operation, error) {
	deletedOperations, err := s.getDeletedOperations()
	if err != nil {
		return nil, fmt.Errorf("failed to getDeletedOperations: %w", err)
	}

	operationsCompositeKey := makeCompositeKey(s.topic, OperationsKey)
	bz, err := s.stateDb.Get(operationsCompositeKey, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get Operations (key: %s): %w", string(operationsCompositeKey), err)
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

func (s *LevelDBState) getDeletedOperations() (map[string]*types.Operation, error) {
	deletedOperationsCompositeKey := makeCompositeKey(s.topic, DeletedOperationsKey)
	bz, err := s.stateDb.Get(deletedOperationsCompositeKey, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get deleted Operations (key: %s): %w", string(deletedOperationsCompositeKey), err)
	}

	var operations map[string]*types.Operation
	if err := json.Unmarshal(bz, &operations); err != nil {
		return nil, fmt.Errorf("failed to unmarshal deleted Operations: %w", err)
	}

	return operations, nil
}

func (s *LevelDBState) getSignatures(dkgID string) (map[string][]types.ReconstructedSignature, error) {
	bz, err := s.stateDb.Get(makeCompositeKey(SignaturesKeyPrefix, dkgID), nil)
	if err != nil {
		if errors.Is(err, leveldb.ErrNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get signatures for dkgID %s: %w", dkgID, err)
	}

	var signatures map[string][]types.ReconstructedSignature
	if err := json.Unmarshal(bz, &signatures); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Operations: %w", err)
	}
	return signatures, nil
}

func (s *LevelDBState) GetSignatures(dkgID string) (map[string][]types.ReconstructedSignature, error) {
	s.Lock()
	defer s.Unlock()

	return s.getSignatures(dkgID)
}

func (s *LevelDBState) GetSignatureByID(dkgID, signatureID string) ([]types.ReconstructedSignature, error) {
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

func (s *LevelDBState) SaveSignatures(signaturesToSave []types.ReconstructedSignature) error {
	s.Lock()
	defer s.Unlock()

	if len(signaturesToSave) == 0 {
		return errors.New("nothing to save")
	}

	signatures, err := s.getSignatures(signaturesToSave[0].DKGRoundID)
	if err != nil {
		return fmt.Errorf("failed to getSignatures: %w", err)
	}
	if signatures == nil {
		signatures = make(map[string][]types.ReconstructedSignature)
	}

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
	}

	signaturesJSON, err := json.Marshal(signatures)
	if err != nil {
		return fmt.Errorf("failed to marshal signatures: %w", err)
	}

	if err := s.stateDb.Put(makeCompositeKey(SignaturesKeyPrefix, signaturesToSave[0].DKGRoundID), signaturesJSON, nil); err != nil {
		return fmt.Errorf("failed to save signatures: %w", err)
	}

	return nil
}
