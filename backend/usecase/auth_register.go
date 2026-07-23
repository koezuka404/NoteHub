package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
)

type RegisterInput struct {
	Name      string
	Email     string
	Password  string
	IPAddress string
}

type RegisterOutput struct {
	ID        uuid.UUID
	Name      string
	Email     string
	Status    entity.UserStatus
	CreatedAt string
}

func (uc *AuthUseCase) Register(ctx context.Context, input RegisterInput) (*RegisterOutput, error) {
	if err := validateRegisterInput(input); err != nil {
		return nil, err
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

	actorID := user.ID
	auditLog, err := entity.NewAuditLog(&actorID, "USER_REGISTERED", "user", &user.ID, nil, now)
	if err != nil {
		return nil, fmt.Errorf("create audit log entity: %w", err)
	}
	auditLog.IPAddress = input.IPAddress
	if err := uc.auditLogs.Create(ctx, &auditLog); err != nil {
		return nil, fmt.Errorf("create audit log: %w", err)
	}

	return &RegisterOutput{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Status:    user.Status,
		CreatedAt: user.CreatedAt.Format(timeFormat),
	}, nil
}
