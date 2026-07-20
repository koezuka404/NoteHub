package router

import (
	"github.com/koezuka404/notehub/controller"
	"github.com/labstack/echo/v4"
)

func registerAuthRoutes(group *echo.Group, auth *controller.AuthController, authMiddleware, csrf echo.MiddlewareFunc) {
	group.POST("/auth/register", auth.Register)
	group.POST("/auth/login", auth.Login)
	group.POST("/auth/refresh", auth.Refresh, csrf)
	group.POST("/auth/logout", auth.Logout, authMiddleware, csrf)
}
