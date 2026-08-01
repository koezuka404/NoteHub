package batch

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/usecase"
)

type DocumentFlushBatch struct {
	flush usecase.IDocumentFlushService
}

func NewDocumentFlushBatch(flush usecase.IDocumentFlushService) *DocumentFlushBatch {
	return &DocumentFlushBatch{flush: flush}
}

func (b *DocumentFlushBatch) FlushAllDirty(ctx context.Context) error {
	if err := b.flush.FlushAllDirty(ctx); err != nil {
		return err
	}
	return nil
}

func (b *DocumentFlushBatch) FlushDocument(ctx context.Context, documentID uuid.UUID) error {
	if err := b.flush.FlushDocument(ctx, documentID); err != nil {
		log.Printf("flush document %s: %v", documentID, err)
		return err
	}
	return nil
}

func (b *DocumentFlushBatch) FlushWorkspaceDocuments(ctx context.Context, workspaceID uuid.UUID) error {
	if err := b.flush.FlushWorkspaceDocuments(ctx, workspaceID); err != nil {
		return err
	}
	return nil
}
