package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
	"gorm.io/gorm"
)

type DocumentVersionRepository struct {
	db *gorm.DB
}

func NewDocumentVersionRepository(db *gorm.DB) *DocumentVersionRepository {
	return &DocumentVersionRepository{db: db}
}

func (r *DocumentVersionRepository) Create(ctx context.Context, version *entity.DocumentVersion) error {
	if err := dbFromContext(ctx, r.db).Create(version).Error; err != nil {
		return fmt.Errorf("insert document version: %w", err)
	}
	return nil
}

func (r *DocumentVersionRepository) FindByID(ctx context.Context, versionID uuid.UUID) (*entity.DocumentVersion, bool, error) {
	var version entity.DocumentVersion
	err := dbFromContext(ctx, r.db).Where("id = ?", versionID).First(&version).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("select document version by id: %w", err)
	}
	return &version, true, nil
}

func (r *DocumentVersionRepository) FindByDocumentID(ctx context.Context, documentID uuid.UUID) ([]entity.DocumentVersion, error) {
	var versions []entity.DocumentVersion
	if err := dbFromContext(ctx, r.db).
		Where("document_id = ?", documentID).
		Order("created_at DESC").
		Find(&versions).Error; err != nil {
		return nil, fmt.Errorf("select document versions by document id: %w", err)
	}
	return versions, nil
}
