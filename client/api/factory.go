package api

import (
	"github.com/gin-gonic/gin"
	"github.com/lidofinance/dc4bc/client/api/http_api"
	"github.com/lidofinance/dc4bc/client/config"
	"github.com/lidofinance/dc4bc/client/services/node"
	"os"
)

type IServerAbstractFactory interface {
	NewServer(router *gin.Engine, config *config.Config) error
	Start() error
	Stop()
}

type InstanceFactory struct {
	apiFactory *http_api.RESTApiProvider
}

func Run(config *config.Config, node node.NodeService) {
	var (
		factoryInstance InstanceFactory
		done            chan bool
	)
	factoryInstance = InstanceFactory{
		apiFactory: &http_api.RESTApiProvider{},
	}

	err := factoryInstance.apiFactory.NewServer(config, node)
	if err != nil {
		os.Exit(1)
	}
	go factoryInstance.apiFactory.Start()

	<-done
}
