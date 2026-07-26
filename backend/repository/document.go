package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DocumentRepository struct {
	db *gorm.DB
}

func NewDocumentRepository(db *gorm.DB) *DocumentRepository {
	return &DocumentRepository{db: db}
}

func (r *DocumentRepository) Create(ctx context.Context, doc *entity.Document) error {
	if err := dbFromContext(ctx, r.db).Create(doc).Error; err != nil {
		return fmt.Errorf("insert document: %w", err)
	}
	return nil
}

func (r *DocumentRepository) FindByID(ctx context.Context, documentID uuid.UUID) (*entity.Document, bool, error) {
	var doc entity.Document
	err := dbFromContext(ctx, r.db).Where("id = ?", documentID).First(&doc).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("select document by id: %w", err)
	}
	return &doc, true, nil
}

func (r *DocumentRepository) FindByIDForUpdate(ctx context.Context, documentID uuid.UUID) (*entity.Document, bool, error) {
	var doc entity.Document
	err := dbFromContext(ctx, r.db).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", documentID).
		First(&doc).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("select document for update: %w", err)
	}
	return &doc, true, nil
}

func (r *DocumentRepository) FindByWorkspaceID(ctx context.Context, workspaceID uuid.UUID) ([]entity.Document, error) {
	var docs []entity.Document
	if err := dbFromContext(ctx, r.db).
		Where("workspace_id = ? AND deleted_at IS NULL", workspaceID).
		Order("updated_at DESC").
		Find(&docs).Error; err != nil {
		return nil, fmt.Errorf("select documents by workspace id: %w", err)
	}
	return docs, nil
}

func (r *DocumentRepository) Update(ctx context.Context, doc *entity.Document) error {
	if err := dbFromContext(ctx, r.db).Save(doc).Error; err != nil {
		return fmt.Errorf("update document: %w", err)
	}
	return nil
}
