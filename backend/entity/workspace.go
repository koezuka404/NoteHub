package entity

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Workspace struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey"`
	Name         string     `gorm:"size:100;not null"`
	HostID       uuid.UUID  `gorm:"type:uuid;not null;index"`
	DeletedAt    *time.Time `gorm:"index"`
	DeletedBy    *uuid.UUID `gorm:"type:uuid;index"`
	DeleteReason string     `gorm:"size:500"`
	CreatedAt    time.Time  `gorm:"not null"`
	UpdatedAt    time.Time  `gorm:"not null"`
}

func (Workspace) TableName() string { return "workspaces" }

func NewWorkspace(hostID uuid.UUID, name string, now time.Time) (Workspace, error) {
	name = strings.TrimSpace(name)
	if hostID == uuid.Nil || name == "" {
		return Workspace{}, fmt.Errorf("host id and workspace name are required")
	}
	return Workspace{
		ID:        uuid.New(),
		Name:      name,
		HostID:    hostID,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (w Workspace) IsDeleted() bool { return w.DeletedAt != nil }

func (w Workspace) IsAvailable(host User) bool {
	return !w.IsDeleted() && host.ID == w.HostID && host.CanAuthenticate()
}

func (w *Workspace) Rename(name string, now time.Time) error {
	if w.IsDeleted() {
		return ErrWorkspaceDeleted
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("workspace name is required")
	}
	w.Name = name
	w.UpdatedAt = now
	return nil
}

func (w *Workspace) LogicalDelete(deletedBy uuid.UUID, reason string, now time.Time) error {
	if w.IsDeleted() {
		return ErrWorkspaceDeleted
	}
	if deletedBy == uuid.Nil {
		return fmt.Errorf("deleted by user id is required")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return fmt.Errorf("delete reason is required")
	}
	w.DeletedAt = timePointer(now)
	w.DeletedBy = &deletedBy
	w.DeleteReason = reason
	w.UpdatedAt = now
	return nil
}
