package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/koezuka404/notehub/config"
	"github.com/labstack/echo/v4"
)

func TestOriginValidationMiddleware_AllowsListedOrigin(t *testing.T) {
	rec := runOriginValidation(t, []string{"https://app.example.com"}, "https://app.example.com", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestOriginValidationMiddleware_AllowsListedReferer(t *testing.T) {
	rec := runOriginValidation(t, []string{"https://app.example.com"}, "", "https://app.example.com/dashboard")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestOriginValidationMiddleware_RejectsUnknownOrigin(t *testing.T) {
	rec := runOriginValidation(t, []string{"https://app.example.com"}, "https://evil.example.com", "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rec.Code)
	}
	payload := decodeErrorResponse(t, rec)
	if payload.Error.Code != "ORIGIN_NOT_ALLOWED" {
		t.Fatalf("code = %q", payload.Error.Code)
	}
}

func TestOriginValidationMiddleware_RejectsMissingOriginAndReferer(t *testing.T) {
	rec := runOriginValidation(t, []string{"https://app.example.com"}, "", "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestOriginValidationMiddleware_DevelopmentDefaults(t *testing.T) {
	rec := runOriginValidationWithConfig(t, &config.Config{
		Environment: config.EnvironmentDevelopment,
	}, "http://localhost:5173", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func runOriginValidation(t *testing.T, origins []string, originHeader, refererHeader string) *httptest.ResponseRecorder {
	t.Helper()
	return runOriginValidationWithConfig(t, &config.Config{
		Environment:    config.EnvironmentProduction,
		AllowedOrigins: origins,
	}, originHeader, refererHeader)
}

func runOriginValidationWithConfig(t *testing.T, cfg *config.Config, originHeader, refererHeader string) *httptest.ResponseRecorder {
	t.Helper()

	e := echo.New()
	e.Use(NewOriginValidationMiddleware(cfg))
	e.POST("/", func(ctx echo.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	if originHeader != "" {
		req.Header.Set("Origin", originHeader)
	}
	if refererHeader != "" {
		req.Header.Set("Referer", refererHeader)
	}

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}
