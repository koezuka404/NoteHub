package entity

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type DocumentVersion struct {
	ID         uuid.UUID           `gorm:"type:uuid;primaryKey"`
	DocumentID uuid.UUID           `gorm:"type:uuid;not null;uniqueIndex:uq_document_versions_document_revision"`
	Revision   uint64              `gorm:"not null;uniqueIndex:uq_document_versions_document_revision"`
	Title      string              `gorm:"size:255;not null"`
	Content    string              `gorm:"type:text;not null"`
	Type       DocumentVersionType `gorm:"size:20;not null;index"`
	CreatedBy  uuid.UUID           `gorm:"type:uuid;not null;index"`
	CreatedAt  time.Time           `gorm:"not null;index"`
}

func (DocumentVersion) TableName() string { return "document_versions" }

func NewDocumentVersion(document Document, versionType DocumentVersionType, actorID uuid.UUID, now time.Time) (DocumentVersion, error) {
	if document.ID == uuid.Nil || actorID == uuid.Nil {
		return DocumentVersion{}, fmt.Errorf("document id and actor id are required")
	}
	if !versionType.IsValid() {
		return DocumentVersion{}, ErrInvalidVersionType
	}
	return DocumentVersion{ID: uuid.New(), DocumentID: document.ID, Revision: document.Revision, Title: document.Title, Content: document.Content, Type: versionType, CreatedBy: actorID, CreatedAt: now}, nil
}
