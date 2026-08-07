package router

import (
	"github.com/koezuka404/notehub/controller"
	"github.com/labstack/echo/v4"
)

func registerWorkspaceRoutes(
	group *echo.Group,
	workspace *controller.WorkspaceController,
	member *controller.MemberController,
	account *controller.AccountController,
	document *controller.DocumentController,
	authMiddleware echo.MiddlewareFunc,
	rateLimit echo.MiddlewareFunc,
) {
	workspaces := group.Group("/workspaces", authMiddleware)
	workspaces.GET("", workspace.List)
	workspaces.POST("", workspace.Create, rateLimit)
	workspaces.GET("/:workspaceId", workspace.Get)
	workspaces.PATCH("/:workspaceId", workspace.Update, rateLimit)
	workspaces.DELETE("/:workspaceId", workspace.Delete, rateLimit)

	workspaces.GET("/:workspaceId/members", member.List)
	workspaces.GET("/:workspaceId/users/search", member.Search, rateLimit)
	workspaces.POST("/:workspaceId/members", member.Add, rateLimit)
	workspaces.DELETE("/:workspaceId/members/:userId", member.Remove, rateLimit)
	workspaces.POST("/:workspaceId/members/:userId/suspend", account.Suspend, rateLimit)
	workspaces.POST("/:workspaceId/members/:userId/reactivate", account.Reactivate, rateLimit)
	workspaces.POST("/:workspaceId/members/:userId/delete-account", account.Delete, rateLimit)

	workspaces.GET("/:workspaceId/documents", document.List)
	workspaces.POST("/:workspaceId/documents", document.Create, rateLimit)
}
