package entity

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID           uuid.UUID       `gorm:"type:uuid;primaryKey"`
	ActorUserID  *uuid.UUID      `gorm:"type:uuid;index:idx_audit_logs_actor_created,priority:1"`
	Action       string          `gorm:"size:100;not null;index"`
	ResourceType string          `gorm:"size:100;not null;index:idx_audit_logs_resource_created,priority:1"`
	ResourceID   *uuid.UUID      `gorm:"type:uuid;index:idx_audit_logs_resource_created,priority:2"`
	IPAddress    string          `gorm:"size:64"`
	UserAgent    string          `gorm:"size:512"`
	Metadata     json.RawMessage `gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt    time.Time       `gorm:"not null;index:idx_audit_logs_actor_created,priority:2;index:idx_audit_logs_resource_created,priority:3"`
}

func (AuditLog) TableName() string { return "audit_logs" }

func NewAuditLog(actorUserID *uuid.UUID, action, resourceType string, resourceID *uuid.UUID, metadata any, now time.Time) (AuditLog, error) {
	action = strings.TrimSpace(action)
	resourceType = strings.TrimSpace(resourceType)
	if action == "" || resourceType == "" {
		return AuditLog{}, fmt.Errorf("action and resource type are required")
	}
	raw := json.RawMessage(`{}`)
	if metadata != nil {
		encoded, err := json.Marshal(metadata)
		if err != nil {
			return AuditLog{}, fmt.Errorf("marshal metadata: %w", err)
		}
		raw = encoded
	}
	return AuditLog{ID: uuid.New(), ActorUserID: actorUserID, Action: action, ResourceType: resourceType, ResourceID: resourceID, Metadata: raw, CreatedAt: now}, nil
}
