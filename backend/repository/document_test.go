package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestDocumentRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDocumentRepository(db)
	ctx := context.Background()
	host := seedUser(t, db)
	ws := seedWorkspace(t, db, host)
	doc := seedDocument(t, db, ws, host)

	found, ok, err := repo.FindByID(ctx, doc.ID)
	if err != nil || !ok || found.Title != "Doc" {
		t.Fatalf("find by id: ok=%v err=%v", ok, err)
	}

	locked, ok, err := repo.FindByIDForUpdate(ctx, doc.ID)
	if err != nil || !ok {
		t.Fatalf("find for update: %v", err)
	}
	if err := locked.Rename("Renamed", host.ID, testNow()); err != nil {
		t.Fatalf("rename: %v", err)
	}
	if err := repo.Update(ctx, locked); err != nil {
		t.Fatalf("update: %v", err)
	}

	docs, err := repo.FindByWorkspaceID(ctx, ws.ID)
	if err != nil || len(docs) != 1 {
		t.Fatalf("find by workspace: len=%d err=%v", len(docs), err)
	}

	if err := doc.LogicalDelete(host.ID, testNow()); err != nil {
		t.Fatalf("logical delete: %v", err)
	}
	if err := repo.Update(ctx, doc); err != nil {
		t.Fatalf("update deleted: %v", err)
	}
	docsAfter, err := repo.FindByWorkspaceID(ctx, ws.ID)
	if err != nil || len(docsAfter) != 0 {
		t.Fatalf("deleted doc excluded: len=%d", len(docsAfter))
	}
}

func TestDocumentRepository_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDocumentRepository(db)
	if _, ok, err := repo.FindByID(context.Background(), uuid.New()); err != nil || ok {
		t.Fatalf("expected not found, ok=%v err=%v", ok, err)
	}
	if _, ok, err := repo.FindByIDForUpdate(context.Background(), uuid.New()); err != nil || ok {
		t.Fatalf("expected not found for update, ok=%v err=%v", ok, err)
	}
}

func TestDocumentRepository_DBErrors(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDocumentRepository(db)
	ctx := context.Background()
	host := seedUser(t, db)
	ws := seedWorkspace(t, db, host)
	doc := seedDocument(t, db, ws, host)
	closeDB(t, db)

	if err := repo.Create(ctx, doc); err == nil {
		t.Fatal("expected create error")
	}
	if _, _, err := repo.FindByID(ctx, doc.ID); err == nil {
		t.Fatal("expected find by id error")
	}
	if _, _, err := repo.FindByIDForUpdate(ctx, doc.ID); err == nil {
		t.Fatal("expected find for update error")
	}
	if _, err := repo.FindByWorkspaceID(ctx, ws.ID); err == nil {
		t.Fatal("expected find by workspace error")
	}
	if err := repo.Update(ctx, doc); err == nil {
		t.Fatal("expected update error")
	}
}

func TestDocumentRepository_WithinTransaction(t *testing.T) {
	db := setupTestDB(t)
	host := seedUser(t, db)
	ws := seedWorkspace(t, db, host)
	doc := seedDocument(t, db, ws, host)
	txMgr := NewTransactionManager(db)
	repo := NewDocumentRepository(db)

	if err := txMgr.WithinTransaction(context.Background(), func(ctx context.Context) error {
		_, ok, err := repo.FindByIDForUpdate(ctx, doc.ID)
		return errOrNotFound(ok, err)
	}); err != nil {
		t.Fatalf("tx find: %v", err)
	}
}
