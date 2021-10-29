package handlers

import (
	"github.com/lidofinance/dc4bc/client/modules/state"
	"github.com/lidofinance/dc4bc/client/services"
	"github.com/lidofinance/dc4bc/client/services/fsmservice"
	"github.com/lidofinance/dc4bc/client/services/node"
	operation_service "github.com/lidofinance/dc4bc/client/services/operation"
	signature_service "github.com/lidofinance/dc4bc/client/services/signature"
)

type HTTPApp struct {
	node      node.NodeService
	fsm       fsmservice.FSMService
	state     state.State
	operation operation_service.OperationService
	signature signature_service.SignatureService
}

func NewHTTPApp(node node.NodeService, sp *services.ServiceProvider) *HTTPApp {
	return &HTTPApp{
		node:      node,
		fsm:       sp.GetFSMService(),
		state:     sp.GetState(),
		operation: sp.GetOperationService(),
		signature: sp.GetSignatureService(),
	}
}
