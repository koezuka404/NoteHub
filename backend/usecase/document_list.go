package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type ListDocumentsInput struct {
	UserID      uuid.UUID
	WorkspaceID uuid.UUID
}

type DocumentListItem struct {
	ID        uuid.UUID
	Title     string
	UpdatedBy uuid.UUID
	UpdatedAt string
}

func (uc *DocumentUseCase) ListDocuments(ctx context.Context, input ListDocumentsInput) ([]DocumentListItem, error) {
	if err := uc.requireMember(ctx, input.UserID, input.WorkspaceID); err != nil {
		return nil, err
	}

	docs, err := uc.docs.FindByWorkspaceID(ctx, input.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}

	items := make([]DocumentListItem, 0, len(docs))
	for _, doc := range docs {
		items = append(items, DocumentListItem{
			ID:        doc.ID,
			Title:     doc.Title,
			UpdatedBy: doc.UpdatedBy,
			UpdatedAt: doc.UpdatedAt.Format(timeFormat),
		})
	}
	return items, nil
}
