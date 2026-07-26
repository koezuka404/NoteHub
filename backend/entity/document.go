package entity

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Document struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey"`
	WorkspaceID uuid.UUID  `gorm:"type:uuid;not null;index"`
	CreatedBy   uuid.UUID  `gorm:"type:uuid;not null;index"`
	UpdatedBy   uuid.UUID  `gorm:"type:uuid;not null;index"`
	Title       string     `gorm:"size:100;not null"`
	Content     string     `gorm:"type:text;not null"`
	Revision    uint64     `gorm:"not null;default:1"`
	DeletedAt   *time.Time `gorm:"index"`
	DeletedBy   *uuid.UUID `gorm:"type:uuid;index"`
	CreatedAt   time.Time  `gorm:"not null"`
	UpdatedAt   time.Time  `gorm:"not null"`
}

func (Document) TableName() string { return "documents" }

func NewDocument(workspaceID, actorID uuid.UUID, title, content string, now time.Time) (Document, error) {
	title = strings.TrimSpace(title)
	if workspaceID == uuid.Nil || actorID == uuid.Nil || title == "" {
		return Document{}, fmt.Errorf("workspace id, actor id and title are required")
	}
	return Document{ID: uuid.New(), WorkspaceID: workspaceID, CreatedBy: actorID, UpdatedBy: actorID, Title: title, Content: content, Revision: 1, CreatedAt: now, UpdatedAt: now}, nil
}

func (d Document) IsDeleted() bool { return d.DeletedAt != nil }

func (d *Document) Rename(title string, actorID uuid.UUID, now time.Time) error {
	if d.IsDeleted() {
		return ErrDocumentDeleted
	}
	title = strings.TrimSpace(title)
	if title == "" || actorID == uuid.Nil {
		return fmt.Errorf("title and actor id are required")
	}
	d.Title = title
	d.UpdatedBy = actorID
	d.UpdatedAt = now
	return nil
}

func (d *Document) ReplaceContent(content string, actorID uuid.UUID, expectedRevision uint64, now time.Time) error {
	if d.IsDeleted() {
		return ErrDocumentDeleted
	}
	if actorID == uuid.Nil {
		return fmt.Errorf("actor id is required")
	}
	if expectedRevision != d.Revision {
		return fmt.Errorf("%w: expected=%d actual=%d", ErrDocumentConflict, expectedRevision, d.Revision)
	}
	d.Content = content
	d.UpdatedBy = actorID
	d.Revision++
	d.UpdatedAt = now
	return nil
}

func (d *Document) LogicalDelete(actorID uuid.UUID, now time.Time) error {
	if d.IsDeleted() {
		return ErrDocumentDeleted
	}
	if actorID == uuid.Nil {
		return fmt.Errorf("actor id is required")
	}
	d.DeletedAt = timePointer(now)
	d.DeletedBy = &actorID
	d.UpdatedBy = actorID
	d.UpdatedAt = now
	return nil
}
