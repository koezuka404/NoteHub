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

	return echomiddleware.CORSWithConfig(echomiddleware.CORSConfig{
		AllowOriginFunc: func(origin string) (bool, error) {
			_, ok := allowed[origin]
			return ok, nil
		},
		AllowCredentials: true,
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type", "X-CSRF-Token", RequestIDHeader},
		ExposeHeaders:    []string{RequestIDHeader},
	})
}
