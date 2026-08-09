package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	appmiddleware "github.com/koezuka404/notehub/middleware"
	"github.com/labstack/echo/v4"
)

var testFixedTime = time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)

type errorPayload struct {
	Error struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		RequestID string `json:"requestId"`
	} `json:"error"`
}

func decodeErrorResponse(t *testing.T, rec *httptest.ResponseRecorder) errorPayload {
	t.Helper()
	var payload errorPayload
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	return payload
}

func newEchoContext(t *testing.T, method, target string, body any) (echo.Context, *httptest.ResponseRecorder) {
	t.Helper()
	e := echo.New()
	var req *http.Request
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		req = httptest.NewRequest(method, target, bytes.NewReader(raw))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	} else {
		req = httptest.NewRequest(method, target, nil)
	}
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

func newEchoContextWithParams(t *testing.T, method, target string, params map[string]string, body any) (echo.Context, *httptest.ResponseRecorder) {
	t.Helper()
	ctx, rec := newEchoContext(t, method, target, body)
	names := make([]string, 0, len(params)*2)
	values := make([]string, 0, len(params)*2)
	for key, value := range params {
		names = append(names, key)
		values = append(values, value)
	}
	ctx.SetParamNames(names...)
	ctx.SetParamValues(values...)
	return ctx, rec
}

func setAuthenticatedUser(ctx echo.Context, userID uuid.UUID) {
	ctx.Set(appmiddleware.ContextUserID, userID)
}

func setLogoutContext(ctx echo.Context, userID, jti uuid.UUID, expiresAt time.Time, csrfValidated bool) {
	setAuthenticatedUser(ctx, userID)
	ctx.Set(appmiddleware.ContextAccessTokenJTI, jti)
	ctx.Set(appmiddleware.ContextAccessTokenExp, expiresAt)
	ctx.Set("csrf_validated", csrfValidated)
}

func serveHandler(t *testing.T, method, path string, handler echo.HandlerFunc, setup func(echo.Context)) *httptest.ResponseRecorder {
	t.Helper()
	e := echo.New()
	var captured echo.Context
	e.Add(method, path, func(ctx echo.Context) error {
		captured = ctx
		if setup != nil {
			setup(ctx)
		}
		return handler(ctx)
	})
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if captured == nil {
		t.Fatal("handler was not invoked")
	}
	return rec
}
