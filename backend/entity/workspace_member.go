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
	JoinedAt    time.Time     `gorm:"not null"`
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
	return WorkspaceMember{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		UserID:      userID,
		Role:        role,
		JoinedAt:    now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (m WorkspaceMember) IsHost() bool { return m.Role == WorkspaceRoleHost }

func (m WorkspaceMember) IsMember() bool { return m.Role == WorkspaceRoleMember }

func (m WorkspaceMember) CanManageMembers() bool { return m.IsHost() }

func (m WorkspaceMember) CanEditDocuments() bool {
	return m.Role == WorkspaceRoleHost || m.Role == WorkspaceRoleMember
}
