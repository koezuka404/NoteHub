package entity

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type WorkspaceMember struct {
	ID          uuid.UUID     `gorm:"type:uuid;primaryKey"`
	WorkspaceID uuid.UUID     `gorm:"type:uuid;not null;uniqueIndex:uq_workspace_members_workspace_user"`
	UserID      uuid.UUID     `gorm:"type:uuid;not null;uniqueIndex:uq_workspace_members_workspace_user;index"`
	Role        WorkspaceRole `gorm:"size:20;not null;index"`
	CreatedAt   time.Time     `gorm:"not null"`
	UpdatedAt   time.Time     `gorm:"not null"`
}

func (WorkspaceMember) TableName() string { return "workspace_members" }

func NewWorkspaceMember(workspaceID, userID uuid.UUID, role WorkspaceRole, now time.Time) (WorkspaceMember, error) {
	if workspaceID == uuid.Nil || userID == uuid.Nil {
		return WorkspaceMember{}, fmt.Errorf("workspace id and user id are required")
	}
	if !role.IsValid() {
		return WorkspaceMember{}, ErrInvalidRole
	}
	return WorkspaceMember{ID: uuid.New(), WorkspaceID: workspaceID, UserID: userID, Role: role, CreatedAt: now, UpdatedAt: now}, nil
}

func (m WorkspaceMember) IsHost() bool { return m.Role == WorkspaceRoleHost }

func (m WorkspaceMember) CanManageMembers() bool { return m.IsHost() }

func (m WorkspaceMember) CanEditDocuments() bool {
	return m.Role == WorkspaceRoleHost || m.Role == WorkspaceRoleEditor
}

func (m WorkspaceMember) CanViewDocuments() bool { return m.Role.IsValid() }

func (m *WorkspaceMember) ChangeRole(role WorkspaceRole, now time.Time) error {
	if !role.IsValid() {
		return ErrInvalidRole
	}
	if m.IsHost() {
		return fmt.Errorf("%w: host role cannot be changed", ErrInvalidStateTransition)
	}
	if role == WorkspaceRoleHost {
		return fmt.Errorf("%w: host role cannot be assigned by ChangeRole", ErrInvalidStateTransition)
	}
	m.Role = role
	m.UpdatedAt = now
	return nil
}
