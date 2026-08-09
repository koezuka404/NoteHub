package db

import (
	"database/sql"

	"github.com/koezuka404/notehub/entity"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var openGormDBFn = func(databaseURL string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
}

var autoMigrateFn = func(tx *gorm.DB) error {
	return tx.AutoMigrate(
		&entity.User{},
		&entity.RefreshToken{},
		&entity.AuditLog{},
		&entity.Workspace{},
		&entity.WorkspaceMember{},
		&entity.Document{},
		&entity.DocumentVersion{},
	)
}

var execMigrationSQL = func(tx *gorm.DB, statement string) error {
	return tx.Exec(statement).Error
}

var getSQLDBFn = func(gdb *gorm.DB) (*sql.DB, error) {
	return gdb.DB()
}
