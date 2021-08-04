package router

import (
	"github.com/labstack/echo/v4"
	h "github.com/lidofinance/dc4bc/client/api/http_api/handlers"
)

func SetRouter(e *echo.Echo, authHandler echo.MiddlewareFunc) {
	e.GET("/getUsername", h.GetUsername)
	e.GET("/getPubKey", h.GetPubKey)

	e.POST("/sendMessage", h.SendMessage)
	e.GET("/getOperations", h.GetOperations)
	e.GET("/getOperationQRPath", h.GetOperationQRPath)

	e.GET("/getSignatures", h.GetSignatures)
	e.GET("/getSignatureByID", nil)

	e.GET("/getOperationQR", nil)
	e.GET("/handleProcessedOperationJSON", nil)
	e.GET("/getOperation", nil)

	e.GET("/startDKG", nil)
	e.GET("/proposeSignMessage", nil)
	e.GET("/approveDKGParticipation", nil)
	e.GET("/reinitDKG", nil)

	e.GET("/saveOffset", nil)
	e.GET("/getOffset", nil)

	e.GET("/getFSMDump", nil)
	e.GET("/getFSMList", nil)

	e.GET("/resetState", nil)

}
