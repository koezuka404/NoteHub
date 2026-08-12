package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

func NewSecFetchSiteMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			secFetchSite := strings.TrimSpace(ctx.Request().Header.Get(echo.HeaderSecFetchSite))
			if secFetchSite == "" {
				return next(ctx)
			}

			switch secFetchSite {
			case "same-origin", "none":
				ctx.Set("sec_fetch_site_validated", true)
				return next(ctx)
			default:
				return writeSecFetchSiteError(ctx)
			}
		}
	}
}

func writeSecFetchSiteError(ctx echo.Context) error {
	return WriteError(ctx, http.StatusForbidden, "SEC_FETCH_SITE_BLOCKED", "リクエスト元が許可されていません")
}
