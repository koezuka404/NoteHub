package usecase

import "github.com/google/uuid"

type IDocumentWebSocketNotifier interface {
	NotifyDocumentRestored(documentID uuid.UUID, content string, sourceVersionID uuid.UUID) error
	NotifyDocumentDeleted(documentID, deletedBy uuid.UUID, deletedAt string) error
	NotifyWorkspaceDeleted(workspaceID, deletedBy uuid.UUID, deletedAt string) error
}
