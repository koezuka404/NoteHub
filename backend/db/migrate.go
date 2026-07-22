package db

import (
	"fmt"

	"github.com/koezuka404/notehub/entity"
	"gorm.io/gorm"
)

func Migrate(gdb *gorm.DB) error {
	return gdb.Transaction(func(tx *gorm.DB) error {
		if err := tx.AutoMigrate(
			&entity.User{},
			&entity.RefreshToken{},
			&entity.AuditLog{},
		); err != nil {
			return fmt.Errorf("auto migrate: %w", err)
		}

		statements := []string{
			`CREATE UNIQUE INDEX IF NOT EXISTS uq_users_email_active
			 ON users (LOWER(email))
			 WHERE deleted_at IS NULL`,

			`CREATE INDEX IF NOT EXISTS idx_users_status
			 ON users (status)`,

			`CREATE UNIQUE INDEX IF NOT EXISTS uq_refresh_tokens_hash
			 ON refresh_tokens (token_hash)`,

			`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_family_status
			 ON refresh_tokens (family_id, status)`,

			`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_status
			 ON refresh_tokens (user_id, status)`,

			`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_at
			 ON refresh_tokens (expires_at)`,

			`CREATE INDEX IF NOT EXISTS idx_audit_logs_actor_created
			 ON audit_logs (actor_user_id, created_at DESC)`,

			`CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_created
			 ON audit_logs (resource_type, resource_id, created_at DESC)`,
		}

		for _, statement := range statements {
			if err := tx.Exec(statement).Error; err != nil {
				return fmt.Errorf("apply migration statement: %w", err)
			}
		}

		return nil
	})
}
