package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/koezuka404/notehub/config"
	"github.com/labstack/echo/v4"
)

func NewOriginOrCSRFMiddleware(cfg *config.Config, csrfCfg CSRFConfig) echo.MiddlewareFunc {
	allowedOrigins := allowedOriginsFromConfig(cfg)
	cookieName := strings.TrimSpace(csrfCfg.CookieName)
	headerName := strings.TrimSpace(csrfCfg.HeaderName)
	if headerName == "" {
		headerName = defaultCSRFHeaderName
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			if originAllowed(ctx, allowedOrigins) {
				ctx.Set("origin_validated", true)
				return next(ctx)
			}
			if csrfDoubleSubmitValid(ctx, cookieName, headerName) {
				ctx.Set("csrf_validated", true)
				return next(ctx)
			}
			return WriteError(ctx, http.StatusForbidden, "CSRF_TOKEN_REQUIRED", "CSRFトークンが必要です")
		}
	}
}

func allowedOriginsFromConfig(cfg *config.Config) map[string]struct{} {
	origins := cfg.AllowedOrigins
	if len(origins) == 0 && cfg.Environment == config.EnvironmentDevelopment {
		origins = []string{"http://localhost:5173", "http://127.0.0.1:5173"}
	}
	allowed := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		allowed[origin] = struct{}{}
	}
	return allowed
}

func originAllowed(ctx echo.Context, allowed map[string]struct{}) bool {
	if len(allowed) == 0 {
		return false
	}

	origin := strings.TrimSpace(ctx.Request().Header.Get("Origin"))
	if origin != "" {
		_, ok := allowed[origin]
		return ok
	}

	referer := strings.TrimSpace(ctx.Request().Header.Get("Referer"))
	if referer == "" {
		return false
	}
	for allowedOrigin := range allowed {
		if referer == allowedOrigin || strings.HasPrefix(referer, allowedOrigin+"/") {
			return true
		}
	}
	return false
}

func csrfDoubleSubmitValid(ctx echo.Context, cookieName, headerName string) bool {
	cookie, err := ctx.Cookie(cookieName)
	if err != nil || cookie.Value == "" {
		return false
	}
	headerToken := ctx.Request().Header.Get(headerName)
	if headerToken == "" {
		return false
	}
	cookieToken := []byte(cookie.Value)
	headerTokenBytes := []byte(headerToken)
	return len(cookieToken) == len(headerTokenBytes) && subtle.ConstantTimeCompare(cookieToken, headerTokenBytes) == 1
}
