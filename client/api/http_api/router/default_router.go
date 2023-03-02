package router

import (
	"github.com/labstack/echo/v4"

	"github.com/lidofinance/dc4bc/client/api/http_api/handlers"
	"github.com/lidofinance/dc4bc/client/services"
	"github.com/lidofinance/dc4bc/client/services/node"
)

func SetRouter(e *echo.Echo, authHandler echo.MiddlewareFunc, node node.NodeService, sp *services.ServiceProvider) {
	h := handlers.NewHTTPApp(node, sp)

	e.GET("/getUsername", h.GetUsername)
	e.GET("/getPubKey", h.GetPubKey)

	e.POST("/sendMessage", h.SendMessage)
	e.GET("/getOperations", h.GetOperations)

	e.GET("/getSignatures", h.GetSignatures)
	e.GET("/getBatches", h.GetBatches)
	e.GET("/getSignatureByID", h.GetSignatureByID)

	e.POST("/handleProcessedOperationJSON", h.ProcessOperation)
	e.GET("/getOperation", h.GetOperation)

	e.POST("/startDKG", h.StartDKG)
	e.POST("/proposeSignMessage", h.ProposeSignMessage)
	e.POST("/proposeSignBatchMessages", h.ProposeSignBatchMessages)
	e.POST("/proposeSignBakedMessages", h.ProposeSignBakedMessages)
	e.POST("/approveDKGParticipation", h.ApproveParticipation)
	e.POST("/reinitDKG", h.ReInitDKG)

	e.POST("/saveOffset", h.SaveStateOffset)
	e.GET("/getOffset", h.GetStateOffset)

	e.GET("/getFSMDump", h.GetFSMDump)
	e.GET("/getFSMList", h.GetFSMList)

	e.POST("/resetState", h.ResetState)

}
