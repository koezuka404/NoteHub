package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
)

type RegisterInput struct {
	Name     string
	Email    string
	Password string
}

type RegisterOutput struct {
	ID        uuid.UUID
	Name      string
	Email     string
	Status    entity.UserStatus
	CreatedAt string
}

func (uc *AuthUseCase) Register(ctx context.Context, input RegisterInput) (*RegisterOutput, error) {
	if !validateRegisterInput(input) {
		return nil, ErrValidation
	}

	email := normalizeEmail(input.Email)
	exists, err := uc.users.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("check email existence: %w", err)
	}
	if exists {
		return nil, ErrEmailAlreadyExists
	}

	passwordHash, err := uc.passwords.Hash(input.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	now := uc.now().UTC()
	user, err := entity.NewUser(strings.TrimSpace(input.Name), email, passwordHash, now)
	if err != nil {
		return nil, fmt.Errorf("create user entity: %w", err)
	}

	if err := uc.users.Create(ctx, &user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return &RegisterOutput{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Status:    user.Status,
		CreatedAt: user.CreatedAt.Format(timeFormat),
	}, nil
}
