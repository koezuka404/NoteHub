package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
)

type RestoreVersionInput struct {
	UserID     uuid.UUID
	DocumentID uuid.UUID
	VersionID  uuid.UUID
}

type RestoreVersionOutput struct {
	DocumentID uuid.UUID
	VersionID  uuid.UUID
	RestoredAt string
}

func (uc *VersionUseCase) RestoreVersion(ctx context.Context, input RestoreVersionInput) (*RestoreVersionOutput, error) {
	doc, err := uc.loadDoc(ctx, input.DocumentID)
	if err != nil {
		return nil, err
	}
	if err := uc.authorizeDoc(ctx, input.UserID, doc); err != nil {
		return nil, err
	}

	target, found, err := uc.versions.FindByID(ctx, input.VersionID)
	if err != nil {
		return nil, fmt.Errorf("find restore version: %w", err)
	}
	if !found || target.DocumentID != input.DocumentID {
		return nil, ErrVersionNotFound
	}

	currentContent, err := uc.currentContent(ctx, doc)
	if err != nil {
		return nil, fmt.Errorf("load current document content: %w", err)
	}

	now := uc.currentTime()
	sourceID := target.ID
	var output *RestoreVersionOutput

	if err := uc.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		locked, found, err := uc.docs.FindByIDForUpdate(txCtx, input.DocumentID)
		if err != nil {
			return fmt.Errorf("find document for restore: %w", err)
		}
		if !found {
			return ErrDocumentNotFound
		}
		if locked.IsDeleted() {
			return ErrDocumentDeleted
		}

		before, err := entity.NewDocumentVersion(*locked, currentContent, entity.DocumentVersionBeforeRestore, input.UserID, nil, now)
		if err != nil {
			return fmt.Errorf("create before_restore version: %w", err)
		}
		if err := uc.versions.Create(txCtx, &before); err != nil {
			return fmt.Errorf("save before_restore version: %w", err)
		}

		if err := locked.ReplaceContent(target.Content, input.UserID, locked.Revision, now); err != nil {
			return err
		}
		if err := uc.docs.Update(txCtx, locked); err != nil {
			return fmt.Errorf("update restored document: %w", err)
		}

		restored, err := entity.NewDocumentVersion(*locked, target.Content, entity.DocumentVersionRestore, input.UserID, &sourceID, now)
		if err != nil {
			return fmt.Errorf("create restore version: %w", err)
		}
		if err := uc.versions.Create(txCtx, &restored); err != nil {
			return fmt.Errorf("save restore version: %w", err)
		}

		output = &RestoreVersionOutput{
			DocumentID: locked.ID,
			VersionID:  restored.ID,
			RestoredAt: now.Format(timeFormat),
		}
		return nil
	}); err != nil {
		return nil, err
	}

	if uc.cache != nil {
		if err := uc.cache.SetContent(ctx, output.DocumentID, target.Content, input.UserID, now); err != nil {
			return nil, fmt.Errorf("update document cache after restore: %w", err)
		}
	}
	if uc.notifier != nil {
		if err := uc.notifier.NotifyDocumentRestored(output.DocumentID, target.Content, target.ID); err != nil {
			return nil, fmt.Errorf("notify document restored: %w", err)
		}
	}
	return output, nil
}
