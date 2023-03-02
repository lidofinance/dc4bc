package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	. "github.com/lidofinance/dc4bc/client/api/dto"
	cs "github.com/lidofinance/dc4bc/client/api/http_api/context_service"
	req "github.com/lidofinance/dc4bc/client/api/http_api/requests"
)

func (a *HTTPApp) SendMessage(c echo.Context) error {
	stx := c.(*cs.ContextService)
	formDTO := &MessageDTO{}
	if err := stx.BindToDTO(&req.MessageForm{}, formDTO); err != nil {
		return stx.JsonError(http.StatusBadRequest, err)
	}

	if err := a.node.SendMessage(formDTO); err != nil {
		return stx.JsonError(http.StatusInternalServerError, err)
	}
	return stx.Json(http.StatusOK, "ok")
}
