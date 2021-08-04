package api

import (
	"github.com/gin-gonic/gin"
	"github.com/lidofinance/dc4bc/client/api/http_api"
	"github.com/lidofinance/dc4bc/client/config"
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

var (
	factoryInstance InstanceFactory
	done            chan bool
)

func Run(config *config.Config) {
	factoryInstance = InstanceFactory{
		apiFactory: &http_api.RESTApiProvider{},
	}

	err := factoryInstance.apiFactory.NewServer(config)
	if err != nil {
		os.Exit(1)
	}
	go factoryInstance.apiFactory.Start()

	<-done
}
