package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

func NewRequireRefreshTokenMiddleware(refreshCookieName string) echo.MiddlewareFunc {
	cookieName := strings.TrimSpace(refreshCookieName)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			cookie, err := ctx.Cookie(cookieName)
			if err != nil || strings.TrimSpace(cookie.Value) == "" {
				return WriteError(ctx, http.StatusUnauthorized, "REFRESH_TOKEN_REQUIRED", "ログインが必要です")
			}
			return next(ctx)
		}
	}
}
