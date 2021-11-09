package handlers

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	. "github.com/lidofinance/dc4bc/client/api/dto"
	cs "github.com/lidofinance/dc4bc/client/api/http_api/context_service"
	req "github.com/lidofinance/dc4bc/client/api/http_api/requests"
)

func (a *HTTPApp) GetOperationQRPath(c echo.Context) error {
	stx := c.(*cs.ContextService)
	formDTO := &OperationIdDTO{}
	if err := stx.BindToDTO(&req.OperationIdForm{}, formDTO); err != nil {
		return err
	}

	operations, err := a.node.GetOperationQRPath(formDTO)
	if err != nil {
		return stx.JsonError(http.StatusInternalServerError, fmt.Errorf("failed to get operations: %v", err))
	}
	return stx.Json(http.StatusOK, operations)
}

func (a *HTTPApp) GetOperationQRFile(c echo.Context) error {
	stx := c.(*cs.ContextService)
	formDTO := &OperationIdDTO{}
	if err := stx.BindToDTO(&req.OperationIdForm{}, formDTO); err != nil {
		return err
	}

	encodedData, err := a.node.GetOperationQRFile(formDTO)
	if err != nil {
		return stx.JsonError(http.StatusInternalServerError, err)
	}
	return stx.Blob(http.StatusOK, "image/png", encodedData)
}
