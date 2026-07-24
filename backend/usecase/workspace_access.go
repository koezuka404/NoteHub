package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
)

type CheckWorkspaceAccessInput struct {
	UserID       uuid.UUID
	WorkspaceID  uuid.UUID
	RequiredRole *entity.WorkspaceRole
}

type WorkspaceAccessResult struct {
	Workspace *entity.Workspace
	Member    *entity.WorkspaceMember
	Host      *entity.User
}

func (uc *WorkspaceUseCase) CheckWorkspaceAccess(ctx context.Context, input CheckWorkspaceAccessInput) (*WorkspaceAccessResult, error) {
	if _, err := uc.findActiveUser(ctx, input.UserID); err != nil {
		return nil, err
	}

	workspace, found, err := uc.workspaces.FindByID(ctx, input.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("find workspace: %w", err)
	}
	if !found {
		return nil, ErrWorkspaceNotFound
	}
	if workspace.IsDeleted() {
		return nil, ErrWorkspaceAlreadyDeleted
	}

	host, found, err := uc.users.FindByID(ctx, workspace.HostID)
	if err != nil {
		return nil, fmt.Errorf("find workspace host: %w", err)
	}
	if !found {
		return nil, ErrWorkspaceNotFound
	}
	if err := checkWorkspaceHostStatus(host); err != nil {
		return nil, err
	}

	member, found, err := uc.members.FindByWorkspaceAndUser(ctx, workspace.ID, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("find workspace member: %w", err)
	}
	if !found {
		return nil, ErrWorkspaceAccessDenied
	}

	if input.RequiredRole != nil && member.Role != *input.RequiredRole {
		if *input.RequiredRole == entity.WorkspaceRoleHost {
			return nil, ErrHostPermissionRequired
		}
		return nil, ErrWorkspacePermissionDenied
	}

	return &WorkspaceAccessResult{
		Workspace: workspace,
		Member:    member,
		Host:      host,
	}, nil
}
