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

	"github.com/koezuka404/notehub/batch"
	"github.com/koezuka404/notehub/config"
	"github.com/koezuka404/notehub/controller"
	infrcrypto "github.com/koezuka404/notehub/crypto"
	"github.com/koezuka404/notehub/db"
	appmiddleware "github.com/koezuka404/notehub/middleware"
	appredis "github.com/koezuka404/notehub/redis"
	"github.com/koezuka404/notehub/repository"
	"github.com/koezuka404/notehub/router"
	"github.com/koezuka404/notehub/usecase"
	appws "github.com/koezuka404/notehub/websocket"
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
	redisClient, err := appredis.NewClientWithTimeout(cfg.RedisURL, cfg.RedisOperationTimeout)
	if err != nil {
		log.Fatalf("redis client: %v", err)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Printf("redis close: %v", err)
		}
	}()
	redisPingCtx, redisPingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer redisPingCancel()
	if err := redisClient.Ping(redisPingCtx); err != nil {
		log.Fatalf("redis ping: %v", err)
	}

	userRepository := repository.NewUserRepository(database)
	refreshTokenRepository := repository.NewRefreshTokenRepository(database)
	auditLogRepository := repository.NewAuditLogRepository(database)
	accessTokenRevocations := appredis.NewAccessTokenRevocationStore(redisClient)
	loginFailures := appredis.NewLoginFailureStore(redisClient, cfg.LoginMaxFailures, cfg.LoginFailureWindow, cfg.LoginLockDuration)
	tokenBuckets := appredis.NewTokenBucketStore(redisClient)
	transactionManager := usecase.NewTransactionManager(database)
	authUseCase := usecase.NewAuthUseCase(
		userRepository,
		refreshTokenRepository,
		auditLogRepository,
		accessTokenRevocations,
		loginFailures,
		transactionManager,
		infrcrypto.NewPasswordService(cfg.BcryptCost),
		jwtService,
		infrcrypto.NewRandomTokenService(),
		infrcrypto.NewTokenHashService(),
		cfg.RefreshTokenTTL,
	)
	authController := controller.NewAuthController(authUseCase, controller.AuthCookieConfig{
		RefreshName: cfg.RefreshTokenCookieName,
		CSRFName:    cfg.CSRFTokenCookieName,
		Domain:      cfg.CookieDomain,
		SameSite:    cfg.CookieSameSite,
		Secure:      cfg.CookieSecure,
		RefreshTTL:  cfg.RefreshTokenTTL,
	})

	workspaceRepository := repository.NewWorkspaceRepository(database)
	workspaceMemberRepository := repository.NewWorkspaceMemberRepository(database)
	documentRepository := repository.NewDocumentRepository(database)
	versionRepository := repository.NewDocumentVersionRepository(database)
	documentCache := appredis.NewDocumentCacheStore(redisClient)
	lockStore := appredis.NewLockStore(redisClient)
	documentAutoSaveUseCase := usecase.NewDocumentAutoSaveUseCase(
		documentRepository,
		versionRepository,
		documentCache,
		lockStore,
		transactionManager,
		30*time.Second,
	)
	documentFlushBatch := batch.NewDocumentFlushBatch(documentAutoSaveUseCase)

	workspaceUseCase := usecase.NewWorkspaceUseCase(
		userRepository,
		workspaceRepository,
		workspaceMemberRepository,
		auditLogRepository,
		transactionManager,
		documentAutoSaveUseCase,
	)
	workspaceController := controller.NewWorkspaceController(workspaceUseCase)
	memberUseCase := usecase.NewMemberUseCase(
		userRepository,
		workspaceMemberRepository,
		auditLogRepository,
		transactionManager,
		workspaceUseCase,
	)
	memberController := controller.NewMemberController(memberUseCase)

	wsSessionStore := appredis.NewWebSocketSessionStore(redisClient)
	sessionTTL := cfg.AccessTokenTTL + time.Minute
	documentEditorsStore := appredis.NewDocumentEditorsStore(redisClient, sessionTTL)
	wsHub := appws.NewHub()
	wsEventPublisher := appws.NewDocumentEventPublisher(wsHub)
	documentWebSocketUseCase := usecase.NewDocumentWebSocketUseCase(
		documentRepository,
		userRepository,
		documentCache,
		wsSessionStore,
		documentEditorsStore,
		workspaceUseCase,
		cfg.WSMaxConnectionsPerDocument,
		sessionTTL,
	)
	webSocketController := controller.NewWebSocketController(
		documentWebSocketUseCase,
		jwtService,
		userRepository,
		accessTokenRevocations,
		wsHub,
		controller.WebSocketControllerConfigFromApp(cfg),
	)
	documentUseCase := usecase.NewDocumentUseCase(
		documentRepository,
		auditLogRepository,
		transactionManager,
		workspaceUseCase,
		documentCache,
		documentAutoSaveUseCase,
	)
	documentController := controller.NewDocumentController(documentUseCase)

	versionUseCase := usecase.NewVersionUseCase(
		documentRepository,
		versionRepository,
		transactionManager,
		workspaceUseCase,
		documentCache,
		wsEventPublisher,
	)
	versionController := controller.NewVersionController(versionUseCase)

	batchCtx, batchCancel := context.WithCancel(context.Background())
	defer batchCancel()
	go batch.NewAutoSaveBatch(documentAutoSaveUseCase, cfg.DocumentAutosaveInterval).Run(batchCtx)

	cleanupStore := appredis.NewCleanupStore(redisClient)
	cleanupUseCase := usecase.NewCleanupUseCase(refreshTokenRepository, cleanupStore, cfg.RefreshTokenRetention)
	go batch.NewCleanupBatch(cleanupUseCase, cfg.CleanupBatchInterval).Run(batchCtx)
	if cfg.BackupEnabled {
		backupUseCase := usecase.NewDatabaseBackupUseCase(
			cfg.DatabaseURL,
			cfg.BackupDirectory,
			cfg.BackupRetention,
			auditLogRepository,
		)
		go batch.NewBackupBatch(backupUseCase, cfg.BackupBatchInterval).Run(batchCtx)
	}

	authMiddleware := appmiddleware.NewAuthMiddleware(jwtService, userRepository, accessTokenRevocations)
	csrfMiddleware := appmiddleware.NewCSRFMiddleware(appmiddleware.CSRFConfig{
		CookieName: cfg.CSRFTokenCookieName,
		HeaderName: "X-CSRF-Token",
	})
	rateLimitMiddleware := appmiddleware.NewRateLimitMiddleware(tokenBuckets, appmiddleware.RateLimitConfig{
		Capacity: cfg.RateLimitCapacity, RefillPerSecond: cfg.RateLimitRefillRate,
	})

	e := echo.New()
	e.Use(appmiddleware.NewCORSMiddleware(cfg))
	router.Register(e, router.Deps{
		Auth:           authController,
		Workspace:      workspaceController,
		Member:         memberController,
		Document:       documentController,
		Version:        versionController,
		WebSocket:      webSocketController,
		AuthMiddleware: authMiddleware,
		CSRF:           csrfMiddleware,
		RateLimit:      rateLimitMiddleware,
	})
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
	batchCancel()
	flushCtx, flushCancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := documentFlushBatch.FlushAllDirty(flushCtx); err != nil {
		log.Printf("shutdown document flush: %v", err)
	}
	flushCancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
