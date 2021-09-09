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
	"io/ioutil"
	"net/http"
)

func (a *HTTPApp) StartDKG(c echo.Context) error {
	var err error
	stx := c.(*cs.ContextService)

	request := &req.StartDKGForm{}

	request.Payload, err = ioutil.ReadAll(stx.Request().Body)
	if err != nil {
		return stx.JsonError(
			http.StatusBadRequest,
			fmt.Errorf("failed to read request body: %v", err),
		)
	}
	defer stx.Request().Body.Close()

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

	err = a.node.StartDKG(formDTO)

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

func (a *HTTPApp) ReInitDKG(c echo.Context) error {
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

	err = a.node.ReInitDKG(formDTO)

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
