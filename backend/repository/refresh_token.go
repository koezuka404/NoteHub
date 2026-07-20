package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RefreshTokenRepository struct{ db *gorm.DB }

func NewRefreshTokenRepository(db *gorm.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}
func (r *RefreshTokenRepository) Create(ctx context.Context, token *entity.RefreshToken) error {
	if err := dbFromContext(ctx, r.db).Create(token).Error; err != nil {
		return fmt.Errorf("insert refresh token: %w", err)
	}
	return nil
}
func (r *RefreshTokenRepository) FindByHashForUpdate(ctx context.Context, hash string) (*entity.RefreshToken, bool, error) {
	var token entity.RefreshToken
	err := dbFromContext(ctx, r.db).Clauses(clause.Locking{Strength: "UPDATE"}).Where("token_hash = ?", hash).First(&token).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("select refresh token for update: %w", err)
	}
	return &token, true, nil
}
func (r *RefreshTokenRepository) Update(ctx context.Context, token *entity.RefreshToken) error {
	if err := dbFromContext(ctx, r.db).Save(token).Error; err != nil {
		return fmt.Errorf("update refresh token: %w", err)
	}
	return nil
}
func (r *RefreshTokenRepository) RevokeFamily(ctx context.Context, familyID uuid.UUID, now time.Time) error {
	result := dbFromContext(ctx, r.db).Model(&entity.RefreshToken{}).
		Where("family_id = ? AND status = ?", familyID, entity.RefreshTokenStatusActive).
		Updates(map[string]any{"status": entity.RefreshTokenStatusRevoked, "revoked_at": now, "updated_at": now})
	if result.Error != nil {
		return fmt.Errorf("revoke refresh token family: %w", result.Error)
	}
	return nil
}
