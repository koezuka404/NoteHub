package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type ListWorkspacesInput struct {
	UserID uuid.UUID
}

type WorkspaceListItem struct {
	ID                uuid.UUID
	Name              string
	HostID            uuid.UUID
	HostName          string
	Role              string
	IsAvailable       bool
	UnavailableReason string
	UpdatedAt         string
}

func (uc *WorkspaceUseCase) ListWorkspaces(ctx context.Context, input ListWorkspacesInput) ([]WorkspaceListItem, error) {
	if _, err := uc.findActiveUser(ctx, input.UserID); err != nil {
		return nil, err
	}

	workspaces, err := uc.workspaces.FindByUserID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}

	items := make([]WorkspaceListItem, 0, len(workspaces))
	for _, workspace := range workspaces {
		member, found, err := uc.members.FindByWorkspaceAndUser(ctx, workspace.ID, input.UserID)
		if err != nil {
			return nil, fmt.Errorf("find workspace member: %w", err)
		}
		if !found {
			continue
		}

		host, found, err := uc.users.FindByID(ctx, workspace.HostID)
		if err != nil {
			return nil, fmt.Errorf("find workspace host: %w", err)
		}
		hostName := ""
		isAvailable := false
		unavailableReason := unavailableHostDeleted
		if found {
			hostName = host.Name
			isAvailable, unavailableReason = workspaceAvailability(host)
		}

		items = append(items, WorkspaceListItem{
			ID:                workspace.ID,
			Name:              workspace.Name,
			HostID:            workspace.HostID,
			HostName:          hostName,
			Role:              string(member.Role),
			IsAvailable:       isAvailable,
			UnavailableReason: unavailableReason,
			UpdatedAt:         workspace.UpdatedAt.Format(timeFormat),
		})
	}
	return items, nil
}
