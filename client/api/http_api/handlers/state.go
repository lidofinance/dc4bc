package handlers

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	. "github.com/lidofinance/dc4bc/client/api/dto"
	cs "github.com/lidofinance/dc4bc/client/api/http_api/context_service"
	req "github.com/lidofinance/dc4bc/client/api/http_api/requests"
)

func (a *HTTPApp) SaveStateOffset(c echo.Context) error {
	stx := c.(*cs.ContextService)
	formDTO := &StateOffsetDTO{}
	if err := stx.BindToDTO(&req.StateOffsetForm{}, formDTO); err != nil {
		return stx.JsonError(http.StatusBadRequest, err)
	}

	if err := a.node.SaveOffset(formDTO); err != nil {
		return stx.JsonError(http.StatusInternalServerError, err)
	}
	return stx.Json(http.StatusOK, "ok")
}

func (a *HTTPApp) GetStateOffset(c echo.Context) error {
	stx := c.(*cs.ContextService)
	offset, err := a.node.GetStateOffset()
	if err != nil {
		return stx.JsonError(http.StatusInternalServerError, fmt.Errorf("failed to load offset: %v", err))
	}
	return stx.Json(http.StatusOK, offset)
}

func (a *HTTPApp) ResetState(c echo.Context) error {
	stx := c.(*cs.ContextService)

	formDTO := &ResetStateDTO{}
	if err := stx.BindToDTO(&req.ResetStateForm{}, formDTO); err != nil {
		return stx.JsonError(http.StatusBadRequest, err)
	}

	newStateDbPath, err := a.fsm.ResetFSMState(formDTO)
	if err != nil {
		return stx.JsonError(http.StatusInternalServerError, err)
	}
	return stx.Json(http.StatusOK, newStateDbPath)
}
