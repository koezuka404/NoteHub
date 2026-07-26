package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
)

type CreateDocumentInput struct {
	UserID      uuid.UUID
	WorkspaceID uuid.UUID
	Title       string
	IPAddress   string
}

type CreateDocumentOutput struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Title       string
	Content     string
	CreatedBy   uuid.UUID
	CreatedAt   string
}

func (uc *DocumentUseCase) CreateDocument(ctx context.Context, input CreateDocumentInput) (*CreateDocumentOutput, error) {
	if err := validTitle(input.Title); err != nil {
		return nil, err
	}
	if err := uc.requireMember(ctx, input.UserID, input.WorkspaceID); err != nil {
		return nil, err
	}

	now := uc.currentTime()
	doc, err := entity.NewDocument(input.WorkspaceID, input.UserID, normalizeTitle(input.Title), "", now)
	if err != nil {
		return nil, fmt.Errorf("create document entity: %w", err)
	}

	if err := uc.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.docs.Create(txCtx, &doc); err != nil {
			return fmt.Errorf("save document: %w", err)
		}
		audit, err := entity.NewAuditLog(&input.UserID, "DOCUMENT_CREATED", "document", &doc.ID, nil, now)
		if err != nil {
			return fmt.Errorf("create audit log entity: %w", err)
		}
		audit.IPAddress = input.IPAddress
		if err := uc.auditLogs.Create(txCtx, &audit); err != nil {
			return fmt.Errorf("save audit log: %w", err)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return &CreateDocumentOutput{
		ID:          doc.ID,
		WorkspaceID: doc.WorkspaceID,
		Title:       doc.Title,
		Content:     doc.Content,
		CreatedBy:   doc.CreatedBy,
		CreatedAt:   doc.CreatedAt.Format(timeFormat),
	}, nil
}
