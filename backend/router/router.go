package router

import (
	"net/http"

	"github.com/koezuka404/notehub/controller"
	"github.com/labstack/echo/v4"
)

type Deps struct {
	Auth *controller.AuthController
	CSRF echo.MiddlewareFunc
}

func Register(e *echo.Echo, deps Deps) {
	e.GET("/health", func(ctx echo.Context) error {
		return ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	api := e.Group("/api")
	registerAuthRoutes(api, deps.Auth, deps.CSRF)
}
