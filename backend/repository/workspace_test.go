package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestWorkspaceRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWorkspaceRepository(db)
	ctx := context.Background()
	host := seedUser(t, db)
	ws := seedWorkspace(t, db, host)

	found, ok, err := repo.FindByID(ctx, ws.ID)
	if err != nil || !ok || found.Name != "Team" {
		t.Fatalf("find by id: ok=%v err=%v", ok, err)
	}

	locked, ok, err := repo.FindByIDForUpdate(ctx, ws.ID)
	if err != nil || !ok {
		t.Fatalf("find for update: %v", err)
	}
	if err := locked.Rename("Renamed", testNow()); err != nil {
		t.Fatalf("rename: %v", err)
	}
	if err := repo.Update(ctx, locked); err != nil {
		t.Fatalf("update: %v", err)
	}

	byHost, err := repo.FindByHostID(ctx, host.ID)
	if err != nil || len(byHost) != 1 {
		t.Fatalf("find by host: %+v err=%v", byHost, err)
	}

	byUser, err := repo.FindByUserID(ctx, host.ID)
	if err != nil || len(byUser) != 1 {
		t.Fatalf("find by user: %+v err=%v", byUser, err)
	}

	now := testNow()
	if err := ws.LogicalDelete(host.ID, "cleanup", now); err != nil {
		t.Fatalf("logical delete: %v", err)
	}
	if err := repo.Update(ctx, ws); err != nil {
		t.Fatalf("update deleted: %v", err)
	}
	byHostAfter, err := repo.FindByHostID(ctx, host.ID)
	if err != nil || len(byHostAfter) != 0 {
		t.Fatalf("deleted workspace excluded: %+v", byHostAfter)
	}
}

func TestWorkspaceRepository_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWorkspaceRepository(db)
	if _, ok, err := repo.FindByID(context.Background(), uuid.New()); err != nil || ok {
		t.Fatalf("expected not found, ok=%v err=%v", ok, err)
	}
	if _, ok, err := repo.FindByIDForUpdate(context.Background(), uuid.New()); err != nil || ok {
		t.Fatalf("expected not found for update, ok=%v err=%v", ok, err)
	}
}

func TestWorkspaceRepository_DBErrors(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWorkspaceRepository(db)
	ctx := context.Background()
	host := seedUser(t, db)
	ws := seedWorkspace(t, db, host)
	closeDB(t, db)

	if err := repo.Create(ctx, ws); err == nil {
		t.Fatal("expected create error")
	}
	if _, _, err := repo.FindByID(ctx, ws.ID); err == nil {
		t.Fatal("expected find by id error")
	}
	if _, _, err := repo.FindByIDForUpdate(ctx, ws.ID); err == nil {
		t.Fatal("expected find for update error")
	}
	if _, err := repo.FindByHostID(ctx, host.ID); err == nil {
		t.Fatal("expected find by host error")
	}
	if _, err := repo.FindByUserID(ctx, host.ID); err == nil {
		t.Fatal("expected find by user error")
	}
	if err := repo.Update(ctx, ws); err == nil {
		t.Fatal("expected update error")
	}
}

func TestWorkspaceRepository_WithinTransaction(t *testing.T) {
	db := setupTestDB(t)
	host := seedUser(t, db)
	ws := seedWorkspace(t, db, host)
	txMgr := NewTransactionManager(db)
	repo := NewWorkspaceRepository(db)

	if err := txMgr.WithinTransaction(context.Background(), func(ctx context.Context) error {
		_, ok, err := repo.FindByIDForUpdate(ctx, ws.ID)
		return errOrNotFound(ok, err)
	}); err != nil {
		t.Fatalf("tx find: %v", err)
	}
}

func errOrNotFound(ok bool, err error) error {
	if err != nil {
		return err
	}
	if !ok {
		return gorm.ErrRecordNotFound
	}
	return nil
}
