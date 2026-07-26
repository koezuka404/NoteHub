package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
)

type UpdateDocumentInput struct {
	UserID     uuid.UUID
	DocumentID uuid.UUID
	Title      string
	IPAddress  string
}

type UpdateDocumentOutput struct {
	ID        uuid.UUID
	Title     string
	UpdatedAt string
}

func (uc *DocumentUseCase) UpdateDocument(ctx context.Context, input UpdateDocumentInput) (*UpdateDocumentOutput, error) {
	if err := validTitle(input.Title); err != nil {
		return nil, err
	}

	doc, err := uc.loadDoc(ctx, input.DocumentID)
	if err != nil {
		return nil, err
	}
	if err := uc.authorizeDoc(ctx, input.UserID, doc); err != nil {
		return nil, err
	}

	now := uc.currentTime()
	var output *UpdateDocumentOutput

	if err := uc.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		locked, found, err := uc.docs.FindByIDForUpdate(txCtx, input.DocumentID)
		if err != nil {
			return fmt.Errorf("find document for update: %w", err)
		}
		if !found {
			return ErrDocumentNotFound
		}
		if locked.IsDeleted() {
			return ErrDocumentDeleted
		}
		if err := locked.Rename(normalizeTitle(input.Title), input.UserID, now); err != nil {
			return err
		}
		if err := uc.docs.Update(txCtx, locked); err != nil {
			return fmt.Errorf("update document: %w", err)
		}

		audit, err := entity.NewAuditLog(&input.UserID, "DOCUMENT_UPDATED", "document", &locked.ID, nil, now)
		if err != nil {
			return fmt.Errorf("create audit log entity: %w", err)
		}
		audit.IPAddress = input.IPAddress
		if err := uc.auditLogs.Create(txCtx, &audit); err != nil {
			return fmt.Errorf("save audit log: %w", err)
		}

		output = &UpdateDocumentOutput{
			ID:        locked.ID,
			Title:     locked.Title,
			UpdatedAt: locked.UpdatedAt.Format(timeFormat),
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return output, nil
}
