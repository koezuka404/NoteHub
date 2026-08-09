package websocket

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func newPublisher(t *testing.T) (*DocumentEventPublisher, *Hub, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	hub := NewHub()
	docID := uuid.New()
	wsID := uuid.New()
	userID := uuid.New()
	pub := NewDocumentEventPublisher(hub)
	pub.now = func() time.Time { return testNow() }
	return pub, hub, docID, wsID, userID
}

func expectEventOnClient(t *testing.T, client *Client, eventType string) {
	t.Helper()
	select {
	case raw := <-client.Send:
		var envelope Envelope
		if err := json.Unmarshal(raw, &envelope); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if envelope.Type != eventType {
			t.Fatalf("type = %q want %q", envelope.Type, eventType)
		}
	case <-time.After(time.Second):
		t.Fatalf("timeout waiting for %s", eventType)
	}
}

func withMarshalError(t *testing.T, fn func() error) {
	t.Helper()
	orig := jsonMarshalFn
	jsonMarshalFn = func(any) ([]byte, error) { return nil, errors.New("marshal failed") }
	t.Cleanup(func() { jsonMarshalFn = orig })
	if err := fn(); err == nil {
		t.Fatal("expected marshal error")
	}
}

func TestNewDocumentEventPublisher(t *testing.T) {
	pub := NewDocumentEventPublisher(NewHub())
	if pub == nil || pub.hub == nil || pub.now == nil {
		t.Fatal("expected publisher")
	}
}

func TestDocumentEventPublisher_NotifyDocumentCreated(t *testing.T) {
	pub, hub, docID, wsID, _ := newPublisher(t)
	watcher := newTestClient(t, hub, uuid.Nil, wsID, uuid.New())
	hub.Register(watcher)

	if err := pub.NotifyDocumentCreated(wsID, docID, "title", "user", testNow().Format(time.RFC3339)); err != nil {
		t.Fatalf("NotifyDocumentCreated: %v", err)
	}
	expectEventOnClient(t, watcher, EventDocumentCreated)
}

func TestDocumentEventPublisher_NotifyDocumentTitleUpdated(t *testing.T) {
	pub, hub, docID, wsID, _ := newPublisher(t)
	watcher := newTestClient(t, hub, uuid.Nil, wsID, uuid.New())
	editor := newTestClient(t, hub, docID, wsID, uuid.New())
	hub.Register(watcher)
	hub.Register(editor)

	ts := testNow().Format(time.RFC3339)
	if err := pub.NotifyDocumentTitleUpdated(wsID, docID, "new", "user", ts); err != nil {
		t.Fatalf("NotifyDocumentTitleUpdated: %v", err)
	}
	expectEventOnClient(t, watcher, EventDocumentUpdated)
	expectEventOnClient(t, editor, EventDocumentUpdated)
}

func TestDocumentEventPublisher_NotifyDocumentRestored(t *testing.T) {
	pub, hub, docID, _, _ := newPublisher(t)
	editor := newTestClient(t, hub, docID, uuid.New(), uuid.New())
	hub.Register(editor)

	if err := pub.NotifyDocumentRestored(docID, "content", uuid.New()); err != nil {
		t.Fatalf("NotifyDocumentRestored: %v", err)
	}
	expectEventOnClient(t, editor, EventDocumentRestored)
}

func TestDocumentEventPublisher_NotifyDocumentDeleted(t *testing.T) {
	pub, hub, docID, _, deletedBy := newPublisher(t)
	editor := newTestClient(t, hub, docID, uuid.New(), uuid.New())
	hub.Register(editor)

	ts := testNow().Format(time.RFC3339)
	if err := pub.NotifyDocumentDeleted(docID, deletedBy, ts); err != nil {
		t.Fatalf("NotifyDocumentDeleted: %v", err)
	}
	expectEventOnClient(t, editor, EventDocumentDeleted)
}

func TestDocumentEventPublisher_NotifyDocumentListDeleted(t *testing.T) {
	pub, hub, docID, wsID, deletedBy := newPublisher(t)
	watcher := newTestClient(t, hub, uuid.Nil, wsID, uuid.New())
	hub.Register(watcher)

	ts := testNow().Format(time.RFC3339)
	if err := pub.NotifyDocumentListDeleted(wsID, docID, deletedBy, ts); err != nil {
		t.Fatalf("NotifyDocumentListDeleted: %v", err)
	}
	expectEventOnClient(t, watcher, EventDocumentDeleted)
}

func TestDocumentEventPublisher_NotifyWorkspaceDeleted(t *testing.T) {
	pub, hub, _, wsID, deletedBy := newPublisher(t)
	watcher := newTestClient(t, hub, uuid.Nil, wsID, uuid.New())
	docEditor := newTestClient(t, hub, uuid.New(), wsID, uuid.New())
	hub.Register(watcher)
	hub.Register(docEditor)

	ts := testNow().Format(time.RFC3339)
	if err := pub.NotifyWorkspaceDeleted(wsID, deletedBy, ts); err != nil {
		t.Fatalf("NotifyWorkspaceDeleted: %v", err)
	}
	expectEventOnClient(t, watcher, EventWorkspaceDeleted)
	expectEventOnClient(t, docEditor, EventWorkspaceDeleted)
}

func TestDocumentEventPublisher_NotifyAccountSuspended(t *testing.T) {
	pub, hub, docID, wsID, userID := newPublisher(t)
	editor := newTestClient(t, hub, docID, wsID, userID)
	hub.Register(editor)

	ts := testNow().Format(time.RFC3339)
	if err := pub.NotifyAccountSuspended(userID, ts); err != nil {
		t.Fatalf("NotifyAccountSuspended: %v", err)
	}
	expectEventOnClient(t, editor, EventAccountSuspended)
}

func TestDocumentEventPublisher_NotifyAccountDeleted(t *testing.T) {
	pub, hub, docID, wsID, userID := newPublisher(t)
	editor := newTestClient(t, hub, docID, wsID, userID)
	hub.Register(editor)

	ts := testNow().Format(time.RFC3339)
	if err := pub.NotifyAccountDeleted(userID, ts); err != nil {
		t.Fatalf("NotifyAccountDeleted: %v", err)
	}
	expectEventOnClient(t, editor, EventAccountDeleted)
}

func TestDocumentEventPublisher_NotifyWorkspaceHostSuspended(t *testing.T) {
	pub, hub, _, wsID, hostID := newPublisher(t)
	watcher := newTestClient(t, hub, uuid.Nil, wsID, uuid.New())
	hub.Register(watcher)

	ts := testNow().Format(time.RFC3339)
	if err := pub.NotifyWorkspaceHostSuspended(wsID, hostID, ts); err != nil {
		t.Fatalf("NotifyWorkspaceHostSuspended: %v", err)
	}
	expectEventOnClient(t, watcher, EventWorkspaceHostSuspended)
}

func TestDocumentEventPublisher_NotifyWorkspaceHostDeleted(t *testing.T) {
	pub, hub, _, wsID, hostID := newPublisher(t)
	watcher := newTestClient(t, hub, uuid.Nil, wsID, uuid.New())
	hub.Register(watcher)

	ts := testNow().Format(time.RFC3339)
	if err := pub.NotifyWorkspaceHostDeleted(wsID, hostID, ts); err != nil {
		t.Fatalf("NotifyWorkspaceHostDeleted: %v", err)
	}
	expectEventOnClient(t, watcher, EventWorkspaceHostDeleted)
}

func TestDocumentEventPublisher_NotifyMemberRemoved(t *testing.T) {
	pub, hub, docID, wsID, userID := newPublisher(t)
	editor := newTestClient(t, hub, docID, wsID, userID)
	hub.Register(editor)

	ts := testNow().Format(time.RFC3339)
	if err := pub.NotifyMemberRemoved(wsID, userID, uuid.New(), ts); err != nil {
		t.Fatalf("NotifyMemberRemoved: %v", err)
	}
	expectEventOnClient(t, editor, EventMemberRemoved)
}

func TestDocumentEventPublisher_MarshalErrors(t *testing.T) {
	pub, hub, docID, wsID, userID := newPublisher(t)
	_ = hub
	ts := testNow().Format(time.RFC3339)

	withMarshalError(t, func() error { return pub.NotifyDocumentCreated(wsID, docID, "t", "u", ts) })
	withMarshalError(t, func() error { return pub.NotifyDocumentTitleUpdated(wsID, docID, "t", "u", ts) })
	withMarshalError(t, func() error { return pub.NotifyDocumentRestored(docID, "c", uuid.New()) })
	withMarshalError(t, func() error { return pub.NotifyDocumentDeleted(docID, userID, ts) })
	withMarshalError(t, func() error { return pub.NotifyDocumentListDeleted(wsID, docID, userID, ts) })
	withMarshalError(t, func() error { return pub.NotifyWorkspaceDeleted(wsID, userID, ts) })
	withMarshalError(t, func() error { return pub.NotifyAccountSuspended(userID, ts) })
	withMarshalError(t, func() error { return pub.NotifyAccountDeleted(userID, ts) })
	withMarshalError(t, func() error { return pub.NotifyWorkspaceHostSuspended(wsID, userID, ts) })
	withMarshalError(t, func() error { return pub.NotifyWorkspaceHostDeleted(wsID, userID, ts) })
	withMarshalError(t, func() error { return pub.NotifyMemberRemoved(wsID, userID, uuid.New(), ts) })
}

func TestDocumentEventPublisher_NotifyDocumentTitleUpdated_SecondMarshalError(t *testing.T) {
	pub, _, docID, wsID, _ := newPublisher(t)
	call := 0
	orig := jsonMarshalFn
	jsonMarshalFn = func(v any) ([]byte, error) {
		call++
		if call == 1 {
			return orig(v)
		}
		return nil, errors.New("second marshal failed")
	}
	t.Cleanup(func() { jsonMarshalFn = orig })

	ts := testNow().Format(time.RFC3339)
	if err := pub.NotifyDocumentTitleUpdated(wsID, docID, "t", "u", ts); err == nil {
		t.Fatal("expected second marshal error")
	}
}
