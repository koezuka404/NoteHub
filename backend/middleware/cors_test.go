package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/koezuka404/notehub/config"
	"github.com/labstack/echo/v4"
)

func TestCORSMiddleware_DevelopmentDefaults(t *testing.T) {
	e := echo.New()
	e.Use(NewCORSMiddleware(&config.Config{Environment: config.EnvironmentDevelopment}))
	e.GET("/", func(ctx echo.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("allow-origin = %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("allow-credentials = %q", got)
	}
}

func TestCORSMiddleware_CustomOrigins(t *testing.T) {
	e := echo.New()
	e.Use(NewCORSMiddleware(&config.Config{
		Environment:    config.EnvironmentProduction,
		AllowedOrigins: []string{"https://notehub.example"},
	}))
	e.GET("/", func(ctx echo.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://notehub.example")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://notehub.example" {
		t.Fatalf("allow-origin = %q", got)
	}
	if got := rec.Header().Get("Access-Control-Expose-Headers"); got == "" {
		t.Fatal("expected exposed headers")
	}
}

func TestCORSMiddleware_AllowsOriginSuffix(t *testing.T) {
	e := echo.New()
	e.Use(NewCORSMiddleware(&config.Config{
		Environment:           config.EnvironmentProduction,
		AllowedOrigins:        []string{"https://note-hub-three.vercel.app"},
		AllowedOriginSuffixes: []string{".vercel.app"},
	}))
	e.GET("/", func(ctx echo.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://note-hub-git-main-koezuka404s-projects.vercel.app")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://note-hub-git-main-koezuka404s-projects.vercel.app" {
		t.Fatalf("allow-origin = %q", got)
	}
}

func TestCORSMiddleware_RejectsUnknownOrigin(t *testing.T) {
	e := echo.New()
	e.Use(NewCORSMiddleware(&config.Config{
		Environment:           config.EnvironmentProduction,
		AllowedOrigins:        []string{"https://note-hub-three.vercel.app"},
		AllowedOriginSuffixes: []string{".vercel.app"},
	}))
	e.GET("/", func(ctx echo.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("allow-origin = %q, want empty", got)
	}
}
