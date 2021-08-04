package handlers

import (
	"github.com/labstack/echo/v4"
	cs "github.com/lidofinance/dc4bc/client/api/http_api/context_service"
	"github.com/lidofinance/dc4bc/client/services"
	"net/http"
)

func GetOperations(c echo.Context) error {
	stx := c.(*cs.ContextService)

	operations, err := services.App().BaseClientService().GetOperations()

	if err == nil {
		return stx.Json(
			http.StatusOK,
			operations,
		)
	} else {
		return stx.JsonError(
			http.StatusInternalServerError,
			err,
		)
	}
}
