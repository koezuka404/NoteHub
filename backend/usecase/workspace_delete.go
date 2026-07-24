package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
)

type DeleteWorkspaceInput struct {
	UserID      uuid.UUID
	WorkspaceID uuid.UUID
	Reason      string
	IPAddress   string
}

type DeleteWorkspaceOutput struct {
	WorkspaceID uuid.UUID
	DeletedAt   string
}

func (uc *WorkspaceUseCase) DeleteWorkspace(ctx context.Context, input DeleteWorkspaceInput) (*DeleteWorkspaceOutput, error) {
	if err := validateDeleteReason(input.Reason); err != nil {
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
	reason := strings.TrimSpace(input.Reason)
	var output *DeleteWorkspaceOutput

	if err := uc.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		workspace, found, err := uc.workspaces.FindByIDForUpdate(txCtx, input.WorkspaceID)
		if err != nil {
			return fmt.Errorf("find workspace for delete: %w", err)
		}
		if !found {
			return ErrWorkspaceNotFound
		}
		if workspace.IsDeleted() {
			return ErrWorkspaceAlreadyDeleted
		}
		if err := workspace.LogicalDelete(input.UserID, reason, now); err != nil {
			return err
		}
		if err := uc.workspaces.Update(txCtx, workspace); err != nil {
			return fmt.Errorf("update deleted workspace: %w", err)
		}

		metadata, err := workspaceDeleteMetadata(reason)
		if err != nil {
			return err
		}
		audit, err := entity.NewAuditLog(&input.UserID, "WORKSPACE_DELETED", "workspace", &workspace.ID, metadata, now)
		if err != nil {
			return fmt.Errorf("create audit log entity: %w", err)
		}
		audit.IPAddress = input.IPAddress
		if err := uc.auditLogs.Create(txCtx, &audit); err != nil {
			return fmt.Errorf("save audit log: %w", err)
		}

		output = &DeleteWorkspaceOutput{
			WorkspaceID: workspace.ID,
			DeletedAt:   workspace.DeletedAt.Format(timeFormat),
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return output, nil
}
