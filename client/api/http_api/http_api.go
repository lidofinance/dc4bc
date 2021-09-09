package http_api

import (
	"context"
	"github.com/labstack/echo/v4"
	echo_middleware "github.com/labstack/echo/v4/middleware"
	"github.com/lidofinance/dc4bc/client/api/http_api/router"
	"github.com/lidofinance/dc4bc/client/config"
	"github.com/lidofinance/dc4bc/client/services/node"
)

type RESTApiProvider struct {
	config       *config.HttpApiConfig
	echoInstance *echo.Echo
}

func (p *RESTApiProvider) NewServer(config *config.Config, node node.NodeService) error {
	p.config = config.HttpApiConfig

	p.echoInstance = echo.New()

	p.echoInstance.HideBanner = true
	p.echoInstance.Debug = false

	p.echoInstance.HTTPErrorHandler = customHTTPErrorHandler

	// Middlewares

	p.echoInstance.Use(echo_middleware.Logger())

	p.echoInstance.Use(contextServiceMiddleware)

	router.SetRouter(p.echoInstance, nil, node)

	return nil
}

func (p *RESTApiProvider) Start() error {
	return p.echoInstance.Start(p.config.ListenAddr)
}

func (p *RESTApiProvider) Stop(ctx context.Context) error {
	return p.echoInstance.Shutdown(ctx)
}
