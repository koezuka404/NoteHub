package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/glebarez/sqlite"
	"github.com/koezuka404/notehub/batch"
	"github.com/koezuka404/notehub/config"
	appredis "github.com/koezuka404/notehub/redis"
	infrcrypto "github.com/koezuka404/notehub/usecase/crypto"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

const testJWTSecret = "01234567890123456789012345678901"

type hookSnapshot struct {
	loadConfig            func() (*config.Config, error)
	openDatabase          func(string) (*gorm.DB, error)
	pingDatabase          func(context.Context, *gorm.DB) error
	migrateDatabase       func(*gorm.DB) error
	closeDatabase         func(*gorm.DB) error
	newJWTService         func(*config.Config) (*infrcrypto.JWTService, error)
	newRedisClient        func(string, time.Duration) (*appredis.Client, error)
	pingRedisClient       func(*appredis.Client, context.Context) error
	closeRedisClient      func(*appredis.Client) error
	getenv                func(string) string
	echoStart             func(*echo.Echo, string) error
	startHTTPServer       func(*echo.Echo, int)
	waitForShutdownSignal func()
	shutdownEcho          func(*echo.Echo, context.Context) error
	flushAllDirty         func(*batch.DocumentFlushBatch, context.Context) error
	logPrintf             func(string, ...any)
	fatal                 func(string, ...any)
	osExit                func(int)
	run                   func() int
	notifyShutdownSignals func(chan os.Signal)
}

func saveHooks() hookSnapshot {
	return hookSnapshot{
		loadConfig:            loadConfigFn,
		openDatabase:          openDatabaseFn,
		pingDatabase:          pingDatabaseFn,
		migrateDatabase:       migrateDatabaseFn,
		closeDatabase:         closeDatabaseFn,
		newJWTService:         newJWTServiceFn,
		newRedisClient:        newRedisClientFn,
		pingRedisClient:       pingRedisClientFn,
		closeRedisClient:      closeRedisClientFn,
		getenv:                getenvFn,
		echoStart:             echoStartFn,
		startHTTPServer:       startHTTPServerFn,
		waitForShutdownSignal: waitForShutdownSignalFn,
		shutdownEcho:          shutdownEchoFn,
		flushAllDirty:         flushAllDirtyFn,
		logPrintf:             logPrintfFn,
		fatal:                 fatalFn,
		osExit:                osExitFn,
		run:                   runFn,
		notifyShutdownSignals: notifyShutdownSignalsFn,
	}
}

func (s hookSnapshot) restore() {
	loadConfigFn = s.loadConfig
	openDatabaseFn = s.openDatabase
	pingDatabaseFn = s.pingDatabase
	migrateDatabaseFn = s.migrateDatabase
	closeDatabaseFn = s.closeDatabase
	newJWTServiceFn = s.newJWTService
	newRedisClientFn = s.newRedisClient
	pingRedisClientFn = s.pingRedisClient
	closeRedisClientFn = s.closeRedisClient
	getenvFn = s.getenv
	echoStartFn = s.echoStart
	startHTTPServerFn = s.startHTTPServer
	waitForShutdownSignalFn = s.waitForShutdownSignal
	shutdownEchoFn = s.shutdownEcho
	flushAllDirtyFn = s.flushAllDirty
	logPrintfFn = s.logPrintf
	fatalFn = s.fatal
	osExitFn = s.osExit
	runFn = s.run
	notifyShutdownSignalsFn = s.notifyShutdownSignals
}

func testFatalHook(t *testing.T) {
	t.Helper()
	fatalFn = func(format string, args ...any) {
		panic(fmt.Sprintf(format, args...))
	}
}

func validTestConfig(redisURL, backupDir string, backupEnabled bool) *config.Config {
	return &config.Config{
		Environment:                 config.EnvironmentTest,
		HTTPPort:                    0,
		DatabaseURL:                 "postgres://example",
		RedisURL:                    redisURL,
		RedisOperationTimeout:       2 * time.Second,
		JWTSecret:                   testJWTSecret,
		JWTIssuer:                   "notehub-api",
		JWTAudience:                 "notehub-client",
		AccessTokenTTL:              15 * time.Minute,
		RefreshTokenTTL:             30 * time.Minute,
		BcryptCost:                  4,
		CookieSameSite:              "Lax",
		RefreshTokenCookieName:      "refresh_token",
		CSRFTokenCookieName:         "csrf_token",
		AllowedOrigins:              []string{"http://localhost:5173"},
		LoginMaxFailures:            3,
		LoginFailureWindow:          time.Minute,
		LoginLockDuration:           5 * time.Minute,
		RateLimitCapacity:           10,
		RateLimitRefillRate:         1,
		WSMaxConnectionsPerDocument: 2,
		DocumentAutosaveInterval:    time.Hour,
		DocumentAutosaveIdleDuration: time.Hour,
		CleanupBatchInterval:        time.Hour,
		RefreshTokenRetention:       24 * time.Hour,
		BackupEnabled:               backupEnabled,
		BackupBatchInterval:         time.Hour,
		BackupDirectory:             backupDir,
		BackupRetention:             24 * time.Hour,
	}
}

func setupSuccessfulRunHooks(t *testing.T, backupEnabled bool, publicURL string) {
	t.Helper()
	testFatalHook(t)

	mr := miniredis.RunT(t)
	cfg := validTestConfig("redis://"+mr.Addr()+"/0", t.TempDir(), backupEnabled)

	loadConfigFn = func() (*config.Config, error) { return cfg, nil }
	openDatabaseFn = func(string) (*gorm.DB, error) {
		return gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	}
	migrateDatabaseFn = func(*gorm.DB) error { return nil }
	newRedisClientFn = func(url string, timeout time.Duration) (*appredis.Client, error) {
		return appredis.NewClientWithTimeout(url, timeout)
	}
	waitForShutdownSignalFn = func() {}
	startHTTPServerFn = defaultStartHTTPServer
	getenvFn = func(key string) string {
		if key == "PUBLIC_HTTP_URL" && publicURL != "" {
			return publicURL
		}
		return ""
	}
}

func TestRun_Success(t *testing.T) {
	hooks := saveHooks()
	t.Cleanup(hooks.restore)

	setupSuccessfulRunHooks(t, true, "http://localhost:8080")
	if code := run(); code != 0 {
		t.Fatalf("run() = %d, want 0", code)
	}
}

func TestRun_SuccessWithoutBackupAndPublicURL(t *testing.T) {
	hooks := saveHooks()
	t.Cleanup(hooks.restore)

	setupSuccessfulRunHooks(t, false, "")
	if code := run(); code != 0 {
		t.Fatalf("run() = %d, want 0", code)
	}
}

func TestRun_ConfigError(t *testing.T) {
	hooks := saveHooks()
	t.Cleanup(hooks.restore)
	testFatalHook(t)

	loadConfigFn = func() (*config.Config, error) {
		return nil, errors.New("config failed")
	}
	if code := run(); code != 1 {
		t.Fatalf("run() = %d, want 1", code)
	}
}

func TestRun_DatabaseOpenError(t *testing.T) {
	hooks := saveHooks()
	t.Cleanup(hooks.restore)
	testFatalHook(t)

	loadConfigFn = func() (*config.Config, error) { return validTestConfig("redis://unused", t.TempDir(), false), nil }
	openDatabaseFn = func(string) (*gorm.DB, error) { return nil, errors.New("open failed") }
	if code := run(); code != 1 {
		t.Fatalf("run() = %d, want 1", code)
	}
}

func TestRun_DatabasePingError(t *testing.T) {
	hooks := saveHooks()
	t.Cleanup(hooks.restore)
	testFatalHook(t)

	loadConfigFn = func() (*config.Config, error) { return validTestConfig("redis://unused", t.TempDir(), false), nil }
	openDatabaseFn = func(string) (*gorm.DB, error) {
		return gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	}
	pingDatabaseFn = func(context.Context, *gorm.DB) error { return errors.New("ping failed") }
	if code := run(); code != 1 {
		t.Fatalf("run() = %d, want 1", code)
	}
}

func TestRun_DatabaseMigrateError(t *testing.T) {
	hooks := saveHooks()
	t.Cleanup(hooks.restore)
	testFatalHook(t)

	loadConfigFn = func() (*config.Config, error) { return validTestConfig("redis://unused", t.TempDir(), false), nil }
	openDatabaseFn = func(string) (*gorm.DB, error) {
		return gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	}
	migrateDatabaseFn = func(*gorm.DB) error { return errors.New("migrate failed") }
	if code := run(); code != 1 {
		t.Fatalf("run() = %d, want 1", code)
	}
}

func TestRun_JWTError(t *testing.T) {
	hooks := saveHooks()
	t.Cleanup(hooks.restore)
	testFatalHook(t)

	loadConfigFn = func() (*config.Config, error) { return validTestConfig("redis://unused", t.TempDir(), false), nil }
	openDatabaseFn = func(string) (*gorm.DB, error) {
		return gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	}
	migrateDatabaseFn = func(*gorm.DB) error { return nil }
	newJWTServiceFn = func(*config.Config) (*infrcrypto.JWTService, error) {
		return nil, errors.New("jwt failed")
	}
	if code := run(); code != 1 {
		t.Fatalf("run() = %d, want 1", code)
	}
}

func TestRun_RedisClientError(t *testing.T) {
	hooks := saveHooks()
	t.Cleanup(hooks.restore)
	testFatalHook(t)

	mr := miniredis.RunT(t)
	setupSuccessfulRunHooks(t, false, "")
	newRedisClientFn = func(string, time.Duration) (*appredis.Client, error) {
		return nil, errors.New("redis failed")
	}
	_ = mr
	if code := run(); code != 1 {
		t.Fatalf("run() = %d, want 1", code)
	}
}

func TestRun_RedisPingError(t *testing.T) {
	hooks := saveHooks()
	t.Cleanup(hooks.restore)
	testFatalHook(t)

	setupSuccessfulRunHooks(t, false, "")
	pingRedisClientFn = func(*appredis.Client, context.Context) error { return errors.New("redis ping failed") }
	if code := run(); code != 1 {
		t.Fatalf("run() = %d, want 1", code)
	}
}

func TestRun_DatabaseCloseError(t *testing.T) {
	hooks := saveHooks()
	t.Cleanup(hooks.restore)

	setupSuccessfulRunHooks(t, false, "")
	closeDatabaseFn = func(*gorm.DB) error { return errors.New("db close failed") }
	if code := run(); code != 0 {
		t.Fatalf("run() = %d, want 0", code)
	}
}

func TestRun_RedisCloseError(t *testing.T) {
	hooks := saveHooks()
	t.Cleanup(hooks.restore)

	setupSuccessfulRunHooks(t, false, "")
	closeRedisClientFn = func(*appredis.Client) error { return errors.New("redis close failed") }
	if code := run(); code != 0 {
		t.Fatalf("run() = %d, want 0", code)
	}
}

func TestRun_FlushAllDirtyError(t *testing.T) {
	hooks := saveHooks()
	t.Cleanup(hooks.restore)

	setupSuccessfulRunHooks(t, false, "")
	flushAllDirtyFn = func(*batch.DocumentFlushBatch, context.Context) error { return errors.New("flush failed") }
	if code := run(); code != 0 {
		t.Fatalf("run() = %d, want 0", code)
	}
}

func TestRun_ShutdownError(t *testing.T) {
	hooks := saveHooks()
	t.Cleanup(hooks.restore)

	setupSuccessfulRunHooks(t, false, "")
	shutdownEchoFn = func(*echo.Echo, context.Context) error { return errors.New("shutdown failed") }
	if code := run(); code != 0 {
		t.Fatalf("run() = %d, want 0", code)
	}
}

func TestRun_ServerStartError(t *testing.T) {
	hooks := saveHooks()
	t.Cleanup(hooks.restore)

	setupSuccessfulRunHooks(t, false, "")
	echoStartFn = func(*echo.Echo, string) error { return errors.New("start failed") }

	var wg sync.WaitGroup
	wg.Add(1)
	fatalFn = func(format string, args ...any) {
		if format == "server: %v" {
			wg.Done()
			return
		}
		panic(fmt.Sprintf(format, args...))
	}

	go run()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for server start fatal")
	}
}

func TestDefaultStartHTTPServer_ServerClosed(t *testing.T) {
	hooks := saveHooks()
	t.Cleanup(hooks.restore)

	e := echo.New()
	echoStartFn = func(*echo.Echo, string) error { return http.ErrServerClosed }
	startHTTPServerFn(e, 0)
	time.Sleep(20 * time.Millisecond)
}

func TestDefaultWaitForShutdownSignal(t *testing.T) {
	hooks := saveHooks()
	t.Cleanup(hooks.restore)

	notifyShutdownSignalsFn = func(c chan os.Signal) {
		go func() { c <- syscall.SIGTERM }()
	}
	defaultWaitForShutdownSignal()
}

type stubFlushService struct{}

func (stubFlushService) FlushAllDirty(context.Context) error { return nil }
func (stubFlushService) FlushDocument(context.Context, uuid.UUID) error {
	return nil
}
func (stubFlushService) FlushWorkspaceDocuments(context.Context, uuid.UUID) error {
	return nil
}

func TestDefaultHookFns(t *testing.T) {
	hooks := saveHooks()
	t.Cleanup(hooks.restore)

	flushBatch := batch.NewDocumentFlushBatch(stubFlushService{})
	if err := flushAllDirtyFn(flushBatch, context.Background()); err != nil {
		t.Fatalf("flushAllDirtyFn: %v", err)
	}

	e := echo.New()
	go func() {
		_ = echoStartFn(e, ":0")
	}()
	time.Sleep(20 * time.Millisecond)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := shutdownEchoFn(e, shutdownCtx); err != nil {
		t.Fatalf("shutdownEchoFn: %v", err)
	}

	stop := make(chan os.Signal, 1)
	notifyShutdownSignalsFn(stop)
	stop <- syscall.SIGTERM
}

func TestDefaultFatalFn(t *testing.T) {
	if os.Getenv("NOTEHUB_TEST_FATAL") == "1" {
		hooks := saveHooks()
		t.Cleanup(hooks.restore)
		fatalFn("test fatal: %s", "boom")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestDefaultFatalFn$")
	cmd.Env = append(os.Environ(), "NOTEHUB_TEST_FATAL=1")
	err := cmd.Run()
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() == 0 {
		t.Fatalf("expected non-zero exit, err=%v", err)
	}
}

func TestMain_InvokesRun(t *testing.T) {
	hooks := saveHooks()
	t.Cleanup(hooks.restore)

	var exitCode int
	osExitFn = func(code int) { exitCode = code }
	runFn = func() int { return 0 }

	main()
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
}
