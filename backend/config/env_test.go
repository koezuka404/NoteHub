package config

import (
	"strings"
	"testing"
	"time"
)

const testJWTSecret = "01234567890123456789012345678901"

func validEnv() map[string]string {
	return map[string]string{
		"APP_ENV":       "development",
		"DATABASE_URL":  "postgres://user:pass@localhost:5432/notehub",
		"JWT_SECRET":    testJWTSecret,
		"REDIS_URL":     "redis://127.0.0.1:6379/0",
		"CORS_ALLOWED_ORIGINS": "http://localhost:5173",
	}
}

func getenvFrom(env map[string]string) func(string) string {
	return func(key string) string { return env[key] }
}

func TestLoadFromEnv_ValidDevelopmentDefaults(t *testing.T) {
	cfg, err := LoadFromEnv(getenvFrom(validEnv()))
	if err != nil {
		t.Fatalf("LoadFromEnv: %v", err)
	}
	if cfg.Environment != EnvironmentDevelopment {
		t.Fatalf("Environment = %q", cfg.Environment)
	}
	if cfg.HTTPPort != 8080 {
		t.Fatalf("HTTPPort = %d", cfg.HTTPPort)
	}
	if cfg.JWTIssuer != "notehub-api" || cfg.JWTAudience != "notehub-client" {
		t.Fatalf("jwt defaults = %q / %q", cfg.JWTIssuer, cfg.JWTAudience)
	}
	if cfg.AccessTokenTTL != 15*time.Minute {
		t.Fatalf("AccessTokenTTL = %v", cfg.AccessTokenTTL)
	}
	if cfg.RefreshTokenTTL != 30*time.Minute {
		t.Fatalf("RefreshTokenTTL = %v", cfg.RefreshTokenTTL)
	}
	if cfg.CookieSecure {
		t.Fatal("development should default CookieSecure to false")
	}
}

func TestLoadFromEnv_ValidProduction(t *testing.T) {
	env := validEnv()
	env["APP_ENV"] = "production"
	env["COOKIE_SECURE"] = "true"
	env["CORS_ALLOWED_ORIGINS"] = "https://app.example.com,https://app.example.com"

	cfg, err := LoadFromEnv(getenvFrom(env))
	if err != nil {
		t.Fatalf("LoadFromEnv: %v", err)
	}
	if cfg.Environment != EnvironmentProduction || !cfg.CookieSecure {
		t.Fatalf("production config = %+v", cfg)
	}
	if len(cfg.AllowedOrigins) != 1 {
		t.Fatalf("AllowedOrigins = %v", cfg.AllowedOrigins)
	}
}

func TestLoadFromEnv_TestEnvironment(t *testing.T) {
	env := validEnv()
	env["APP_ENV"] = "test"
	cfg, err := LoadFromEnv(getenvFrom(env))
	if err != nil {
		t.Fatalf("LoadFromEnv: %v", err)
	}
	if cfg.Environment != EnvironmentTest {
		t.Fatalf("Environment = %q", cfg.Environment)
	}
}

func TestLoadFromEnv_InvalidEnvironment(t *testing.T) {
	env := validEnv()
	env["APP_ENV"] = "staging"
	if _, err := LoadFromEnv(getenvFrom(env)); err == nil {
		t.Fatal("expected invalid APP_ENV error")
	}
}

func TestLoadAndMustLoad(t *testing.T) {
	for key, value := range validEnv() {
		t.Setenv(key, value)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DatabaseURL == "" {
		t.Fatal("expected loaded config")
	}

	got := MustLoad()
	if got.HTTPPort != cfg.HTTPPort {
		t.Fatalf("MustLoad HTTPPort = %d", got.HTTPPort)
	}
}

func TestMustLoad_PanicsOnInvalidConfig(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "short")

	defer func() {
		if recover() == nil {
			t.Fatal("expected MustLoad panic")
		}
	}()
	MustLoad()
}

func TestResolveAccessTokenTTL(t *testing.T) {
	t.Run("explicit duration", func(t *testing.T) {
		env := map[string]string{"ACCESS_TOKEN_TTL": "30m"}
		if ttl := resolveAccessTokenTTL(getenvFrom(env)); ttl != 30*time.Minute {
			t.Fatalf("ttl = %v", ttl)
		}
	})
	t.Run("minutes fallback", func(t *testing.T) {
		env := map[string]string{"ACCESS_TOKEN_TTL_MINUTES": "20"}
		if ttl := resolveAccessTokenTTL(getenvFrom(env)); ttl != 20*time.Minute {
			t.Fatalf("ttl = %v", ttl)
		}
	})
	t.Run("invalid minutes uses default", func(t *testing.T) {
		env := map[string]string{"ACCESS_TOKEN_TTL_MINUTES": "bad"}
		if ttl := resolveAccessTokenTTL(getenvFrom(env)); ttl != 15*time.Minute {
			t.Fatalf("ttl = %v", ttl)
		}
	})
}

func TestResolveRefreshTokenTTL(t *testing.T) {
	t.Run("explicit duration", func(t *testing.T) {
		env := map[string]string{"REFRESH_TOKEN_TTL": "48h"}
		if ttl := resolveRefreshTokenTTL(EnvironmentDevelopment, getenvFrom(env)); ttl != 48*time.Hour {
			t.Fatalf("ttl = %v", ttl)
		}
	})
	t.Run("production days", func(t *testing.T) {
		env := map[string]string{"REFRESH_TOKEN_TTL_DAYS": "21"}
		if ttl := resolveRefreshTokenTTL(EnvironmentProduction, getenvFrom(env)); ttl != 21*24*time.Hour {
			t.Fatalf("ttl = %v", ttl)
		}
	})
	t.Run("production invalid days default", func(t *testing.T) {
		env := map[string]string{"REFRESH_TOKEN_TTL_DAYS": "0"}
		if ttl := resolveRefreshTokenTTL(EnvironmentProduction, getenvFrom(env)); ttl != 14*24*time.Hour {
			t.Fatalf("ttl = %v", ttl)
		}
	})
	t.Run("development minutes", func(t *testing.T) {
		env := map[string]string{"REFRESH_TOKEN_TTL_MINUTES": "45"}
		if ttl := resolveRefreshTokenTTL(EnvironmentDevelopment, getenvFrom(env)); ttl != 45*time.Minute {
			t.Fatalf("ttl = %v", ttl)
		}
	})
	t.Run("development invalid minutes default", func(t *testing.T) {
		env := map[string]string{"REFRESH_TOKEN_TTL_MINUTES": "bad"}
		if ttl := resolveRefreshTokenTTL(EnvironmentDevelopment, getenvFrom(env)); ttl != 30*time.Minute {
			t.Fatalf("ttl = %v", ttl)
		}
	})
}

func TestParseEnvironment(t *testing.T) {
	cases := []struct {
		raw   string
		want  Environment
		isErr bool
	}{
		{"development", EnvironmentDevelopment, false},
		{" TEST ", EnvironmentTest, false},
		{"Production", EnvironmentProduction, false},
		{"invalid", "", true},
	}
	for _, tc := range cases {
		got, err := parseEnvironment(tc.raw)
		if tc.isErr {
			if err == nil {
				t.Fatalf("parseEnvironment(%q) expected error", tc.raw)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Fatalf("parseEnvironment(%q) = %q, %v", tc.raw, got, err)
		}
	}
}

func TestHelperFunctions(t *testing.T) {
	if got := valueOrDefault("  ", "fallback"); got != "fallback" {
		t.Fatalf("valueOrDefault = %q", got)
	}
	if got := valueOrDefault(" value ", ""); got != "value" {
		t.Fatalf("valueOrDefault trim = %q", got)
	}
	if got := intValue("", 7); got != 7 {
		t.Fatalf("intValue fallback = %d", got)
	}
	if got := intValue("bad", 7); got != -1 {
		t.Fatalf("intValue invalid = %d", got)
	}
	if got := boolValue("", true); !got {
		t.Fatal("boolValue fallback")
	}
	if got := boolValue("maybe", true); got {
		t.Fatal("boolValue invalid should be false")
	}
	if got := durationValue("", 3*time.Second); got != 3*time.Second {
		t.Fatalf("durationValue fallback = %v", got)
	}
	if got := durationValue("nope", 3*time.Second); got != -1 {
		t.Fatalf("durationValue invalid = %v", got)
	}
	if got := floatValue("", 2.5); got != 2.5 {
		t.Fatalf("floatValue fallback = %v", got)
	}
	if got := floatValue("bad", 2.5); got != -1 {
		t.Fatalf("floatValue invalid = %v", got)
	}
	if got := floatValue("1.5", 2.5); got != 1.5 {
		t.Fatalf("floatValue parsed = %v", got)
	}
	if got := splitCSV(" https://a.example , ,https://a.example,https://b.example "); len(got) != 2 {
		t.Fatalf("splitCSV = %v", got)
	}
}

func TestConfigValidate(t *testing.T) {
	base := func() Config {
		env := validEnv()
		cfg, err := LoadFromEnv(getenvFrom(env))
		if err != nil {
			t.Fatalf("base config: %v", err)
		}
		return *cfg
	}

	assertErr := func(t *testing.T, cfg Config, contains string) {
		t.Helper()
		err := cfg.Validate()
		if err == nil || !strings.Contains(err.Error(), contains) {
			t.Fatalf("Validate() = %v, want substring %q", err, contains)
		}
	}

	t.Run("HTTP_PORT", func(t *testing.T) {
		cfg := base()
		cfg.HTTPPort = 0
		assertErr(t, cfg, "HTTP_PORT")
	})

	t.Run("DATABASE_URL missing", func(t *testing.T) {
		cfg := base()
		cfg.DatabaseURL = ""
		assertErr(t, cfg, "DATABASE_URL is required")
	})

	t.Run("DATABASE_URL invalid", func(t *testing.T) {
		cfg := base()
		cfg.DatabaseURL = "://bad"
		assertErr(t, cfg, "DATABASE_URL is invalid")
	})

	t.Run("REDIS_OPERATION_TIMEOUT", func(t *testing.T) {
		cfg := base()
		cfg.RedisOperationTimeout = 0
		assertErr(t, cfg, "REDIS_OPERATION_TIMEOUT")
		cfg.RedisOperationTimeout = 31 * time.Second
		assertErr(t, cfg, "REDIS_OPERATION_TIMEOUT")
	})

	t.Run("JWT fields", func(t *testing.T) {
		cfg := base()
		cfg.JWTSecret = "short"
		assertErr(t, cfg, "JWT_SECRET")
		cfg = base()
		cfg.JWTIssuer = " "
		assertErr(t, cfg, "JWT_ISSUER")
		cfg = base()
		cfg.JWTAudience = " "
		assertErr(t, cfg, "JWT_AUDIENCE")
	})

	t.Run("token TTL", func(t *testing.T) {
		cfg := base()
		cfg.AccessTokenTTL = 0
		assertErr(t, cfg, "ACCESS_TOKEN_TTL")
		cfg = base()
		cfg.AccessTokenTTL = 25 * time.Hour
		assertErr(t, cfg, "ACCESS_TOKEN_TTL")
		cfg = base()
		cfg.RefreshTokenTTL = cfg.AccessTokenTTL
		assertErr(t, cfg, "REFRESH_TOKEN_TTL")
	})

	t.Run("bcrypt and cookies", func(t *testing.T) {
		cfg := base()
		cfg.BcryptCost = 9
		assertErr(t, cfg, "BCRYPT_COST")
		cfg = base()
		cfg.CookieSameSite = "Invalid"
		assertErr(t, cfg, "COOKIE_SAME_SITE")
		cfg = base()
		cfg.RefreshTokenCookieName = " "
		assertErr(t, cfg, "REFRESH_TOKEN_COOKIE_NAME")
		cfg = base()
		cfg.CSRFTokenCookieName = " "
		assertErr(t, cfg, "CSRF_TOKEN_COOKIE_NAME")
	})

	t.Run("login and rate limit", func(t *testing.T) {
		cfg := base()
		cfg.LoginMaxFailures = 0
		assertErr(t, cfg, "LOGIN_MAX_FAILURES")
		cfg = base()
		cfg.LoginFailureWindow = 0
		assertErr(t, cfg, "LOGIN_FAILURE_WINDOW")
		cfg = base()
		cfg.LoginLockDuration = 0
		assertErr(t, cfg, "LOGIN_LOCK_DURATION")
		cfg = base()
		cfg.RateLimitCapacity = 0
		assertErr(t, cfg, "RATE_LIMIT_CAPACITY")
		cfg = base()
		cfg.RateLimitRefillRate = 0
		assertErr(t, cfg, "RATE_LIMIT_REFILL_PER_SECOND")
	})

	t.Run("websocket and autosave", func(t *testing.T) {
		cfg := base()
		cfg.WSMaxConnectionsPerDocument = 0
		assertErr(t, cfg, "WS_MAX_CONNECTIONS_PER_DOCUMENT")
		cfg = base()
		cfg.DocumentAutosaveInterval = 500 * time.Millisecond
		assertErr(t, cfg, "DOCUMENT_AUTOSAVE_INTERVAL")
		cfg = base()
		cfg.DocumentAutosaveIdleDuration = 0
		assertErr(t, cfg, "DOCUMENT_AUTOSAVE_IDLE_DURATION")
	})

	t.Run("batch and backup", func(t *testing.T) {
		cfg := base()
		cfg.CleanupBatchInterval = time.Second
		assertErr(t, cfg, "CLEANUP_BATCH_INTERVAL")
		cfg = base()
		cfg.RefreshTokenRetention = time.Hour
		assertErr(t, cfg, "REFRESH_TOKEN_RETENTION")
		cfg = base()
		cfg.BackupBatchInterval = 30 * time.Minute
		assertErr(t, cfg, "BACKUP_BATCH_INTERVAL")
		cfg = base()
		cfg.BackupRetention = time.Hour
		assertErr(t, cfg, "BACKUP_RETENTION")
		cfg = base()
		cfg.BackupEnabled = true
		cfg.BackupDirectory = " "
		assertErr(t, cfg, "BACKUP_DIR is required")
	})

	t.Run("production rules", func(t *testing.T) {
		cfg := base()
		cfg.Environment = EnvironmentProduction
		cfg.RedisURL = ""
		assertErr(t, cfg, "REDIS_URL is required in production")
		cfg = base()
		cfg.Environment = EnvironmentProduction
		cfg.CookieSecure = false
		assertErr(t, cfg, "COOKIE_SECURE must be true in production")
		cfg = base()
		cfg.Environment = EnvironmentProduction
		cfg.AllowedOrigins = nil
		assertErr(t, cfg, "CORS_ALLOWED_ORIGINS is required in production")
		cfg = base()
		cfg.Environment = EnvironmentProduction
		cfg.AllowedOrigins = []string{"*"}
		assertErr(t, cfg, "cannot contain *")
		cfg = base()
		cfg.Environment = EnvironmentProduction
		cfg.AllowedOrigins = []string{"http://insecure.example"}
		assertErr(t, cfg, "production origin must be HTTPS")
		cfg = base()
		cfg.Environment = EnvironmentProduction
		cfg.CookieSecure = true
		cfg.AllowedOrigins = []string{"https://app.example.com"}
		if err := cfg.Validate(); err != nil {
			t.Fatalf("expected valid production config: %v", err)
		}
	})

	t.Run("valid base", func(t *testing.T) {
		if err := base().Validate(); err != nil {
			t.Fatalf("Validate: %v", err)
		}
	})
}

func TestLoadFromEnv_InvalidValues(t *testing.T) {
	cases := []struct {
		name string
		set  func(map[string]string)
	}{
		{"invalid http port", func(env map[string]string) { env["HTTP_PORT"] = "bad" }},
		{"invalid redis timeout", func(env map[string]string) { env["REDIS_OPERATION_TIMEOUT"] = "bad" }},
		{"invalid bcrypt cost", func(env map[string]string) { env["BCRYPT_COST"] = "bad" }},
		{"invalid rate limit", func(env map[string]string) { env["RATE_LIMIT_REFILL_PER_SECOND"] = "bad" }},
		{"invalid access token ttl", func(env map[string]string) { env["ACCESS_TOKEN_TTL"] = "25h" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := validEnv()
			tc.set(env)
			if _, err := LoadFromEnv(getenvFrom(env)); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
