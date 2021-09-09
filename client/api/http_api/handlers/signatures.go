package handlers

import (
	"fmt"
	dto "github.com/censync/go-dto"
	"github.com/censync/go-validator"
	"github.com/labstack/echo/v4"
	. "github.com/lidofinance/dc4bc/client/api/dto"
	cs "github.com/lidofinance/dc4bc/client/api/http_api/context_service"
	req "github.com/lidofinance/dc4bc/client/api/http_api/requests"
	"net/http"
)

func (a *HTTPApp) GetSignatures(c echo.Context) error {
	stx := c.(*cs.ContextService)

	request := &req.DkgIdForm{}

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

	formDTO := &DkgIdDTO{}

	err = dto.RequestToDTO(formDTO, request)

	if err != nil {
		return stx.JsonError(
			http.StatusBadRequest,
			err,
		)
	}

	signatures, err := a.node.GetSignatures(formDTO)

	if err == nil {
		return stx.Json(
			http.StatusOK,
			signatures,
		)
	} else {
		return stx.JsonError(
			http.StatusInternalServerError,
			fmt.Errorf("failed to get operations: %v", err),
		)
	}
}

func (a *HTTPApp) GetSignatureByID(c echo.Context) error {
	stx := c.(*cs.ContextService)

	request := &req.SignatureByIDForm{}

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

	formDTO := &SignatureByIdDTO{}

	err = dto.RequestToDTO(formDTO, request)

	if err != nil {
		return stx.JsonError(
			http.StatusBadRequest,
			err,
		)
	}

	signatures, err := a.node.GetSignatureByID(formDTO)

	if err == nil {
		return stx.Json(
			http.StatusOK,
			signatures,
		)
	} else {
		return stx.JsonError(
			http.StatusInternalServerError,
			fmt.Errorf("failed to get operations: %v", err),
		)
	}
}

func (a *HTTPApp) ProposeSignData(c echo.Context) error {
	stx := c.(*cs.ContextService)

	request := &req.ProposeSignDataForm{}
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

	formDTO := &ProposeSignDataDTO{}

	err = dto.RequestToDTO(formDTO, request)

	if err != nil {
		return stx.JsonError(
			http.StatusBadRequest,
			err,
		)
	}

	err = a.node.ProposeSignData(formDTO)

	if err == nil {
		return stx.Json(
			http.StatusOK,
			"ok",
		)
	} else {
		return stx.JsonError(
			http.StatusInternalServerError,
			err,
		)
	}
}
