package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
)

type DeleteDocumentInput struct {
	UserID     uuid.UUID
	DocumentID uuid.UUID
	IPAddress  string
}

type DeleteDocumentOutput struct {
	DocumentID uuid.UUID
	DeletedAt  string
}

func (uc *DocumentUseCase) DeleteDocument(ctx context.Context, input DeleteDocumentInput) (*DeleteDocumentOutput, error) {
	doc, err := uc.loadDoc(ctx, input.DocumentID)
	if err != nil {
		return nil, err
	}
	if err := uc.authorizeDoc(ctx, input.UserID, doc); err != nil {
		return nil, err
	}

	now := uc.currentTime()
	var output *DeleteDocumentOutput

	if err := uc.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		locked, found, err := uc.docs.FindByIDForUpdate(txCtx, input.DocumentID)
		if err != nil {
			return fmt.Errorf("find document for delete: %w", err)
		}
		if !found {
			return ErrDocumentNotFound
		}
		if locked.IsDeleted() {
			return ErrDocumentDeleted
		}
		if err := locked.LogicalDelete(input.UserID, now); err != nil {
			return err
		}
		if err := uc.docs.Update(txCtx, locked); err != nil {
			return fmt.Errorf("update deleted document: %w", err)
		}

		audit, err := entity.NewAuditLog(&input.UserID, "DOCUMENT_DELETED", "document", &locked.ID, nil, now)
		if err != nil {
			return fmt.Errorf("create audit log entity: %w", err)
		}
		audit.IPAddress = input.IPAddress
		if err := uc.auditLogs.Create(txCtx, &audit); err != nil {
			return fmt.Errorf("save audit log: %w", err)
		}

		output = &DeleteDocumentOutput{
			DocumentID: locked.ID,
			DeletedAt:  locked.DeletedAt.Format(timeFormat),
		}
		return nil
	}); err != nil {
		return nil, err
	}

	if uc.cache != nil {
		if err := uc.cache.Clear(ctx, input.DocumentID); err != nil {
			return nil, fmt.Errorf("clear document cache: %w", err)
		}
	}
	return output, nil
}
