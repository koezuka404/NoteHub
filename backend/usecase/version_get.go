package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type GetVersionInput struct {
	UserID     uuid.UUID
	DocumentID uuid.UUID
	VersionID  uuid.UUID
}

type GetVersionOutput struct {
	ID         uuid.UUID
	DocumentID uuid.UUID
	Content    string
	Type       string
	CreatedBy  uuid.UUID
	CreatedAt  string
}

func (uc *VersionUseCase) GetVersion(ctx context.Context, input GetVersionInput) (*GetVersionOutput, error) {
	doc, err := uc.loadDoc(ctx, input.DocumentID)
	if err != nil {
		return nil, err
	}
	if err := uc.authorizeDoc(ctx, input.UserID, doc); err != nil {
		return nil, err
	}

	version, found, err := uc.versions.FindByID(ctx, input.VersionID)
	if err != nil {
		return nil, fmt.Errorf("find document version: %w", err)
	}
	if !found || version.DocumentID != input.DocumentID {
		return nil, ErrVersionNotFound
	}

	return &GetVersionOutput{
		ID:         version.ID,
		DocumentID: version.DocumentID,
		Content:    version.Content,
		Type:       string(version.Type),
		CreatedBy:  version.CreatedBy,
		CreatedAt:  version.CreatedAt.Format(timeFormat),
	}, nil
}
