package handlers

import (
	"fmt"
	"github.com/labstack/echo/v4"
	cs "github.com/lidofinance/dc4bc/client/api/http_api/context_service"
	req "github.com/lidofinance/dc4bc/client/api/http_api/requests"
	"github.com/lidofinance/dc4bc/client/services"
	"net/http"
)

func GetOperationQRPath(c echo.Context) error {
	stx := c.(*cs.ContextService)

	operationId := stx.QueryParam(req.QueryParamOperationID)

	operations, err := services.App().BaseClientService().GetOperationQRPath(operationId)

	if err == nil {
		return stx.Json(
			http.StatusOK,
			operations,
		)
	} else {
		return stx.JsonError(
			http.StatusInternalServerError,
			fmt.Errorf("failed to get operations: %v", err),
		)
	}
}
