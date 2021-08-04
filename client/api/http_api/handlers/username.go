package handlers

import (
	"github.com/labstack/echo/v4"
	cs "github.com/lidofinance/dc4bc/client/api/http_api/context_service"
	"github.com/lidofinance/dc4bc/client/services"
	"net/http"
)

func GetUsername(c echo.Context) error {
	stx := c.(*cs.ContextService)

	username := services.App().BaseClientService().GetUsername()

	return stx.Json(
		http.StatusOK,
		username,
	)
}
