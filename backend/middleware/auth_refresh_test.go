package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestRequireRefreshTokenMiddleware_MissingCookie(t *testing.T) {
	e := echo.New()
	e.Use(NewRequireRefreshTokenMiddleware("notehub_refresh_token"))
	e.POST("/", func(ctx echo.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
	payload := decodeErrorResponse(t, rec)
	if payload.Error.Code != "REFRESH_TOKEN_REQUIRED" {
		t.Fatalf("code = %q", payload.Error.Code)
	}
}

func TestRequireRefreshTokenMiddleware_Success(t *testing.T) {
	e := echo.New()
	e.Use(NewRequireRefreshTokenMiddleware("notehub_refresh_token"))
	e.POST("/", func(ctx echo.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Cookie", "notehub_refresh_token=abc; Path=/")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}
