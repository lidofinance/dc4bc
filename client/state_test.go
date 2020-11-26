package client_test

import (
	"os"
	"testing"
	"time"

	"github.com/lidofinance/dc4bc/client/types"

	"github.com/lidofinance/dc4bc/client"
	"github.com/stretchr/testify/require"
)

func TestLevelDBState_SaveOffset(t *testing.T) {
	var (
		req    = require.New(t)
		dbPath = "/tmp/dc4bc_test_SaveOffset"
		topic  = "test_topic"
	)
	defer os.RemoveAll(dbPath)

	stg, err := client.NewLevelDBState(dbPath, topic)
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

	stg, err := client.NewLevelDBState(dbPath, topic)
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

	stg, err := client.NewLevelDBState(dbPath, topic)
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

	stg, err := client.NewLevelDBState(dbPath, topic)
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

	err = stg.DeleteOperation(operation.ID)
	req.NoError(err)

	_, err = stg.GetOperationByID(operation.ID)
	req.Error(err)
}
