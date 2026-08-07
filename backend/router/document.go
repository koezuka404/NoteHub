package router

import (
	"github.com/koezuka404/notehub/controller"
	"github.com/labstack/echo/v4"
)

func registerDocumentRoutes(
	group *echo.Group,
	document *controller.DocumentController,
	version *controller.VersionController,
	authMiddleware echo.MiddlewareFunc,
	rateLimit echo.MiddlewareFunc,
) {
	documents := group.Group("/documents", authMiddleware)
	documents.GET("/:documentId", document.Get)
	documents.PATCH("/:documentId", document.Update, rateLimit)
	documents.DELETE("/:documentId", document.Delete, rateLimit)

	documents.GET("/:documentId/versions", version.List)
	documents.POST("/:documentId/versions", version.Save, rateLimit)
	documents.GET("/:documentId/versions/:versionId", version.Get)
	documents.POST("/:documentId/versions/:versionId/restore", version.Restore, rateLimit)
}
