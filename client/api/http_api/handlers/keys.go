package handlers

import (
	"github.com/labstack/echo/v4"
	cs "github.com/lidofinance/dc4bc/client/api/http_api/context_service"
	"github.com/lidofinance/dc4bc/client/services"
	"net/http"
)

func GetPubKey(c echo.Context) error {
	stx := c.(*cs.ContextService)

	pubKey := services.App().BaseClientService().GetPubKey()

	return stx.Json(
		http.StatusOK,
		pubKey,
	)
}
