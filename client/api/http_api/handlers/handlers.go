package handlers

import (
	"github.com/lidofinance/dc4bc/client/modules/state"
	"github.com/lidofinance/dc4bc/client/services"
	"github.com/lidofinance/dc4bc/client/services/fsmservice"
	"github.com/lidofinance/dc4bc/client/services/node"
)

type HTTPApp struct {
	node  node.NodeService
	fsm   fsmservice.FSMService
	state state.State
}

func NewHTTPApp(node node.NodeService, sp *services.ServiceProvider) *HTTPApp {
	return &HTTPApp{
		node:  node,
		fsm:   sp.GetFSMService(),
		state: sp.GetState(),
	}
}
