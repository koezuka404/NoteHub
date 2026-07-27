package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type ListVersionsInput struct {
	UserID     uuid.UUID
	DocumentID uuid.UUID
}

type VersionListItem struct {
	ID        uuid.UUID
	Type      string
	CreatedBy uuid.UUID
	CreatedAt string
}

func (uc *VersionUseCase) ListVersions(ctx context.Context, input ListVersionsInput) ([]VersionListItem, error) {
	doc, err := uc.loadDoc(ctx, input.DocumentID)
	if err != nil {
		return nil, err
	}
	if err := uc.authorizeDoc(ctx, input.UserID, doc); err != nil {
		return nil, err
	}

	versions, err := uc.versions.FindByDocumentID(ctx, input.DocumentID)
	if err != nil {
		return nil, fmt.Errorf("list document versions: %w", err)
	}

	items := make([]VersionListItem, 0, len(versions))
	for _, v := range versions {
		items = append(items, VersionListItem{
			ID:        v.ID,
			Type:      string(v.Type),
			CreatedBy: v.CreatedBy,
			CreatedAt: v.CreatedAt.Format(timeFormat),
		})
	}
	return items, nil
}
