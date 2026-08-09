package entity

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func activeWorkspace() (Workspace, User) {
	host := activeUser()
	ws, _ := NewWorkspace(host.ID, "Team", testNow())
	return ws, host
}

func TestWorkspace_TableName(t *testing.T) {
	if (Workspace{}).TableName() != "workspaces" {
		t.Fatalf("table name = %q", (Workspace{}).TableName())
	}
}

func TestNewWorkspace(t *testing.T) {
	hostID := uuid.New()
	now := testNow()

	ws, err := NewWorkspace(hostID, "  My Workspace  ", now)
	if err != nil {
		t.Fatalf("NewWorkspace: %v", err)
	}
	if ws.Name != "My Workspace" || ws.HostID != hostID {
		t.Fatalf("unexpected workspace: %+v", ws)
	}
}

func TestNewWorkspace_ValidationErrors(t *testing.T) {
	now := testNow()
	if _, err := NewWorkspace(uuid.Nil, "name", now); err == nil {
		t.Fatal("expected error for nil host id")
	}
	if _, err := NewWorkspace(uuid.New(), "  ", now); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestWorkspace_IsDeletedAndAvailable(t *testing.T) {
	ws, host := activeWorkspace()
	if ws.IsDeleted() {
		t.Fatal("new workspace should not be deleted")
	}
	if !ws.IsAvailable(host) {
		t.Fatal("workspace should be available for active host")
	}

	suspendedHost := host
	_ = suspendedHost.Suspend(testNow())
	if ws.IsAvailable(suspendedHost) {
		t.Fatal("suspended host should make workspace unavailable")
	}

	otherHost := activeUser()
	if ws.IsAvailable(otherHost) {
		t.Fatal("non-host user should not make workspace available")
	}

	now := testNow()
	_ = ws.LogicalDelete(host.ID, "cleanup", now)
	if !ws.IsDeleted() || ws.IsAvailable(host) {
		t.Fatal("deleted workspace should not be available")
	}
}

func TestWorkspace_Rename(t *testing.T) {
	ws, _ := activeWorkspace()
	now := testNow().Add(1)

	if err := ws.Rename("  Renamed  ", now); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if ws.Name != "Renamed" || !ws.UpdatedAt.Equal(now) {
		t.Fatalf("unexpected workspace: %+v", ws)
	}
}

func TestWorkspace_Rename_Errors(t *testing.T) {
	ws, host := activeWorkspace()
	now := testNow()

	if err := ws.Rename("  ", now); err == nil {
		t.Fatal("expected error for empty name")
	}
	_ = ws.LogicalDelete(host.ID, "done", now)
	if err := ws.Rename("x", now); !errors.Is(err, ErrWorkspaceDeleted) {
		t.Fatalf("expected ErrWorkspaceDeleted, got %v", err)
	}
}

func TestWorkspace_LogicalDelete(t *testing.T) {
	ws, host := activeWorkspace()
	now := testNow()

	if err := ws.LogicalDelete(host.ID, "  no longer needed  ", now); err != nil {
		t.Fatalf("LogicalDelete: %v", err)
	}
	if ws.DeletedAt == nil || ws.DeletedBy == nil || ws.DeleteReason != "no longer needed" {
		t.Fatalf("unexpected workspace: %+v", ws)
	}
}

func TestWorkspace_LogicalDelete_Errors(t *testing.T) {
	ws, host := activeWorkspace()
	now := testNow()

	if err := ws.LogicalDelete(uuid.Nil, "reason", now); err == nil {
		t.Fatal("expected error for nil deleted by")
	}
	if err := ws.LogicalDelete(host.ID, "  ", now); err == nil {
		t.Fatal("expected error for empty reason")
	}
	_ = ws.LogicalDelete(host.ID, "done", now)
	if err := ws.LogicalDelete(host.ID, "again", now); !errors.Is(err, ErrWorkspaceDeleted) {
		t.Fatalf("expected ErrWorkspaceDeleted, got %v", err)
	}
}
