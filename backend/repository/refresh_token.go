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

func (r *RefreshTokenRepository) MarkExpiredBefore(ctx context.Context, now time.Time) (int64, error) {
	result := dbFromContext(ctx, r.db).Model(&entity.RefreshToken{}).
		Where("expires_at < ? AND status IN ?", now, []entity.RefreshTokenStatus{
			entity.RefreshTokenStatusActive,
			entity.RefreshTokenStatusRotated,
		}).
		Updates(map[string]any{"status": entity.RefreshTokenStatusExpired, "updated_at": now})
	if result.Error != nil {
		return 0, fmt.Errorf("mark expired refresh tokens: %w", result.Error)
	}
	return result.RowsAffected, nil
}

func (r *RefreshTokenRepository) DeleteStaleBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	result := dbFromContext(ctx, r.db).
		Where("status IN ? AND updated_at < ?", []entity.RefreshTokenStatus{
			entity.RefreshTokenStatusExpired,
			entity.RefreshTokenStatusRevoked,
			entity.RefreshTokenStatusRotated,
		}, cutoff).
		Delete(&entity.RefreshToken{})
	if result.Error != nil {
		return 0, fmt.Errorf("delete stale refresh tokens: %w", result.Error)
	}
	return result.RowsAffected, nil
}
