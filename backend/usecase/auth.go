package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
)

type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	FindByID(ctx context.Context, id uuid.UUID) (user *entity.User, found bool, err error)
	FindByEmail(ctx context.Context, email string) (user *entity.User, found bool, err error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	IncrementAuthVersion(ctx context.Context, userID uuid.UUID, now time.Time) error
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *entity.RefreshToken) error
	FindByHashForUpdate(ctx context.Context, tokenHash string) (token *entity.RefreshToken, found bool, err error)
	Update(ctx context.Context, token *entity.RefreshToken) error
	RevokeFamily(ctx context.Context, familyID uuid.UUID, now time.Time) error
}

type AuditLogRepository interface {
	Create(ctx context.Context, log *entity.AuditLog) error
}

type AccessTokenRevocationStore interface {
	Revoke(ctx context.Context, jti uuid.UUID, ttl time.Duration) error
	IsRevoked(ctx context.Context, jti uuid.UUID) (bool, error)
}

type LoginFailureStore interface {
	IsLocked(ctx context.Context, email string) (locked bool, retryAfter time.Duration, err error)
	RecordFailure(ctx context.Context, email string) (locked bool, retryAfter time.Duration, err error)
	Reset(ctx context.Context, email string) error
}

type TransactionManager interface {
	WithinTransaction(ctx context.Context, fn func(context.Context) error) error
}

type PasswordService interface {
	Hash(password string) (string, error)
	Compare(passwordHash, password string) error
}

type AccessTokenService interface {
	GenerateAccessToken(userID uuid.UUID, authVersion uint, now time.Time) (token string, expiresAt time.Time, err error)
}

type RandomTokenService interface {
	GenerateRefreshToken() (string, error)
	GenerateCSRFToken() (string, error)
}

type TokenHashService interface{ Hash(token string) string }

type AuthInputPort interface {
	Register(ctx context.Context, input RegisterInput) (*RegisterOutput, error)
	Login(ctx context.Context, input LoginInput) (*LoginOutput, error)
	Refresh(ctx context.Context, input RefreshInput) (*RefreshOutput, error)
	Logout(ctx context.Context, input LogoutInput) (*LogoutOutput, error)
	GetCurrentUser(ctx context.Context, input GetCurrentUserInput) (*GetCurrentUserOutput, error)
}

type AuthUseCase struct {
	users           UserRepository
	refreshTokens   RefreshTokenRepository
	auditLogs       AuditLogRepository
	revokedAccess   AccessTokenRevocationStore
	loginFailures   LoginFailureStore
	transactions    TransactionManager
	passwords       PasswordService
	accessTokens    AccessTokenService
	randomTokens    RandomTokenService
	tokenHashes     TokenHashService
	refreshTokenTTL time.Duration
	now             func() time.Time
}

func NewAuthUseCase(
	users UserRepository,
	refreshTokens RefreshTokenRepository,
	auditLogs AuditLogRepository,
	revokedAccess AccessTokenRevocationStore,
	loginFailures LoginFailureStore,
	transactions TransactionManager,
	passwords PasswordService,
	accessTokens AccessTokenService,
	randomTokens RandomTokenService,
	tokenHashes TokenHashService,
	refreshTokenTTL time.Duration,
) *AuthUseCase {
	return &AuthUseCase{
		users: users, refreshTokens: refreshTokens, auditLogs: auditLogs,
		revokedAccess: revokedAccess, loginFailures: loginFailures, transactions: transactions,
		passwords: passwords, accessTokens: accessTokens, randomTokens: randomTokens,
		tokenHashes: tokenHashes, refreshTokenTTL: refreshTokenTTL, now: time.Now,
	}
}
