package client

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/depools/dc4bc/client/types"

	"github.com/depools/dc4bc/fsm/state_machines"

	"github.com/syndtr/goleveldb/leveldb"
)

const (
	offsetKey           = "offset"
	operationsKey       = "operations"
	fsmStateKey         = "fsm_state"
	signaturesKeyPrefix = "signatures"
)

type State interface {
	SaveOffset(uint64) error
	LoadOffset() (uint64, error)

	SaveFSM(dkgRoundID string, dump []byte) error
	LoadFSM(dkgRoundID string) (*state_machines.FSMInstance, bool, error)

	PutOperation(operation *types.Operation) error
	DeleteOperation(operationID string) error
	GetOperations() (map[string]*types.Operation, error)
	GetOperationByID(operationID string) (*types.Operation, error)

	SaveSignature(signature types.ReconstructedSignature) error
	GetSignatureByDataHash(dkgID, signatureID string) ([]types.ReconstructedSignature, error)
	GetSignatures(dkgID string) (map[string][]types.ReconstructedSignature, error)
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
		if err := state.initJsonKey(operationsKey, map[string]*types.Operation{}); err != nil {
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

	if _, err := state.stateDb.Get([]byte(fsmStateKey), nil); err != nil {
		if err := db.Put([]byte(fsmStateKey), []byte{}, nil); err != nil {
			return nil, fmt.Errorf("failed to init %s storage: %w", offsetKey, err)
		}
	}

	return state, nil
}

func (s *LevelDBState) initJsonKey(key string, data interface{}) error {
	if _, err := s.stateDb.Get([]byte(key), nil); err != nil {
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

	if err := s.stateDb.Put([]byte(fsmStateKey), fsmInstancesBz, nil); err != nil {
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

func (s *LevelDBState) PutOperation(operation *types.Operation) error {
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

// DeleteOperation deletes operation from an operation pool
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
	bz, err := s.stateDb.Get([]byte(operationsKey), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get Operations (key: %s): %w", operationsKey, err)
	}

	var operations map[string]*types.Operation
	if err := json.Unmarshal(bz, &operations); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Operations: %w", err)
	}

	return operations, nil
}

func makeSignatureKey(dkgID string) []byte {
	return []byte(fmt.Sprintf("%s_%s", signaturesKeyPrefix, dkgID))
}

func (s *LevelDBState) getSignatures(dkgID string) (map[string][]types.ReconstructedSignature, error) {
	bz, err := s.stateDb.Get(makeSignatureKey(dkgID), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
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

func (s *LevelDBState) GetSignatureByDataHash(dkgID, signatureID string) ([]types.ReconstructedSignature, error) {
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

func (s *LevelDBState) SaveSignature(signature types.ReconstructedSignature) error {
	s.Lock()
	defer s.Unlock()

	signatures, err := s.getSignatures(signature.DKGRoundID)
	if err != nil {
		return fmt.Errorf("failed to getSignatures: %w", err)
	}
	if signatures == nil {
		signatures = make(map[string][]types.ReconstructedSignature)
	}

	dataHash := md5.Sum(signature.Data)
	dataHashString := hex.EncodeToString(dataHash[:])

	sig := signatures[dataHashString]
	sig = append(sig, signature)
	signatures[dataHashString] = sig

	signaturesJSON, err := json.Marshal(signatures)
	if err != nil {
		return fmt.Errorf("failed to marshal signatures: %w", err)
	}

	if err := s.stateDb.Put(makeSignatureKey(signature.DKGRoundID), signaturesJSON, nil); err != nil {
		return fmt.Errorf("failed to save signatures: %w", err)
	}

	return nil
}
