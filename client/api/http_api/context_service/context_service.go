package context_service

import (
	"fmt"
	"net/http"

	"github.com/censync/go-dto"
	"github.com/censync/go-validator"
	"github.com/labstack/echo/v4"
)

type ContextService struct {
	echo.Context
}

func New(c echo.Context) *ContextService {
	return &ContextService{
		c,
	}
}

type CSJsonResp struct {
	Result interface{} `json:"result"`
}

// Custom error
type CSErrorResp struct {
	Result       interface{} `json:"result"`
	ErrorMessage string      `json:"error_message,omitempty"`
}

func (e *CSErrorResp) Error() string {
	if e == nil {
		return ""
	}
	return e.ErrorMessage
}

// BindToRequest populates the request fields based on the context path and query parameters and body
// and validates the result.
func (cs *ContextService) BindToRequest(request interface{}) error {
	if err := cs.Bind(request); err != nil {
		return cs.JsonError(http.StatusBadRequest, fmt.Errorf("failed to read request body: %v", err))
	}
	if err := validator.Validate(request); !err.IsEmpty() {
		return cs.JsonError(http.StatusBadRequest, err.Error())
	}
	return nil
}

// BindToDTO builds a request of the given form based on the context and converts it to a DTO.
func (cs *ContextService) BindToDTO(requestForm, dtoForm interface{}) error {
	if err := cs.BindToRequest(requestForm); err != nil {
		return err
	}
	if err := dto.RequestToDTO(dtoForm, requestForm); err != nil {
		return cs.JsonError(http.StatusBadRequest, err)
	}
	return nil
}

func (cs *ContextService) Json(code int, data interface{}) error {
	if data != nil {
		return cs.JSON(code, &CSJsonResp{
			Result: data,
		})
	} else {
		return cs.JSON(code, &CSJsonResp{
			Result: struct{}{},
		})
	}
}

func (cs *ContextService) JsonEmpty(code int) error {
	return cs.JSON(code, &CSJsonResp{
		Result: struct{}{},
	})
}

func (cs *ContextService) JsonError(code int, err error) error {
	if err == nil {
		return cs.JSON(code, &CSErrorResp{
			Result:       struct{}{},
			ErrorMessage: "undefined error",
		})
	} else {
		return cs.JSON(code, &CSErrorResp{
			Result:       struct{}{},
			ErrorMessage: err.Error(),
		})
	}
}
