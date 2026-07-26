package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type GetDocumentInput struct {
	UserID     uuid.UUID
	DocumentID uuid.UUID
}

type GetDocumentOutput struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Title       string
	Content     string
	UpdatedBy   uuid.UUID
	UpdatedAt   string
}

func (uc *DocumentUseCase) GetDocument(ctx context.Context, input GetDocumentInput) (*GetDocumentOutput, error) {
	doc, err := uc.loadDoc(ctx, input.DocumentID)
	if err != nil {
		return nil, err
	}
	if err := uc.authorizeDoc(ctx, input.UserID, doc); err != nil {
		return nil, err
	}

	content, err := uc.docContent(ctx, doc)
	if err != nil {
		return nil, fmt.Errorf("resolve document content: %w", err)
	}

	return &GetDocumentOutput{
		ID:          doc.ID,
		WorkspaceID: doc.WorkspaceID,
		Title:       doc.Title,
		Content:     content,
		UpdatedBy:   doc.UpdatedBy,
		UpdatedAt:   doc.UpdatedAt.Format(timeFormat),
	}, nil
}
