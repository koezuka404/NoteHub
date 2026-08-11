package middleware

import (
	"net/http"
	"strings"

	"github.com/koezuka404/notehub/config"
	"github.com/labstack/echo/v4"
)

func NewOriginValidationMiddleware(cfg *config.Config) echo.MiddlewareFunc {
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

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			if len(allowed) == 0 {
				return next(ctx)
			}

			origin := strings.TrimSpace(ctx.Request().Header.Get("Origin"))
			if origin != "" {
				if _, ok := allowed[origin]; ok {
					ctx.Set("origin_validated", true)
					return next(ctx)
				}
				return writeOriginError(ctx)
			}

			referer := strings.TrimSpace(ctx.Request().Header.Get("Referer"))
			if referer != "" {
				for allowedOrigin := range allowed {
					if referer == allowedOrigin || strings.HasPrefix(referer, allowedOrigin+"/") {
						ctx.Set("origin_validated", true)
						return next(ctx)
					}
				}
			}

			return writeOriginError(ctx)
		}
	}
}

func writeOriginError(ctx echo.Context) error {
	return WriteError(ctx, http.StatusForbidden, "ORIGIN_NOT_ALLOWED", "リクエスト元が許可されていません")
}
