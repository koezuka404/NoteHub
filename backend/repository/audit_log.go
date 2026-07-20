package repository

import (
	"context"
	"fmt"

	"github.com/koezuka404/notehub/entity"
	"gorm.io/gorm"
)

type AuditLogRepository struct{ db *gorm.DB }

func NewAuditLogRepository(db *gorm.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

func (r *AuditLogRepository) Create(ctx context.Context, log *entity.AuditLog) error {
	if err := dbFromContext(ctx, r.db).Create(log).Error; err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}
