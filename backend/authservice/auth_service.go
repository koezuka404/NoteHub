package authservice

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/crypto"
	"github.com/koezuka404/notehub/redis"
)

type AuthService struct {
	// IPasswordService
	passwords *crypto.PasswordService
	// IAccessTokenService
	jwt *crypto.JWTService
	// IRandomTokenService
	random *crypto.RandomTokenService
	// ITokenHashService
	hasher *crypto.TokenHashService
	// IAccessTokenRevocationStore
	revocations *redis.AccessTokenRevocationStore
	// ILoginFailureStore
	loginFails *redis.LoginFailureStore
}

func NewAuthService(
	passwords *crypto.PasswordService,
	jwt *crypto.JWTService,
	random *crypto.RandomTokenService,
	hasher *crypto.TokenHashService,
	revocations *redis.AccessTokenRevocationStore,
	loginFails *redis.LoginFailureStore,
) *AuthService {
	return &AuthService{
		passwords:   passwords,
		jwt:         jwt,
		random:      random,
		hasher:      hasher,
		revocations: revocations,
		loginFails:  loginFails,
	}
}

// IPasswordService

func (s *AuthService) HashPassword(password string) (string, error) {
	return s.passwords.Hash(password)
}

func (s *AuthService) ComparePassword(passwordHash, password string) error {
	return s.passwords.Compare(passwordHash, password)
}

// IAccessTokenService

func (s *AuthService) GenerateAccessToken(userID uuid.UUID, authVersion uint, now time.Time) (string, time.Time, error) {
	return s.jwt.GenerateAccessToken(userID, authVersion, now)
}

// IRandomTokenService

func (s *AuthService) GenerateRefreshToken() (string, error) {
	return s.random.GenerateRefreshToken()
}

func (s *AuthService) GenerateCSRFToken() (string, error) {
	return s.random.GenerateCSRFToken()
}

// ITokenHashService

func (s *AuthService) HashToken(token string) string {
	return s.hasher.Hash(token)
}

// IAccessTokenRevocationStore

func (s *AuthService) RevokeAccessToken(ctx context.Context, jti uuid.UUID, ttl time.Duration) error {
	return s.revocations.Revoke(ctx, jti, ttl)
}

func (s *AuthService) IsAccessTokenRevoked(ctx context.Context, jti uuid.UUID) (bool, error) {
	return s.revocations.IsRevoked(ctx, jti)
}

// ILoginFailureStore

func (s *AuthService) IsLoginLocked(ctx context.Context, email string) (bool, time.Duration, error) {
	if s.loginFails == nil {
		return false, 0, nil
	}
	return s.loginFails.IsLocked(ctx, email)
}

func (s *AuthService) RecordLoginFailure(ctx context.Context, email string) (bool, time.Duration, error) {
	if s.loginFails == nil {
		return false, 0, nil
	}
	return s.loginFails.RecordFailure(ctx, email)
}

func (s *AuthService) ResetLoginFailures(ctx context.Context, email string) error {
	if s.loginFails == nil {
		return nil
	}
	return s.loginFails.Reset(ctx, email)
}
