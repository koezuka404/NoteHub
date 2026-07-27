package entity

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type DocumentVersion struct {
	ID              uuid.UUID           `gorm:"type:uuid;primaryKey"`
	DocumentID      uuid.UUID           `gorm:"type:uuid;not null;index"`
	Revision        uint64              `gorm:"not null;index"`
	Title           string              `gorm:"size:100;not null"`
	Content         string              `gorm:"type:text;not null"`
	Type            DocumentVersionType `gorm:"size:20;not null;index"`
	SourceVersionID *uuid.UUID          `gorm:"type:uuid;index"`
	CreatedBy       uuid.UUID           `gorm:"type:uuid;not null;index"`
	CreatedAt       time.Time           `gorm:"not null;index"`
}

func (DocumentVersion) TableName() string { return "document_versions" }

func NewDocumentVersion(
	doc Document,
	content string,
	versionType DocumentVersionType,
	actorID uuid.UUID,
	sourceVersionID *uuid.UUID,
	now time.Time,
) (DocumentVersion, error) {
	if doc.ID == uuid.Nil || actorID == uuid.Nil {
		return DocumentVersion{}, fmt.Errorf("document id and actor id are required")
	}
	if !versionType.IsValid() {
		return DocumentVersion{}, ErrInvalidVersionType
	}
	return DocumentVersion{
		ID:              uuid.New(),
		DocumentID:      doc.ID,
		Revision:        doc.Revision,
		Title:           doc.Title,
		Content:         content,
		Type:            versionType,
		SourceVersionID: sourceVersionID,
		CreatedBy:       actorID,
		CreatedAt:       now,
	}, nil
}
