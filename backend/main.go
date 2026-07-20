package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/koezuka404/notehub/config"
	"github.com/koezuka404/notehub/controller"
	infrcrypto "github.com/koezuka404/notehub/crypto"
	"github.com/koezuka404/notehub/db"
	"github.com/koezuka404/notehub/repository"
	"github.com/koezuka404/notehub/router"
	"github.com/koezuka404/notehub/usecase"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	database, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database open: %v", err)
	}
	defer func() {
		if err := db.Close(database); err != nil {
			log.Printf("database close: %v", err)
		}
	}()
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := db.Ping(pingCtx, database); err != nil {
		log.Fatalf("database ping: %v", err)
	}
	if err := db.Migrate(database); err != nil {
		log.Fatalf("database migrate: %v", err)
	}

	jwtService, err := infrcrypto.NewJWTService(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTAudience, cfg.AccessTokenTTL)
	if err != nil {
		log.Fatalf("jwt: %v", err)
	}
	userRepository := repository.NewUserRepository(database)
	refreshTokenRepository := repository.NewRefreshTokenRepository(database)
	transactionManager := repository.NewTransactionManager(database)
	authUseCase := usecase.NewAuthUseCase(
		userRepository,
		refreshTokenRepository,
		transactionManager,
		infrcrypto.NewPasswordService(cfg.BcryptCost),
		jwtService,
		infrcrypto.NewRandomTokenService(),
		infrcrypto.NewTokenHashService(),
		cfg.RefreshTokenTTL,
	)
	authController := controller.NewAuthController(authUseCase, controller.AuthCookieConfig{
		RefreshName: cfg.RefreshTokenCookieName,
		Domain:      cfg.CookieDomain,
		SameSite:    cfg.CookieSameSite,
		Secure:      cfg.CookieSecure,
		RefreshTTL:  cfg.RefreshTokenTTL,
	})

	e := echo.New()
	router.Register(e, router.Deps{Auth: authController})
	go func() {
		address := ":" + strconv.Itoa(cfg.HTTPPort)
		log.Printf("listening on %s", address)
		if err := e.Start(address); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
