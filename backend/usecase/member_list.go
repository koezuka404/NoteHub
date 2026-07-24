package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
)

type ListMembersInput struct {
	UserID      uuid.UUID
	WorkspaceID uuid.UUID
}

type MemberListItem struct {
	UserID   uuid.UUID
	Name     string
	Email    string
	Status   string
	Role     string
	JoinedAt string
}

func (uc *MemberUseCase) ListMembers(ctx context.Context, input ListMembersInput) ([]MemberListItem, error) {
	if err := uc.requireMember(ctx, input.UserID, input.WorkspaceID); err != nil {
		return nil, err
	}

	members, err := uc.members.FindByWorkspaceID(ctx, input.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("list workspace members: %w", err)
	}

	items := make([]MemberListItem, 0, len(members))
	for _, member := range members {
		user, found, err := uc.users.FindByID(ctx, member.UserID)
		if err != nil {
			return nil, fmt.Errorf("find member user: %w", err)
		}
		name, email, status := deletedName, "", string(entity.UserStatusDeleted)
		if found {
			name, email, status = listUser(user)
		}
		items = append(items, MemberListItem{
			UserID:   member.UserID,
			Name:     name,
			Email:    email,
			Status:   status,
			Role:     string(member.Role),
			JoinedAt: member.JoinedAt.Format(timeFormat),
		})
	}
	return items, nil
}
