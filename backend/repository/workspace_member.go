package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
	"gorm.io/gorm"
)

type WorkspaceMemberRepository struct {
	db *gorm.DB
}

func NewWorkspaceMemberRepository(db *gorm.DB) *WorkspaceMemberRepository {
	return &WorkspaceMemberRepository{db: db}
}

func (r *WorkspaceMemberRepository) Create(ctx context.Context, member *entity.WorkspaceMember) error {
	if err := dbFromContext(ctx, r.db).Create(member).Error; err != nil {
		return fmt.Errorf("insert workspace member: %w", err)
	}
	return nil
}

func (r *WorkspaceMemberRepository) FindByWorkspaceAndUser(ctx context.Context, workspaceID, userID uuid.UUID) (*entity.WorkspaceMember, bool, error) {
	var member entity.WorkspaceMember
	err := dbFromContext(ctx, r.db).
		Where("workspace_id = ? AND user_id = ?", workspaceID, userID).
		First(&member).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("select workspace member: %w", err)
	}
	return &member, true, nil
}

func (r *WorkspaceMemberRepository) FindByWorkspaceID(ctx context.Context, workspaceID uuid.UUID) ([]entity.WorkspaceMember, error) {
	var members []entity.WorkspaceMember
	if err := dbFromContext(ctx, r.db).
		Where("workspace_id = ?", workspaceID).
		Order("CASE WHEN role = 'host' THEN 0 ELSE 1 END, joined_at ASC").
		Find(&members).Error; err != nil {
		return nil, fmt.Errorf("select workspace members: %w", err)
	}
	return members, nil
}

func (r *WorkspaceMemberRepository) Exists(ctx context.Context, workspaceID, userID uuid.UUID) (bool, error) {
	var count int64
	if err := dbFromContext(ctx, r.db).
		Model(&entity.WorkspaceMember{}).
		Where("workspace_id = ? AND user_id = ?", workspaceID, userID).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("count workspace member: %w", err)
	}
	return count > 0, nil
}

func (r *WorkspaceMemberRepository) Delete(ctx context.Context, workspaceID, userID uuid.UUID) error {
	result := dbFromContext(ctx, r.db).
		Where("workspace_id = ? AND user_id = ?", workspaceID, userID).
		Delete(&entity.WorkspaceMember{})
	if result.Error != nil {
		return fmt.Errorf("delete workspace member: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
