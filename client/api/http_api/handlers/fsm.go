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

func GetFSMDump(c echo.Context) error {
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

	fsmDump, err := services.App().BaseClientService().GetFSMDump(formDTO)

	if err == nil {
		return stx.Json(
			http.StatusOK,
			fsmDump,
		)
	} else {
		return stx.JsonError(
			http.StatusInternalServerError,
			err,
		)
	}
}

func GetFSMList(c echo.Context) error {
	stx := c.(*cs.ContextService)

	fsmDump, err := services.App().BaseClientService().GetFSMList()

	if err == nil {
		return stx.Json(
			http.StatusOK,
			fsmDump,
		)
	} else {
		return stx.JsonError(
			http.StatusInternalServerError,
			err,
		)
	}
}
