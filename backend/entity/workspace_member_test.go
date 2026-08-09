package entity

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestWorkspaceMember_TableName(t *testing.T) {
	if (WorkspaceMember{}).TableName() != "workspace_members" {
		t.Fatalf("table name = %q", (WorkspaceMember{}).TableName())
	}
}

func TestNewWorkspaceMember(t *testing.T) {
	wsID := uuid.New()
	userID := uuid.New()
	now := testNow()

	member, err := NewWorkspaceMember(wsID, userID, WorkspaceRoleMember, now)
	if err != nil {
		t.Fatalf("NewWorkspaceMember: %v", err)
	}
	if member.WorkspaceID != wsID || member.UserID != userID || member.Role != WorkspaceRoleMember {
		t.Fatalf("unexpected member: %+v", member)
	}
}

func TestNewWorkspaceMember_ValidationErrors(t *testing.T) {
	now := testNow()
	if _, err := NewWorkspaceMember(uuid.Nil, uuid.New(), WorkspaceRoleHost, now); err == nil {
		t.Fatal("expected error for nil workspace id")
	}
	if _, err := NewWorkspaceMember(uuid.New(), uuid.Nil, WorkspaceRoleHost, now); err == nil {
		t.Fatal("expected error for nil user id")
	}
	if _, err := NewWorkspaceMember(uuid.New(), uuid.New(), WorkspaceRole("guest"), now); !errors.Is(err, ErrInvalidRole) {
		t.Fatalf("expected ErrInvalidRole, got %v", err)
	}
}

func TestWorkspaceMember_RoleHelpers(t *testing.T) {
	host, err := NewWorkspaceMember(uuid.New(), uuid.New(), WorkspaceRoleHost, testNow())
	if err != nil {
		t.Fatalf("NewWorkspaceMember: %v", err)
	}
	if !host.IsHost() || host.IsMember() || !host.CanManageMembers() || !host.CanEditDocuments() {
		t.Fatal("host role helpers mismatch")
	}

	member, err := NewWorkspaceMember(uuid.New(), uuid.New(), WorkspaceRoleMember, testNow())
	if err != nil {
		t.Fatalf("NewWorkspaceMember: %v", err)
	}
	if member.IsHost() || !member.IsMember() || member.CanManageMembers() || !member.CanEditDocuments() {
		t.Fatal("member role helpers mismatch")
	}
}
