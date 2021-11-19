package handlers

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"github.com/labstack/echo/v4"
	. "github.com/lidofinance/dc4bc/client/api/dto"
	cs "github.com/lidofinance/dc4bc/client/api/http_api/context_service"
	req "github.com/lidofinance/dc4bc/client/api/http_api/requests"
)

func (a *HTTPApp) GetSignatures(c echo.Context) error {
	stx := c.(*cs.ContextService)
	formDTO := &DkgIdDTO{}
	if err := stx.BindToDTO(&req.DkgIdForm{}, formDTO); err != nil {
		return err
	}

	signatures, err := a.signature.GetSignatures(formDTO)
	if err != nil {
		return stx.JsonError(http.StatusInternalServerError, fmt.Errorf("failed to get signatures: %w", err))
	}
	return stx.Json(http.StatusOK, signatures)
}

func (a *HTTPApp) GetSignatureByID(c echo.Context) error {
	stx := c.(*cs.ContextService)
	formDTO := &SignatureByIdDTO{}
	if err := stx.BindToDTO(&req.SignatureByIDForm{}, formDTO); err != nil {
		return err
	}

	signatures, err := a.signature.GetSignatureByID(formDTO)
	if err != nil {
		return stx.JsonError(http.StatusInternalServerError, fmt.Errorf("failed to get signatures: %w", err))
	}
	return stx.Json(http.StatusOK, signatures)
}

func (a *HTTPApp) ProposeSignMessage(c echo.Context) error {
	stx := c.(*cs.ContextService)
	formDTO := &ProposeSignMessageDTO{}
	if err := stx.BindToDTO(&req.ProposeSignMessageForm{}, formDTO); err != nil {
		return err
	}

	batch := ProposeSignBatchMessagesDTO{
		DkgID: formDTO.DkgID,
		Data: map[string][]byte{
			uuid.New().String(): formDTO.Data,
		},
	}

	if err := a.node.ProposeSignMessages(&batch); err != nil {
		return stx.JsonError(http.StatusInternalServerError, err)
	}
	return stx.Json(http.StatusOK, "ok")
}

func (a *HTTPApp) ProposeSignBatchMessages(c echo.Context) error {
	stx := c.(*cs.ContextService)
	formDTO := &ProposeSignBatchMessagesDTO{}
	if err := stx.BindToDTO(&req.ProposeSignBatchMessagesForm{}, formDTO); err != nil {
		return err
	}

	if err := a.node.ProposeSignMessages(formDTO); err != nil {
		return stx.JsonError(http.StatusInternalServerError, err)
	}
	return stx.Json(http.StatusOK, "ok")
}
