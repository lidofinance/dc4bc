package fsmservice

import (
	"errors"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/lidofinance/dc4bc/client/api/dto"
	"github.com/lidofinance/dc4bc/mocks/clientMocks"
	"github.com/lidofinance/dc4bc/mocks/storageMocks"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestClient_ResetState(t *testing.T) {
	var (
		req  = require.New(t)
		ctrl = gomock.NewController(t)
	)
	defer ctrl.Finish()

	userName := "test_client"

	state := clientMocks.NewMockState(ctrl)
	stg := storageMocks.NewMockStorage(ctrl)
	fsm := NewFSMService(state, stg, "")

	resetReq := dto.ResetStateDTO{
		NewStateDBDSN:      "./dc4bc_client_state_new",
		UseOffset:          true,
		KafkaConsumerGroup: fmt.Sprintf("%s_%d", userName, time.Now().Unix()),
		Messages:           []string{"11", "12"},
	}

	stg.EXPECT().IgnoreMessages(resetReq.Messages, resetReq.UseOffset).Times(1).Return(errors.New(""))
	_, err := fsm.ResetFSMState(&resetReq)
	req.Error(err)

	stg.EXPECT().IgnoreMessages(resetReq.Messages, resetReq.UseOffset).Times(1).Return(nil)
	state.EXPECT().Reset(resetReq.NewStateDBDSN).Times(1).Return(resetReq.NewStateDBDSN, nil)
	newStatePath, err := fsm.ResetFSMState(&resetReq)
	req.NoError(err)
	req.Equal(newStatePath, resetReq.NewStateDBDSN)
}
