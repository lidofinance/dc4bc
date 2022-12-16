package operation

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/lidofinance/dc4bc/client/modules/state"
	"github.com/lidofinance/dc4bc/client/types"
)

func TestPutOperation(t *testing.T) {
	var (
		req    = require.New(t)
		dbPath = "/tmp/dc4bc_test_PutOperation"
		topic  = "test_topic"
	)
	defer os.RemoveAll(dbPath)

	stg, err := state.NewLevelDBState(dbPath, topic)
	req.NoError(err)

	repo, err := NewOperationRepo(stg, topic)
	req.NoError(err)

	operation := &types.Operation{
		ID:        "operation_id",
		Type:      types.DKGCommits,
		Payload:   []byte("operation_payload"),
		CreatedAt: time.Now(),
	}
	err = repo.PutOperation(operation)
	req.NoError(err)

	loadedOperation, err := repo.GetOperationByID(operation.ID)
	req.NoError(err)
	req.Equal(operation.ID, loadedOperation.ID)
	req.Equal(operation.Type, loadedOperation.Type)
	req.Equal(operation.Payload, loadedOperation.Payload)

	err = repo.PutOperation(operation)
	req.Error(err)
}

func TestGetOperations(t *testing.T) {
	var (
		req    = require.New(t)
		dbPath = "/tmp/dc4bc_test_PutOperation"
		topic  = "test_topic"
	)
	defer os.RemoveAll(dbPath)

	stg, err := state.NewLevelDBState(dbPath, topic)
	req.NoError(err)

	repo, err := NewOperationRepo(stg, topic)
	req.NoError(err)

	operation := &types.Operation{
		ID:        "operation_1",
		Type:      types.DKGCommits,
		Payload:   []byte("operation_payload"),
		CreatedAt: time.Now(),
	}
	err = repo.PutOperation(operation)
	req.NoError(err)

	operation.ID = "operation_2"
	err = repo.PutOperation(operation)
	req.NoError(err)

	operations, err := repo.GetOperations()
	req.NoError(err)
	req.Len(operations, 2)
}

func TestDeleteOperation(t *testing.T) {
	var (
		req    = require.New(t)
		dbPath = "/tmp/dc4bc_test_DeleteOperation"
		topic  = "test_topic"
	)
	defer os.RemoveAll(dbPath)

	stg, err := state.NewLevelDBState(dbPath, topic)
	req.NoError(err)

	repo, err := NewOperationRepo(stg, topic)
	req.NoError(err)

	operation := &types.Operation{
		ID:        "operation_id",
		Type:      types.DKGCommits,
		Payload:   []byte("operation_payload"),
		CreatedAt: time.Now(),
	}
	err = repo.PutOperation(operation)
	req.NoError(err)

	_, err = repo.GetOperationByID(operation.ID)
	req.NoError(err)

	err = repo.DeleteOperation(operation)
	req.NoError(err)

	_, err = repo.GetOperationByID(operation.ID)
	req.Error(err)
}
