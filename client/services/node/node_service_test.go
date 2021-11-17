package node

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"testing"
	"time"

	"github.com/lidofinance/dc4bc/mocks/serviceMocks"

	"github.com/google/uuid"

	"github.com/lidofinance/dc4bc/client/config"
	"github.com/lidofinance/dc4bc/client/modules/keystore"
	"github.com/lidofinance/dc4bc/client/modules/logger"
	"github.com/lidofinance/dc4bc/client/services"
	"github.com/lidofinance/dc4bc/fsm/state_machines"
	spf "github.com/lidofinance/dc4bc/fsm/state_machines/signature_proposal_fsm"
	"github.com/lidofinance/dc4bc/fsm/types/requests"
	"github.com/lidofinance/dc4bc/storage"

	"github.com/golang/mock/gomock"
	"github.com/lidofinance/dc4bc/mocks/clientMocks"
	"github.com/lidofinance/dc4bc/mocks/storageMocks"
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
	fsmService := serviceMocks.NewMockFSMService(ctrl)
	opService := serviceMocks.NewMockOperationService(ctrl)

	testClientKeyPair := keystore.NewKeyPair()
	keyStore.EXPECT().LoadKeys(userName, "").Times(1).Return(testClientKeyPair, nil)

	opService.EXPECT().PutOperation(gomock.Any()).Times(1).Return(nil)

	sp := services.ServiceProvider{}
	sp.SetLogger(logger.NewLogger(userName))
	sp.SetState(state)
	sp.SetKeyStore(keyStore)
	sp.SetStorage(stg)
	sp.SetFSMService(fsmService)
	sp.SetOperationService(opService)

	// minimal config to make test
	cfg := config.Config{
		Username: userName,
		KafkaStorageConfig: &config.KafkaStorageConfig{
			Topic: "topic",
		},
	}

	clt, err := NewNode(ctx, &cfg, &sp)
	req.NoError(err)

	t.Run("test_process_dkg_init", func(t *testing.T) {
		fsm, err := state_machines.Create(dkgRoundID)
		req.NoError(err)
		fsmService.EXPECT().GetFSMInstance(dkgRoundID).Times(1).Return(fsm, nil)

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

		fsmService.EXPECT().SaveFSM(gomock.Any(), gomock.Any()).Times(1).Return(nil)

		err = clt.ProcessMessage(message)
		req.NoError(err)
	})
}


//func TestClient_GetOperationsList(t *testing.T) {
//	var (
//		ctx  = context.Background()
//		req  = require.New(t)
//		ctrl = gomock.NewController(t)
//	)
//	defer ctrl.Finish()
//
//	userName := "test_client"
//
//	keyStore := clientMocks.NewMockKeyStore(ctrl)
//	testClientKeyPair := keystore.NewKeyPair()
//	keyStore.EXPECT().LoadKeys(userName, "").Times(1).Return(testClientKeyPair, nil)
//
//	state := clientMocks.NewMockState(ctrl)
//	stg := storageMocks.NewMockStorage(ctrl)
//
//
//	sp := services.ServiceProvider{}
//	sp.SetLogger(logger.NewLogger(userName))
//	sp.SetState(state)
//	sp.SetKeyStore(keyStore)
//	sp.SetStorage(stg)
//
//
//	// minimal config to make test
//	cfg := config.Config{
//		Username: userName,
//	}
//
//	clt, err := NewNode(ctx, &cfg, &sp)
//	req.NoError(err)
//
//
//
//
//
//	operation := &types.Operation{
//		ID:        "operation_id",
//		Type:      types.DKGCommits,
//		Payload:   []byte("operation_payload"),
//		CreatedAt: time.Now(),
//	}
//
//}
