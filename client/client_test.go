package client_test

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/depools/dc4bc/client"
	"github.com/depools/dc4bc/fsm/state_machines"
	spf "github.com/depools/dc4bc/fsm/state_machines/signature_proposal_fsm"
	"github.com/depools/dc4bc/fsm/types/requests"
	"github.com/depools/dc4bc/mocks/clientMocks"
	"github.com/depools/dc4bc/mocks/qrMocks"
	"github.com/depools/dc4bc/mocks/storageMocks"
	"github.com/depools/dc4bc/storage"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestClient_ProcessMessage(t *testing.T) {
	var (
		ctx  = context.Background()
		req  = require.New(t)
		ctrl = gomock.NewController(t)
	)
	defer ctrl.Finish()

	dkgRoundID := "dkg_round_id"
	state := clientMocks.NewMockState(ctrl)
	keyStore := clientMocks.NewMockKeyStore(ctrl)
	stg := storageMocks.NewMockStorage(ctrl)
	qrProcessor := qrMocks.NewMockProcessor(ctrl)

	testClientKeyPair := client.NewKeyPair()
	keyStore.EXPECT().LoadKeys("test_client", "").Times(1).Return(testClientKeyPair, nil)

	fsm, err := state_machines.Create(dkgRoundID)
	state.EXPECT().LoadFSM(dkgRoundID).Times(1).Return(fsm, true, nil)

	clt, err := client.NewClient(
		ctx,
		"test_client",
		state,
		stg,
		keyStore,
		qrProcessor,
	)
	req.NoError(err)

	t.Run("test_process_init_dkg", func(t *testing.T) {
		senderUserName := "sender_username"
		senderKeyPair := client.NewKeyPair()
		messageData := requests.SignatureProposalParticipantsListRequest{
			Participants: []*requests.SignatureProposalParticipantsEntry{
				{
					Addr:      senderUserName,
					PubKey:    senderKeyPair.Pub,
					DkgPubKey: make([]byte, 128),
				},
				{
					Addr:      "111",
					PubKey:    client.NewKeyPair().Pub,
					DkgPubKey: make([]byte, 128),
				},
				{
					Addr:      "222",
					PubKey:    client.NewKeyPair().Pub,
					DkgPubKey: make([]byte, 128),
				},
				{
					Addr:      "333",
					PubKey:    client.NewKeyPair().Pub,
					DkgPubKey: make([]byte, 128),
				},
			},
			CreatedAt:        time.Now(),
			SigningThreshold: 2,
		}
		messageDataBz, err := json.Marshal(messageData)
		req.NoError(err)

		message := storage.Message{
			ID:         uuid.New().String(),
			DkgRoundID: dkgRoundID,
			Offset:     1,
			Event:      string(spf.EventInitProposal),
			Data:       messageDataBz,
			SenderAddr: senderUserName,
		}
		message.Signature = ed25519.Sign(senderKeyPair.Priv, message.Bytes())

		state.EXPECT().SaveOffset(uint64(1)).Times(1).Return(nil)
		state.EXPECT().SaveFSM(gomock.Any(), gomock.Any()).Times(1).Return(nil)

		err = clt.ProcessMessage(message)
		req.NoError(err)
	})
}

func TestClient_GetOperationsList(t *testing.T) {
	var (
		ctx  = context.Background()
		req  = require.New(t)
		ctrl = gomock.NewController(t)
	)
	defer ctrl.Finish()

	state := clientMocks.NewMockState(ctrl)
	keyStore := clientMocks.NewMockKeyStore(ctrl)
	stg := storageMocks.NewMockStorage(ctrl)
	qrProcessor := qrMocks.NewMockProcessor(ctrl)

	clt, err := client.NewClient(
		ctx,
		"test_client",
		state,
		stg,
		keyStore,
		qrProcessor,
	)
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
	keyStore := clientMocks.NewMockKeyStore(ctrl)
	stg := storageMocks.NewMockStorage(ctrl)
	qrProcessor := qrMocks.NewMockProcessor(ctrl)

	clt, err := client.NewClient(
		ctx,
		"test_client",
		state,
		stg,
		keyStore,
		qrProcessor,
	)
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
	keyStore := clientMocks.NewMockKeyStore(ctrl)
	stg := storageMocks.NewMockStorage(ctrl)
	qrProcessor := qrMocks.NewMockProcessor(ctrl)

	clt, err := client.NewClient(
		ctx,
		"test_client",
		state,
		stg,
		keyStore,
		qrProcessor,
	)
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
	stg.EXPECT().Send(gomock.Any()).Times(1)

	keyPair := client.NewKeyPair()
	keyStore.EXPECT().LoadKeys("test_client", "").Times(1).Return(keyPair, nil)

	err = clt.ReadProcessedOperation()
	req.NoError(err)
}
