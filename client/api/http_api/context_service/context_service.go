package context_service

import (
	"fmt"

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
	return fmt.Sprintf("%s", e.ErrorMessage)
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
