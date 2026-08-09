package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
)

func TestDocumentVersionRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDocumentVersionRepository(db)
	ctx := context.Background()
	host := seedUser(t, db)
	ws := seedWorkspace(t, db, host)
	doc := seedDocument(t, db, ws, host)

	version, err := entity.NewDocumentVersion(*doc, "snapshot", entity.DocumentVersionManualSave, host.ID, nil, testNow())
	if err != nil {
		t.Fatalf("new version: %v", err)
	}
	if err := repo.Create(ctx, &version); err != nil {
		t.Fatalf("create: %v", err)
	}

	found, ok, err := repo.FindByID(ctx, version.ID)
	if err != nil || !ok || found.Content != "snapshot" {
		t.Fatalf("find by id: ok=%v err=%v", ok, err)
	}

	versions, err := repo.FindByDocumentID(ctx, doc.ID)
	if err != nil || len(versions) != 1 {
		t.Fatalf("find by document id: len=%d err=%v", len(versions), err)
	}
}

func TestDocumentVersionRepository_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDocumentVersionRepository(db)
	if _, ok, err := repo.FindByID(context.Background(), uuid.New()); err != nil || ok {
		t.Fatalf("expected not found, ok=%v err=%v", ok, err)
	}
}

func TestDocumentVersionRepository_DBErrors(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDocumentVersionRepository(db)
	ctx := context.Background()
	host := seedUser(t, db)
	ws := seedWorkspace(t, db, host)
	doc := seedDocument(t, db, ws, host)
	version, err := entity.NewDocumentVersion(*doc, "snapshot", entity.DocumentVersionManualSave, host.ID, nil, testNow())
	if err != nil {
		t.Fatalf("new version: %v", err)
	}
	closeDB(t, db)

	if err := repo.Create(ctx, &version); err == nil {
		t.Fatal("expected create error")
	}
	if _, _, err := repo.FindByID(ctx, version.ID); err == nil {
		t.Fatal("expected find by id error")
	}
	if _, err := repo.FindByDocumentID(ctx, doc.ID); err == nil {
		t.Fatal("expected find by document id error")
	}
}
