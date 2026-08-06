package repository

import (
	"context"
	"fmt"

	"github.com/koezuka404/notehub/entity"
	"gorm.io/gorm"
)

type AuditLogRepository interface {
	Create(ctx context.Context, log *entity.AuditLog) error
}

type auditLogRepository struct{ db *gorm.DB }

func NewAuditLogRepository(db *gorm.DB) AuditLogRepository {
	return &auditLogRepository{db: db}
}

func (r *auditLogRepository) Create(ctx context.Context, log *entity.AuditLog) error {
	if err := dbFromContext(ctx, r.db).Create(log).Error; err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}
