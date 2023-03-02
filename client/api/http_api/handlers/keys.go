package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	cs "github.com/lidofinance/dc4bc/client/api/http_api/context_service"
)

func (a *HTTPApp) GetPubKey(c echo.Context) error {
	stx := c.(*cs.ContextService)

	pubKey := a.node.GetPubKey()

	return stx.Json(
		http.StatusOK,
		pubKey,
	)
}
