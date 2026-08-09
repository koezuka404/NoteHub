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
	UserID       *uuid.UUID      `gorm:"column:user_id;type:uuid;index:idx_audit_logs_user_created,priority:1"`
	Action       string          `gorm:"size:100;not null;index"`
	TargetUserID *uuid.UUID      `gorm:"column:target_user_id;type:uuid"`
	WorkspaceID  *uuid.UUID      `gorm:"column:workspace_id;type:uuid"`
	DocumentID   *uuid.UUID      `gorm:"column:document_id;type:uuid"`
	ResourceType string          `gorm:"size:100;not null;index:idx_audit_logs_resource_created,priority:1"`
	ResourceID   *uuid.UUID      `gorm:"type:uuid;index:idx_audit_logs_resource_created,priority:2"`
	IPAddress    string          `gorm:"size:64"`
	UserAgent    string          `gorm:"size:512"`
	Metadata     json.RawMessage `gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt    time.Time       `gorm:"not null;index:idx_audit_logs_user_created,priority:2;index:idx_audit_logs_resource_created,priority:3"`
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
	log := AuditLog{
		ID:           uuid.New(),
		UserID:       actorUserID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Metadata:     raw,
		CreatedAt:    now,
	}
	switch resourceType {
	case "user":
		log.TargetUserID = resourceID
	case "workspace", "workspace_member":
		if resourceType == "workspace" {
			log.WorkspaceID = resourceID
		}
	case "document":
		log.DocumentID = resourceID
	}
	return log, nil
}

func ApplyDocumentAuditContext(audit *AuditLog, workspaceID uuid.UUID, ipAddress, userAgent string) {
	audit.WorkspaceID = &workspaceID
	if audit.DocumentID == nil && audit.ResourceID != nil {
		audit.DocumentID = audit.ResourceID
	}
	audit.IPAddress = ipAddress
	audit.UserAgent = userAgent
}
