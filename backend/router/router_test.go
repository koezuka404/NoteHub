package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/koezuka404/notehub/config"
	"github.com/koezuka404/notehub/controller"
	"github.com/labstack/echo/v4"
)

func passthroughMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return next
}

func testDeps(ws *controller.WebSocketController) Deps {
	return Deps{
		Auth:                &controller.AuthController{},
		Workspace:           &controller.WorkspaceController{},
		Member:              &controller.MemberController{},
		Account:             &controller.AccountController{},
		Document:            &controller.DocumentController{},
		Version:             &controller.VersionController{},
		WebSocket:           ws,
		AuthMiddleware:      passthroughMiddleware,
		RequireRefreshToken: passthroughMiddleware,
		SecFetchSite:        passthroughMiddleware,
		CSRF:                passthroughMiddleware,
		RateLimit:           passthroughMiddleware,
	}
}

func TestNew(t *testing.T) {
	cfg := &config.Config{Environment: config.EnvironmentDevelopment}
	e := New(cfg, testDeps(&controller.WebSocketController{}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /health status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestNew_IgnoresSpoofedForwardedIP(t *testing.T) {
	cfg := &config.Config{Environment: config.EnvironmentDevelopment}
	e := New(cfg, testDeps(&controller.WebSocketController{}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.RemoteAddr = "203.0.113.10:1234"
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	req.Header.Set("X-Vercel-Forwarded-For", "1.2.3.4")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /health status = %d, want %d", rec.Code, http.StatusOK)
	}

	ctx := e.NewContext(req, rec)
	if got := ctx.RealIP(); got != "203.0.113.10" {
		t.Fatalf("RealIP = %q, want untrusted remote address", got)
	}
}

func TestNew_UsesDedicatedHeaderFromTrustedProxy(t *testing.T) {
	cfg := &config.Config{
		Environment:       config.EnvironmentDevelopment,
		TrustedProxyCIDRs: []string{"76.76.21.0/24"},
	}
	e := New(cfg, testDeps(&controller.WebSocketController{}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.RemoteAddr = "76.76.21.10:443"
	req.Header.Set("X-Vercel-Forwarded-For", "198.51.100.20")
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	ctx := e.NewContext(req, rec)
	if got := ctx.RealIP(); got != "198.51.100.20" {
		t.Fatalf("RealIP = %q, want dedicated Vercel header", got)
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
