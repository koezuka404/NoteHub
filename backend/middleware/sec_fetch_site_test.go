package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestSecFetchSiteMiddleware_SuccessWithSameOrigin(t *testing.T) {
	rec, ctx := runSecFetchSiteWithContext(t, "same-origin", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if validated, ok := ctx.Get("sec_fetch_site_validated").(bool); !ok || !validated {
		t.Fatal("expected sec_fetch_site_validated in context")
	}
}

func TestSecFetchSiteMiddleware_AllowsNone(t *testing.T) {
	rec := runSecFetchSite(t, "none", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestSecFetchSiteMiddleware_PassesCrossSiteForCSRFFallback(t *testing.T) {
	rec := runSecFetchSite(t, "cross-site", "https://evil.example.com", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestSecFetchSiteMiddleware_PassesSameSiteForCSRFFallback(t *testing.T) {
	rec := runSecFetchSite(t, "same-site", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestSecFetchSiteMiddleware_SkipsWhenHeaderMissing(t *testing.T) {
	rec := runSecFetchSite(t, "", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestSecFetchSiteMiddleware_RejectsUnknownValue(t *testing.T) {
	rec := runSecFetchSite(t, "unknown", "", "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestSecFetchSiteWithCSRFFallback_RejectsCrossSiteWithoutToken(t *testing.T) {
	e := echo.New()
	e.Use(NewSecFetchSiteMiddleware())
	e.Use(NewCSRFMiddleware(CSRFConfig{CookieName: "csrf"}))
	e.POST("/", func(ctx echo.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set(echo.HeaderSecFetchSite, "cross-site")
	req.Header.Set(echo.HeaderOrigin, "https://app.example.com")

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rec.Code)
	}
	payload := decodeErrorResponse(t, rec)
	if payload.Error.Code != "CSRF_TOKEN_REQUIRED" {
		t.Fatalf("code = %q", payload.Error.Code)
	}
}

func runSecFetchSite(t *testing.T, secFetchSite, originHeader, refererHeader string) *httptest.ResponseRecorder {
	t.Helper()
	rec, _ := runSecFetchSiteWithContext(t, secFetchSite, originHeader, refererHeader)
	return rec
}

func runSecFetchSiteWithContext(t *testing.T, secFetchSite, originHeader, refererHeader string) (*httptest.ResponseRecorder, echo.Context) {
	t.Helper()

	e := echo.New()
	var captured echo.Context
	e.Use(NewSecFetchSiteMiddleware())
	e.POST("/", func(ctx echo.Context) error {
		captured = ctx
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	if secFetchSite != "" {
		req.Header.Set(echo.HeaderSecFetchSite, secFetchSite)
	}
	if originHeader != "" {
		req.Header.Set(echo.HeaderOrigin, originHeader)
	}
	if refererHeader != "" {
		req.Header.Set("Referer", refererHeader)
	}

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec, captured
}
