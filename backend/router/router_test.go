package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/koezuka404/notehub/controller"
	"github.com/labstack/echo/v4"
)

func passthroughMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return next
}

func testDeps(ws *controller.WebSocketController) Deps {
	return Deps{
		Auth:           &controller.AuthController{},
		Workspace:      &controller.WorkspaceController{},
		Member:         &controller.MemberController{},
		Account:        &controller.AccountController{},
		Document:       &controller.DocumentController{},
		Version:        &controller.VersionController{},
		WebSocket:      ws,
		AuthMiddleware:      passthroughMiddleware,
		RequireRefreshToken: passthroughMiddleware,
		SecFetchSite:        passthroughMiddleware,
		CSRF:                passthroughMiddleware,
		RateLimit:           passthroughMiddleware,
	}
}

func TestRegister(t *testing.T) {
	e := echo.New()
	Register(e, testDeps(&controller.WebSocketController{}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /health status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestRegister_WithoutWebSocket(t *testing.T) {
	e := echo.New()
	Register(e, testDeps(nil))
}
