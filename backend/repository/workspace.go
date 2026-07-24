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

type WorkspaceRepository struct {
	db *gorm.DB
}

func NewWorkspaceRepository(db *gorm.DB) *WorkspaceRepository {
	return &WorkspaceRepository{db: db}
}

func (r *WorkspaceRepository) Create(ctx context.Context, workspace *entity.Workspace) error {
	if err := dbFromContext(ctx, r.db).Create(workspace).Error; err != nil {
		return fmt.Errorf("insert workspace: %w", err)
	}
	return nil
}

func (r *WorkspaceRepository) FindByID(ctx context.Context, workspaceID uuid.UUID) (*entity.Workspace, bool, error) {
	var workspace entity.Workspace
	err := dbFromContext(ctx, r.db).Where("id = ?", workspaceID).First(&workspace).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("select workspace by id: %w", err)
	}
	return &workspace, true, nil
}

func (r *WorkspaceRepository) FindByIDForUpdate(ctx context.Context, workspaceID uuid.UUID) (*entity.Workspace, bool, error) {
	var workspace entity.Workspace
	err := dbFromContext(ctx, r.db).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", workspaceID).
		First(&workspace).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("select workspace for update: %w", err)
	}
	return &workspace, true, nil
}

func (r *WorkspaceRepository) FindByHostID(ctx context.Context, hostID uuid.UUID) ([]entity.Workspace, error) {
	var workspaces []entity.Workspace
	if err := dbFromContext(ctx, r.db).
		Where("host_id = ? AND deleted_at IS NULL", hostID).
		Order("updated_at DESC").
		Find(&workspaces).Error; err != nil {
		return nil, fmt.Errorf("select workspaces by host id: %w", err)
	}
	return workspaces, nil
}

func (r *WorkspaceRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]entity.Workspace, error) {
	var workspaces []entity.Workspace
	if err := dbFromContext(ctx, r.db).
		Joins("INNER JOIN workspace_members ON workspace_members.workspace_id = workspaces.id").
		Where("workspace_members.user_id = ? AND workspaces.deleted_at IS NULL", userID).
		Order("workspaces.updated_at DESC").
		Find(&workspaces).Error; err != nil {
		return nil, fmt.Errorf("select workspaces by user id: %w", err)
	}
	return workspaces, nil
}

func (r *WorkspaceRepository) Update(ctx context.Context, workspace *entity.Workspace) error {
	if err := dbFromContext(ctx, r.db).Save(workspace).Error; err != nil {
		return fmt.Errorf("update workspace: %w", err)
	}
	return nil
}
