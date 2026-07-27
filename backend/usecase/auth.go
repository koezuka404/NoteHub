package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
)

type IUserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	FindByID(ctx context.Context, id uuid.UUID) (user *entity.User, found bool, err error)
	FindByEmail(ctx context.Context, email string) (user *entity.User, found bool, err error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	IncrementAuthVersion(ctx context.Context, userID uuid.UUID, now time.Time) error
}

type IRefreshTokenRepository interface {
	Create(ctx context.Context, token *entity.RefreshToken) error
	FindByHashForUpdate(ctx context.Context, tokenHash string) (token *entity.RefreshToken, found bool, err error)
	Update(ctx context.Context, token *entity.RefreshToken) error
	RevokeFamily(ctx context.Context, familyID uuid.UUID, now time.Time) error
}

type IAuditLogRepository interface {
	Create(ctx context.Context, log *entity.AuditLog) error
}

type IAccessTokenRevocationStore interface {
	Revoke(ctx context.Context, jti uuid.UUID, ttl time.Duration) error
	IsRevoked(ctx context.Context, jti uuid.UUID) (bool, error)
}

type ILoginFailureStore interface {
	IsLocked(ctx context.Context, email string) (locked bool, retryAfter time.Duration, err error)
	RecordFailure(ctx context.Context, email string) (locked bool, retryAfter time.Duration, err error)
	Reset(ctx context.Context, email string) error
}

type IPasswordService interface {
	Hash(password string) (string, error)
	Compare(passwordHash, password string) error
}

type IAccessTokenService interface {
	GenerateAccessToken(userID uuid.UUID, authVersion uint, now time.Time) (token string, expiresAt time.Time, err error)
}

type IRandomTokenService interface {
	GenerateRefreshToken() (string, error)
	GenerateCSRFToken() (string, error)
}

type ITokenHashService interface{ Hash(token string) string }

type IAuthUsecase interface {
	Register(ctx context.Context, input RegisterInput) (*RegisterOutput, error)
	Login(ctx context.Context, input LoginInput) (*LoginOutput, error)
	Refresh(ctx context.Context, input RefreshInput) (*RefreshOutput, error)
	Logout(ctx context.Context, input LogoutInput) (*LogoutOutput, error)
	GetCurrentUser(ctx context.Context, input GetCurrentUserInput) (*GetCurrentUserOutput, error)
}

type AuthUseCase struct {
	users           IUserRepository
	refreshTokens   IRefreshTokenRepository
	auditLogs       IAuditLogRepository
	revokedAccess   IAccessTokenRevocationStore
	loginFailures   ILoginFailureStore
	transactions    ITransactionManager
	passwords       IPasswordService
	accessTokens    IAccessTokenService
	randomTokens    IRandomTokenService
	tokenHashes     ITokenHashService
	refreshTokenTTL time.Duration
	now             func() time.Time
}

func NewAuthUseCase(
	users IUserRepository,
	refreshTokens IRefreshTokenRepository,
	auditLogs IAuditLogRepository,
	revokedAccess IAccessTokenRevocationStore,
	loginFailures ILoginFailureStore,
	transactions ITransactionManager,
	passwords IPasswordService,
	accessTokens IAccessTokenService,
	randomTokens IRandomTokenService,
	tokenHashes ITokenHashService,
	refreshTokenTTL time.Duration,
) *AuthUseCase {
	return &AuthUseCase{
		users: users, refreshTokens: refreshTokens, auditLogs: auditLogs,
		revokedAccess: revokedAccess, loginFailures: loginFailures, transactions: transactions,
		passwords: passwords, accessTokens: accessTokens, randomTokens: randomTokens,
		tokenHashes: tokenHashes, refreshTokenTTL: refreshTokenTTL, now: time.Now,
	}
}
