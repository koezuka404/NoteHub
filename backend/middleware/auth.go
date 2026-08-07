package middleware

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	infrcrypto "github.com/koezuka404/notehub/usecase/crypto"
	"github.com/koezuka404/notehub/dto"
	"github.com/koezuka404/notehub/entity"
	"github.com/labstack/echo/v4"
)

const (
	ContextAuthUser       = "auth_user"
	ContextUserID         = "user_id"
	ContextAccessTokenJTI = "access_token_jti"
	ContextAccessTokenExp = "access_token_exp"
	ContextAuthVersion    = "auth_version"
)

// Concrete interfaces avoid coupling Middleware to repository implementations.
type IAuthUserFinder interface {
	FindByID(ctx context.Context, id uuid.UUID) (*entity.User, bool, error)
}

type IAccessTokenValidator interface {
	ValidateAccessToken(raw string, now time.Time) (infrcrypto.AccessTokenClaims, error)
}

type IAccessTokenRevocationChecker interface {
	IsRevoked(ctx context.Context, jti uuid.UUID) (bool, error)
}

type AuthMiddleware struct {
	tokens  IAccessTokenValidator
	users   IAuthUserFinder
	revoked IAccessTokenRevocationChecker
	now     func() time.Time
}

func NewAuthMiddleware(tokens IAccessTokenValidator, users IAuthUserFinder, revoked IAccessTokenRevocationChecker) echo.MiddlewareFunc {
	middleware := &AuthMiddleware{tokens: tokens, users: users, revoked: revoked, now: time.Now}
	return middleware.Handle
}

func (m *AuthMiddleware) Handle(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		authorization := strings.TrimSpace(ctx.Request().Header.Get(echo.HeaderAuthorization))
		if authorization == "" {
			return writeAuthMiddlewareError(ctx, 401, "ACCESS_TOKEN_REQUIRED", "アクセストークンが必要です")
		}
		parts := strings.Fields(authorization)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
			return writeAuthMiddlewareError(ctx, 401, "ACCESS_TOKEN_INVALID", "アクセストークンが不正です")
		}
		claims, err := m.tokens.ValidateAccessToken(parts[1], m.now().UTC())
		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				return writeAuthMiddlewareError(ctx, 401, "ACCESS_TOKEN_EXPIRED", "アクセストークンの有効期限が切れています")
			}
			return writeAuthMiddlewareError(ctx, 401, "ACCESS_TOKEN_INVALID", "アクセストークンが不正です")
		}
		isRevoked, err := m.revoked.IsRevoked(ctx.Request().Context(), claims.JTI)
		if err != nil {
			return writeAuthMiddlewareError(ctx, 503, "AUTH_SERVICE_UNAVAILABLE", "認証サービスを利用できません")
		}
		if isRevoked {
			return writeAuthMiddlewareError(ctx, 401, "ACCESS_TOKEN_REVOKED", "アクセストークンは失効しています")
		}
		user, found, err := m.users.FindByID(ctx.Request().Context(), claims.UserID)
		if err != nil {
			return writeAuthMiddlewareError(ctx, 500, "DATABASE_ERROR", "データベース処理に失敗しました")
		}
		if !found {
			return writeAuthMiddlewareError(ctx, 401, "ACCESS_TOKEN_INVALID", "アクセストークンが不正です")
		}
		if !user.CanAuthenticate() {
			return writeAuthMiddlewareError(ctx, 401, "ACCOUNT_UNAVAILABLE", "このアカウントは利用できません")
		}
		if user.AuthVersion != claims.AuthVersion {
			return writeAuthMiddlewareError(ctx, 401, "ACCESS_TOKEN_REVOKED", "アクセストークンは失効しています")
		}
		ctx.Set(ContextAuthUser, user)
		ctx.Set(ContextUserID, claims.UserID)
		ctx.Set(ContextAccessTokenJTI, claims.JTI)
		ctx.Set(ContextAccessTokenExp, claims.ExpiresAt)
		ctx.Set(ContextAuthVersion, claims.AuthVersion)
		return next(ctx)
	}
}

func writeAuthMiddlewareError(ctx echo.Context, status int, code, message string) error {
	return ctx.JSON(status, dto.ErrorResponse{Error: dto.ErrorBody{Code: code, Message: message}})
}
