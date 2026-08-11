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
	Account        *controller.AccountController
	Document       *controller.DocumentController
	Version        *controller.VersionController
	WebSocket      *controller.WebSocketController
	AuthMiddleware      echo.MiddlewareFunc
	RequireRefreshToken echo.MiddlewareFunc
	OriginValidation    echo.MiddlewareFunc
	CSRF                echo.MiddlewareFunc
	RateLimit           echo.MiddlewareFunc
}

func Register(e *echo.Echo, deps Deps) {
	e.GET("/health", func(ctx echo.Context) error {
		return ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	if deps.WebSocket != nil {
		registerWebSocketRoutes(e, deps.WebSocket)
	}
	api := e.Group("/api")
	registerAuthRoutes(api, deps.Auth, deps.AuthMiddleware, deps.RequireRefreshToken, deps.OriginValidation, deps.CSRF, deps.RateLimit)
	registerWorkspaceRoutes(api, deps.Workspace, deps.Member, deps.Account, deps.Document, deps.AuthMiddleware, deps.RateLimit)
	registerDocumentRoutes(api, deps.Document, deps.Version, deps.AuthMiddleware, deps.RateLimit)
}
