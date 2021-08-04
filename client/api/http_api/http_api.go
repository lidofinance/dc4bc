package http_api

import (
	"fmt"
	"github.com/labstack/echo/v4"
	echo_middleware "github.com/labstack/echo/v4/middleware"
	"github.com/lidofinance/dc4bc/client/api/http_api/router"
	"github.com/lidofinance/dc4bc/client/config"
)

type RESTApiProvider struct {
	config       *config.HttpApiConfig
	echoInstance *echo.Echo
}

func (p *RESTApiProvider) NewServer(config *config.Config) error {
	p.config = config.HttpApiConfig

	p.echoInstance = echo.New()

	p.echoInstance.HideBanner = true
	p.echoInstance.Debug = false

	p.echoInstance.HTTPErrorHandler = customHTTPErrorHandler

	// Middlewares

	p.echoInstance.Use(echo_middleware.Logger())

	p.echoInstance.Use(contextServiceMiddleware)

	router.SetRouter(p.echoInstance, nil)

	return nil
}

func (p *RESTApiProvider) Start() error {
	return p.echoInstance.Start(fmt.Sprintf("%s:%d", p.config.Host, p.config.Port))
}
