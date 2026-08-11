package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/koezuka404/notehub/config"
	"github.com/labstack/echo/v4"
)

func TestOriginOrCSRFMiddleware_AllowsOrigin(t *testing.T) {
	rec := runOriginOrCSRF(t, productionOriginConfig(), CSRFConfig{CookieName: "csrf"}, "https://app.example.com", "", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestOriginOrCSRFMiddleware_AllowsReferer(t *testing.T) {
	rec := runOriginOrCSRF(t, productionOriginConfig(), CSRFConfig{CookieName: "csrf"}, "", "https://app.example.com/dashboard", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestOriginOrCSRFMiddleware_AllowsExactReferer(t *testing.T) {
	rec := runOriginOrCSRF(t, productionOriginConfig(), CSRFConfig{CookieName: "csrf"}, "", "https://app.example.com", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestOriginOrCSRFMiddleware_DevelopmentDefaults(t *testing.T) {
	rec := runOriginOrCSRF(
		t,
		&config.Config{Environment: config.EnvironmentDevelopment},
		CSRFConfig{CookieName: "csrf"},
		"http://127.0.0.1:5173",
		"",
		"",
		"",
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestOriginOrCSRFMiddleware_AllowsCSRF(t *testing.T) {
	rec := runOriginOrCSRF(t, productionOriginConfig(), CSRFConfig{CookieName: "csrf"}, "", "", "csrf=secret; Path=/", "secret")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestOriginOrCSRFMiddleware_AllowsCSRFWithDefaultHeaderName(t *testing.T) {
	rec := runOriginOrCSRF(t, productionOriginConfig(), CSRFConfig{CookieName: "csrf", HeaderName: ""}, "", "", "csrf=secret; Path=/", "secret")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestOriginOrCSRFMiddleware_RejectsMissingBoth(t *testing.T) {
	rec := runOriginOrCSRF(t, productionOriginConfig(), CSRFConfig{CookieName: "csrf"}, "", "", "", "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rec.Code)
	}
	payload := decodeErrorResponse(t, rec)
	if payload.Error.Code != "CSRF_TOKEN_REQUIRED" {
		t.Fatalf("code = %q", payload.Error.Code)
	}
}

func TestOriginOrCSRFMiddleware_RejectsWhenNoAllowedOriginsAndNoCSRF(t *testing.T) {
	rec := runOriginOrCSRF(
		t,
		&config.Config{Environment: config.EnvironmentProduction, AllowedOrigins: nil},
		CSRFConfig{CookieName: "csrf"},
		"",
		"",
		"",
		"",
	)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestOriginOrCSRFMiddleware_RejectsUnknownOrigin(t *testing.T) {
	rec := runOriginOrCSRF(t, productionOriginConfig(), CSRFConfig{CookieName: "csrf"}, "https://evil.example.com", "", "", "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestOriginOrCSRFMiddleware_RejectsUnknownReferer(t *testing.T) {
	rec := runOriginOrCSRF(t, productionOriginConfig(), CSRFConfig{CookieName: "csrf"}, "", "https://evil.example.com/path", "", "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestOriginOrCSRFMiddleware_RejectsCSRFCookieOnly(t *testing.T) {
	rec := runOriginOrCSRF(t, productionOriginConfig(), CSRFConfig{CookieName: "csrf"}, "", "", "csrf=secret; Path=/", "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestOriginOrCSRFMiddleware_RejectsCSRFHeaderOnly(t *testing.T) {
	rec := runOriginOrCSRF(t, productionOriginConfig(), CSRFConfig{CookieName: "csrf"}, "", "", "", "secret")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestOriginOrCSRFMiddleware_RejectsEmptyCSRFCookie(t *testing.T) {
	rec := runOriginOrCSRF(t, productionOriginConfig(), CSRFConfig{CookieName: "csrf"}, "", "", "csrf=; Path=/", "secret")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestOriginOrCSRFMiddleware_RejectsMismatchedCSRFTokens(t *testing.T) {
	rec := runOriginOrCSRF(t, productionOriginConfig(), CSRFConfig{CookieName: "csrf"}, "", "", "csrf=secret; Path=/", "other")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestOriginOrCSRFMiddleware_SkipsBlankConfiguredOrigins(t *testing.T) {
	rec := runOriginOrCSRF(
		t,
		&config.Config{Environment: config.EnvironmentProduction, AllowedOrigins: []string{" ", "https://app.example.com"}},
		CSRFConfig{CookieName: "csrf"},
		"https://app.example.com",
		"",
		"",
		"",
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func productionOriginConfig() *config.Config {
	return &config.Config{
		Environment:    config.EnvironmentProduction,
		AllowedOrigins: []string{"https://app.example.com"},
	}
}

func runOriginOrCSRF(
	t *testing.T,
	cfg *config.Config,
	csrfCfg CSRFConfig,
	origin, referer, cookieHeader, csrfHeader string,
) *httptest.ResponseRecorder {
	t.Helper()

	e := echo.New()
	e.Use(NewOriginOrCSRFMiddleware(cfg, csrfCfg))
	e.POST("/", func(ctx echo.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if referer != "" {
		req.Header.Set("Referer", referer)
	}
	if cookieHeader != "" {
		req.Header.Set("Cookie", cookieHeader)
	}
	if csrfHeader != "" {
		headerName := csrfCfg.HeaderName
		if headerName == "" {
			headerName = defaultCSRFHeaderName
		}
		req.Header.Set(headerName, csrfHeader)
	}

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}
