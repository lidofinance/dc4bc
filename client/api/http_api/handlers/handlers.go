package handlers

import (
	"github.com/lidofinance/dc4bc/client/services/node"
)

type HTTPApp struct {
	node node.NodeService
}

func NewHTTPApp(node node.NodeService) *HTTPApp {
	return &HTTPApp{
		node: node,
	}
}
