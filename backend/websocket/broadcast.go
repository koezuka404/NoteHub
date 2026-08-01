package websocket

import (
	"time"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/usecase"
)

type DocumentEventPublisher struct {
	hub *Hub
	now func() time.Time
}

func NewDocumentEventPublisher(hub *Hub) *DocumentEventPublisher {
	return &DocumentEventPublisher{hub: hub, now: time.Now}
}

func (p *DocumentEventPublisher) NotifyDocumentRestored(documentID uuid.UUID, content string, sourceVersionID uuid.UUID) error {
	payload, err := MarshalEvent(EventDocumentRestored, DocumentRestoredData{
		DocumentID:      documentID.String(),
		Content:         content,
		SourceVersionID: sourceVersionID.String(),
	}, p.now())
	if err != nil {
		return err
	}
	p.hub.BroadcastDocument(documentID, payload)
	return nil
}

var _ usecase.IDocumentWebSocketNotifier = (*DocumentEventPublisher)(nil)
