package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type DocumentContentState struct {
	Content   string
	UpdatedBy uuid.UUID
	UpdatedAt time.Time
}

type IDocumentCache interface {
	GetContentState(ctx context.Context, documentID uuid.UUID) (DocumentContentState, bool, error)
	SetContent(ctx context.Context, documentID uuid.UUID, content string, updatedBy uuid.UUID, updatedAt time.Time) error
	IsDirty(ctx context.Context, documentID uuid.UUID) (bool, error)
	MarkClean(ctx context.Context, documentID uuid.UUID) error
	GetRevision(ctx context.Context, documentID uuid.UUID) (uint64, error)
	ListDirtyDocumentIDs(ctx context.Context) ([]uuid.UUID, error)
	Clear(ctx context.Context, documentID uuid.UUID) error
}

type IAutoSaveLockStore interface {
	TryLock(ctx context.Context, key string, ttl time.Duration) (bool, error)
	Unlock(ctx context.Context, key string) error
}
