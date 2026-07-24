package usecase

import (
	"context"

	"github.com/google/uuid"
)

type GetWorkspaceInput struct {
	UserID      uuid.UUID
	WorkspaceID uuid.UUID
}

type WorkspaceHostOutput struct {
	ID     uuid.UUID
	Name   string
	Status string
}

type GetWorkspaceOutput struct {
	ID        uuid.UUID
	Name      string
	Host      WorkspaceHostOutput
	Role      string
	CreatedAt string
	UpdatedAt string
}

func (uc *WorkspaceUseCase) GetWorkspace(ctx context.Context, input GetWorkspaceInput) (*GetWorkspaceOutput, error) {
	access, err := uc.CheckWorkspaceAccess(ctx, CheckWorkspaceAccessInput{
		UserID:      input.UserID,
		WorkspaceID: input.WorkspaceID,
	})
	if err != nil {
		return nil, err
	}

	return &GetWorkspaceOutput{
		ID:   access.Workspace.ID,
		Name: access.Workspace.Name,
		Host: WorkspaceHostOutput{
			ID:     access.Host.ID,
			Name:   access.Host.Name,
			Status: string(access.Host.Status),
		},
		Role:      string(access.Member.Role),
		CreatedAt: access.Workspace.CreatedAt.Format(timeFormat),
		UpdatedAt: access.Workspace.UpdatedAt.Format(timeFormat),
	}, nil
}
