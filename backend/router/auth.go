package router

import (
	"github.com/koezuka404/notehub/controller"
	"github.com/labstack/echo/v4"
)

func registerAuthRoutes(group *echo.Group, auth *controller.AuthController, authMiddleware, requireRefresh, originValidation, csrf, rateLimit echo.MiddlewareFunc) {
	group.POST("/auth/register", auth.Register, rateLimit)
	group.POST("/auth/login", auth.Login, rateLimit)
	group.POST("/auth/refresh", auth.Refresh, requireRefresh, originValidation, rateLimit)
	group.POST("/auth/logout", auth.Logout, authMiddleware, csrf, rateLimit)
	group.GET("/me", auth.Me, authMiddleware)
}
