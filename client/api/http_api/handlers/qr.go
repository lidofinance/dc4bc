package handlers

import (
	"fmt"
	"net/http"

	"github.com/censync/go-dto"
	"github.com/censync/go-validator"
	"github.com/labstack/echo/v4"
	. "github.com/lidofinance/dc4bc/client/api/dto"
	cs "github.com/lidofinance/dc4bc/client/api/http_api/context_service"
	req "github.com/lidofinance/dc4bc/client/api/http_api/requests"
)

func (a *HTTPApp) GetOperationQRPath(c echo.Context) error {
	stx := c.(*cs.ContextService)

	request := &req.OperationIdForm{}

	err := stx.Bind(request)

	if err != nil {
		return stx.JsonError(
			http.StatusBadRequest,
			fmt.Errorf("failed to read request body: %v", err),
		)
	}

	if err := validator.Validate(request); !err.IsEmpty() {
		return stx.JsonError(
			http.StatusBadRequest,
			err.Error(),
		)
	}

	formDTO := &OperationIdDTO{}

	err = dto.RequestToDTO(formDTO, request)

	if err != nil {
		return stx.JsonError(
			http.StatusBadRequest,
			err,
		)
	}

	operations, err := a.node.GetOperationQRPath(formDTO)

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

func (a *HTTPApp) GetOperationQRFile(c echo.Context) error {
	stx := c.(*cs.ContextService)

	request := &req.OperationIdForm{}

	err := stx.Bind(request)

	if err != nil {
		return stx.JsonError(
			http.StatusBadRequest,
			fmt.Errorf("failed to read request body: %v", err),
		)
	}

	if err := validator.Validate(request); !err.IsEmpty() {
		return stx.JsonError(
			http.StatusBadRequest,
			err.Error(),
		)
	}

	formDTO := &OperationIdDTO{}

	err = dto.RequestToDTO(formDTO, request)

	if err != nil {
		return stx.JsonError(
			http.StatusBadRequest,
			err,
		)
	}

	encodedData, err := a.node.GetOperationQRFile(formDTO)

	if err == nil {
		return stx.Blob(http.StatusOK, "image/png", encodedData)
	} else {
		return stx.JsonError(
			http.StatusInternalServerError,
			err,
		)
	}
}
