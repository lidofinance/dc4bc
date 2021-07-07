package client_test

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	operations2 "github.com/lidofinance/dc4bc/client/operations"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/lidofinance/dc4bc/client"
	"github.com/lidofinance/dc4bc/fsm/state_machines"
	spf "github.com/lidofinance/dc4bc/fsm/state_machines/signature_proposal_fsm"
	"github.com/lidofinance/dc4bc/fsm/types/requests"
	"github.com/lidofinance/dc4bc/mocks/clientMocks"
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

	testClientKeyPair := client.NewKeyPair()
	keyStore.EXPECT().LoadKeys(userName, "").Times(1).Return(testClientKeyPair, nil)

	clt, err := client.NewClient(
		ctx,
		userName,
		state,
		stg,
		keyStore,
	)
	req.NoError(err)

	t.Run("test_process_dkg_init", func(t *testing.T) {
		fsm, err := state_machines.Create(dkgRoundID)
		req.NoError(err)
		state.EXPECT().LoadFSM(dkgRoundID).Times(1).Return(fsm, true, nil)

		senderKeyPair := client.NewKeyPair()
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
					PubKey:    client.NewKeyPair().Pub,
					DkgPubKey: make([]byte, 128),
				},
				{
					Username:  "222",
					PubKey:    client.NewKeyPair().Pub,
					DkgPubKey: make([]byte, 128),
				},
				{
					Username:  "333",
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
			SenderAddr: senderAddr,
		}
		message.Signature = ed25519.Sign(senderKeyPair.Priv, message.Bytes())

		state.EXPECT().SaveFSM(gomock.Any(), gomock.Any()).Times(1).Return(nil)
		state.EXPECT().PutOperation(gomock.Any()).Times(1).Return(nil)

		baseClient := clt.(*client.BaseClient)
		err = baseClient.ProcessMessage(message)

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
	testClientKeyPair := client.NewKeyPair()
	keyStore.EXPECT().LoadKeys(userName, "").Times(1).Return(testClientKeyPair, nil)

	state := clientMocks.NewMockState(ctrl)
	stg := storageMocks.NewMockStorage(ctrl)

	clt, err := client.NewClient(
		ctx,
		userName,
		state,
		stg,
		keyStore,
	)
	req.NoError(err)

	state.EXPECT().GetOperations().Times(1).Return(map[string]*operations2.Operation{}, nil)
	operations, err := clt.GetState().GetOperations()
	req.NoError(err)
	req.Len(operations, 0)

	operation := &operations2.Operation{
		ID:        "operation_id",
		Type:      operations2.DKGCommits,
		Payload:   []byte("operation_payload"),
		CreatedAt: time.Now(),
	}
	state.EXPECT().GetOperations().Times(1).Return(
		map[string]*operations2.Operation{operation.ID: operation}, nil)
	operations, err = clt.GetState().GetOperations()
	req.NoError(err)
	req.Len(operations, 1)
	req.Equal(operation, operations[operation.ID])
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
	testClientKeyPair := client.NewKeyPair()
	keyStore.EXPECT().LoadKeys(userName, "").Times(1).Return(testClientKeyPair, nil)

	state := clientMocks.NewMockState(ctrl)
	stg := storageMocks.NewMockStorage(ctrl)

	clt, err := client.NewClient(
		ctx,
		userName,
		state,
		stg,
		keyStore,
	)
	req.NoError(err)

	var (
		newStateDBDSN      = "./dc4bc_client_state_new"
		useOffset          = true
		kafkaConsumerGroup = fmt.Sprintf("%s_%d", userName, time.Now().Unix())
		messages           = []string{"11", "12"}
	)

	stg.EXPECT().IgnoreMessages(messages, useOffset).Times(1).Return(errors.New(""))
	_, err = clt.ResetState(newStateDBDSN, kafkaConsumerGroup, messages, useOffset)
	req.Error(err)

	stg.EXPECT().IgnoreMessages(messages, useOffset).Times(1).Return(nil)
	state.EXPECT().NewStateFromOld(newStateDBDSN).Times(1).Return(state, newStateDBDSN, nil)
	newStatePath, err := clt.ResetState(newStateDBDSN, kafkaConsumerGroup, messages, useOffset)
	req.NoError(err)
	req.Equal(newStatePath, newStateDBDSN)
}
