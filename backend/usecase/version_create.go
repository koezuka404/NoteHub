package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
)

type CreateVersionInput struct {
	DocumentID  uuid.UUID
	Content     string
	CreatedBy   uuid.UUID
	VersionType entity.DocumentVersionType
}

func (uc *VersionUseCase) CreateVersion(ctx context.Context, input CreateVersionInput) error {
	doc, err := uc.loadDoc(ctx, input.DocumentID)
	if err != nil {
		return err
	}

	now := uc.currentTime()
	version, err := entity.NewDocumentVersion(*doc, input.Content, input.VersionType, input.CreatedBy, nil, now)
	if err != nil {
		return fmt.Errorf("create document version entity: %w", err)
	}
	if err := uc.versions.Create(ctx, &version); err != nil {
		return fmt.Errorf("save document version: %w", err)
	}
	return nil
}
