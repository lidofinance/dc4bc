package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	. "github.com/lidofinance/dc4bc/client/api/dto"
	cs "github.com/lidofinance/dc4bc/client/api/http_api/context_service"
	req "github.com/lidofinance/dc4bc/client/api/http_api/requests"
)

func (a *HTTPApp) GetFSMDump(c echo.Context) error {
	stx := c.(*cs.ContextService)
	formDTO := &DkgIdDTO{}
	if err := stx.BindToDTO(&req.DkgIdForm{}, formDTO); err != nil {
		return stx.JsonError(http.StatusBadRequest, err)
	}

	fsmDump, err := a.fsm.GetFSMDump(formDTO)
	if err != nil {
		return stx.JsonError(http.StatusInternalServerError, err)

	}
	return stx.Json(http.StatusOK, fsmDump)
}

func (a *HTTPApp) GetFSMList(c echo.Context) error {
	stx := c.(*cs.ContextService)
	fsmDump, err := a.fsm.GetFSMList()
	if err != nil {
		return stx.JsonError(http.StatusInternalServerError, err)
	}
	return stx.Json(http.StatusOK, fsmDump)
}
