package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
)

type GetCurrentUserInput struct {
	UserID uuid.UUID
}

type GetCurrentUserOutput struct {
	ID     uuid.UUID
	Name   string
	Email  string
	Status entity.UserStatus
}

func (uc *AuthUseCase) GetCurrentUser(ctx context.Context, input GetCurrentUserInput) (*GetCurrentUserOutput, error) {
	user, found, err := uc.users.FindByID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	if !found || !user.CanAuthenticate() {
		return nil, ErrAccountSuspended
	}
	return &GetCurrentUserOutput{
		ID:     user.ID,
		Name:   user.Name,
		Email:  user.Email,
		Status: user.Status,
	}, nil
}
