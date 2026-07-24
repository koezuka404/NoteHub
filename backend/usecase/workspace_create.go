package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
)

type CreateWorkspaceInput struct {
	UserID    uuid.UUID
	Name      string
	IPAddress string
}

type CreateWorkspaceOutput struct {
	ID        uuid.UUID
	Name      string
	HostID    uuid.UUID
	CreatedAt string
}

func (uc *WorkspaceUseCase) CreateWorkspace(ctx context.Context, input CreateWorkspaceInput) (*CreateWorkspaceOutput, error) {
	if err := validateWorkspaceName(input.Name); err != nil {
		return nil, err
	}
	user, err := uc.findActiveUser(ctx, input.UserID)
	if err != nil {
		return nil, err
	}

	now := uc.currentTime()
	workspace, err := entity.NewWorkspace(user.ID, normalizeWorkspaceName(input.Name), now)
	if err != nil {
		return nil, fmt.Errorf("create workspace entity: %w", err)
	}
	member, err := entity.NewWorkspaceMember(workspace.ID, user.ID, entity.WorkspaceRoleHost, now)
	if err != nil {
		return nil, fmt.Errorf("create workspace host member: %w", err)
	}

	if err := uc.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.workspaces.Create(txCtx, &workspace); err != nil {
			return fmt.Errorf("save workspace: %w", err)
		}
		if err := uc.members.Create(txCtx, &member); err != nil {
			return fmt.Errorf("save workspace host: %w", err)
		}
		audit, err := entity.NewAuditLog(&input.UserID, "WORKSPACE_CREATED", "workspace", &workspace.ID, nil, now)
		if err != nil {
			return fmt.Errorf("create audit log entity: %w", err)
		}
		audit.IPAddress = input.IPAddress
		if err := uc.auditLogs.Create(txCtx, &audit); err != nil {
			return fmt.Errorf("save audit log: %w", err)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return &CreateWorkspaceOutput{
		ID:        workspace.ID,
		Name:      workspace.Name,
		HostID:    workspace.HostID,
		CreatedAt: workspace.CreatedAt.Format(timeFormat),
	}, nil
}
