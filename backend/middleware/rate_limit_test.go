package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
)

type mockTokenBucketLimiter struct {
	allowed    bool
	retryAfter time.Duration
	err        error
}

func (m *mockTokenBucketLimiter) Allow(context.Context, string, int, float64, time.Time) (bool, time.Duration, error) {
	return m.allowed, m.retryAfter, m.err
}

func runRateLimit(t *testing.T, limiter ITokenBucketLimiter) *httptest.ResponseRecorder {
	t.Helper()

	e := echo.New()
	e.Use(NewRateLimitMiddleware(limiter, RateLimitConfig{Capacity: 10, RefillPerSecond: 1}))
	e.GET("/resource", func(ctx echo.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	req.RemoteAddr = "203.0.113.10:1234"
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestRateLimitMiddleware_ServiceUnavailable(t *testing.T) {
	rec := runRateLimit(t, &mockTokenBucketLimiter{err: errors.New("redis down")})
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", rec.Code)
	}
	payload := decodeErrorResponse(t, rec)
	if payload.Error.Code != "RATE_LIMIT_SERVICE_UNAVAILABLE" {
		t.Fatalf("code = %q", payload.Error.Code)
	}
}

func TestRateLimitMiddleware_Exceeded(t *testing.T) {
	rec := runRateLimit(t, &mockTokenBucketLimiter{allowed: false, retryAfter: 2500 * time.Millisecond})
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := rec.Header().Get("Retry-After"); got != "3" {
		t.Fatalf("retry-after = %q", got)
	}
	payload := decodeErrorResponse(t, rec)
	if payload.Error.Code != "RATE_LIMIT_EXCEEDED" {
		t.Fatalf("code = %q", payload.Error.Code)
	}
}

func TestRateLimitMiddleware_ExceededMinimumRetryAfter(t *testing.T) {
	rec := runRateLimit(t, &mockTokenBucketLimiter{allowed: false, retryAfter: 100 * time.Millisecond})
	if got := rec.Header().Get("Retry-After"); got != "1" {
		t.Fatalf("retry-after = %q", got)
	}
}

func TestRateLimitMiddleware_ExceededZeroRetryAfter(t *testing.T) {
	rec := runRateLimit(t, &mockTokenBucketLimiter{allowed: false, retryAfter: 0})
	if got := rec.Header().Get("Retry-After"); got != "1" {
		t.Fatalf("retry-after = %q", got)
	}
}

func TestRateLimitMiddleware_ExceededExactSecondRetryAfter(t *testing.T) {
	rec := runRateLimit(t, &mockTokenBucketLimiter{allowed: false, retryAfter: time.Second})
	if got := rec.Header().Get("Retry-After"); got != "1" {
		t.Fatalf("retry-after = %q", got)
	}
}

func TestRateLimitMiddleware_Allowed(t *testing.T) {
	rec := runRateLimit(t, &mockTokenBucketLimiter{allowed: true})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}
