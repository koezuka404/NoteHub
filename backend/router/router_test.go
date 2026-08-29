package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
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
	cfg := &config.Config{
		Environment: config.EnvironmentDevelopment,
	}

	e := New(
		cfg,
		testDeps(&controller.WebSocketController{}),
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/health",
		nil,
	)

	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"GET /health status = %d, want %d",
			rec.Code,
			http.StatusOK,
		)
	}
}

func TestNew_RejectsOversizedRequestBody(t *testing.T) {
	cfg := &config.Config{
		Environment:         config.EnvironmentDevelopment,
		MaxRequestBodyBytes: 8,
	}

	e := New(
		cfg,
		testDeps(&controller.WebSocketController{}),
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/health",
		strings.NewReader("123456789"),
	)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestNew_IgnoresSpoofedForwardedIP(t *testing.T) {
	cfg := &config.Config{
		Environment: config.EnvironmentDevelopment,
	}

	e := New(
		cfg,
		testDeps(&controller.WebSocketController{}),
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/health",
		nil,
	)

	// 実際の接続元IP。
	req.RemoteAddr = "203.0.113.10:1234"

	// 攻撃者が勝手に付与したヘッダー。
	req.Header.Set(
		"X-Forwarded-For",
		"1.2.3.4",
	)

	req.Header.Set(
		"X-Vercel-Forwarded-For",
		"1.2.3.4",
	)

	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"GET /health status = %d, want %d",
			rec.Code,
			http.StatusOK,
		)
	}

	ctx := e.NewContext(req, rec)

	got := ctx.RealIP()

	if got != "203.0.113.10" {
		t.Fatalf(
			"RealIP = %q, want %q",
			got,
			"203.0.113.10",
		)
	}
}

func TestNew_UsesDedicatedHeaderFromTrustedProxy(t *testing.T) {
	cfg := &config.Config{
		Environment: config.EnvironmentDevelopment,

		// Vercelのプロキシとして信頼するCIDR。
		TrustedProxyCIDRs: []string{
			"76.76.21.0/24",
		},
	}

	e := New(
		cfg,
		testDeps(&controller.WebSocketController{}),
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/health",
		nil,
	)

	// 信頼済みプロキシからのアクセス。
	req.RemoteAddr = "76.76.21.10:443"

	// Vercel専用ヘッダー。
	req.Header.Set(
		"X-Vercel-Forwarded-For",
		"198.51.100.20",
	)

	// 一般的なX-Forwarded-Forには別IPを設定。
	// このヘッダーは採用されないことを確認する。
	req.Header.Set(
		"X-Forwarded-For",
		"1.2.3.4",
	)

	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	ctx := e.NewContext(req, rec)

	got := ctx.RealIP()

	if got != "198.51.100.20" {
		t.Fatalf(
			"RealIP = %q, want %q",
			got,
			"198.51.100.20",
		)
	}
}

func TestNew_UsesConfiguredClientIPHeader(t *testing.T) {
	cfg := &config.Config{
		Environment: config.EnvironmentDevelopment,
		TrustedProxyCIDRs: []string{
			"76.76.21.0/24",
		},
		ClientIPHeader: "X-NoteHub-Client-IP",
	}

	e := New(
		cfg,
		testDeps(&controller.WebSocketController{}),
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/health",
		nil,
	)
	req.RemoteAddr = "76.76.21.10:443"
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	req.Header.Set("X-Vercel-Forwarded-For", "9.9.9.9")
	req.Header.Set("X-NoteHub-Client-IP", "198.51.100.20")

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	ctx := e.NewContext(req, rec)
	got := ctx.RealIP()
	if got != "198.51.100.20" {
		t.Fatalf("RealIP = %q, want %q", got, "198.51.100.20")
	}
}

func TestNew_DoesNotTrustVercelHeaderFromUntrustedProxy(t *testing.T) {
	cfg := &config.Config{
		Environment: config.EnvironmentDevelopment,

		TrustedProxyCIDRs: []string{
			"76.76.21.0/24",
		},
	}

	e := New(
		cfg,
		testDeps(&controller.WebSocketController{}),
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/health",
		nil,
	)

	// 信頼済みCIDRではない接続元。
	req.RemoteAddr = "203.0.113.10:1234"

	req.Header.Set(
		"X-Vercel-Forwarded-For",
		"198.51.100.20",
	)

	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	ctx := e.NewContext(req, rec)

	got := ctx.RealIP()

	if got != "203.0.113.10" {
		t.Fatalf(
			"RealIP = %q, want %q",
			got,
			"203.0.113.10",
		)
	}
}

func TestNew_TrustedProxyWithoutForwardedHeader(t *testing.T) {
	cfg := &config.Config{
		Environment: config.EnvironmentDevelopment,

		TrustedProxyCIDRs: []string{
			"76.76.21.0/24",
		},
	}

	e := New(
		cfg,
		testDeps(&controller.WebSocketController{}),
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/health",
		nil,
	)

	req.RemoteAddr = "76.76.21.10:443"

	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	ctx := e.NewContext(req, rec)

	got := ctx.RealIP()

	if got != "76.76.21.10" {
		t.Fatalf(
			"RealIP = %q, want %q",
			got,
			"76.76.21.10",
		)
	}
}

func TestNew_UsesFirstValidForwardedIP(t *testing.T) {
	cfg := &config.Config{
		Environment: config.EnvironmentDevelopment,

		TrustedProxyCIDRs: []string{
			"76.76.21.0/24",
		},
	}

	e := New(
		cfg,
		testDeps(&controller.WebSocketController{}),
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/health",
		nil,
	)

	req.RemoteAddr = "76.76.21.10:443"

	req.Header.Set(
		"X-Vercel-Forwarded-For",
		"198.51.100.20, 198.51.100.21",
	)

	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	ctx := e.NewContext(req, rec)

	got := ctx.RealIP()

	if got != "198.51.100.20" {
		t.Fatalf(
			"RealIP = %q, want %q",
			got,
			"198.51.100.20",
		)
	}
}

func TestRegister(t *testing.T) {
	e := echo.New()

	Register(
		e,
		testDeps(&controller.WebSocketController{}),
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/health",
		nil,
	)

	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"GET /health status = %d, want %d",
			rec.Code,
			http.StatusOK,
		)
	}
}

func TestRegister_WithoutWebSocket(t *testing.T) {
	e := echo.New()

	Register(
		e,
		testDeps(nil),
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/health",
		nil,
	)

	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"GET /health status = %d, want %d",
			rec.Code,
			http.StatusOK,
		)
	}
}
