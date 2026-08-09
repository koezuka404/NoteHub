package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestCSRFMiddleware_MissingCookie(t *testing.T) {
	rec := runCSRF(t, CSRFConfig{CookieName: "csrf"}, "", "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rec.Code)
	}
	payload := decodeErrorResponse(t, rec)
	if payload.Error.Code != "CSRF_TOKEN_REQUIRED" {
		t.Fatalf("code = %q", payload.Error.Code)
	}
}

func TestCSRFMiddleware_EmptyCookieValue(t *testing.T) {
	rec := runCSRF(t, CSRFConfig{CookieName: "csrf"}, "csrf=; Path=/", "token")
	payload := decodeErrorResponse(t, rec)
	if payload.Error.Code != "CSRF_TOKEN_REQUIRED" {
		t.Fatalf("code = %q", payload.Error.Code)
	}
}

func TestCSRFMiddleware_MissingHeader(t *testing.T) {
	rec := runCSRF(t, CSRFConfig{CookieName: "csrf"}, "csrf=abc; Path=/", "")
	payload := decodeErrorResponse(t, rec)
	if payload.Error.Code != "CSRF_TOKEN_REQUIRED" {
		t.Fatalf("code = %q", payload.Error.Code)
	}
}

func TestCSRFMiddleware_InvalidToken(t *testing.T) {
	rec := runCSRF(t, CSRFConfig{CookieName: "csrf"}, "csrf=abc; Path=/", "xyz")
	payload := decodeErrorResponse(t, rec)
	if payload.Error.Code != "CSRF_TOKEN_INVALID" {
		t.Fatalf("code = %q", payload.Error.Code)
	}
}

func TestCSRFMiddleware_SuccessWithDefaultHeader(t *testing.T) {
	rec, ctx := runCSRFWithContext(t, CSRFConfig{CookieName: "csrf"}, "csrf=secret; Path=/", "secret")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if validated, ok := ctx.Get("csrf_validated").(bool); !ok || !validated {
		t.Fatal("expected csrf_validated in context")
	}
}

func TestCSRFMiddleware_CustomHeaderName(t *testing.T) {
	rec, _ := runCSRFWithContext(
		t,
		CSRFConfig{CookieName: "csrf", HeaderName: "X-Custom-CSRF"},
		"csrf=secret; Path=/",
		"secret",
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func runCSRF(t *testing.T, config CSRFConfig, cookieHeader, csrfHeader string) *httptest.ResponseRecorder {
	rec, _ := runCSRFWithContext(t, config, cookieHeader, csrfHeader)
	return rec
}

func runCSRFWithContext(t *testing.T, config CSRFConfig, cookieHeader, csrfHeader string) (*httptest.ResponseRecorder, echo.Context) {
	t.Helper()

	e := echo.New()
	var captured echo.Context
	e.Use(NewCSRFMiddleware(config))
	e.POST("/", func(ctx echo.Context) error {
		captured = ctx
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	if cookieHeader != "" {
		req.Header.Set("Cookie", cookieHeader)
	}
	headerName := config.HeaderName
	if headerName == "" {
		headerName = defaultCSRFHeaderName
	}
	if csrfHeader != "" {
		req.Header.Set(headerName, csrfHeader)
	}

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec, captured
}
