package router

import (
	"github.com/koezuka404/notehub/controller"
	"github.com/labstack/echo/v4"
)

func registerWebSocketRoutes(e *echo.Echo, ws *controller.WebSocketController) {
	e.GET("/ws/documents/:documentId", ws.HandleDocument)
}
