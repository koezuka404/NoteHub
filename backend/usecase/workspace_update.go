package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
)

type UpdateWorkspaceInput struct {
	UserID      uuid.UUID
	WorkspaceID uuid.UUID
	Name        string
	IPAddress   string
}

type UpdateWorkspaceOutput struct {
	ID        uuid.UUID
	Name      string
	UpdatedAt string
}

func (uc *WorkspaceUseCase) UpdateWorkspace(ctx context.Context, input UpdateWorkspaceInput) (*UpdateWorkspaceOutput, error) {
	if err := validateWorkspaceName(input.Name); err != nil {
		return nil, err
	}

	hostRole := entity.WorkspaceRoleHost
	if _, err := uc.CheckWorkspaceAccess(ctx, CheckWorkspaceAccessInput{
		UserID:       input.UserID,
		WorkspaceID:  input.WorkspaceID,
		RequiredRole: &hostRole,
	}); err != nil {
		return nil, err
	}

	now := uc.currentTime()
	var output *UpdateWorkspaceOutput

	if err := uc.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		workspace, found, err := uc.workspaces.FindByIDForUpdate(txCtx, input.WorkspaceID)
		if err != nil {
			return fmt.Errorf("find workspace for update: %w", err)
		}
		if !found {
			return ErrWorkspaceNotFound
		}
		if workspace.IsDeleted() {
			return ErrWorkspaceAlreadyDeleted
		}
		if err := workspace.Rename(normalizeWorkspaceName(input.Name), now); err != nil {
			return err
		}
		if err := uc.workspaces.Update(txCtx, workspace); err != nil {
			return fmt.Errorf("update workspace: %w", err)
		}

		audit, err := entity.NewAuditLog(&input.UserID, "WORKSPACE_UPDATED", "workspace", &workspace.ID, nil, now)
		if err != nil {
			return fmt.Errorf("create audit log entity: %w", err)
		}
		audit.IPAddress = input.IPAddress
		if err := uc.auditLogs.Create(txCtx, &audit); err != nil {
			return fmt.Errorf("save audit log: %w", err)
		}

		output = &UpdateWorkspaceOutput{
			ID:        workspace.ID,
			Name:      workspace.Name,
			UpdatedAt: workspace.UpdatedAt.Format(timeFormat),
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return output, nil
}
