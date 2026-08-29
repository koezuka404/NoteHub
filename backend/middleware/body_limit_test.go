package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestNewBodyLimitMiddleware_ContentLengthTooLarge(t *testing.T) {
	e := echo.New()

	handler := NewBodyLimitMiddleware(10)(
		func(c echo.Context) error {
			t.Fatal("handler should not be called")
			return nil
		},
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/",
		strings.NewReader("12345678901"),
	)
	req.ContentLength = 11

	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	if err := handler(ctx); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
	if !strings.Contains(rec.Body.String(), "REQUEST_BODY_TOO_LARGE") {
		t.Fatalf("response body = %s", rec.Body.String())
	}
}

func TestNewBodyLimitMiddleware_WithinLimit(t *testing.T) {
	e := echo.New()

	handler := NewBodyLimitMiddleware(10)(
		func(c echo.Context) error {
			body, err := io.ReadAll(c.Request().Body)
			if err != nil {
				return err
			}
			return c.String(http.StatusOK, string(body))
		},
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/",
		strings.NewReader("1234567890"),
	)
	req.ContentLength = 10

	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	if err := handler(ctx); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec.Body.String() != "1234567890" {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestNewBodyLimitMiddleware_ActualReadTooLarge(t *testing.T) {
	e := echo.New()

	handler := NewBodyLimitMiddleware(10)(
		func(c echo.Context) error {
			t.Fatal("handler should not be called")
			return nil
		},
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/",
		strings.NewReader("12345678901"),
	)
	req.ContentLength = -1

	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	if err := handler(ctx); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestNewBodyLimitMiddleware_Default(t *testing.T) {
	e := echo.New()

	handler := NewBodyLimitMiddleware(0)(
		func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		},
	)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.ContentLength = DefaultMaxRequestBodyBytes + 1

	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	if err := handler(ctx); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestNewBodyLimitMiddleware_SkipsGet(t *testing.T) {
	e := echo.New()

	handler := NewBodyLimitMiddleware(1)(
		func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.ContentLength = 100

	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	if err := handler(ctx); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}
