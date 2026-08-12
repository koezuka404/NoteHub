package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

const defaultCSRFHeaderName = "X-CSRF-Token"

type CSRFConfig struct {
	CookieName string
	HeaderName string
}

func NewCSRFMiddleware(config CSRFConfig) echo.MiddlewareFunc {
	cookieName := strings.TrimSpace(config.CookieName)
	headerName := strings.TrimSpace(config.HeaderName)
	if headerName == "" {
		headerName = defaultCSRFHeaderName
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			cookie, err := ctx.Cookie(cookieName)
			if err != nil || cookie.Value == "" {
				return writeCSRFError(ctx, "CSRF_TOKEN_REQUIRED", "CSRFトークンが必要です")
			}

			headerToken := ctx.Request().Header.Get(headerName)
			if headerToken == "" {
				return writeCSRFError(ctx, "CSRF_TOKEN_REQUIRED", "CSRFトークンが必要です")
			}

			cookieToken := []byte(cookie.Value)
			headerTokenBytes := []byte(headerToken)
			if len(cookieToken) != len(headerTokenBytes) || subtle.ConstantTimeCompare(cookieToken, headerTokenBytes) != 1 {
				return writeCSRFError(ctx, "CSRF_TOKEN_INVALID", "CSRFトークンが不正です")
			}

			ctx.Set("csrf_validated", true)
			return next(ctx)
		}
	}
}

func writeCSRFError(ctx echo.Context, code, message string) error {
	return WriteError(ctx, http.StatusForbidden, code, message)
}
