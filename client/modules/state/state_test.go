package state_test

import (
	"os"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/lidofinance/dc4bc/client/modules/state"

	"github.com/lidofinance/dc4bc/client/types"

	"github.com/stretchr/testify/require"
)

func TestLevelDBState_SaveOffset(t *testing.T) {
	var (
		req    = require.New(t)
		dbPath = "/tmp/dc4bc_test_SaveOffset"
		topic  = "test_topic"
	)
	defer os.RemoveAll(dbPath)

	stg, err := state.NewLevelDBState(dbPath, topic)
	req.NoError(err)

	var offset uint64 = 1
	err = stg.SaveOffset(offset)
	req.NoError(err)

	loadedOffset, err := stg.LoadOffset()
	req.NoError(err)
	req.Equal(offset, loadedOffset)
}

func TestLevelDBState_PutOperation(t *testing.T) {
	var (
		req    = require.New(t)
		dbPath = "/tmp/dc4bc_test_PutOperation"
		topic  = "test_topic"
	)
	defer os.RemoveAll(dbPath)

	stg, err := state.NewLevelDBState(dbPath, topic)
	req.NoError(err)

	operation := &types.Operation{
		ID:        "operation_id",
		Type:      types.DKGCommits,
		Payload:   []byte("operation_payload"),
		CreatedAt: time.Now(),
	}
	err = stg.PutOperation(operation)
	req.NoError(err)

	loadedOperation, err := stg.GetOperationByID(operation.ID)
	req.NoError(err)
	req.Equal(operation.ID, loadedOperation.ID)
	req.Equal(operation.Type, loadedOperation.Type)
	req.Equal(operation.Payload, loadedOperation.Payload)

	err = stg.PutOperation(operation)
	req.Error(err)
}

func TestLevelDBState_GetOperations(t *testing.T) {
	var (
		req    = require.New(t)
		dbPath = "/tmp/dc4bc_test_PutOperation"
		topic  = "test_topic"
	)
	defer os.RemoveAll(dbPath)

	stg, err := state.NewLevelDBState(dbPath, topic)
	req.NoError(err)

	operation := &types.Operation{
		ID:        "operation_1",
		Type:      types.DKGCommits,
		Payload:   []byte("operation_payload"),
		CreatedAt: time.Now(),
	}
	err = stg.PutOperation(operation)
	req.NoError(err)

	operation.ID = "operation_2"
	err = stg.PutOperation(operation)
	req.NoError(err)

	operations, err := stg.GetOperations()
	req.NoError(err)
	req.Len(operations, 2)
}

func TestLevelDBState_DeleteOperation(t *testing.T) {
	var (
		req    = require.New(t)
		dbPath = "/tmp/dc4bc_test_DeleteOperation"
		topic  = "test_topic"
	)
	defer os.RemoveAll(dbPath)

	stg, err := state.NewLevelDBState(dbPath, topic)
	req.NoError(err)

	operation := &types.Operation{
		ID:        "operation_id",
		Type:      types.DKGCommits,
		Payload:   []byte("operation_payload"),
		CreatedAt: time.Now(),
	}
	err = stg.PutOperation(operation)
	req.NoError(err)

	_, err = stg.GetOperationByID(operation.ID)
	req.NoError(err)

	err = stg.DeleteOperation(operation)
	req.NoError(err)

	_, err = stg.GetOperationByID(operation.ID)
	req.Error(err)
}

func TestLevelDBState_NewStateFromOld(t *testing.T) {
	var (
		req    = require.New(t)
		dbPath = "/tmp/dc4bc_test_NewStateFromOld"
		topic  = "test_topic"
		re     = regexp.MustCompile(dbPath + `_(?P<ts>\d+)`)
	)
	defer os.RemoveAll(dbPath)

	state, err := state.NewLevelDBState(dbPath, topic)
	req.NoError(err)

	var offset uint64 = 1
	err = state.SaveOffset(offset)
	req.NoError(err)

	loadedOffset, err := state.LoadOffset()
	req.NoError(err)
	req.Equal(offset, loadedOffset)

	timeBefore := time.Now().Unix()
	newState, newStateDbPath, err := state.NewStateFromOld("")
	timeAfter := time.Now().Unix()

	req.NoError(err)

	submatches := re.FindStringSubmatch(newStateDbPath)
	req.Greater(len(submatches), 0)

	ts, err := strconv.Atoi(submatches[1])
	req.NoError(err)
	req.GreaterOrEqual(int64(ts), timeBefore)
	req.LessOrEqual(int64(ts), timeAfter)

	newLoadedOffset, err := newState.LoadOffset()
	req.NoError(err)
	req.NotEqual(newLoadedOffset, loadedOffset)
}
