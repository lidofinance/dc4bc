package handlers

import (
	"fmt"
	"github.com/censync/go-dto"
	"github.com/censync/go-validator"
	"github.com/labstack/echo/v4"
	. "github.com/lidofinance/dc4bc/client/api/dto"
	cs "github.com/lidofinance/dc4bc/client/api/http_api/context_service"
	req "github.com/lidofinance/dc4bc/client/api/http_api/requests"
	"github.com/lidofinance/dc4bc/client/services"
	"net/http"
)

func SaveStateOffset(c echo.Context) error {
	stx := c.(*cs.ContextService)

	request := &req.StateOffsetForm{}

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

	formDTO := &StateOffsetDTO{}

	err = dto.RequestToDTO(formDTO, request)

	if err != nil {
		return stx.JsonError(
			http.StatusBadRequest,
			err,
		)
	}

	err = services.App().BaseClientService().SaveOffset(formDTO)

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

func GetStateOffset(c echo.Context) error {
	stx := c.(*cs.ContextService)

	offset, err := services.App().BaseClientService().GetStateOffset()

	if err == nil {
		return stx.Json(
			http.StatusOK,
			offset,
		)
	} else {
		return stx.JsonError(
			http.StatusInternalServerError,
			fmt.Errorf("failed to load offset: %v", err),
		)
	}
}
func ResetState(c echo.Context) error {
	stx := c.(*cs.ContextService)

	request := &req.ResetStateForm{}

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

	formDTO := &ResetStateDTO{}

	err = dto.RequestToDTO(formDTO, request)

	if err != nil {
		return stx.JsonError(
			http.StatusBadRequest,
			err,
		)
	}

	newStateDbPath, err := services.App().BaseClientService().ResetFSMState(formDTO)

	if err == nil {
		return stx.Json(
			http.StatusOK,
			newStateDbPath,
		)
	} else {
		return stx.JsonError(
			http.StatusInternalServerError,
			err,
		)
	}
}
