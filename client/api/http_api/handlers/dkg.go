package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/censync/go-dto"
	"github.com/censync/go-validator"
	"github.com/labstack/echo/v4"
	. "github.com/lidofinance/dc4bc/client/api/dto"
	cs "github.com/lidofinance/dc4bc/client/api/http_api/context_service"
	req "github.com/lidofinance/dc4bc/client/api/http_api/requests"
)

func (a *HTTPApp) StartDKG(c echo.Context) error {
	var err error
	ctx := c.(*cs.ContextService)
	request := &req.StartDKGForm{}
	request.Payload, err = ioutil.ReadAll(ctx.Request().Body)
	if err != nil {
		return ctx.JsonError(http.StatusBadRequest, fmt.Errorf("failed to read request body: %v", err))
	}
	defer ctx.Request().Body.Close()

	if err := validator.Validate(request); !err.IsEmpty() {
		return ctx.JsonError(http.StatusBadRequest, fmt.Errorf("invalid request: %w", err.Error()))
	}

	formDTO := &StartDkgDTO{}
	if err = dto.RequestToDTO(formDTO, request); err != nil {
		return ctx.JsonError(http.StatusBadRequest, err)
	}

	if err = a.node.StartDKG(formDTO); err != nil {
		return ctx.JsonError(http.StatusInternalServerError, err)
	}
	return ctx.Json(http.StatusOK, "ok")
}

func (a *HTTPApp) ReInitDKG(c echo.Context) error {
	ctx := c.(*cs.ContextService)
	request := &req.ReInitDKGForm{}
	err := ctx.BindToRequest(request)
	if err != nil {
		return ctx.JsonError(http.StatusBadRequest, err)
	}

	formDTO := &ReInitDKGDTO{ID: request.ID}
	formDTO.Payload, err = json.Marshal(request)
	if err != nil {
		return ctx.JsonError(http.StatusBadRequest, fmt.Errorf("failed to marshal request body: %v", err))
	}

	if err = a.node.ReInitDKG(formDTO); err != nil {
		return ctx.JsonError(http.StatusInternalServerError, err)
	}
	return ctx.Json(http.StatusOK, "ok")
}
