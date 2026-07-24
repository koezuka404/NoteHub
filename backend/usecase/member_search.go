package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
)

type SearchUserInput struct {
	UserID      uuid.UUID
	WorkspaceID uuid.UUID
	Email       string
}

type SearchUserOutput struct {
	ID     uuid.UUID
	Email  string
	Name   string
	Status entity.UserStatus
}

func (uc *MemberUseCase) SearchUser(ctx context.Context, input SearchUserInput) (*SearchUserOutput, error) {
	if err := uc.requireHost(ctx, input.UserID, input.WorkspaceID); err != nil {
		return nil, err
	}

	email, err := validEmail(input.Email)
	if err != nil {
		return nil, err
	}

	user, found, err := uc.users.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if !found || !user.CanAuthenticate() {
		return nil, ErrUserNotFound
	}
	if user.ID == input.UserID {
		return nil, ErrUserNotFound
	}

	exists, err := uc.members.Exists(ctx, input.WorkspaceID, user.ID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUserNotFound
	}

	return &SearchUserOutput{
		ID:     user.ID,
		Email:  user.Email,
		Name:   user.Name,
		Status: user.Status,
	}, nil
}
