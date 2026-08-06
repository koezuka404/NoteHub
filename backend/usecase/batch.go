package usecase

import (
	"bytes"
	"context"
	"fmt"
	"github.com/koezuka404/notehub/entity"
	"github.com/koezuka404/notehub/repository"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// cleanup.go

type ICleanupRedisStore interface {
	CleanupEphemeralKeys(ctx context.Context) (int, error)
}

type CleanupUseCase struct {
	refreshTokens repository.RefreshTokenRepository
	redis         ICleanupRedisStore
	retention     time.Duration
	now           func() time.Time
}

func NewCleanupUseCase(
	refreshTokens repository.RefreshTokenRepository,
	redis ICleanupRedisStore,
	retention time.Duration,
) *CleanupUseCase {
	if retention <= 0 {
		retention = 30 * 24 * time.Hour
	}
	return &CleanupUseCase{
		refreshTokens: refreshTokens,
		redis:         redis,
		retention:     retention,
		now:           time.Now,
	}
}

func (u *CleanupUseCase) RunOnce(ctx context.Context) error {
	now := u.now()

	marked, err := u.refreshTokens.MarkExpiredBefore(ctx, now)
	if err != nil {
		return err
	}

	cutoff := now.Add(-u.retention)
	deleted, err := u.refreshTokens.DeleteStaleBefore(ctx, cutoff)
	if err != nil {
		return err
	}

	removed, err := u.redis.CleanupEphemeralKeys(ctx)
	if err != nil {
		return err
	}

	if marked > 0 || deleted > 0 || removed > 0 {
		log.Printf(
			"cleanup batch: marked %d expired refresh tokens, deleted %d stale refresh tokens, removed %d redis keys",
			marked,
			deleted,
			removed,
		)
	}
	return nil
}

// database_backup.go

type DatabaseBackupUseCase struct {
	databaseURL string
	backupDir   string
	retention   time.Duration
	audit       repository.AuditLogRepository
	now         func() time.Time
}

func NewDatabaseBackupUseCase(
	databaseURL string,
	backupDir string,
	retention time.Duration,
	audit repository.AuditLogRepository,
) *DatabaseBackupUseCase {
	if retention <= 0 {
		retention = 7 * 24 * time.Hour
	}
	return &DatabaseBackupUseCase{
		databaseURL: strings.TrimSpace(databaseURL),
		backupDir:   strings.TrimSpace(backupDir),
		retention:   retention,
		audit:       audit,
		now:         time.Now,
	}
}

func (u *DatabaseBackupUseCase) RunOnce(ctx context.Context) error {
	if u.databaseURL == "" {
		return fmt.Errorf("database url is required")
	}
	if u.backupDir == "" {
		return fmt.Errorf("backup directory is required")
	}
	if err := os.MkdirAll(u.backupDir, 0o700); err != nil {
		return fmt.Errorf("create backup directory: %w", err)
	}

	now := u.now().UTC()
	filename := fmt.Sprintf("notehub-backup-%s.sql", now.Format("20060102-150405"))
	outputPath := filepath.Join(u.backupDir, filename)

	if err := u.runPgDump(ctx, outputPath); err != nil {
		return err
	}

	removed, err := u.pruneOldBackups(now)
	if err != nil {
		log.Printf("backup batch: prune old backups: %v", err)
	}

	audit, err := entity.NewAuditLog(nil, "BACKUP_EXECUTED", "database", nil, map[string]any{
		"filename": filename,
		"path":     outputPath,
		"removed":  removed,
	}, now)
	if err != nil {
		return fmt.Errorf("build backup audit log: %w", err)
	}
	if err := u.audit.Create(ctx, &audit); err != nil {
		return fmt.Errorf("record backup audit log: %w", err)
	}

	log.Printf("backup batch: created %s (removed %d old backups)", outputPath, removed)
	return nil
}

func (u *DatabaseBackupUseCase) runPgDump(ctx context.Context, outputPath string) error {
	cmd := exec.CommandContext(
		ctx,
		"pg_dump",
		u.databaseURL,
		"-F", "p",
		"-f", outputPath,
		"--no-owner",
		"--no-acl",
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_dump: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func (u *DatabaseBackupUseCase) pruneOldBackups(now time.Time) (int, error) {
	cutoff := now.Add(-u.retention)
	entries, err := os.ReadDir(u.backupDir)
	if err != nil {
		return 0, fmt.Errorf("read backup directory: %w", err)
	}

	removed := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "notehub-backup-") || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return removed, fmt.Errorf("stat backup file %s: %w", entry.Name(), err)
		}
		if info.ModTime().UTC().After(cutoff) {
			continue
		}
		path := filepath.Join(u.backupDir, entry.Name())
		if err := os.Remove(path); err != nil {
			return removed, fmt.Errorf("remove old backup %s: %w", path, err)
		}
		removed++
	}
	return removed, nil
}
