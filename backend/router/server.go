package router

import (
	"github.com/koezuka404/notehub/config"
	appmiddleware "github.com/koezuka404/notehub/middleware"
	"github.com/labstack/echo/v4"
)

func New(cfg *config.Config, deps Deps) *echo.Echo {
	e := echo.New()
	e.IPExtractor = echo.ExtractIPFromXFFHeader()
	e.Use(appmiddleware.NewRecoveryMiddleware())
	e.Use(appmiddleware.NewRequestIDMiddleware())
	e.Use(appmiddleware.NewLoggingMiddleware())
	e.Use(appmiddleware.NewCORSMiddleware(cfg))
	Register(e, deps)
	return e
}
