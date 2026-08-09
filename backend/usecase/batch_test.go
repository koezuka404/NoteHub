package usecase

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewCleanupUseCase_DefaultRetention(t *testing.T) {
	uc := NewCleanupUseCase(&stubRefreshRepoFull{}, &stubCleanupRedis{}, 0)
	if uc.retention != 30*24*time.Hour {
		t.Fatalf("retention = %v", uc.retention)
	}
}

func TestCleanupUseCase_RunOnce_MarkError(t *testing.T) {
	refresh := &stubRefreshRepoFull{markErr: fmt.Errorf("mark failed")}
	uc := NewCleanupUseCase(refresh, &stubCleanupRedis{}, time.Hour)
	uc.now = usecaseTestNow

	if err := uc.RunOnce(context.Background()); err == nil {
		t.Fatal("expected mark error")
	}
}

func TestCleanupUseCase_RunOnce_DeleteError(t *testing.T) {
	refresh := &stubRefreshRepoFull{deleteErr: fmt.Errorf("delete failed")}
	uc := NewCleanupUseCase(refresh, &stubCleanupRedis{}, time.Hour)
	uc.now = usecaseTestNow

	if err := uc.RunOnce(context.Background()); err == nil {
		t.Fatal("expected delete error")
	}
}

func TestCleanupUseCase_RunOnce_RedisError(t *testing.T) {
	redis := &stubCleanupRedis{err: fmt.Errorf("redis failed")}
	uc := NewCleanupUseCase(&stubRefreshRepoFull{}, redis, time.Hour)
	uc.now = usecaseTestNow

	if err := uc.RunOnce(context.Background()); err == nil {
		t.Fatal("expected redis error")
	}
}

func TestCleanupUseCase_RunOnce_Success(t *testing.T) {
	redis := &stubCleanupRedis{removed: 3}
	uc := NewCleanupUseCase(&stubRefreshRepoFull{}, redis, time.Hour)
	uc.now = usecaseTestNow

	if err := uc.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
}

func TestNewDatabaseBackupUseCase_DefaultRetention(t *testing.T) {
	uc := NewDatabaseBackupUseCase("postgres://x", "/tmp", 0, &mockAuditLogRepo{})
	if uc.retention != 7*24*time.Hour {
		t.Fatalf("retention = %v", uc.retention)
	}
}

func TestDatabaseBackupUseCase_Validation(t *testing.T) {
	uc := NewDatabaseBackupUseCase("", "", time.Hour, &mockAuditLogRepo{})
	if err := uc.RunOnce(context.Background()); err == nil {
		t.Fatal("expected database url error")
	}

	uc = NewDatabaseBackupUseCase("postgres://x", "", time.Hour, &mockAuditLogRepo{})
	if err := uc.RunOnce(context.Background()); err == nil {
		t.Fatal("expected backup dir error")
	}
}

func TestDatabaseBackupUseCase_PruneOldBackups(t *testing.T) {
	dir := t.TempDir()
	oldFile := filepath.Join(dir, "notehub-backup-old.sql")
	if err := os.WriteFile(oldFile, []byte("backup"), 0o600); err != nil {
		t.Fatalf("write old backup: %v", err)
	}
	oldTime := usecaseTestNow().Add(-10 * 24 * time.Hour)
	if err := os.Chtimes(oldFile, oldTime, oldTime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	newFile := filepath.Join(dir, "notehub-backup-new.sql")
	if err := os.WriteFile(newFile, []byte("backup"), 0o600); err != nil {
		t.Fatalf("write new backup: %v", err)
	}

	uc := NewDatabaseBackupUseCase("postgres://x", dir, 7*24*time.Hour, &mockAuditLogRepo{})
	uc.now = usecaseTestNow

	removed, err := uc.pruneOldBackups(usecaseTestNow())
	if err != nil {
		t.Fatalf("pruneOldBackups: %v", err)
	}
	if removed != 1 {
		t.Fatalf("removed = %d", removed)
	}
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Fatal("expected old backup removed")
	}
	if _, err := os.Stat(newFile); err != nil {
		t.Fatalf("expected new backup kept: %v", err)
	}
}

func TestDatabaseBackupUseCase_RunOnce_Success(t *testing.T) {
	dir := t.TempDir()
	binDir := t.TempDir()
	scriptPath := filepath.Join(binDir, "pg_dump")
	script := fmt.Sprintf("#!/bin/sh\nshift\nwhile [ $# -gt 0 ]; do\n  case \"$1\" in\n    -f) OUT=\"$2\"; shift 2 ;;\n    *) shift ;;\n  esac\ndone\necho 'backup data' > \"$OUT\"\n")
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake pg_dump: %v", err)
	}

	oldPath := os.Getenv("PATH")
	t.Cleanup(func() { _ = os.Setenv("PATH", oldPath) })
	if err := os.Setenv("PATH", binDir+string(os.PathListSeparator)+oldPath); err != nil {
		t.Fatalf("set PATH: %v", err)
	}

	audit := &mockAuditLogRepo{}
	uc := NewDatabaseBackupUseCase("postgres://user:pass@localhost/db", dir, 7*24*time.Hour, audit)
	uc.now = usecaseTestNow

	if err := uc.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if len(audit.logs) != 1 || audit.logs[0].Action != "BACKUP_EXECUTED" {
		t.Fatalf("expected backup audit log, got %+v", audit.logs)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read backup dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 backup file, got %d", len(entries))
	}
}
