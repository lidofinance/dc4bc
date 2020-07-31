package client_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/depools/dc4bc/mocks/qrMocks"

	"github.com/depools/dc4bc/client"

	"github.com/depools/dc4bc/mocks/clientMocks"
	"github.com/depools/dc4bc/mocks/storageMocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestClient_GetOperationsList(t *testing.T) {
	var (
		ctx  = context.Background()
		req  = require.New(t)
		ctrl = gomock.NewController(t)
	)
	defer ctrl.Finish()

	state := clientMocks.NewMockState(ctrl)
	storage := storageMocks.NewMockStorage(ctrl)
	qrProcessor := qrMocks.NewMockProcessor(ctrl)

	clt, err := client.NewClient(ctx, nil, state, storage, qrProcessor)
	req.NoError(err)

	state.EXPECT().GetOperations().Times(1).Return(map[string]*client.Operation{}, nil)
	operations, err := clt.GetOperations()
	req.NoError(err)
	req.Len(operations, 0)

	operation := &client.Operation{
		ID:        "operation_id",
		Type:      client.DKGCommits,
		Payload:   []byte("operation_payload"),
		CreatedAt: time.Now(),
	}
	state.EXPECT().GetOperations().Times(1).Return(
		map[string]*client.Operation{operation.ID: operation}, nil)
	operations, err = clt.GetOperations()
	req.NoError(err)
	req.Len(operations, 1)
	req.Equal(operation, operations[operation.ID])
}

func TestClient_GetOperationQRPath(t *testing.T) {
	var (
		ctx  = context.Background()
		req  = require.New(t)
		ctrl = gomock.NewController(t)
	)
	defer ctrl.Finish()

	state := clientMocks.NewMockState(ctrl)
	storage := storageMocks.NewMockStorage(ctrl)
	qrProcessor := qrMocks.NewMockProcessor(ctrl)

	clt, err := client.NewClient(ctx, nil, state, storage, qrProcessor)
	req.NoError(err)

	operation := &client.Operation{
		ID:        "operation_id",
		Type:      client.DKGCommits,
		Payload:   []byte("operation_payload"),
		CreatedAt: time.Now(),
	}

	var expectedQrPath = filepath.Join(client.QrCodesDir, operation.ID)
	defer os.Remove(expectedQrPath)

	state.EXPECT().GetOperationByID(operation.ID).Times(1).Return(
		nil, errors.New(""))
	_, err = clt.GetOperationQRPath(operation.ID)
	req.Error(err)

	state.EXPECT().GetOperationByID(operation.ID).Times(1).Return(
		operation, nil)
	qrProcessor.EXPECT().WriteQR(expectedQrPath, gomock.Any()).Times(1).Return(nil)
	qrPath, err := clt.GetOperationQRPath(operation.ID)
	req.NoError(err)
	req.Equal(expectedQrPath, qrPath)
}

func TestClient_ReadProcessedOperation(t *testing.T) {
	var (
		ctx  = context.Background()
		req  = require.New(t)
		ctrl = gomock.NewController(t)
	)
	defer ctrl.Finish()

	state := clientMocks.NewMockState(ctrl)
	storage := storageMocks.NewMockStorage(ctrl)
	qrProcessor := qrMocks.NewMockProcessor(ctrl)

	clt, err := client.NewClient(ctx, nil, state, storage, qrProcessor)
	req.NoError(err)

	operation := &client.Operation{
		ID:        "operation_id",
		Type:      client.DKGCommits,
		Payload:   []byte("operation_payload"),
		Result:    []byte("operation_result"),
		CreatedAt: time.Now(),
	}
	processedOperation := &client.Operation{
		ID:        "operation_id",
		Type:      client.DKGCommits,
		Payload:   []byte("operation_payload"),
		Result:    []byte("operation_result"),
		CreatedAt: time.Now(),
	}
	processedOperationBz, err := json.Marshal(processedOperation)
	req.NoError(err)

	qrProcessor.EXPECT().ReadQR().Return(processedOperationBz, nil).Times(1)
	state.EXPECT().GetOperationByID(processedOperation.ID).Times(1).Return(operation, nil)
	state.EXPECT().DeleteOperation(processedOperation.ID).Times(1)
	storage.EXPECT().Send(gomock.Any()).Times(1)
	err = clt.ReadProcessedOperation()
	req.NoError(err)
}
