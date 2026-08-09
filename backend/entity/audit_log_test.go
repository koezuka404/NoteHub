package entity

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestAuditLog_TableName(t *testing.T) {
	if (AuditLog{}).TableName() != "audit_logs" {
		t.Fatalf("table name = %q", (AuditLog{}).TableName())
	}
}

func TestNewAuditLog(t *testing.T) {
	now := testNow()
	actorID := uuid.New()
	resourceID := uuid.New()

	log, err := NewAuditLog(&actorID, "LOGIN", "user", &resourceID, map[string]string{"ok": "true"}, now)
	if err != nil {
		t.Fatalf("NewAuditLog: %v", err)
	}
	if log.UserID == nil || *log.UserID != actorID {
		t.Fatal("expected actor user id")
	}
	if log.TargetUserID == nil || *log.TargetUserID != resourceID {
		t.Fatal("expected target user id for user resource")
	}
	if log.Action != "LOGIN" || log.ResourceType != "user" {
		t.Fatalf("unexpected action/resource: %q / %q", log.Action, log.ResourceType)
	}
	if !json.Valid(log.Metadata) {
		t.Fatal("metadata should be valid json")
	}
}

func TestNewAuditLog_NilMetadata(t *testing.T) {
	now := testNow()
	log, err := NewAuditLog(nil, "LOGOUT", "document", nil, nil, now)
	if err != nil {
		t.Fatalf("NewAuditLog: %v", err)
	}
	if log.UserID != nil {
		t.Fatal("expected nil user id")
	}
	if string(log.Metadata) != "{}" {
		t.Fatalf("metadata = %s", log.Metadata)
	}
}

func TestNewAuditLog_WorkspaceResource(t *testing.T) {
	now := testNow()
	wsID := uuid.New()
	log, err := NewAuditLog(nil, "WORKSPACE_CREATED", "workspace", &wsID, nil, now)
	if err != nil {
		t.Fatalf("NewAuditLog: %v", err)
	}
	if log.WorkspaceID == nil || *log.WorkspaceID != wsID {
		t.Fatal("expected workspace id")
	}
}

func TestNewAuditLog_WorkspaceMemberResource(t *testing.T) {
	now := testNow()
	memberID := uuid.New()
	log, err := NewAuditLog(nil, "MEMBER_ADDED", "workspace_member", &memberID, nil, now)
	if err != nil {
		t.Fatalf("NewAuditLog: %v", err)
	}
	if log.WorkspaceID != nil {
		t.Fatal("workspace_member should not set workspace id")
	}
}

func TestNewAuditLog_DocumentResource(t *testing.T) {
	now := testNow()
	docID := uuid.New()
	log, err := NewAuditLog(nil, "DOCUMENT_CREATED", "document", &docID, nil, now)
	if err != nil {
		t.Fatalf("NewAuditLog: %v", err)
	}
	if log.DocumentID == nil || *log.DocumentID != docID {
		t.Fatal("expected document id")
	}
}

func TestNewAuditLog_ValidationErrors(t *testing.T) {
	now := testNow()
	if _, err := NewAuditLog(nil, " ", "user", nil, nil, now); err == nil {
		t.Fatal("expected error for empty action")
	}
	if _, err := NewAuditLog(nil, "LOGIN", " ", nil, nil, now); err == nil {
		t.Fatal("expected error for empty resource type")
	}
}

func TestNewAuditLog_MetadataMarshalError(t *testing.T) {
	now := testNow()
	type badMetadata struct {
		Ch chan int
	}
	_, err := NewAuditLog(nil, "LOGIN", "user", nil, badMetadata{Ch: make(chan int)}, now)
	if err == nil {
		t.Fatal("expected marshal error")
	}
}

func TestApplyDocumentAuditContext(t *testing.T) {
	resourceID := uuid.New()
	workspaceID := uuid.New()
	log := AuditLog{ResourceID: &resourceID}

	ApplyDocumentAuditContext(&log, workspaceID, "127.0.0.1", "test-agent")

	if log.WorkspaceID == nil || *log.WorkspaceID != workspaceID {
		t.Fatal("expected workspace id")
	}
	if log.DocumentID == nil || *log.DocumentID != resourceID {
		t.Fatal("expected document id from resource id")
	}
	if log.IPAddress != "127.0.0.1" || log.UserAgent != "test-agent" {
		t.Fatalf("ip=%q ua=%q", log.IPAddress, log.UserAgent)
	}
}

func TestApplyDocumentAuditContext_PreservesDocumentID(t *testing.T) {
	docID := uuid.New()
	otherID := uuid.New()
	log := AuditLog{DocumentID: &docID, ResourceID: &otherID}
	ApplyDocumentAuditContext(&log, uuid.New(), "10.0.0.1", "agent")
	if log.DocumentID == nil || *log.DocumentID != docID {
		t.Fatal("existing document id should be preserved")
	}
}
