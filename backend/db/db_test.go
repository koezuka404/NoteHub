package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func openTestSQLite(t *testing.T) *gorm.DB {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return gdb
}

func TestOpen_EmptyURL(t *testing.T) {
	if _, err := Open(""); err == nil {
		t.Fatal("expected empty DATABASE_URL error")
	}
}

func TestOpen_InvalidURL(t *testing.T) {
	if _, err := Open("not-a-valid-dsn"); err == nil {
		t.Fatal("expected open postgres error")
	}
}

func TestOpen_Success(t *testing.T) {
	orig := openGormDBFn
	openGormDBFn = func(string) (*gorm.DB, error) {
		return gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	}
	t.Cleanup(func() { openGormDBFn = orig })

	gdb, err := Open("postgres://example")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = Close(gdb) })

	sqlDB, err := SQLDB(gdb)
	if err != nil {
		t.Fatalf("SQLDB: %v", err)
	}
	if err := Ping(context.Background(), gdb); err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if err := sqlDB.PingContext(context.Background()); err != nil {
		t.Fatalf("sql ping: %v", err)
	}
}

func TestOpen_GetSQLDBError(t *testing.T) {
	origOpen := openGormDBFn
	origSQL := getSQLDBFn
	openGormDBFn = func(string) (*gorm.DB, error) {
		return gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	}
	getSQLDBFn = func(*gorm.DB) (*sql.DB, error) {
		return nil, errors.New("get sql db failed")
	}
	t.Cleanup(func() {
		openGormDBFn = origOpen
		getSQLDBFn = origSQL
	})

	if _, err := Open("postgres://example"); err == nil {
		t.Fatal("expected get sql db error")
	}
}

func TestDefaultOpenGormDBFn(t *testing.T) {
	_, err := openGormDBFn("postgres://invalid:invalid@127.0.0.1:1/nope?connect_timeout=1")
	if err == nil {
		t.Fatal("expected postgres open error")
	}
}

func TestPingCloseSQLDB_Error(t *testing.T) {
	orig := getSQLDBFn
	getSQLDBFn = func(*gorm.DB) (*sql.DB, error) {
		return nil, errors.New("sql db unavailable")
	}
	t.Cleanup(func() { getSQLDBFn = orig })

	gdb := openTestSQLite(t)
	ctx := context.Background()

	if err := Ping(ctx, gdb); err == nil {
		t.Fatal("expected Ping error")
	}
	if err := Close(gdb); err == nil {
		t.Fatal("expected Close error")
	}
	if _, err := SQLDB(gdb); err == nil {
		t.Fatal("expected SQLDB error")
	}
}

func TestDefaultExecMigrationSQL(t *testing.T) {
	gdb := openTestSQLite(t)
	err := gdb.Transaction(func(tx *gorm.DB) error {
		return execMigrationSQL(tx, "SELECT 1")
	})
	if err != nil {
		t.Fatalf("execMigrationSQL: %v", err)
	}
}

func TestDefaultAutoMigrateFn(t *testing.T) {
	gdb := openTestSQLite(t)
	if err := gdb.Transaction(autoMigrateFn); err != nil {
		t.Fatalf("autoMigrateFn: %v", err)
	}
}

func TestMigrate_Success(t *testing.T) {
	orig := execMigrationSQL
	execMigrationSQL = func(*gorm.DB, string) error { return nil }
	t.Cleanup(func() { execMigrationSQL = orig })

	gdb := openTestSQLite(t)
	if err := Migrate(gdb); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
}

func TestMigrate_AutoMigrateError(t *testing.T) {
	orig := autoMigrateFn
	autoMigrateFn = func(*gorm.DB) error { return errors.New("auto migrate failed") }
	t.Cleanup(func() { autoMigrateFn = orig })

	gdb := openTestSQLite(t)
	if err := Migrate(gdb); err == nil {
		t.Fatal("expected auto migrate error")
	}
}

func TestMigrate_StatementError(t *testing.T) {
	orig := execMigrationSQL
	execMigrationSQL = func(*gorm.DB, string) error { return errors.New("statement failed") }
	t.Cleanup(func() { execMigrationSQL = orig })

	gdb := openTestSQLite(t)
	if err := Migrate(gdb); err == nil {
		t.Fatal("expected migration statement error")
	}
}

func TestOpen_ConfiguresPool(t *testing.T) {
	orig := openGormDBFn
	openGormDBFn = func(string) (*gorm.DB, error) {
		return gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	}
	t.Cleanup(func() { openGormDBFn = orig })

	gdb, err := Open("postgres://example")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = Close(gdb) })

	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatalf("DB: %v", err)
	}
	if sqlDB.Stats().MaxOpenConnections != 50 {
		t.Fatalf("MaxOpenConnections = %d", sqlDB.Stats().MaxOpenConnections)
	}
}
