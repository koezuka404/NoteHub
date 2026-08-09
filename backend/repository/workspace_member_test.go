package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
	"gorm.io/gorm"
)

func TestWorkspaceMemberRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWorkspaceMemberRepository(db)
	ctx := context.Background()
	host := seedUser(t, db)
	ws := seedWorkspace(t, db, host)

	memberUser, err := entity.NewUser("Bob", "bob@example.com", "hash", testNow())
	if err != nil {
		t.Fatalf("new user: %v", err)
	}
	if err := NewUserRepository(db).Create(ctx, &memberUser); err != nil {
		t.Fatalf("create user: %v", err)
	}
	member, err := entity.NewWorkspaceMember(ws.ID, memberUser.ID, entity.WorkspaceRoleMember, testNow())
	if err != nil {
		t.Fatalf("new member: %v", err)
	}
	if err := repo.Create(ctx, &member); err != nil {
		t.Fatalf("create member: %v", err)
	}

	found, ok, err := repo.FindByWorkspaceAndUser(ctx, ws.ID, memberUser.ID)
	if err != nil || !ok || found.Role != entity.WorkspaceRoleMember {
		t.Fatalf("find by workspace and user: ok=%v err=%v", ok, err)
	}

	members, err := repo.FindByWorkspaceID(ctx, ws.ID)
	if err != nil || len(members) != 2 {
		t.Fatalf("find by workspace id: len=%d err=%v", len(members), err)
	}
	if members[0].Role != entity.WorkspaceRoleHost {
		t.Fatalf("host should sort first, got %q", members[0].Role)
	}

	exists, err := repo.Exists(ctx, ws.ID, memberUser.ID)
	if err != nil || !exists {
		t.Fatalf("exists: exists=%v err=%v", exists, err)
	}

	if err := repo.Delete(ctx, ws.ID, memberUser.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

func TestWorkspaceMemberRepository_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWorkspaceMemberRepository(db)
	ctx := context.Background()

	if _, ok, err := repo.FindByWorkspaceAndUser(ctx, uuid.New(), uuid.New()); err != nil || ok {
		t.Fatalf("expected not found member, ok=%v err=%v", ok, err)
	}
	if exists, err := repo.Exists(ctx, uuid.New(), uuid.New()); err != nil || exists {
		t.Fatalf("expected not exists, exists=%v err=%v", exists, err)
	}
	if err := repo.Delete(ctx, uuid.New(), uuid.New()); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected delete not found, got %v", err)
	}
}

func TestWorkspaceMemberRepository_DBErrors(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWorkspaceMemberRepository(db)
	ctx := context.Background()
	host := seedUser(t, db)
	ws := seedWorkspace(t, db, host)
	member, err := entity.NewWorkspaceMember(ws.ID, host.ID, entity.WorkspaceRoleHost, testNow())
	if err != nil {
		t.Fatalf("new member: %v", err)
	}
	closeDB(t, db)

	if err := repo.Create(ctx, &member); err == nil {
		t.Fatal("expected create error")
	}
	if _, _, err := repo.FindByWorkspaceAndUser(ctx, ws.ID, host.ID); err == nil {
		t.Fatal("expected find error")
	}
	if _, err := repo.FindByWorkspaceID(ctx, ws.ID); err == nil {
		t.Fatal("expected list error")
	}
	if _, err := repo.Exists(ctx, ws.ID, host.ID); err == nil {
		t.Fatal("expected exists error")
	}
	if err := repo.Delete(ctx, ws.ID, host.ID); err == nil {
		t.Fatal("expected delete error")
	}
}
