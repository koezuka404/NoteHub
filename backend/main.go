package main

import (
	"context"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/koezuka404/notehub/authservice"
	"github.com/koezuka404/notehub/batch"
	"github.com/koezuka404/notehub/controller"
	appmiddleware "github.com/koezuka404/notehub/middleware"
	appredis "github.com/koezuka404/notehub/redis"
	"github.com/koezuka404/notehub/repository"
	"github.com/koezuka404/notehub/router"
	"github.com/koezuka404/notehub/usecase"
	infrcrypto "github.com/koezuka404/notehub/usecase/crypto"
	appws "github.com/koezuka404/notehub/websocket"
)

func main() {
	osExitFn(runFn())
}

func run() (exitCode int) {
	defer func() {
		if recover() != nil {
			exitCode = 1
		}
	}()

	cfg, err := loadConfigFn()
	if err != nil {
		fatalFn("config: %v", err)
	}
	database, err := openDatabaseFn(cfg.DatabaseURL)
	if err != nil {
		fatalFn("database open: %v", err)
	}
	defer func() {
		if err := closeDatabaseFn(database); err != nil {
			logPrintfFn("database close: %v", err)
		}
	}()
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := pingDatabaseFn(pingCtx, database); err != nil {
		fatalFn("database ping: %v", err)
	}
	if err := migrateDatabaseFn(database); err != nil {
		fatalFn("database migrate: %v", err)
	}

	jwtService, err := newJWTServiceFn(cfg)
	if err != nil {
		fatalFn("jwt: %v", err)
	}
	redisClient, err := newRedisClientFn(cfg.RedisURL, cfg.RedisOperationTimeout)
	if err != nil {
		fatalFn("redis client: %v", err)
	}
	defer func() {
		if err := closeRedisClientFn(redisClient); err != nil {
			logPrintfFn("redis close: %v", err)
		}
	}()
	redisPingCtx, redisPingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer redisPingCancel()
	if err := pingRedisClientFn(redisClient, redisPingCtx); err != nil {
		fatalFn("redis ping: %v", err)
	}

	userRepository := repository.NewUserRepository(database)
	refreshTokenRepository := repository.NewRefreshTokenRepository(database)
	auditLogRepository := repository.NewAuditLogRepository(database)
	accessTokenRevocations := appredis.NewAccessTokenRevocationStore(redisClient)
	loginFailures := appredis.NewLoginFailureStore(redisClient, cfg.LoginMaxFailures, cfg.LoginFailureWindow, cfg.LoginLockDuration)
	tokenBuckets := appredis.NewTokenBucketStore(redisClient)
	transactionManager := repository.NewTransactionManager(database)

	authService := authservice.NewAuthService(
		infrcrypto.NewPasswordService(cfg.BcryptCost),
		jwtService,
		infrcrypto.NewRandomTokenService(),
		infrcrypto.NewTokenHashService(),
		accessTokenRevocations,
		loginFailures,
	)
	authUseCase := usecase.NewAuthUseCase(
		userRepository,
		refreshTokenRepository,
		auditLogRepository,
		authService,
		transactionManager,
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
		cfg.DocumentAutosaveIdleDuration,
	)
	documentFlushBatch := batch.NewDocumentFlushBatch(documentAutoSaveUseCase)

	wsSessionStore := appredis.NewWebSocketSessionStore(redisClient)
	sessionTTL := cfg.AccessTokenTTL + time.Minute
	documentEditorsStore := appredis.NewDocumentEditorsStore(redisClient, sessionTTL)
	wsHub := appws.NewHub()
	wsEventPublisher := appws.NewDocumentEventPublisher(wsHub)

	workspaceUseCase := usecase.NewWorkspaceUseCase(
		userRepository,
		workspaceRepository,
		workspaceMemberRepository,
		auditLogRepository,
		transactionManager,
		documentAutoSaveUseCase,
		wsEventPublisher,
	)
	workspaceController := controller.NewWorkspaceController(workspaceUseCase)
	memberUseCase := usecase.NewMemberUseCase(
		userRepository,
		workspaceMemberRepository,
		auditLogRepository,
		transactionManager,
		workspaceUseCase,
		wsEventPublisher,
	)
	memberController := controller.NewMemberController(memberUseCase)
	accountUseCase := usecase.NewAccountUseCase(
		userRepository,
		refreshTokenRepository,
		workspaceMemberRepository,
		workspaceRepository,
		auditLogRepository,
		transactionManager,
		workspaceUseCase,
		wsEventPublisher,
	)
	accountController := controller.NewAccountController(accountUseCase)

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
		wsEventPublisher,
	)
	documentController := controller.NewDocumentController(documentUseCase)

	versionUseCase := usecase.NewVersionUseCase(
		documentRepository,
		versionRepository,
		transactionManager,
		workspaceUseCase,
		documentCache,
		wsEventPublisher,
		auditLogRepository,
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
	requireRefreshMiddleware := appmiddleware.NewRequireRefreshTokenMiddleware(cfg.RefreshTokenCookieName)
	originValidationMiddleware := appmiddleware.NewOriginValidationMiddleware(cfg)
	csrfMiddleware := appmiddleware.NewCSRFMiddleware(appmiddleware.CSRFConfig{
		CookieName: cfg.CSRFTokenCookieName,
		HeaderName: "X-CSRF-Token",
	})
	rateLimitMiddleware := appmiddleware.NewRateLimitMiddleware(tokenBuckets, appmiddleware.RateLimitConfig{
		Capacity: cfg.RateLimitCapacity, RefillPerSecond: cfg.RateLimitRefillRate,
	})

	e := echo.New()
	e.IPExtractor = echo.ExtractIPFromXFFHeader()
	e.Use(appmiddleware.NewRecoveryMiddleware())
	e.Use(appmiddleware.NewRequestIDMiddleware())
	e.Use(appmiddleware.NewLoggingMiddleware())
	e.Use(appmiddleware.NewCORSMiddleware(cfg))
	router.Register(e, router.Deps{
		Auth:                authController,
		Workspace:           workspaceController,
		Member:              memberController,
		Account:             accountController,
		Document:            documentController,
		Version:             versionController,
		WebSocket:           webSocketController,
		AuthMiddleware:      authMiddleware,
		RequireRefreshToken: requireRefreshMiddleware,
		OriginValidation:    originValidationMiddleware,
		CSRF:                csrfMiddleware,
		RateLimit:           rateLimitMiddleware,
	})
	startHTTPServerFn(e, cfg.HTTPPort)
	waitForShutdownSignalFn()
	batchCancel()
	flushCtx, flushCancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := flushAllDirtyFn(documentFlushBatch, flushCtx); err != nil {
		logPrintfFn("shutdown document flush: %v", err)
	}
	flushCancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := shutdownEchoFn(e, shutdownCtx); err != nil {
		logPrintfFn("shutdown: %v", err)
	}
	return 0
}
