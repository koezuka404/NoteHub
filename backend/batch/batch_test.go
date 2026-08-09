package batch

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
	"github.com/koezuka404/notehub/usecase"
)

type mockFlushService struct {
	flushAllErr error
	flushDocErr error
	flushWSErr  error
}

func (m *mockFlushService) FlushAllDirty(context.Context) error {
	return m.flushAllErr
}

func (m *mockFlushService) FlushDocument(context.Context, uuid.UUID) error {
	return m.flushDocErr
}

func (m *mockFlushService) FlushWorkspaceDocuments(context.Context, uuid.UUID) error {
	return m.flushWSErr
}

type noopDocumentCache struct{}

func (noopDocumentCache) GetContentState(context.Context, uuid.UUID) (usecase.DocumentContentState, bool, error) {
	return usecase.DocumentContentState{}, false, nil
}
func (noopDocumentCache) SetContent(context.Context, uuid.UUID, string, uuid.UUID, time.Time) error {
	return nil
}
func (noopDocumentCache) IsDirty(context.Context, uuid.UUID) (bool, error) { return false, nil }
func (noopDocumentCache) MarkClean(context.Context, uuid.UUID) error       { return nil }
func (noopDocumentCache) GetRevision(context.Context, uuid.UUID) (uint64, error) {
	return 0, nil
}
func (noopDocumentCache) ListDirtyDocumentIDs(context.Context) ([]uuid.UUID, error) {
	return nil, nil
}
func (noopDocumentCache) ListIdleDirtyDocumentIDs(context.Context, time.Duration) ([]uuid.UUID, error) {
	return nil, nil
}
func (noopDocumentCache) Clear(context.Context, uuid.UUID) error { return nil }

type stubRefreshTokenRepo struct{}

func (stubRefreshTokenRepo) Create(context.Context, *entity.RefreshToken) error { return nil }
func (stubRefreshTokenRepo) FindByHashForUpdate(context.Context, string) (*entity.RefreshToken, bool, error) {
	return nil, false, nil
}
func (stubRefreshTokenRepo) Update(context.Context, *entity.RefreshToken) error { return nil }
func (stubRefreshTokenRepo) RevokeFamily(context.Context, uuid.UUID, time.Time) error {
	return nil
}
func (stubRefreshTokenRepo) RevokeAllByUserID(context.Context, uuid.UUID, time.Time) error {
	return nil
}
func (stubRefreshTokenRepo) MarkExpiredBefore(context.Context, time.Time) (int64, error) {
	return 0, nil
}
func (stubRefreshTokenRepo) DeleteStaleBefore(context.Context, time.Time) (int64, error) {
	return 0, nil
}

type stubCleanupRedis struct{}

func (stubCleanupRedis) CleanupEphemeralKeys(context.Context) (int, error) { return 0, nil }

func TestDefaultRunHooks(t *testing.T) {
	autoSave := NewAutoSaveBatch(
		usecase.NewDocumentAutoSaveUseCase(nil, nil, noopDocumentCache{}, nil, nil, 0, 0),
		time.Second,
	)
	if err := runAutoSaveOnce(autoSave, context.Background()); err != nil {
		t.Fatalf("runAutoSaveOnce: %v", err)
	}

	backup := NewBackupBatch(usecase.NewDatabaseBackupUseCase("", "", 0, nil), time.Second)
	if err := runBackupOnce(backup, context.Background()); err == nil {
		t.Fatal("expected backup hook error for empty database url")
	}

	cleanup := NewCleanupBatch(
		usecase.NewCleanupUseCase(stubRefreshTokenRepo{}, stubCleanupRedis{}, time.Hour),
		time.Second,
	)
	if err := runCleanupOnce(cleanup, context.Background()); err != nil {
		t.Fatalf("runCleanupOnce: %v", err)
	}
}

func TestNewAutoSaveBatch_DefaultInterval(t *testing.T) {
	batch := NewAutoSaveBatch(&usecase.DocumentAutoSaveUseCase{}, 0)
	if batch.interval != 5*time.Second {
		t.Fatalf("interval = %v", batch.interval)
	}
}

func TestNewBackupBatch_DefaultInterval(t *testing.T) {
	batch := NewBackupBatch(&usecase.DatabaseBackupUseCase{}, 0)
	if batch.interval != 24*time.Hour {
		t.Fatalf("interval = %v", batch.interval)
	}
}

func TestNewCleanupBatch_DefaultInterval(t *testing.T) {
	batch := NewCleanupBatch(&usecase.CleanupUseCase{}, 0)
	if batch.interval != time.Hour {
		t.Fatalf("interval = %v", batch.interval)
	}
}

func TestAutoSaveBatch_Run(t *testing.T) {
	orig := runAutoSaveOnce
	t.Cleanup(func() { runAutoSaveOnce = orig })

	calls := 0
	runAutoSaveOnce = func(*AutoSaveBatch, context.Context) error {
		calls++
		if calls == 1 {
			return errors.New("autosave failed")
		}
		return nil
	}

	batch := NewAutoSaveBatch(&usecase.DocumentAutoSaveUseCase{}, time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		batch.Run(ctx)
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not stop after cancel")
	}
	if calls == 0 {
		t.Fatal("expected at least one autosave tick")
	}
}

func TestBackupBatch_Run(t *testing.T) {
	orig := runBackupOnce
	t.Cleanup(func() { runBackupOnce = orig })

	calls := 0
	runBackupOnce = func(*BackupBatch, context.Context) error {
		calls++
		if calls == 1 {
			return errors.New("backup failed")
		}
		return nil
	}

	batch := NewBackupBatch(&usecase.DatabaseBackupUseCase{}, time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		batch.Run(ctx)
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not stop after cancel")
	}
	if calls == 0 {
		t.Fatal("expected at least one backup tick")
	}
}

func TestCleanupBatch_Run(t *testing.T) {
	orig := runCleanupOnce
	t.Cleanup(func() { runCleanupOnce = orig })

	calls := 0
	runCleanupOnce = func(*CleanupBatch, context.Context) error {
		calls++
		if calls == 1 {
			return errors.New("cleanup failed")
		}
		return nil
	}

	batch := NewCleanupBatch(&usecase.CleanupUseCase{}, time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		batch.Run(ctx)
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not stop after cancel")
	}
	if calls == 0 {
		t.Fatal("expected at least one cleanup tick")
	}
}

func TestNewDocumentFlushBatch(t *testing.T) {
	flush := &mockFlushService{}
	batch := NewDocumentFlushBatch(flush)
	if batch == nil || batch.flush == nil {
		t.Fatal("expected document flush batch")
	}
}

func TestDocumentFlushBatch_FlushAllDirty(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		batch := NewDocumentFlushBatch(&mockFlushService{})
		if err := batch.FlushAllDirty(ctx); err != nil {
			t.Fatalf("FlushAllDirty: %v", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		want := errors.New("flush all failed")
		batch := NewDocumentFlushBatch(&mockFlushService{flushAllErr: want})
		if err := batch.FlushAllDirty(ctx); !errors.Is(err, want) {
			t.Fatalf("FlushAllDirty() = %v", err)
		}
	})
}

func TestDocumentFlushBatch_FlushDocument(t *testing.T) {
	ctx := context.Background()
	docID := uuid.New()

	t.Run("success", func(t *testing.T) {
		batch := NewDocumentFlushBatch(&mockFlushService{})
		if err := batch.FlushDocument(ctx, docID); err != nil {
			t.Fatalf("FlushDocument: %v", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		want := errors.New("flush document failed")
		batch := NewDocumentFlushBatch(&mockFlushService{flushDocErr: want})
		if err := batch.FlushDocument(ctx, docID); !errors.Is(err, want) {
			t.Fatalf("FlushDocument() = %v", err)
		}
	})
}

func TestDocumentFlushBatch_FlushWorkspaceDocuments(t *testing.T) {
	ctx := context.Background()
	wsID := uuid.New()

	t.Run("success", func(t *testing.T) {
		batch := NewDocumentFlushBatch(&mockFlushService{})
		if err := batch.FlushWorkspaceDocuments(ctx, wsID); err != nil {
			t.Fatalf("FlushWorkspaceDocuments: %v", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		want := errors.New("flush workspace failed")
		batch := NewDocumentFlushBatch(&mockFlushService{flushWSErr: want})
		if err := batch.FlushWorkspaceDocuments(ctx, wsID); !errors.Is(err, want) {
			t.Fatalf("FlushWorkspaceDocuments() = %v", err)
		}
	})
}
