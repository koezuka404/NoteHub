package websocket

import (
	"encoding/json"
	"time"
)

const (
	EventConnected        = "connected"
	EventDocumentSync     = "document_sync"
	EventDocumentEdit     = "document_edit"
	EventDocumentUpdated  = "document_updated"
	EventEditorJoined     = "editor_joined"
	EventEditorLeft       = "editor_left"
	EventEditorsSync      = "editors_sync"
	EventDocumentRestored = "document_restored"
	EventDocumentDeleted  = "document_deleted"
	EventWorkspaceDeleted = "workspace_deleted"
	EventError            = "error"
)

const (
	ReasonDocumentDeleted  = "DOCUMENT_DELETED"
	ReasonWorkspaceDeleted = "WORKSPACE_DELETED"
)

type Envelope struct {
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data"`
	Timestamp string          `json:"timestamp"`
}

func MarshalEvent(eventType string, data any, now time.Time) ([]byte, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return json.Marshal(Envelope{
		Type:      eventType,
		Data:      raw,
		Timestamp: now.UTC().Format(time.RFC3339),
	})
}

type ConnectedData struct {
	ConnectionID string `json:"connection_id"`
	DocumentID   string `json:"document_id"`
}

type DocumentSyncData struct {
	DocumentID string `json:"document_id"`
	Content    string `json:"content"`
	UpdatedAt  string `json:"updated_at"`
}

type DocumentEditData struct {
	Content string `json:"content"`
}

type DocumentUpdatedData struct {
	DocumentID string `json:"document_id"`
	Content    string `json:"content"`
	UpdatedBy  string `json:"updated_by"`
	UpdatedAt  string `json:"updated_at"`
}

type EditorEventData struct {
	UserID string `json:"user_id"`
	Name   string `json:"name"`
}

type EditorsSyncData struct {
	Editors []EditorEventData `json:"editors"`
}

type DocumentRestoredData struct {
	DocumentID      string `json:"document_id"`
	Content         string `json:"content"`
	SourceVersionID string `json:"source_version_id"`
}

type DocumentDeletedData struct {
	DocumentID string `json:"document_id"`
	DeletedBy  string `json:"deleted_by"`
	DeletedAt  string `json:"deleted_at"`
	Reason     string `json:"reason"`
}

type WorkspaceDeletedData struct {
	WorkspaceID string `json:"workspace_id"`
	DeletedBy   string `json:"deleted_by"`
	DeletedAt   string `json:"deleted_at"`
	Reason      string `json:"reason"`
}

type ErrorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ClientMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}
