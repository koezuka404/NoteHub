package redis

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/usecase"
)

func TestNewDocumentEditorsStore_DefaultTTL(t *testing.T) {
	client, _ := newTestClient(t)
	store := NewDocumentEditorsStore(client, 0)
	if store.keyTTL != 16*time.Minute {
		t.Fatalf("keyTTL = %v", store.keyTTL)
	}
}

func TestDocumentEditorsStore(t *testing.T) {
	client, _ := newTestClient(t)
	store := NewDocumentEditorsStore(client, time.Minute)
	ctx := context.Background()
	docID := uuid.New()
	userA := uuid.New()
	userB := uuid.New()

	editors, err := store.List(ctx, docID)
	if err != nil || len(editors) != 0 {
		t.Fatalf("List empty = %v, %v", editors, err)
	}

	added, editors, err := store.Add(ctx, docID, usecase.DocumentEditorInfo{UserID: userA, Name: "Alice"})
	if err != nil || !added || len(editors) != 1 {
		t.Fatalf("Add first = %v, %v, %v", added, editors, err)
	}

	added, editors, err = store.Add(ctx, docID, usecase.DocumentEditorInfo{UserID: userA, Name: "Alice"})
	if err != nil || added || len(editors) != 1 {
		t.Fatalf("Add duplicate = %v, %v, %v", added, editors, err)
	}

	added, editors, err = store.Add(ctx, docID, usecase.DocumentEditorInfo{UserID: userB, Name: "Bob"})
	if err != nil || !added || len(editors) != 2 {
		t.Fatalf("Add second = %v, %v, %v", added, editors, err)
	}

	if err := store.RefreshTTL(ctx, docID); err != nil {
		t.Fatalf("RefreshTTL: %v", err)
	}
	if err := store.RefreshTTL(ctx, uuid.New()); err != nil {
		t.Fatalf("RefreshTTL missing: %v", err)
	}

	removed, editors, err := store.Remove(ctx, docID, uuid.New())
	if err != nil || removed || len(editors) != 2 {
		t.Fatalf("Remove missing = %v, %v, %v", removed, editors, err)
	}

	removed, editors, err = store.Remove(ctx, docID, userA)
	if err != nil || !removed || len(editors) != 1 {
		t.Fatalf("Remove first = %v, %v, %v", removed, editors, err)
	}

	removed, editors, err = store.Remove(ctx, docID, userB)
	if err != nil || !removed || editors != nil {
		t.Fatalf("Remove last = %v, %v, %v", removed, editors, err)
	}
}

func TestDocumentEditorsStore_ListSkipsInvalidUserID(t *testing.T) {
	client, mr := newTestClient(t)
	store := NewDocumentEditorsStore(client, time.Minute)
	docID := uuid.New()
	mr.HSet(editorsKey(docID), "bad-id", "Ghost")

	editors, err := store.List(context.Background(), docID)
	if err != nil || len(editors) != 0 {
		t.Fatalf("List() = %v, %v", editors, err)
	}
}

func TestDocumentEditorsStore_Errors(t *testing.T) {
	client, _ := newTestClient(t)
	store := NewDocumentEditorsStore(client, time.Minute)
	client.client.Close()
	ctx := context.Background()
	docID := uuid.New()
	editor := usecase.DocumentEditorInfo{UserID: uuid.New(), Name: "X"}

	if _, err := store.List(ctx, docID); err == nil {
		t.Fatal("expected list error")
	}
	if _, _, err := store.Add(ctx, docID, editor); err == nil {
		t.Fatal("expected add error")
	}
	if _, _, err := store.Remove(ctx, docID, editor.UserID); err == nil {
		t.Fatal("expected remove error")
	}
	if err := store.RefreshTTL(ctx, docID); err == nil {
		t.Fatal("expected refresh ttl error")
	}
}
