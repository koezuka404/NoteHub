package batch

import (
	"context"
	"log"
	"time"

	"github.com/koezuka404/notehub/usecase"
)

type BackupBatch struct {
	backup   *usecase.DatabaseBackupUseCase
	interval time.Duration
}

func NewBackupBatch(backup *usecase.DatabaseBackupUseCase, interval time.Duration) *BackupBatch {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	return &BackupBatch{backup: backup, interval: interval}
}

func (b *BackupBatch) Run(ctx context.Context) {
	ticker := time.NewTicker(b.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
			if err := b.backup.RunOnce(runCtx); err != nil {
				log.Printf("backup batch: %v", err)
			}
			cancel()
		}
	}
}
