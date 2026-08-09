package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
	infrcrypto "github.com/koezuka404/notehub/usecase/crypto"
	"github.com/labstack/echo/v4"
)

type mockTokenValidator struct {
	claims infrcrypto.AccessTokenClaims
	err    error
}

func (m *mockTokenValidator) ValidateAccessToken(string, time.Time) (infrcrypto.AccessTokenClaims, error) {
	return m.claims, m.err
}

type mockUserFinder struct {
	user  *entity.User
	found bool
	err   error
}

func (m *mockUserFinder) FindByID(context.Context, uuid.UUID) (*entity.User, bool, error) {
	return m.user, m.found, m.err
}

type mockRevocationChecker struct {
	revoked bool
	err     error
}

func (m *mockRevocationChecker) IsRevoked(context.Context, uuid.UUID) (bool, error) {
	return m.revoked, m.err
}

func runAuthMiddleware(
	t *testing.T,
	tokens IAccessTokenValidator,
	users IAuthUserFinder,
	revoked IAccessTokenRevocationChecker,
	authorization string,
) (*httptest.ResponseRecorder, echo.Context) {
	t.Helper()

	e := echo.New()
	var captured echo.Context
	e.Use(NewAuthMiddleware(tokens, users, revoked))
	e.GET("/", func(ctx echo.Context) error {
		captured = ctx
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if authorization != "" {
		req.Header.Set(echo.HeaderAuthorization, authorization)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec, captured
}

func validAuthUser(claims infrcrypto.AccessTokenClaims) entity.User {
	return entity.User{
		ID:          claims.UserID,
		Status:      entity.UserStatusActive,
		AuthVersion: claims.AuthVersion,
	}
}

func TestAuthMiddleware_MissingAuthorization(t *testing.T) {
	rec, _ := runAuthMiddleware(t, &mockTokenValidator{}, &mockUserFinder{}, &mockRevocationChecker{}, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
	payload := decodeErrorResponse(t, rec)
	if payload.Error.Code != "ACCESS_TOKEN_REQUIRED" {
		t.Fatalf("code = %q", payload.Error.Code)
	}
}

func TestAuthMiddleware_InvalidBearerFormat(t *testing.T) {
	rec, _ := runAuthMiddleware(t, &mockTokenValidator{}, &mockUserFinder{}, &mockRevocationChecker{}, "Token abc")
	payload := decodeErrorResponse(t, rec)
	if payload.Error.Code != "ACCESS_TOKEN_INVALID" {
		t.Fatalf("code = %q", payload.Error.Code)
	}
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	rec, _ := runAuthMiddleware(
		t,
		&mockTokenValidator{err: jwt.ErrTokenExpired},
		&mockUserFinder{},
		&mockRevocationChecker{},
		"Bearer token",
	)
	payload := decodeErrorResponse(t, rec)
	if payload.Error.Code != "ACCESS_TOKEN_EXPIRED" {
		t.Fatalf("code = %q", payload.Error.Code)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	rec, _ := runAuthMiddleware(
		t,
		&mockTokenValidator{err: errors.New("invalid")},
		&mockUserFinder{},
		&mockRevocationChecker{},
		"Bearer token",
	)
	payload := decodeErrorResponse(t, rec)
	if payload.Error.Code != "ACCESS_TOKEN_INVALID" {
		t.Fatalf("code = %q", payload.Error.Code)
	}
}

func TestAuthMiddleware_RevocationCheckError(t *testing.T) {
	claims := infrcrypto.AccessTokenClaims{UserID: uuid.New(), JTI: uuid.New(), AuthVersion: 1}
	rec, _ := runAuthMiddleware(
		t,
		&mockTokenValidator{claims: claims},
		&mockUserFinder{},
		&mockRevocationChecker{err: errors.New("redis down")},
		"Bearer token",
	)
	payload := decodeErrorResponse(t, rec)
	if payload.Error.Code != "AUTH_SERVICE_UNAVAILABLE" {
		t.Fatalf("code = %q", payload.Error.Code)
	}
}

func TestAuthMiddleware_RevokedToken(t *testing.T) {
	claims := infrcrypto.AccessTokenClaims{UserID: uuid.New(), JTI: uuid.New(), AuthVersion: 1}
	rec, _ := runAuthMiddleware(
		t,
		&mockTokenValidator{claims: claims},
		&mockUserFinder{},
		&mockRevocationChecker{revoked: true},
		"Bearer token",
	)
	payload := decodeErrorResponse(t, rec)
	if payload.Error.Code != "ACCESS_TOKEN_REVOKED" {
		t.Fatalf("code = %q", payload.Error.Code)
	}
}

func TestAuthMiddleware_UserLookupError(t *testing.T) {
	claims := infrcrypto.AccessTokenClaims{UserID: uuid.New(), JTI: uuid.New(), AuthVersion: 1}
	rec, _ := runAuthMiddleware(
		t,
		&mockTokenValidator{claims: claims},
		&mockUserFinder{err: errors.New("db error")},
		&mockRevocationChecker{},
		"Bearer token",
	)
	payload := decodeErrorResponse(t, rec)
	if payload.Error.Code != "DATABASE_ERROR" {
		t.Fatalf("code = %q", payload.Error.Code)
	}
}

func TestAuthMiddleware_UserNotFound(t *testing.T) {
	claims := infrcrypto.AccessTokenClaims{UserID: uuid.New(), JTI: uuid.New(), AuthVersion: 1}
	rec, _ := runAuthMiddleware(
		t,
		&mockTokenValidator{claims: claims},
		&mockUserFinder{found: false},
		&mockRevocationChecker{},
		"Bearer token",
	)
	payload := decodeErrorResponse(t, rec)
	if payload.Error.Code != "ACCESS_TOKEN_INVALID" {
		t.Fatalf("code = %q", payload.Error.Code)
	}
}

func TestAuthMiddleware_AccountUnavailable(t *testing.T) {
	claims := infrcrypto.AccessTokenClaims{UserID: uuid.New(), JTI: uuid.New(), AuthVersion: 1}
	user := validAuthUser(claims)
	user.Status = entity.UserStatusSuspended
	rec, _ := runAuthMiddleware(
		t,
		&mockTokenValidator{claims: claims},
		&mockUserFinder{user: &user, found: true},
		&mockRevocationChecker{},
		"Bearer token",
	)
	payload := decodeErrorResponse(t, rec)
	if payload.Error.Code != "ACCOUNT_UNAVAILABLE" {
		t.Fatalf("code = %q", payload.Error.Code)
	}
}

func TestAuthMiddleware_AuthVersionMismatch(t *testing.T) {
	claims := infrcrypto.AccessTokenClaims{UserID: uuid.New(), JTI: uuid.New(), AuthVersion: 1}
	user := validAuthUser(claims)
	user.AuthVersion = 2
	rec, _ := runAuthMiddleware(
		t,
		&mockTokenValidator{claims: claims},
		&mockUserFinder{user: &user, found: true},
		&mockRevocationChecker{},
		"Bearer token",
	)
	payload := decodeErrorResponse(t, rec)
	if payload.Error.Code != "ACCESS_TOKEN_REVOKED" {
		t.Fatalf("code = %q", payload.Error.Code)
	}
}

func TestAuthMiddleware_Success(t *testing.T) {
	claims := infrcrypto.AccessTokenClaims{
		UserID:      uuid.New(),
		JTI:         uuid.New(),
		ExpiresAt:   time.Now().Add(time.Hour),
		AuthVersion: 3,
	}
	user := validAuthUser(claims)

	rec, ctx := runAuthMiddleware(
		t,
		&mockTokenValidator{claims: claims},
		&mockUserFinder{user: &user, found: true},
		&mockRevocationChecker{},
		"Bearer valid-token",
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if ctx == nil {
		t.Fatal("expected handler to run")
	}
	if got, ok := ctx.Get(ContextAuthUser).(*entity.User); !ok || got.ID != user.ID {
		t.Fatal("expected auth user in context")
	}
	if got, ok := ctx.Get(ContextUserID).(uuid.UUID); !ok || got != claims.UserID {
		t.Fatalf("user id = %v", got)
	}
	if got, ok := ctx.Get(ContextAccessTokenJTI).(uuid.UUID); !ok || got != claims.JTI {
		t.Fatalf("jti = %v", got)
	}
	if got, ok := ctx.Get(ContextAccessTokenExp).(time.Time); !ok || !got.Equal(claims.ExpiresAt) {
		t.Fatalf("exp = %v", got)
	}
	if got, ok := ctx.Get(ContextAuthVersion).(uint); !ok || got != claims.AuthVersion {
		t.Fatalf("auth version = %d", got)
	}
}
