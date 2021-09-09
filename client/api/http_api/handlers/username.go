package handlers

import (
	"github.com/labstack/echo/v4"
	cs "github.com/lidofinance/dc4bc/client/api/http_api/context_service"
	"net/http"
)

func (a *HTTPApp) GetUsername(c echo.Context) error {
	stx := c.(*cs.ContextService)

	username := a.node.GetUsername()

	return stx.Json(
		http.StatusOK,
		username,
	)
}
