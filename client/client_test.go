package client_test

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/lidofinance/dc4bc/client/modules/keystore"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/lidofinance/dc4bc/client"
	"github.com/lidofinance/dc4bc/client/types"
	"github.com/lidofinance/dc4bc/fsm/state_machines"
	spf "github.com/lidofinance/dc4bc/fsm/state_machines/signature_proposal_fsm"
	"github.com/lidofinance/dc4bc/fsm/types/requests"
	"github.com/lidofinance/dc4bc/mocks/clientMocks"
	"github.com/lidofinance/dc4bc/mocks/qrMocks"
	"github.com/lidofinance/dc4bc/mocks/storageMocks"
	"github.com/lidofinance/dc4bc/storage"
	"github.com/stretchr/testify/require"
)

func TestClient_ProcessMessage(t *testing.T) {
	var (
		ctx  = context.Background()
		req  = require.New(t)
		ctrl = gomock.NewController(t)
	)
	defer ctrl.Finish()

	userName := "user_name"
	dkgRoundID := "dkg_round_id"
	state := clientMocks.NewMockState(ctrl)
	keyStore := clientMocks.NewMockKeyStore(ctrl)
	stg := storageMocks.NewMockStorage(ctrl)
	qrProcessor := qrMocks.NewMockProcessor(ctrl)

	testClientKeyPair := keystore.NewKeyPair()
	keyStore.EXPECT().LoadKeys(userName, "").Times(1).Return(testClientKeyPair, nil)

	clt, err := client.NewClient(
		ctx,
		userName,
		state,
		stg,
		keyStore,
		qrProcessor,
	)
	req.NoError(err)

	t.Run("test_process_dkg_init", func(t *testing.T) {
		fsm, err := state_machines.Create(dkgRoundID)
		req.NoError(err)
		state.EXPECT().LoadFSM(dkgRoundID).Times(1).Return(fsm, true, nil)

		senderKeyPair := keystore.NewKeyPair()
		senderAddr := senderKeyPair.GetAddr()
		messageData := requests.SignatureProposalParticipantsListRequest{
			Participants: []*requests.SignatureProposalParticipantsEntry{
				{
					Username:  senderAddr,
					PubKey:    senderKeyPair.Pub,
					DkgPubKey: make([]byte, 128),
				},
				{
					Username:  "111",
					PubKey:    keystore.NewKeyPair().Pub,
					DkgPubKey: make([]byte, 128),
				},
				{
					Username:  "222",
					PubKey:    keystore.NewKeyPair().Pub,
					DkgPubKey: make([]byte, 128),
				},
				{
					Username:  "333",
					PubKey:    keystore.NewKeyPair().Pub,
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
			SenderAddr: senderAddr,
		}
		message.Signature = ed25519.Sign(senderKeyPair.Priv, message.Bytes())

		state.EXPECT().SaveFSM(gomock.Any(), gomock.Any()).Times(1).Return(nil)
		state.EXPECT().PutOperation(gomock.Any()).Times(1).Return(nil)

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

	userName := "test_client"

	keyStore := clientMocks.NewMockKeyStore(ctrl)
	testClientKeyPair := keystore.NewKeyPair()
	keyStore.EXPECT().LoadKeys(userName, "").Times(1).Return(testClientKeyPair, nil)

	state := clientMocks.NewMockState(ctrl)
	stg := storageMocks.NewMockStorage(ctrl)
	qrProcessor := qrMocks.NewMockProcessor(ctrl)

	clt, err := client.NewClient(
		ctx,
		userName,
		state,
		stg,
		keyStore,
		qrProcessor,
	)
	req.NoError(err)

	state.EXPECT().GetOperations().Times(1).Return(map[string]*types.Operation{}, nil)
	operations, err := clt.GetOperations()
	req.NoError(err)
	req.Len(operations, 0)

	operation := &types.Operation{
		ID:        "operation_id",
		Type:      types.DKGCommits,
		Payload:   []byte("operation_payload"),
		CreatedAt: time.Now(),
	}
	state.EXPECT().GetOperations().Times(1).Return(
		map[string]*types.Operation{operation.ID: operation}, nil)
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

	userName := "test_client"

	keyStore := clientMocks.NewMockKeyStore(ctrl)
	testClientKeyPair := keystore.NewKeyPair()
	keyStore.EXPECT().LoadKeys(userName, "").Times(1).Return(testClientKeyPair, nil)

	state := clientMocks.NewMockState(ctrl)
	stg := storageMocks.NewMockStorage(ctrl)
	qrProcessor := qrMocks.NewMockProcessor(ctrl)

	clt, err := client.NewClient(
		ctx,
		userName,
		state,
		stg,
		keyStore,
		qrProcessor,
	)
	req.NoError(err)

	operation := &types.Operation{
		ID:        "operation_id",
		Type:      types.DKGCommits,
		Payload:   []byte("operation_payload"),
		CreatedAt: time.Now(),
	}

	var expectedQrPath = filepath.Join(client.QrCodesDir, fmt.Sprintf("dc4bc_qr_%s.gif", operation.ID))
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

func TestClient_ResetState(t *testing.T) {
	var (
		ctx  = context.Background()
		req  = require.New(t)
		ctrl = gomock.NewController(t)
	)
	defer ctrl.Finish()

	userName := "test_client"

	keyStore := clientMocks.NewMockKeyStore(ctrl)
	testClientKeyPair := keystore.NewKeyPair()
	keyStore.EXPECT().LoadKeys(userName, "").Times(1).Return(testClientKeyPair, nil)

	state := clientMocks.NewMockState(ctrl)
	stg := storageMocks.NewMockStorage(ctrl)
	qrProcessor := qrMocks.NewMockProcessor(ctrl)

	clt, err := client.NewClient(
		ctx,
		userName,
		state,
		stg,
		keyStore,
		qrProcessor,
	)
	req.NoError(err)

	resetReq := &client.ResetStateRequest{
		NewStateDBDSN:      "./dc4bc_client_state_new",
		UseOffset:          true,
		KafkaConsumerGroup: fmt.Sprintf("%s_%d", userName, time.Now().Unix()),
		Messages:           []string{"11", "12"},
	}

	stg.EXPECT().IgnoreMessages(resetReq.Messages, resetReq.UseOffset).Times(1).Return(errors.New(""))
	_, err = clt.ResetState(resetReq.NewStateDBDSN, resetReq.KafkaConsumerGroup, resetReq.Messages, resetReq.UseOffset)
	req.Error(err)

	stg.EXPECT().IgnoreMessages(resetReq.Messages, resetReq.UseOffset).Times(1).Return(nil)
	state.EXPECT().NewStateFromOld(resetReq.NewStateDBDSN).Times(1).Return(state, resetReq.NewStateDBDSN, nil)
	newStatePath, err := clt.ResetState(resetReq.NewStateDBDSN, resetReq.KafkaConsumerGroup, resetReq.Messages, resetReq.UseOffset)
	req.NoError(err)
	req.Equal(newStatePath, resetReq.NewStateDBDSN)
}
