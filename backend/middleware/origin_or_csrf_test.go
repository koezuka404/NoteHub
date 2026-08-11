package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/koezuka404/notehub/config"
	"github.com/labstack/echo/v4"
)

func TestOriginOrCSRFMiddleware_AllowsOrigin(t *testing.T) {
	rec := runOriginOrCSRF(t, "https://app.example.com", "", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestOriginOrCSRFMiddleware_AllowsCSRF(t *testing.T) {
	rec := runOriginOrCSRF(t, "", "csrf=secret; Path=/", "secret", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestOriginOrCSRFMiddleware_RejectsMissingBoth(t *testing.T) {
	rec := runOriginOrCSRF(t, "", "", "", "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rec.Code)
	}
}

func runOriginOrCSRF(t *testing.T, origin, cookieHeader, csrfHeader, referer string) *httptest.ResponseRecorder {
	t.Helper()

	e := echo.New()
	e.Use(NewOriginOrCSRFMiddleware(
		&config.Config{Environment: config.EnvironmentProduction, AllowedOrigins: []string{"https://app.example.com"}},
		CSRFConfig{CookieName: "csrf"},
	))
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
		req.Header.Set(defaultCSRFHeaderName, csrfHeader)
	}

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}
