package batch

import "context"

var (
	runAutoSaveOnce = func(b *AutoSaveBatch, ctx context.Context) error { return b.autosave.RunOnce(ctx) }
	runBackupOnce   = func(b *BackupBatch, ctx context.Context) error { return b.backup.RunOnce(ctx) }
	runCleanupOnce  = func(b *CleanupBatch, ctx context.Context) error { return b.cleanup.RunOnce(ctx) }
)
