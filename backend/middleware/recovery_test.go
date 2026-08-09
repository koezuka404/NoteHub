package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestRecoveryMiddleware_HandlesPanic(t *testing.T) {
	e := echo.New()
	e.Use(NewRequestIDMiddleware(), NewRecoveryMiddleware())
	e.GET("/panic", func(ctx echo.Context) error {
		panic("boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
	payload := decodeErrorResponse(t, rec)
	if payload.Error.Code != "INTERNAL_ERROR" {
		t.Fatalf("code = %q", payload.Error.Code)
	}
	if payload.Error.RequestID == "" {
		t.Fatal("expected request id in error response")
	}
}

func TestRecoveryMiddleware_PassesThroughNormalHandler(t *testing.T) {
	e := echo.New()
	e.Use(NewRecoveryMiddleware())
	e.GET("/ok", func(ctx echo.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}
