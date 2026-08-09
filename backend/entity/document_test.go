package entity

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func activeDocument() Document {
	wsID := uuid.New()
	actorID := uuid.New()
	doc, _ := NewDocument(wsID, actorID, "Title", "content", testNow())
	return doc
}

func TestDocument_TableName(t *testing.T) {
	if (Document{}).TableName() != "documents" {
		t.Fatalf("table name = %q", (Document{}).TableName())
	}
}

func TestNewDocument(t *testing.T) {
	wsID := uuid.New()
	actorID := uuid.New()
	now := testNow()

	doc, err := NewDocument(wsID, actorID, "  My Doc  ", "body", now)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}
	if doc.Title != "My Doc" || doc.Content != "body" || doc.Revision != 1 {
		t.Fatalf("unexpected doc: %+v", doc)
	}
	if doc.WorkspaceID != wsID || doc.CreatedBy != actorID || doc.UpdatedBy != actorID {
		t.Fatal("unexpected ids")
	}
}

func TestNewDocument_ValidationErrors(t *testing.T) {
	now := testNow()
	actorID := uuid.New()
	wsID := uuid.New()

	if _, err := NewDocument(uuid.Nil, actorID, "t", "c", now); err == nil {
		t.Fatal("expected error for nil workspace id")
	}
	if _, err := NewDocument(wsID, uuid.Nil, "t", "c", now); err == nil {
		t.Fatal("expected error for nil actor id")
	}
	if _, err := NewDocument(wsID, actorID, "  ", "c", now); err == nil {
		t.Fatal("expected error for empty title")
	}
}

func TestDocument_IsDeleted(t *testing.T) {
	doc := activeDocument()
	if doc.IsDeleted() {
		t.Fatal("new document should not be deleted")
	}
	now := testNow()
	if err := doc.LogicalDelete(doc.CreatedBy, now); err != nil {
		t.Fatalf("LogicalDelete: %v", err)
	}
	if !doc.IsDeleted() {
		t.Fatal("deleted document should report deleted")
	}
}

func TestDocument_Rename(t *testing.T) {
	doc := activeDocument()
	actorID := uuid.New()
	now := testNow().Add(1)

	if err := doc.Rename("  New Title  ", actorID, now); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if doc.Title != "New Title" || doc.UpdatedBy != actorID || !doc.UpdatedAt.Equal(now) {
		t.Fatalf("unexpected doc after rename: %+v", doc)
	}
}

func TestDocument_Rename_Errors(t *testing.T) {
	doc := activeDocument()
	now := testNow()
	actorID := uuid.New()

	if err := doc.Rename("", actorID, now); err == nil {
		t.Fatal("expected error for empty title")
	}
	if err := doc.Rename("x", uuid.Nil, now); err == nil {
		t.Fatal("expected error for nil actor")
	}
	_ = doc.LogicalDelete(actorID, now)
	if err := doc.Rename("x", actorID, now); !errors.Is(err, ErrDocumentDeleted) {
		t.Fatalf("expected ErrDocumentDeleted, got %v", err)
	}
}

func TestDocument_ReplaceContent(t *testing.T) {
	doc := activeDocument()
	actorID := uuid.New()
	now := testNow().Add(1)
	rev := doc.Revision

	if err := doc.ReplaceContent("updated", actorID, rev, now); err != nil {
		t.Fatalf("ReplaceContent: %v", err)
	}
	if doc.Content != "updated" || doc.Revision != rev+1 || doc.UpdatedBy != actorID {
		t.Fatalf("unexpected doc: %+v", doc)
	}
}

func TestDocument_ReplaceContent_Errors(t *testing.T) {
	doc := activeDocument()
	now := testNow()
	actorID := uuid.New()

	if err := doc.ReplaceContent("x", uuid.Nil, doc.Revision, now); err == nil {
		t.Fatal("expected error for nil actor")
	}
	if err := doc.ReplaceContent("x", actorID, doc.Revision+1, now); !errors.Is(err, ErrDocumentConflict) {
		t.Fatalf("expected ErrDocumentConflict, got %v", err)
	}
	_ = doc.LogicalDelete(actorID, now)
	if err := doc.ReplaceContent("x", actorID, doc.Revision, now); !errors.Is(err, ErrDocumentDeleted) {
		t.Fatalf("expected ErrDocumentDeleted, got %v", err)
	}
}

func TestDocument_LogicalDelete(t *testing.T) {
	doc := activeDocument()
	actorID := uuid.New()
	now := testNow()

	if err := doc.LogicalDelete(actorID, now); err != nil {
		t.Fatalf("LogicalDelete: %v", err)
	}
	if doc.DeletedAt == nil || doc.DeletedBy == nil || *doc.DeletedBy != actorID {
		t.Fatal("expected delete metadata")
	}
}

func TestDocument_LogicalDelete_Errors(t *testing.T) {
	doc := activeDocument()
	now := testNow()
	actorID := uuid.New()

	if err := doc.LogicalDelete(uuid.Nil, now); err == nil {
		t.Fatal("expected error for nil actor")
	}
	_ = doc.LogicalDelete(actorID, now)
	if err := doc.LogicalDelete(actorID, now); !errors.Is(err, ErrDocumentDeleted) {
		t.Fatalf("expected ErrDocumentDeleted, got %v", err)
	}
}
