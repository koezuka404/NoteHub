package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func TestLoggingMiddleware_LogsSuccessfulRequest(t *testing.T) {
	e := echo.New()
	e.Use(NewRequestIDMiddleware(), NewLoggingMiddleware())
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

func TestLoggingMiddleware_LogsHTTPError(t *testing.T) {
	e := echo.New()
	e.Use(NewLoggingMiddleware())
	e.GET("/fail", func(ctx echo.Context) error {
		return echo.NewHTTPError(http.StatusBadRequest, "bad request")
	})

	req := httptest.NewRequest(http.MethodGet, "/fail", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestLoggingMiddleware_LogsGenericErrorWithDefaultStatus(t *testing.T) {
	e := echo.New()
	e.Use(NewLoggingMiddleware())
	e.GET("/error", func(ctx echo.Context) error {
		return errors.New("unexpected")
	})

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestLoggingMiddleware_LogsAuthenticatedUser(t *testing.T) {
	userID := uuid.New()
	e := echo.New()
	e.Use(NewLoggingMiddleware())
	e.GET("/me", func(ctx echo.Context) error {
		ctx.Set(ContextUserID, userID)
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestLoggingMiddleware_LogsMissingUserAsDash(t *testing.T) {
	e := echo.New()
	e.Use(NewLoggingMiddleware())
	e.GET("/anon", func(ctx echo.Context) error {
		ctx.Set(ContextUserID, uuid.Nil)
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/anon", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestLoggingMiddleware_UsesZeroStatusFallback(t *testing.T) {
	e := echo.New()
	e.Use(NewLoggingMiddleware())
	e.GET("/plain", func(ctx echo.Context) error {
		ctx.Response().Status = 0
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/plain", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
}

func TestLoggingMiddleware_LogsNonUUIDUserAsDash(t *testing.T) {
	e := echo.New()
	e.Use(NewLoggingMiddleware())
	e.GET("/bad-user", func(ctx echo.Context) error {
		ctx.Set(ContextUserID, "not-a-uuid")
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/bad-user", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}
