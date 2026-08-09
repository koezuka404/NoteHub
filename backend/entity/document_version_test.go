package entity

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestDocumentVersion_TableName(t *testing.T) {
	if (DocumentVersion{}).TableName() != "document_versions" {
		t.Fatalf("table name = %q", (DocumentVersion{}).TableName())
	}
}

func TestNewDocumentVersion(t *testing.T) {
	doc := activeDocument()
	actorID := uuid.New()
	sourceID := uuid.New()
	now := testNow()

	version, err := NewDocumentVersion(doc, "snapshot", DocumentVersionManualSave, actorID, &sourceID, now)
	if err != nil {
		t.Fatalf("NewDocumentVersion: %v", err)
	}
	if version.DocumentID != doc.ID || version.Revision != doc.Revision {
		t.Fatal("unexpected document linkage")
	}
	if version.Title != doc.Title || version.Content != "snapshot" {
		t.Fatalf("unexpected version content: %+v", version)
	}
	if version.Type != DocumentVersionManualSave || version.CreatedBy != actorID {
		t.Fatal("unexpected type or actor")
	}
	if version.SourceVersionID == nil || *version.SourceVersionID != sourceID {
		t.Fatal("expected source version id")
	}
}

func TestNewDocumentVersion_ValidationErrors(t *testing.T) {
	doc := activeDocument()
	now := testNow()
	actorID := uuid.New()

	doc.ID = uuid.Nil
	if _, err := NewDocumentVersion(doc, "c", DocumentVersionAutoSave, actorID, nil, now); err == nil {
		t.Fatal("expected error for nil document id")
	}

	doc = activeDocument()
	if _, err := NewDocumentVersion(doc, "c", DocumentVersionAutoSave, uuid.Nil, nil, now); err == nil {
		t.Fatal("expected error for nil actor id")
	}
	if _, err := NewDocumentVersion(doc, "c", DocumentVersionType("bad"), actorID, nil, now); !errors.Is(err, ErrInvalidVersionType) {
		t.Fatalf("expected ErrInvalidVersionType, got %v", err)
	}
}
