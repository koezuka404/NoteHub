package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type IDocumentCache interface {
	GetContent(ctx context.Context, documentID uuid.UUID) (string, bool, error)
	SetContent(ctx context.Context, documentID uuid.UUID, content string, updatedBy uuid.UUID, updatedAt time.Time) error
	IsDirty(ctx context.Context, documentID uuid.UUID) (bool, error)
	MarkClean(ctx context.Context, documentID uuid.UUID) error
	Clear(ctx context.Context, documentID uuid.UUID) error
}
