package usecase

import "github.com/google/uuid"

type IDocumentWebSocketNotifier interface {
	NotifyDocumentRestored(documentID uuid.UUID, content string, sourceVersionID uuid.UUID) error
}
