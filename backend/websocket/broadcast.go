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

func (p *DocumentEventPublisher) NotifyDocumentDeleted(documentID, deletedBy uuid.UUID, deletedAt string) error {
	payload, err := MarshalEvent(EventDocumentDeleted, DocumentDeletedData{
		DocumentID: documentID.String(),
		DeletedBy:  deletedBy.String(),
		DeletedAt:  deletedAt,
		Reason:     ReasonDocumentDeleted,
	}, p.now())
	if err != nil {
		return err
	}
	p.hub.DisconnectDocument(documentID, payload)
	return nil
}

func (p *DocumentEventPublisher) NotifyWorkspaceDeleted(workspaceID, deletedBy uuid.UUID, deletedAt string) error {
	payload, err := MarshalEvent(EventWorkspaceDeleted, WorkspaceDeletedData{
		WorkspaceID: workspaceID.String(),
		DeletedBy:   deletedBy.String(),
		DeletedAt:   deletedAt,
		Reason:      ReasonWorkspaceDeleted,
	}, p.now())
	if err != nil {
		return err
	}
	p.hub.DisconnectWorkspace(workspaceID, payload)
	return nil
}

func (p *DocumentEventPublisher) NotifyAccountSuspended(userID uuid.UUID, suspendedAt string) error {
	payload, err := MarshalEvent(EventAccountSuspended, AccountSuspendedData{
		UserID:      userID.String(),
		SuspendedAt: suspendedAt,
		Reason:      ReasonAccountSuspended,
	}, p.now())
	if err != nil {
		return err
	}
	p.hub.DisconnectUser(userID, payload)
	return nil
}

func (p *DocumentEventPublisher) NotifyAccountDeleted(userID uuid.UUID, deletedAt string) error {
	payload, err := MarshalEvent(EventAccountDeleted, AccountDeletedData{
		UserID:    userID.String(),
		DeletedAt: deletedAt,
		Reason:    ReasonAccountDeleted,
	}, p.now())
	if err != nil {
		return err
	}
	p.hub.DisconnectUser(userID, payload)
	return nil
}

func (p *DocumentEventPublisher) NotifyWorkspaceHostSuspended(workspaceID, hostUserID uuid.UUID, suspendedAt string) error {
	payload, err := MarshalEvent(EventWorkspaceHostSuspended, WorkspaceHostLockedData{
		WorkspaceID: workspaceID.String(),
		HostUserID:  hostUserID.String(),
		LockedAt:    suspendedAt,
		Reason:      ReasonWorkspaceHostSuspended,
	}, p.now())
	if err != nil {
		return err
	}
	p.hub.DisconnectWorkspace(workspaceID, payload)
	return nil
}

func (p *DocumentEventPublisher) NotifyWorkspaceHostDeleted(workspaceID, hostUserID uuid.UUID, deletedAt string) error {
	payload, err := MarshalEvent(EventWorkspaceHostDeleted, WorkspaceHostLockedData{
		WorkspaceID: workspaceID.String(),
		HostUserID:  hostUserID.String(),
		LockedAt:    deletedAt,
		Reason:      ReasonWorkspaceHostDeleted,
	}, p.now())
	if err != nil {
		return err
	}
	p.hub.DisconnectWorkspace(workspaceID, payload)
	return nil
}

var _ usecase.IDocumentWebSocketNotifier = (*DocumentEventPublisher)(nil)
var _ usecase.IAccountWebSocketNotifier = (*DocumentEventPublisher)(nil)
