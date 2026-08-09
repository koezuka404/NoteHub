package authservice

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	infrcrypto "github.com/koezuka404/notehub/usecase/crypto"
	appredis "github.com/koezuka404/notehub/redis"
	"golang.org/x/crypto/bcrypt"
)

const testJWTSecret = "01234567890123456789012345678901"

func newTestAuthService(t *testing.T, withLoginFailures bool) *AuthService {
	t.Helper()

	mr := miniredis.RunT(t)
	redisClient, err := appredis.NewClientWithTimeout("redis://"+mr.Addr()+"/0", 2*time.Second)
	if err != nil {
		t.Fatalf("NewClientWithTimeout: %v", err)
	}
	t.Cleanup(func() { _ = redisClient.Close() })

	jwtService, err := infrcrypto.NewJWTService(testJWTSecret, "notehub", "notehub-client", 15*time.Minute)
	if err != nil {
		t.Fatalf("NewJWTService: %v", err)
	}

	var loginFails *appredis.LoginFailureStore
	if withLoginFailures {
		loginFails = appredis.NewLoginFailureStore(redisClient, 3, time.Minute, 5*time.Minute)
	}

	return NewAuthService(
		infrcrypto.NewPasswordService(bcrypt.MinCost),
		jwtService,
		infrcrypto.NewRandomTokenService(),
		infrcrypto.NewTokenHashService(),
		appredis.NewAccessTokenRevocationStore(redisClient),
		loginFails,
	)
}

func TestNewAuthService(t *testing.T) {
	svc := newTestAuthService(t, true)
	if svc == nil || svc.passwords == nil || svc.jwt == nil || svc.random == nil ||
		svc.hasher == nil || svc.revocations == nil || svc.loginFails == nil {
		t.Fatal("expected fully initialized auth service")
	}
}

func TestAuthService_PasswordMethods(t *testing.T) {
	svc := newTestAuthService(t, false)

	hash, err := svc.HashPassword("Pass1234")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if err := svc.ComparePassword(hash, "Pass1234"); err != nil {
		t.Fatalf("ComparePassword match: %v", err)
	}
	if err := svc.ComparePassword(hash, "wrong"); err == nil {
		t.Fatal("expected password mismatch")
	}
}

func TestAuthService_GenerateAccessToken(t *testing.T) {
	svc := newTestAuthService(t, false)
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	userID := uuid.New()

	token, expiresAt, err := svc.GenerateAccessToken(userID, 2, now)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}
	if token == "" || !expiresAt.After(now) {
		t.Fatalf("token=%q expiresAt=%v", token, expiresAt)
	}
}

func TestAuthService_RandomTokens(t *testing.T) {
	svc := newTestAuthService(t, false)

	refresh, err := svc.GenerateRefreshToken()
	if err != nil || refresh == "" {
		t.Fatalf("GenerateRefreshToken: %q, %v", refresh, err)
	}
	csrf, err := svc.GenerateCSRFToken()
	if err != nil || csrf == "" {
		t.Fatalf("GenerateCSRFToken: %q, %v", csrf, err)
	}
}

func TestAuthService_HashToken(t *testing.T) {
	svc := newTestAuthService(t, false)
	if got := svc.HashToken("token-value"); got == "" || got == "token-value" {
		t.Fatalf("HashToken = %q", got)
	}
}

func TestAuthService_AccessTokenRevocation(t *testing.T) {
	svc := newTestAuthService(t, false)
	ctx := context.Background()
	jti := uuid.New()

	if err := svc.RevokeAccessToken(ctx, jti, time.Minute); err != nil {
		t.Fatalf("RevokeAccessToken: %v", err)
	}
	revoked, err := svc.IsAccessTokenRevoked(ctx, jti)
	if err != nil {
		t.Fatalf("IsAccessTokenRevoked: %v", err)
	}
	if !revoked {
		t.Fatal("expected revoked access token")
	}

	revoked, err = svc.IsAccessTokenRevoked(ctx, uuid.Nil)
	if err != nil || revoked {
		t.Fatalf("nil jti should not be revoked: revoked=%v err=%v", revoked, err)
	}
}

func TestAuthService_LoginFailureNilStore(t *testing.T) {
	svc := newTestAuthService(t, false)
	ctx := context.Background()

	locked, retryAfter, err := svc.IsLoginLocked(ctx, "user@example.com")
	if err != nil || locked || retryAfter != 0 {
		t.Fatalf("IsLoginLocked() = %v, %v, %v", locked, retryAfter, err)
	}

	locked, retryAfter, err = svc.RecordLoginFailure(ctx, "user@example.com")
	if err != nil || locked || retryAfter != 0 {
		t.Fatalf("RecordLoginFailure() = %v, %v, %v", locked, retryAfter, err)
	}

	if err := svc.ResetLoginFailures(ctx, "user@example.com"); err != nil {
		t.Fatalf("ResetLoginFailures: %v", err)
	}
}

func TestAuthService_LoginFailureWithStore(t *testing.T) {
	svc := newTestAuthService(t, true)
	ctx := context.Background()
	email := "locked@example.com"

	for i := 0; i < 2; i++ {
		locked, _, err := svc.RecordLoginFailure(ctx, email)
		if err != nil {
			t.Fatalf("RecordLoginFailure attempt %d: %v", i+1, err)
		}
		if locked {
			t.Fatalf("unexpected lock on attempt %d", i+1)
		}
	}

	locked, retryAfter, err := svc.RecordLoginFailure(ctx, email)
	if err != nil {
		t.Fatalf("RecordLoginFailure lock attempt: %v", err)
	}
	if !locked || retryAfter <= 0 {
		t.Fatalf("expected lock: locked=%v retryAfter=%v", locked, retryAfter)
	}

	locked, retryAfter, err = svc.IsLoginLocked(ctx, email)
	if err != nil {
		t.Fatalf("IsLoginLocked: %v", err)
	}
	if !locked || retryAfter <= 0 {
		t.Fatalf("expected locked account: locked=%v retryAfter=%v", locked, retryAfter)
	}

	if err := svc.ResetLoginFailures(ctx, email); err != nil {
		t.Fatalf("ResetLoginFailures: %v", err)
	}

	locked, retryAfter, err = svc.IsLoginLocked(ctx, email)
	if err != nil {
		t.Fatalf("IsLoginLocked after reset: %v", err)
	}
	if locked || retryAfter != 0 {
		t.Fatalf("expected unlocked after reset: locked=%v retryAfter=%v", locked, retryAfter)
	}
}
