package config

import (
	"errors"
	"fmt"
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

	DatabaseURL string
	RedisURL    string

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

	WSMaxConnectionsPerDocument int
	DocumentAutosaveInterval    time.Duration
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
	environment, err := parseEnvironment(valueOrDefault(getenv("APP_ENV"), "development"))
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Environment:                 environment,
		HTTPPort:                    intValue(getenv("HTTP_PORT"), 8080),
		DatabaseURL:                 strings.TrimSpace(getenv("DATABASE_URL")),
		RedisURL:                    strings.TrimSpace(getenv("REDIS_URL")),
		JWTSecret:                   getenv("JWT_SECRET"),
		JWTIssuer:                   valueOrDefault(getenv("JWT_ISSUER"), "notehub-api"),
		JWTAudience:                 valueOrDefault(getenv("JWT_AUDIENCE"), "notehub-web"),
		AccessTokenTTL:              durationValue(getenv("ACCESS_TOKEN_TTL"), 15*time.Minute),
		RefreshTokenTTL:             durationValue(getenv("REFRESH_TOKEN_TTL"), 30*24*time.Hour),
		BcryptCost:                  intValue(getenv("BCRYPT_COST"), 12),
		CookieSecure:                boolValue(getenv("COOKIE_SECURE"), environment == EnvironmentProduction),
		CookieDomain:                strings.TrimSpace(getenv("COOKIE_DOMAIN")),
		CookieSameSite:              valueOrDefault(getenv("COOKIE_SAME_SITE"), "Lax"),
		RefreshTokenCookieName:      valueOrDefault(getenv("REFRESH_TOKEN_COOKIE_NAME"), "notehub_refresh_token"),
		CSRFTokenCookieName:         valueOrDefault(getenv("CSRF_TOKEN_COOKIE_NAME"), "notehub_csrf_token"),
		AllowedOrigins:              splitCSV(getenv("CORS_ALLOWED_ORIGINS")),
		WSMaxConnectionsPerDocument: intValue(getenv("WS_MAX_CONNECTIONS_PER_DOCUMENT"), 3),
		DocumentAutosaveInterval:    durationValue(getenv("DOCUMENT_AUTOSAVE_INTERVAL"), 5*time.Second),
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
		errs = append(errs, fmt.Errorf("ACCESS_TOKEN_TTL must be greater than 0 and no more than 24h"))
	}
	if c.RefreshTokenTTL <= c.AccessTokenTTL {
		errs = append(errs, fmt.Errorf("REFRESH_TOKEN_TTL must be greater than ACCESS_TOKEN_TTL"))
	}
	if c.BcryptCost < 10 || c.BcryptCost > 16 {
		errs = append(errs, fmt.Errorf("BCRYPT_COST must be between 10 and 16"))
	}
	if c.CookieSameSite != "Lax" && c.CookieSameSite != "Strict" && c.CookieSameSite != "None" {
		errs = append(errs, fmt.Errorf("COOKIE_SAME_SITE must be Lax, Strict, or None"))
	}
	if strings.TrimSpace(c.RefreshTokenCookieName) == "" {
		errs = append(errs, fmt.Errorf("REFRESH_TOKEN_COOKIE_NAME is required"))
	}
	if strings.TrimSpace(c.CSRFTokenCookieName) == "" {
		errs = append(errs, fmt.Errorf("CSRF_TOKEN_COOKIE_NAME is required"))
	}
	if c.WSMaxConnectionsPerDocument < 1 || c.WSMaxConnectionsPerDocument > 20 {
		errs = append(errs, fmt.Errorf("WS_MAX_CONNECTIONS_PER_DOCUMENT must be between 1 and 20"))
	}
	if c.DocumentAutosaveInterval < time.Second {
		errs = append(errs, fmt.Errorf("DOCUMENT_AUTOSAVE_INTERVAL must be at least 1s"))
	}

	if c.Environment == EnvironmentProduction {
		if c.RedisURL == "" {
			errs = append(errs, fmt.Errorf("REDIS_URL is required in production"))
		}
		if !c.CookieSecure {
			errs = append(errs, fmt.Errorf("COOKIE_SECURE must be true in production"))
		}
		if len(c.AllowedOrigins) == 0 {
			errs = append(errs, fmt.Errorf("CORS_ALLOWED_ORIGINS is required in production"))
		}
		for _, origin := range c.AllowedOrigins {
			if origin == "*" {
				errs = append(errs, fmt.Errorf("CORS_ALLOWED_ORIGINS cannot contain * in production"))
			}
			parsed, err := url.Parse(origin)
			if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
				errs = append(errs, fmt.Errorf("production origin must be HTTPS: %s", origin))
			}
		}
	}

	return errors.Join(errs...)
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
		return "", fmt.Errorf("APP_ENV must be development, test, or production")
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
