package state

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
)

const (
	OffsetKey            = "offset"
	OperationsKey        = "operations"
	DeletedOperationsKey = "deleted_operations"
	FSMStateKey          = "fsm_state"
)

// State is the node's state (it keeps the offset, the signatures and
// the Operation pool.
type State interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
	Delete(key string) error
	Reset(stateDbPath string) (string, error)

	SaveOffset(uint64) error
	LoadOffset() (uint64, error)
}

type LevelDBState struct {
	sync.Mutex
	stateDb     *leveldb.DB
	topic       string
	stateDbPath string
}

func NewLevelDBState(stateDbPath string, topic string) (*LevelDBState, error) {
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

	// Init state key for offset bytes.
	offsetCompositeKey := MakeCompositeKey(topic, OffsetKey)
	if _, err := state.stateDb.Get(offsetCompositeKey, nil); err != nil {
		bz := make([]byte, 8)
		binary.LittleEndian.PutUint64(bz, 0)
		if err := db.Put(offsetCompositeKey, bz, nil); err != nil {
			return nil, fmt.Errorf("failed to init %s storage: %w", string(offsetCompositeKey), err)
		}
	}

	fsmStateCompositeKey := MakeCompositeKey(topic, FSMStateKey)
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
	s.stateDb = newstate.stateDb
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

	if err := s.stateDb.Put(MakeCompositeKey(s.topic, OffsetKey), bz, nil); err != nil {
		return fmt.Errorf("failed to set offset: %w", err)
	}

	return nil
}

func (s *LevelDBState) LoadOffset() (uint64, error) {
	bz, err := s.stateDb.Get(MakeCompositeKey(s.topic, OffsetKey), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to read offset: %w", err)
	}

	offset := binary.LittleEndian.Uint64(bz)

	return offset, nil
}

func MakeCompositeKey(prefix, key string) []byte {
	return []byte(fmt.Sprintf("%s_%s", prefix, key))
}

func MakeCompositeKeyString(prefix, key string) string {
	return fmt.Sprintf("%s_%s", prefix, key)
}
