package entity

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID           uuid.UUID          `gorm:"type:uuid;primaryKey"`
	UserID       uuid.UUID          `gorm:"type:uuid;not null;index:idx_refresh_tokens_user_status"`
	TokenHash    string             `gorm:"size:64;not null;uniqueIndex"`
	FamilyID     uuid.UUID          `gorm:"type:uuid;not null;index:idx_refresh_tokens_family_status"`
	Status       RefreshTokenStatus `gorm:"size:20;not null;default:active;index:idx_refresh_tokens_user_status;index:idx_refresh_tokens_family_status"`
	ExpiresAt    time.Time          `gorm:"not null;index"`
	RotatedAt    *time.Time
	RevokedAt    *time.Time
	LastUsedAt   *time.Time
	ReplacedByID *uuid.UUID `gorm:"type:uuid"`
	IPAddress    string     `gorm:"size:45"`
	UserAgent    string     `gorm:"type:text"`
	CreatedAt    time.Time  `gorm:"not null"`
	UpdatedAt    time.Time  `gorm:"not null"`
}

func (RefreshToken) TableName() string { return "refresh_tokens" }

func NewRefreshToken(userID uuid.UUID, tokenHash string, familyID uuid.UUID, expiresAt, now time.Time) (RefreshToken, error) {
	tokenHash = strings.TrimSpace(tokenHash)
	if userID == uuid.Nil || len(tokenHash) != 64 || !expiresAt.After(now) {
		return RefreshToken{}, fmt.Errorf("valid user id, SHA-256 token hash and future expiration are required")
	}
	if familyID == uuid.Nil {
		familyID = uuid.New()
	}
	return RefreshToken{ID: uuid.New(), UserID: userID, TokenHash: tokenHash, FamilyID: familyID, Status: RefreshTokenStatusActive, ExpiresAt: expiresAt, CreatedAt: now, UpdatedAt: now}, nil
}

func (t RefreshToken) IsExpired(now time.Time) bool { return !t.ExpiresAt.After(now) }
func (t RefreshToken) IsActive(now time.Time) bool {
	return t.Status == RefreshTokenStatusActive && t.RotatedAt == nil && t.RevokedAt == nil && !t.IsExpired(now)
}
func (t *RefreshToken) Rotate(replacementID uuid.UUID, now time.Time) error {
	if replacementID == uuid.Nil {
		return ErrReplacementTokenID
	}
	if t.IsExpired(now) {
		return ErrRefreshTokenExpired
	}
	if !t.IsActive(now) {
		return ErrRefreshTokenInactive
	}
	t.Status = RefreshTokenStatusRotated
	t.RotatedAt = timePointer(now)
	t.LastUsedAt = timePointer(now)
	t.ReplacedByID = &replacementID
	t.UpdatedAt = now
	return nil
}
func (t *RefreshToken) Revoke(now time.Time) error {
	if t.Status == RefreshTokenStatusRevoked {
		return nil
	}
	t.Status = RefreshTokenStatusRevoked
	t.RevokedAt = timePointer(now)
	t.UpdatedAt = now
	return nil
}
