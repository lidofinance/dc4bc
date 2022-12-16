package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	cs "github.com/lidofinance/dc4bc/client/api/http_api/context_service"
)

func (a *HTTPApp) GetUsername(c echo.Context) error {
	stx := c.(*cs.ContextService)
	return stx.Json(http.StatusOK, a.node.GetUsername())
}
