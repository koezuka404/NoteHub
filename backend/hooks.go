package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/koezuka404/notehub/batch"
	"github.com/koezuka404/notehub/config"
	infrcrypto "github.com/koezuka404/notehub/usecase/crypto"
	"github.com/koezuka404/notehub/db"
	appredis "github.com/koezuka404/notehub/redis"
	"github.com/labstack/echo/v4"
)

var (
	loadConfigFn = config.Load
	openDatabaseFn = db.Open
	pingDatabaseFn = db.Ping
	migrateDatabaseFn = db.Migrate
	closeDatabaseFn = db.Close
	newJWTServiceFn = func(cfg *config.Config) (*infrcrypto.JWTService, error) {
		return infrcrypto.NewJWTService(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTAudience, cfg.AccessTokenTTL)
	}
	newRedisClientFn = appredis.NewClientWithTimeout
	pingRedisClientFn = func(client *appredis.Client, ctx context.Context) error {
		return client.Ping(ctx)
	}
	closeRedisClientFn = func(client *appredis.Client) error {
		return client.Close()
	}
	getenvFn = os.Getenv
	echoStartFn = func(e *echo.Echo, address string) error {
		return e.Start(address)
	}
	startHTTPServerFn = defaultStartHTTPServer
	waitForShutdownSignalFn = defaultWaitForShutdownSignal
	shutdownEchoFn = func(e *echo.Echo, ctx context.Context) error {
		return e.Shutdown(ctx)
	}
	flushAllDirtyFn = func(flushBatch *batch.DocumentFlushBatch, ctx context.Context) error {
		return flushBatch.FlushAllDirty(ctx)
	}
	logPrintfFn = log.Printf
	fatalFn = func(format string, args ...any) {
		log.Fatalf(format, args...)
	}
	osExitFn = os.Exit
	runFn = run
)

func defaultStartHTTPServer(e *echo.Echo, port int) {
	go func() {
		address := ":" + strconv.Itoa(port)
		if publicURL := getenvFn("PUBLIC_HTTP_URL"); publicURL != "" {
			logPrintfFn("NoteHub backend ready at %s", publicURL)
			logPrintfFn("Health check: %s/health", publicURL)
		}
		logPrintfFn("listening on %s", address)
		if err := echoStartFn(e, address); err != nil && err != http.ErrServerClosed {
			fatalFn("server: %v", err)
		}
	}()
}

func defaultWaitForShutdownSignal() {
	stop := make(chan os.Signal, 1)
	notifyShutdownSignalsFn(stop)
	<-stop
}

var notifyShutdownSignalsFn = func(c chan os.Signal) {
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
}
