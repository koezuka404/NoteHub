package router

import (
	"net/http"

	"github.com/koezuka404/notehub/controller"
	"github.com/labstack/echo/v4"
)

type Deps struct {
	Auth           *controller.AuthController
	Workspace      *controller.WorkspaceController
	Member         *controller.MemberController
	AuthMiddleware echo.MiddlewareFunc
	CSRF           echo.MiddlewareFunc
	RateLimit      echo.MiddlewareFunc
}

func Register(e *echo.Echo, deps Deps) {
	e.GET("/health", func(ctx echo.Context) error {
		return ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	api := e.Group("/api")
	registerAuthRoutes(api, deps.Auth, deps.AuthMiddleware, deps.CSRF, deps.RateLimit)
	registerWorkspaceRoutes(api, deps.Workspace, deps.Member, deps.AuthMiddleware, deps.RateLimit)
}
