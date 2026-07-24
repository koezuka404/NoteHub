package router

import (
	"github.com/koezuka404/notehub/controller"
	"github.com/labstack/echo/v4"
)

func registerWorkspaceRoutes(
	group *echo.Group,
	workspace *controller.WorkspaceController,
	authMiddleware echo.MiddlewareFunc,
	rateLimit echo.MiddlewareFunc,
) {
	workspaces := group.Group("/workspaces", authMiddleware)
	workspaces.GET("", workspace.List)
	workspaces.POST("", workspace.Create, rateLimit)
	workspaces.GET("/:workspaceId", workspace.Get)
	workspaces.PATCH("/:workspaceId", workspace.Update, rateLimit)
	workspaces.DELETE("/:workspaceId", workspace.Delete, rateLimit)
}
