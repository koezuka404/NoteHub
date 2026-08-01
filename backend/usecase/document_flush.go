package usecase

import (
	"context"

	"github.com/google/uuid"
)

type IDocumentFlushService interface {
	FlushDocument(ctx context.Context, documentID uuid.UUID) error
	FlushAllDirty(ctx context.Context) error
	FlushWorkspaceDocuments(ctx context.Context, workspaceID uuid.UUID) error
}
