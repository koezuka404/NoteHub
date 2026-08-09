package entity

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func validTokenHash() string {
	return strings.Repeat("a", 64)
}

func activeRefreshToken(t *testing.T) RefreshToken {
	t.Helper()
	now := testNow()
	token, err := NewRefreshToken(uuid.New(), validTokenHash(), uuid.New(), now.Add(time.Hour), now)
	if err != nil {
		t.Fatalf("NewRefreshToken: %v", err)
	}
	return token
}

func TestRefreshToken_TableName(t *testing.T) {
	if (RefreshToken{}).TableName() != "refresh_tokens" {
		t.Fatalf("table name = %q", (RefreshToken{}).TableName())
	}
}

func TestNewRefreshToken(t *testing.T) {
	now := testNow()
	userID := uuid.New()
	familyID := uuid.New()
	expires := now.Add(time.Hour)

	token, err := NewRefreshToken(userID, validTokenHash(), familyID, expires, now)
	if err != nil {
		t.Fatalf("NewRefreshToken: %v", err)
	}
	if token.UserID != userID || token.FamilyID != familyID || token.Status != RefreshTokenStatusActive {
		t.Fatalf("unexpected token: %+v", token)
	}
}

func TestNewRefreshToken_GeneratesFamilyID(t *testing.T) {
	now := testNow()
	token, err := NewRefreshToken(uuid.New(), validTokenHash(), uuid.Nil, now.Add(time.Hour), now)
	if err != nil {
		t.Fatalf("NewRefreshToken: %v", err)
	}
	if token.FamilyID == uuid.Nil {
		t.Fatal("expected generated family id")
	}
}

func TestNewRefreshToken_ValidationErrors(t *testing.T) {
	now := testNow()
	expires := now.Add(time.Hour)

	if _, err := NewRefreshToken(uuid.Nil, validTokenHash(), uuid.New(), expires, now); err == nil {
		t.Fatal("expected error for nil user id")
	}
	if _, err := NewRefreshToken(uuid.New(), "short", uuid.New(), expires, now); err == nil {
		t.Fatal("expected error for invalid hash length")
	}
	if _, err := NewRefreshToken(uuid.New(), validTokenHash(), uuid.New(), now, now); err == nil {
		t.Fatal("expected error for non-future expiration")
	}
}

func TestRefreshToken_IsExpiredAndActive(t *testing.T) {
	now := testNow()
	token := activeRefreshToken(t)

	if token.IsExpired(now) {
		t.Fatal("token should not be expired yet")
	}
	if !token.IsActive(now) {
		t.Fatal("token should be active")
	}

	expired := token
	expired.ExpiresAt = now.Add(-time.Minute)
	if !expired.IsExpired(now) || expired.IsActive(now) {
		t.Fatal("expired token should not be active")
	}

	revoked := activeRefreshToken(t)
	_ = revoked.Revoke(now)
	if revoked.IsActive(now) {
		t.Fatal("revoked token should not be active")
	}
}

func TestRefreshToken_Rotate(t *testing.T) {
	token := activeRefreshToken(t)
	now := testNow().Add(time.Minute)
	replacementID := uuid.New()

	if err := token.Rotate(replacementID, now); err != nil {
		t.Fatalf("Rotate: %v", err)
	}
	if token.Status != RefreshTokenStatusRotated || token.ReplacedByID == nil || *token.ReplacedByID != replacementID {
		t.Fatalf("unexpected token after rotate: %+v", token)
	}
	if token.RotatedAt == nil || token.LastUsedAt == nil {
		t.Fatal("expected rotated and last used timestamps")
	}
}

func TestRefreshToken_Rotate_Errors(t *testing.T) {
	token := activeRefreshToken(t)
	now := testNow()

	if err := token.Rotate(uuid.Nil, now); !errors.Is(err, ErrReplacementTokenID) {
		t.Fatalf("expected ErrReplacementTokenID, got %v", err)
	}

	expired := activeRefreshToken(t)
	expired.ExpiresAt = now.Add(-time.Hour)
	if err := expired.Rotate(uuid.New(), now); !errors.Is(err, ErrRefreshTokenExpired) {
		t.Fatalf("expected ErrRefreshTokenExpired, got %v", err)
	}

	inactive := activeRefreshToken(t)
	_ = inactive.Revoke(now)
	if err := inactive.Rotate(uuid.New(), now); !errors.Is(err, ErrRefreshTokenInactive) {
		t.Fatalf("expected ErrRefreshTokenInactive, got %v", err)
	}
}

func TestRefreshToken_Revoke(t *testing.T) {
	token := activeRefreshToken(t)
	now := testNow()

	if err := token.Revoke(now); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if token.Status != RefreshTokenStatusRevoked || token.RevokedAt == nil {
		t.Fatal("expected revoked state")
	}
	if err := token.Revoke(now); err != nil {
		t.Fatalf("second Revoke should be no-op: %v", err)
	}
}
