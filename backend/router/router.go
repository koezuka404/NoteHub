package router

import (
	"net/http"

	"github.com/koezuka404/notehub/controller"
	"github.com/labstack/echo/v4"
)

type Deps struct {
	Auth                *controller.AuthController
	Workspace           *controller.WorkspaceController
	Member              *controller.MemberController
	Account             *controller.AccountController
	Document            *controller.DocumentController
	Version             *controller.VersionController
	WebSocket           *controller.WebSocketController
	AuthMiddleware      echo.MiddlewareFunc
	RequireRefreshToken echo.MiddlewareFunc
	SecFetchSite        echo.MiddlewareFunc
	CSRF                echo.MiddlewareFunc
	RateLimit           echo.MiddlewareFunc
}

func Register(e *echo.Echo, deps Deps) {
	e.GET("/health", func(ctx echo.Context) error {
		return ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	if deps.WebSocket != nil {
		e.GET("/ws/documents/:documentId", deps.WebSocket.HandleDocument)
		e.GET("/ws/workspaces/:workspaceId", deps.WebSocket.HandleWorkspace)
	}

	api := e.Group("/api")

	api.GET("/auth/csrf", deps.Auth.IssueCSRF, deps.SecFetchSite, deps.RateLimit)
	api.POST("/auth/register", deps.Auth.Register, deps.SecFetchSite, deps.CSRF, deps.RateLimit)
	api.POST("/auth/login", deps.Auth.Login, deps.SecFetchSite, deps.CSRF, deps.RateLimit)
	api.POST("/auth/refresh", deps.Auth.Refresh, deps.RequireRefreshToken, deps.SecFetchSite, deps.RateLimit)
	api.POST("/auth/logout", deps.Auth.Logout, deps.AuthMiddleware, deps.SecFetchSite, deps.CSRF, deps.RateLimit)
	api.GET("/me", deps.Auth.Me, deps.AuthMiddleware)

	workspaces := api.Group("/workspaces", deps.AuthMiddleware)
	workspaces.GET("", deps.Workspace.List)
	workspaces.POST("", deps.Workspace.Create, deps.RateLimit)
	workspaces.GET("/:workspaceId", deps.Workspace.Get)
	workspaces.PATCH("/:workspaceId", deps.Workspace.Update, deps.RateLimit)
	workspaces.DELETE("/:workspaceId", deps.Workspace.Delete, deps.RateLimit)

	workspaces.GET("/:workspaceId/members", deps.Member.List)
	workspaces.GET("/:workspaceId/users/search", deps.Member.Search, deps.RateLimit)
	workspaces.POST("/:workspaceId/members", deps.Member.Add, deps.RateLimit)
	workspaces.DELETE("/:workspaceId/members/:userId", deps.Member.Remove, deps.RateLimit)
	workspaces.POST("/:workspaceId/members/:userId/suspend", deps.Account.Suspend, deps.RateLimit)
	workspaces.POST("/:workspaceId/members/:userId/reactivate", deps.Account.Reactivate, deps.RateLimit)
	workspaces.POST("/:workspaceId/members/:userId/delete-account", deps.Account.Delete, deps.RateLimit)

	workspaces.GET("/:workspaceId/documents", deps.Document.List)
	workspaces.POST("/:workspaceId/documents", deps.Document.Create, deps.RateLimit)

	documents := api.Group("/documents", deps.AuthMiddleware)
	documents.GET("/:documentId", deps.Document.Get)
	documents.PATCH("/:documentId", deps.Document.Update, deps.RateLimit)
	documents.DELETE("/:documentId", deps.Document.Delete, deps.RateLimit)

	documents.GET("/:documentId/versions", deps.Version.List)
	documents.POST("/:documentId/versions", deps.Version.Save, deps.RateLimit)
	documents.GET("/:documentId/versions/:versionId", deps.Version.Get)
	documents.POST("/:documentId/versions/:versionId/restore", deps.Version.Restore, deps.RateLimit)
}
