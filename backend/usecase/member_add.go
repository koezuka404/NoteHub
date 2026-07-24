package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
	"gorm.io/gorm"
)

type AddMemberInput struct {
	UserID       uuid.UUID
	WorkspaceID  uuid.UUID
	TargetUserID uuid.UUID
	IPAddress    string
}

type AddMemberOutput struct {
	UserID   uuid.UUID
	Name     string
	Email    string
	Role     string
	JoinedAt string
}

func (uc *MemberUseCase) AddMember(ctx context.Context, input AddMemberInput) (*AddMemberOutput, error) {
	if err := uc.requireHost(ctx, input.UserID, input.WorkspaceID); err != nil {
		return nil, err
	}

	if input.TargetUserID == input.UserID {
		return nil, ErrCannotAddSelf
	}

	target, found, err := uc.users.FindByID(ctx, input.TargetUserID)
	if err != nil {
		return nil, fmt.Errorf("find target user: %w", err)
	}
	if !found {
		return nil, ErrTargetUserNotFound
	}
	if !target.CanAuthenticate() {
		return nil, ErrTargetAccountUnavailable
	}

	exists, err := uc.members.Exists(ctx, input.WorkspaceID, input.TargetUserID)
	if err != nil {
		return nil, fmt.Errorf("check workspace member: %w", err)
	}
	if exists {
		return nil, ErrMemberAlreadyExists
	}

	now := uc.currentTime()
	member, err := entity.NewWorkspaceMember(input.WorkspaceID, input.TargetUserID, entity.WorkspaceRoleMember, now)
	if err != nil {
		return nil, fmt.Errorf("create workspace member entity: %w", err)
	}

	var output *AddMemberOutput
	if err := uc.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.members.Create(txCtx, &member); err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return ErrMemberAlreadyExists
			}
			return fmt.Errorf("save workspace member: %w", err)
		}

		audit, err := entity.NewAuditLog(&input.UserID, "MEMBER_ADDED", "workspace_member", &member.ID, nil, now)
		if err != nil {
			return fmt.Errorf("create audit log entity: %w", err)
		}
		audit.IPAddress = input.IPAddress
		if err := uc.auditLogs.Create(txCtx, &audit); err != nil {
			return fmt.Errorf("save audit log: %w", err)
		}

		output = &AddMemberOutput{
			UserID:   target.ID,
			Name:     target.Name,
			Email:    target.Email,
			Role:     string(entity.WorkspaceRoleMember),
			JoinedAt: member.JoinedAt.Format(timeFormat),
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return output, nil
}
