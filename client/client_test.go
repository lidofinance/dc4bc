package client_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/p2p-org/dc4bc/client"

	"github.com/golang/mock/gomock"
	"github.com/p2p-org/dc4bc/mocks/clientMocks"
	"github.com/p2p-org/dc4bc/mocks/storageMocks"
	"github.com/stretchr/testify/require"
)

func TestClient_GetOperationQRPath(t *testing.T) {
	var (
		ctx  = context.Background()
		req  = require.New(t)
		ctrl = gomock.NewController(t)
	)
	defer ctrl.Finish()

	state := clientMocks.NewMockState(ctrl)
	storage := storageMocks.NewMockStorage(ctrl)

	clt, err := client.NewClient(ctx, nil, state, storage)
	req.NoError(err)

	operation := &client.Operation{
		ID:        "operation_id",
		Type:      client.DKGCommits,
		Payload:   []byte("operation_payload"),
		CreatedAt: time.Now(),
	}

	state.EXPECT().GetOperationByID(operation.ID).Times(1).Return(
		nil, errors.New(""))
	_, err = clt.GetOperationQRPath(operation.ID)
	req.Error(err)
}
