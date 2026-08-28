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

	err := handler(ctx)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf(
			"status = %d, want %d",
			rec.Code,
			http.StatusRequestEntityTooLarge,
		)
	}

	if !strings.Contains(
		rec.Body.String(),
		"REQUEST_BODY_TOO_LARGE",
	) {
		t.Fatalf(
			"response body does not contain REQUEST_BODY_TOO_LARGE: %s",
			rec.Body.String(),
		)
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

			return c.String(
				http.StatusOK,
				string(body),
			)
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

	err := handler(ctx)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"status = %d, want %d",
			rec.Code,
			http.StatusOK,
		)
	}

	if rec.Body.String() != "1234567890" {
		t.Fatalf(
			"body = %q, want %q",
			rec.Body.String(),
			"1234567890",
		)
	}
}

func TestNewBodyLimitMiddleware_ActualReadTooLarge(t *testing.T) {
	e := echo.New()

	handler := NewBodyLimitMiddleware(10)(
		func(c echo.Context) error {
			_, err := io.ReadAll(c.Request().Body)

			if err != nil {
				return WriteError(
					c,
					http.StatusRequestEntityTooLarge,
					"REQUEST_BODY_TOO_LARGE",
					"request body is too large",
				)
			}

			return c.NoContent(http.StatusOK)
		},
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/",
		strings.NewReader("12345678901"),
	)

	// Content-Lengthを設定しないケースを想定。
	req.ContentLength = -1

	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	err := handler(ctx)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf(
			"status = %d, want %d",
			rec.Code,
			http.StatusRequestEntityTooLarge,
		)
	}
}

func TestNewBodyLimitMiddleware_Default(t *testing.T) {
	e := echo.New()

	handler := NewBodyLimitMiddleware(0)(
		func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		},
	)

	// デフォルト2MiBが設定されていることを確認するため、
	// 2MiBを超えるContent-Lengthを指定する。
	req := httptest.NewRequest(
		http.MethodPost,
		"/",
		nil,
	)

	req.ContentLength = DefaultMaxRequestBodyBytes + 1

	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	err := handler(ctx)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf(
			"status = %d, want %d",
			rec.Code,
			http.StatusRequestEntityTooLarge,
		)
	}
}
