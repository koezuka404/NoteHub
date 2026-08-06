package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/koezuka404/notehub/entity"
	"gorm.io/gorm"
)

type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.User, bool, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, bool, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	IncrementAuthVersion(ctx context.Context, userID uuid.UUID, now time.Time) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *entity.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*entity.User, bool, error) {
	var user entity.User
	err := r.db.WithContext(ctx).
		Where("LOWER(email) = LOWER(?)", email).
		First(&user).
		Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("select user by email: %w", err)
	}
	return &user, true, nil
}

func (r *userRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&entity.User{}).
		Where("LOWER(email) = LOWER(?) AND deleted_at IS NULL", email).
		Count(&count).
		Error; err != nil {
		return false, fmt.Errorf("count user by email: %w", err)
	}
	return count > 0, nil
}

func (r *userRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.User, bool, error) {
	var user entity.User
	err := dbFromContext(ctx, r.db).Where("id = ?", id).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("select user by id: %w", err)
	}
	return &user, true, nil
}

func (r *userRepository) IncrementAuthVersion(ctx context.Context, userID uuid.UUID, now time.Time) error {
	result := dbFromContext(ctx, r.db).Model(&entity.User{}).Where("id = ?", userID).Updates(map[string]any{
		"auth_version": gorm.Expr("auth_version + 1"), "updated_at": now,
	})
	if result.Error != nil {
		return fmt.Errorf("increment auth version: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
