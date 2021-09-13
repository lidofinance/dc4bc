package http_api

import (
	"fmt"
	"net/http"

	. "github.com/labstack/echo/v4"
	cs "github.com/lidofinance/dc4bc/client/api/http_api/context_service"
)

func contextServiceMiddleware(next HandlerFunc) HandlerFunc {
	return func(ctx Context) error {
		return next(cs.New(ctx))
	}
}

// Custom error handler
func customHTTPErrorHandler(err error, c Context) {
	csError, ok := err.(*cs.CSErrorResp)
	if !ok {
		if he, ok := err.(*HTTPError); ok {
			csError = &cs.CSErrorResp{
				ErrorMessage: fmt.Sprintf("%s", he.Message),
			}
		} else {
			csError = &cs.CSErrorResp{
				ErrorMessage: http.StatusText(http.StatusInternalServerError),
			}
		}
	}

	// Send response
	if !c.Response().Committed {
		if c.Request().Method == http.MethodHead {
			err = c.NoContent(http.StatusInternalServerError)
		} else {
			err = c.JSON(http.StatusInternalServerError, csError)
		}
		if err != nil {
			// TODO: Add logging
			// e.Logger.Error(err)
		}
	}
}
