package middleware

import (
	"strings"

	"github.com/koezuka404/notehub/config"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
)

func NewCORSMiddleware(cfg *config.Config) echo.MiddlewareFunc {
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

	suffixes := make([]string, 0, len(cfg.AllowedOriginSuffixes))
	for _, suffix := range cfg.AllowedOriginSuffixes {
		suffix = strings.TrimSpace(suffix)
		if suffix == "" {
			continue
		}
		suffixes = append(suffixes, suffix)
	}

	return echomiddleware.CORSWithConfig(echomiddleware.CORSConfig{
		AllowOriginFunc: func(origin string) (bool, error) {
			if _, ok := allowed[origin]; ok {
				return true, nil
			}
			if !strings.HasPrefix(origin, "https://") {
				return false, nil
			}
			for _, suffix := range suffixes {
				if strings.HasSuffix(origin, suffix) {
					return true, nil
				}
			}
			return false, nil
		},
		AllowCredentials: true,
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type", "X-CSRF-Token", RequestIDHeader},
		ExposeHeaders:    []string{RequestIDHeader},
	})
}
