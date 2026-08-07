package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestRequestIDMiddleware_UsesIncomingHeader(t *testing.T) {
	e := echo.New()
	e.Use(NewRequestIDMiddleware())
	e.GET("/health", func(ctx echo.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set(RequestIDHeader, "client-request-id")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if got := rec.Header().Get(RequestIDHeader); got != "client-request-id" {
		t.Fatalf("response header = %q, want client-request-id", got)
	}
}

func TestWriteError_IncludesRequestID(t *testing.T) {
	e := echo.New()
	e.Use(NewRequestIDMiddleware())
	e.GET("/error", func(ctx echo.Context) error {
		return WriteError(ctx, http.StatusBadRequest, "VALIDATION_ERROR", "入力値が不正です")
	})

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	var payload struct {
		Error struct {
			Code      string `json:"code"`
			Message   string `json:"message"`
			RequestID string `json:"requestId"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Error.Code != "VALIDATION_ERROR" {
		t.Fatalf("code = %q", payload.Error.Code)
	}
	if payload.Error.RequestID == "" {
		t.Fatal("expected requestId in error response")
	}
	if payload.Error.RequestID != rec.Header().Get(RequestIDHeader) {
		t.Fatalf("requestId mismatch: body=%q header=%q", payload.Error.RequestID, rec.Header().Get(RequestIDHeader))
	}
}

func TestNormalizeRequestID_RejectsInvalidCharacters(t *testing.T) {
	if got := normalizeRequestID("bad id"); got != "" {
		t.Fatalf("normalizeRequestID() = %q, want empty", got)
	}
}
