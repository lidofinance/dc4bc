package handlers

import (
	"encoding/json"
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

func StartDKG(c echo.Context) error {
	stx := c.(*cs.ContextService)

	request := &req.StartDKGForm{}

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

	formDTO := &StartDkgDTO{}

	err = dto.RequestToDTO(formDTO, request)

	if err != nil {
		return stx.JsonError(
			http.StatusBadRequest,
			err,
		)
	}

	err = services.App().BaseClientService().StartDKG(formDTO)

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

func ReInitDKG(c echo.Context) error {
	stx := c.(*cs.ContextService)

	request := &req.ReInitDKGForm{}

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

	formDTO := &ReInitDKGDTO{
		ID: request.ID,
	}

	formDTO.Payload, err = json.Marshal(request)

	if err != nil {
		return stx.JsonError(
			http.StatusBadRequest,
			fmt.Errorf("failed to marshal request body: %v", err),
		)
	}

	err = services.App().BaseClientService().ReInitDKG(formDTO)

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
