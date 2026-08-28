package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Environment string

const (
	EnvironmentDevelopment Environment = "development"
	EnvironmentTest        Environment = "test"
	EnvironmentProduction  Environment = "production"
)

type Config struct {
	Environment Environment
	HTTPPort    int

	DatabaseURL           string
	RedisURL              string
	RedisOperationTimeout time.Duration

	JWTSecret       string
	JWTIssuer       string
	JWTAudience     string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	BcryptCost int

	CookieSecure           bool
	CookieDomain           string
	CookieSameSite         string
	RefreshTokenCookieName string
	CSRFTokenCookieName    string
	AllowedOrigins         []string
	AllowedOriginSuffixes  []string

	LoginMaxFailures    int
	LoginFailureWindow  time.Duration
	LoginLockDuration   time.Duration
	RateLimitCapacity   int
	RateLimitRefillRate float64
	TrustedProxyCIDRs   []string

	WSMaxConnectionsPerDocument  int
	DocumentAutosaveInterval     time.Duration
	DocumentAutosaveIdleDuration time.Duration

	CleanupBatchInterval  time.Duration
	RefreshTokenRetention time.Duration
	BackupEnabled         bool
	BackupBatchInterval   time.Duration
	BackupDirectory       string
	BackupRetention       time.Duration
}

func Load() (*Config, error) {
	return LoadFromEnv(os.Getenv)
}

func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		panic(fmt.Sprintf("invalid configuration: %v", err))
	}
	return cfg
}

func LoadFromEnv(getenv func(string) string) (*Config, error) {
	environment, err := parseEnvironment(
		valueOrDefault(getenv("APP_ENV"), "development"),
	)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Environment:                  environment,
		HTTPPort:                     resolveHTTPPort(getenv),
		DatabaseURL:                  strings.TrimSpace(getenv("DATABASE_URL")),
		RedisURL:                     strings.TrimSpace(getenv("REDIS_URL")),
		RedisOperationTimeout:        durationValue(getenv("REDIS_OPERATION_TIMEOUT"), 2*time.Second),
		JWTSecret:                    getenv("JWT_SECRET"),
		JWTIssuer:                    valueOrDefault(getenv("JWT_ISSUER"), "notehub-api"),
		JWTAudience:                  valueOrDefault(getenv("JWT_AUDIENCE"), "notehub-client"),
		AccessTokenTTL:               resolveAccessTokenTTL(getenv),
		RefreshTokenTTL:              resolveRefreshTokenTTL(environment, getenv),
		BcryptCost:                   intValue(getenv("BCRYPT_COST"), 12),
		CookieSecure:                 boolValue(getenv("COOKIE_SECURE"), environment == EnvironmentProduction),
		CookieDomain:                 strings.TrimSpace(getenv("COOKIE_DOMAIN")),
		CookieSameSite:               normalizeCookieSameSite(valueOrDefault(getenv("COOKIE_SAME_SITE"), "Lax")),
		RefreshTokenCookieName:       valueOrDefault(getenv("REFRESH_TOKEN_COOKIE_NAME"), "notehub_refresh_token"),
		CSRFTokenCookieName:          valueOrDefault(getenv("CSRF_TOKEN_COOKIE_NAME"), "notehub_csrf_token"),
		AllowedOrigins:               splitCSV(getenv("CORS_ALLOWED_ORIGINS")),
		AllowedOriginSuffixes:        splitCSV(getenv("CORS_ALLOWED_ORIGIN_SUFFIXES")),
		LoginMaxFailures:             intValue(getenv("LOGIN_MAX_FAILURES"), 5),
		LoginFailureWindow:           durationValue(getenv("LOGIN_FAILURE_WINDOW"), time.Hour),
		LoginLockDuration:            durationValue(getenv("LOGIN_LOCK_DURATION"), time.Hour),
		RateLimitCapacity:            intValue(getenv("RATE_LIMIT_CAPACITY"), 10),
		RateLimitRefillRate:          floatValue(getenv("RATE_LIMIT_REFILL_PER_SECOND"), 1),
		TrustedProxyCIDRs:            splitCSV(getenv("TRUSTED_PROXY_CIDRS")),
		WSMaxConnectionsPerDocument:  intValue(getenv("WS_MAX_CONNECTIONS_PER_DOCUMENT"), 3),
		DocumentAutosaveInterval:     durationValue(getenv("DOCUMENT_AUTOSAVE_INTERVAL"), 10*time.Second),
		DocumentAutosaveIdleDuration: durationValue(getenv("DOCUMENT_AUTOSAVE_IDLE_DURATION"), 60*time.Second),
		CleanupBatchInterval:         durationValue(getenv("CLEANUP_BATCH_INTERVAL"), time.Hour),
		RefreshTokenRetention:        durationValue(getenv("REFRESH_TOKEN_RETENTION"), 30*24*time.Hour),
		BackupEnabled:                boolValue(getenv("BACKUP_ENABLED"), false),
		BackupBatchInterval:          durationValue(getenv("BACKUP_BATCH_INTERVAL"), 24*time.Hour),
		BackupDirectory:              valueOrDefault(getenv("BACKUP_DIR"), "./backups"),
		BackupRetention:              durationValue(getenv("BACKUP_RETENTION"), 7*24*time.Hour),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	var errs []error

	if c.HTTPPort < 1 || c.HTTPPort > 65535 {
		errs = append(errs, fmt.Errorf("HTTP_PORT must be between 1 and 65535"))
	}

	if c.DatabaseURL == "" {
		errs = append(errs, fmt.Errorf("DATABASE_URL is required"))
	} else if _, err := url.ParseRequestURI(c.DatabaseURL); err != nil {
		errs = append(errs, fmt.Errorf("DATABASE_URL is invalid: %w", err))
	}

	if c.RedisOperationTimeout <= 0 || c.RedisOperationTimeout > 30*time.Second {
		errs = append(errs, fmt.Errorf(
			"REDIS_OPERATION_TIMEOUT must be greater than 0 and no more than 30s",
		))
	}

	if len([]byte(c.JWTSecret)) < 32 {
		errs = append(errs, fmt.Errorf("JWT_SECRET must be at least 32 bytes"))
	}

	if strings.TrimSpace(c.JWTIssuer) == "" {
		errs = append(errs, fmt.Errorf("JWT_ISSUER is required"))
	}

	if strings.TrimSpace(c.JWTAudience) == "" {
		errs = append(errs, fmt.Errorf("JWT_AUDIENCE is required"))
	}

	if c.AccessTokenTTL <= 0 || c.AccessTokenTTL > 24*time.Hour {
		errs = append(errs, fmt.Errorf(
			"ACCESS_TOKEN_TTL must be greater than 0 and no more than 24h",
		))
	}

	if c.RefreshTokenTTL <= c.AccessTokenTTL {
		errs = append(errs, fmt.Errorf(
			"REFRESH_TOKEN_TTL must be greater than ACCESS_TOKEN_TTL",
		))
	}

	if c.BcryptCost < 10 || c.BcryptCost > 16 {
		errs = append(errs, fmt.Errorf(
			"BCRYPT_COST must be between 10 and 16",
		))
	}

	if c.CookieSameSite != "Lax" &&
		c.CookieSameSite != "Strict" &&
		c.CookieSameSite != "None" {
		errs = append(errs, fmt.Errorf(
			"COOKIE_SAME_SITE must be Lax, Strict, or None",
		))
	}

	if strings.TrimSpace(c.RefreshTokenCookieName) == "" {
		errs = append(errs, fmt.Errorf(
			"REFRESH_TOKEN_COOKIE_NAME is required",
		))
	}

	if strings.TrimSpace(c.CSRFTokenCookieName) == "" {
		errs = append(errs, fmt.Errorf(
			"CSRF_TOKEN_COOKIE_NAME is required",
		))
	}

	if c.LoginMaxFailures < 1 || c.LoginMaxFailures > 100 {
		errs = append(errs, fmt.Errorf(
			"LOGIN_MAX_FAILURES must be between 1 and 100",
		))
	}

	if c.LoginFailureWindow <= 0 {
		errs = append(errs, fmt.Errorf(
			"LOGIN_FAILURE_WINDOW must be greater than 0",
		))
	}

	if c.LoginLockDuration <= 0 {
		errs = append(errs, fmt.Errorf(
			"LOGIN_LOCK_DURATION must be greater than 0",
		))
	}

	if c.RateLimitCapacity < 1 {
		errs = append(errs, fmt.Errorf(
			"RATE_LIMIT_CAPACITY must be greater than 0",
		))
	}

	if c.RateLimitRefillRate <= 0 {
		errs = append(errs, fmt.Errorf(
			"RATE_LIMIT_REFILL_PER_SECOND must be greater than 0",
		))
	}

	for _, cidr := range c.TrustedProxyCIDRs {
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			errs = append(errs, fmt.Errorf(
				"TRUSTED_PROXY_CIDRS contains invalid CIDR: %s",
				cidr,
			))
		}
	}

	if c.WSMaxConnectionsPerDocument < 1 ||
		c.WSMaxConnectionsPerDocument > 20 {
		errs = append(errs, fmt.Errorf(
			"WS_MAX_CONNECTIONS_PER_DOCUMENT must be between 1 and 20",
		))
	}

	if c.DocumentAutosaveInterval < time.Second {
		errs = append(errs, fmt.Errorf(
			"DOCUMENT_AUTOSAVE_INTERVAL must be at least 1s",
		))
	}

	if c.DocumentAutosaveIdleDuration < time.Second {
		errs = append(errs, fmt.Errorf(
			"DOCUMENT_AUTOSAVE_IDLE_DURATION must be at least 1s",
		))
	}

	if c.CleanupBatchInterval < time.Minute {
		errs = append(errs, fmt.Errorf(
			"CLEANUP_BATCH_INTERVAL must be at least 1m",
		))
	}

	if c.RefreshTokenRetention < 24*time.Hour {
		errs = append(errs, fmt.Errorf(
			"REFRESH_TOKEN_RETENTION must be at least 24h",
		))
	}

	if c.BackupBatchInterval < time.Hour {
		errs = append(errs, fmt.Errorf(
			"BACKUP_BATCH_INTERVAL must be at least 1h",
		))
	}

	if c.BackupRetention < 24*time.Hour {
		errs = append(errs, fmt.Errorf(
			"BACKUP_RETENTION must be at least 24h",
		))
	}

	if c.BackupEnabled && strings.TrimSpace(c.BackupDirectory) == "" {
		errs = append(errs, fmt.Errorf(
			"BACKUP_DIR is required when BACKUP_ENABLED is true",
		))
	}

	if c.Environment == EnvironmentProduction {
		if c.RedisURL == "" {
			errs = append(errs, fmt.Errorf(
				"REDIS_URL is required in production",
			))
		}

		if !c.CookieSecure {
			errs = append(errs, fmt.Errorf(
				"COOKIE_SECURE must be true in production",
			))
		}

		if len(c.AllowedOrigins) == 0 {
			errs = append(errs, fmt.Errorf(
				"CORS_ALLOWED_ORIGINS is required in production",
			))
		}

		for _, origin := range c.AllowedOrigins {
			if origin == "*" {
				errs = append(errs, fmt.Errorf(
					"CORS_ALLOWED_ORIGINS cannot contain * in production",
				))
			}

			parsed, err := url.Parse(origin)
			if err != nil ||
				parsed.Scheme != "https" ||
				parsed.Host == "" {
				errs = append(errs, fmt.Errorf(
					"production origin must be HTTPS: %s",
					origin,
				))
			}
		}

		for _, suffix := range c.AllowedOriginSuffixes {
			if !strings.HasPrefix(suffix, ".") ||
				strings.Contains(suffix, " ") {
				errs = append(errs, fmt.Errorf(
					"CORS_ALLOWED_ORIGIN_SUFFIXES entries must start with '.': %s",
					suffix,
				))
			}
		}

		if len(c.TrustedProxyCIDRs) == 0 {
			errs = append(errs, fmt.Errorf(
				"TRUSTED_PROXY_CIDRS is required in production",
			))
		}
	}

	return errors.Join(errs...)
}

func resolveHTTPPort(getenv func(string) string) int {
	if port := strings.TrimSpace(getenv("PORT")); port != "" {
		return intValue(port, 8080)
	}

	return intValue(getenv("HTTP_PORT"), 8080)
}

func parseEnvironment(raw string) (Environment, error) {
	switch Environment(strings.ToLower(strings.TrimSpace(raw))) {
	case EnvironmentDevelopment:
		return EnvironmentDevelopment, nil

	case EnvironmentTest:
		return EnvironmentTest, nil

	case EnvironmentProduction:
		return EnvironmentProduction, nil

	default:
		return "", fmt.Errorf(
			"APP_ENV must be development, test, or production",
		)
	}
}

func normalizeCookieSameSite(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "none":
		return "None"

	case "strict":
		return "Strict"

	case "lax":
		return "Lax"

	default:
		return strings.TrimSpace(value)
	}
}

func valueOrDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}

	return strings.TrimSpace(value)
}

func intValue(raw string, fallback int) int {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}

	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return -1
	}

	return value
}

func boolValue(raw string, fallback bool) bool {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}

	value, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err != nil {
		return false
	}

	return value
}

func durationValue(raw string, fallback time.Duration) time.Duration {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}

	value, err := time.ParseDuration(strings.TrimSpace(raw))
	if err != nil {
		return -1
	}

	return value
}

func splitCSV(raw string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0)

	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)

		if item == "" {
			continue
		}

		if _, exists := seen[item]; exists {
			continue
		}

		seen[item] = struct{}{}
		result = append(result, item)
	}

	return result
}

func floatValue(raw string, fallback float64) float64 {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}

	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return -1
	}

	return value
}

func resolveAccessTokenTTL(getenv func(string) string) time.Duration {
	if ttl := durationValue(getenv("ACCESS_TOKEN_TTL"), 0); ttl > 0 {
		return ttl
	}

	minutes := intValue(getenv("ACCESS_TOKEN_TTL_MINUTES"), 15)
	if minutes <= 0 {
		return 15 * time.Minute
	}

	return time.Duration(minutes) * time.Minute
}

func resolveRefreshTokenTTL(
	environment Environment,
	getenv func(string) string,
) time.Duration {
	if ttl := durationValue(getenv("REFRESH_TOKEN_TTL"), 0); ttl > 0 {
		return ttl
	}

	if environment == EnvironmentProduction {
		days := intValue(getenv("REFRESH_TOKEN_TTL_DAYS"), 14)

		if days <= 0 {
			return 14 * 24 * time.Hour
		}

		return time.Duration(days) * 24 * time.Hour
	}

	minutes := intValue(
		getenv("REFRESH_TOKEN_TTL_MINUTES"),
		30,
	)

	if minutes <= 0 {
		return 30 * time.Minute
	}

	return time.Duration(minutes) * time.Minute
}
