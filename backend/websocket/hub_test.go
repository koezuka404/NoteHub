package websocket

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewHub(t *testing.T) {
	hub := NewHub()
	if hub == nil || hub.documents == nil || hub.workspaceWatchers == nil {
		t.Fatal("NewHub returned incomplete hub")
	}
}

func TestHub_RegisterUnregisterDocumentClient(t *testing.T) {
	hub := NewHub()
	docID := uuid.New()
	wsID := uuid.New()
	userID := uuid.New()
	client := newTestClient(t, hub, docID, wsID, userID)

	hub.Register(client)
	if _, ok := hub.documents[docID][client]; !ok {
		t.Fatal("document client not registered")
	}

	hub.Unregister(client)
	if len(hub.documents[docID]) != 0 {
		t.Fatal("expected empty document clients map entry removed")
	}
}

func TestHub_RegisterUnregisterWorkspaceClient(t *testing.T) {
	hub := NewHub()
	wsID := uuid.New()
	userID := uuid.New()
	client := newTestClient(t, hub, uuid.Nil, wsID, userID)

	hub.Register(client)
	if _, ok := hub.workspaceWatchers[wsID][client]; !ok {
		t.Fatal("workspace client not registered")
	}

	hub.Unregister(client)
	if len(hub.workspaceWatchers[wsID]) != 0 {
		t.Fatal("expected empty workspace clients map entry removed")
	}
}

func TestHub_UnregisterNilClientsMap(t *testing.T) {
	hub := NewHub()
	client := newTestClient(t, hub, uuid.New(), uuid.New(), uuid.New())
	hub.Unregister(client)

	wsClient := newTestClient(t, hub, uuid.Nil, uuid.New(), uuid.New())
	hub.Unregister(wsClient)
}

func TestHub_BroadcastWorkspaceSkipsNotReady(t *testing.T) {
	hub := NewHub()
	wsID := uuid.New()
	notReady := newTestClient(t, hub, uuid.Nil, wsID, uuid.New())
	notReady.Ready.Store(false)
	hub.Register(notReady)

	hub.BroadcastWorkspace(wsID, []byte(`{"type":"x"}`))
	select {
	case <-notReady.Send:
		t.Fatal("not-ready client should not receive broadcast")
	default:
	}
}

func TestHub_BroadcastWorkspaceEmpty(t *testing.T) {
	hub := NewHub()
	hub.BroadcastWorkspace(uuid.New(), []byte(`{"type":"x"}`))
}

func TestHub_BroadcastDocumentExceptSkipsNotReady(t *testing.T) {
	hub := NewHub()
	docID := uuid.New()
	notReady := newTestClient(t, hub, docID, uuid.New(), uuid.New())
	notReady.Ready.Store(false)
	hub.Register(notReady)

	hub.BroadcastDocumentExcept(docID, nil, []byte(`{"type":"x"}`))
	select {
	case <-notReady.Send:
		t.Fatal("not-ready client should not receive broadcast")
	default:
	}
}

func TestHub_DisconnectDocumentSkipsNotReady(t *testing.T) {
	hub := NewHub()
	docID := uuid.New()
	client := newTestClient(t, hub, docID, uuid.New(), uuid.New())
	client.Ready.Store(false)
	hub.Register(client)

	hub.DisconnectDocument(docID, []byte(`{"type":"x"}`))
	select {
	case <-client.Send:
		t.Fatal("not-ready client should not receive disconnect")
	default:
	}
}

func TestHub_BroadcastDocumentSkipsNotReady(t *testing.T) {
	hub := NewHub()
	docID := uuid.New()
	ready := newTestClient(t, hub, docID, uuid.New(), uuid.New())
	notReady := newTestClient(t, hub, docID, uuid.New(), uuid.New())
	notReady.Ready.Store(false)
	hub.Register(ready)
	hub.Register(notReady)

	payload := []byte(`{"type":"test"}`)
	hub.BroadcastDocument(docID, payload)

	select {
	case got := <-ready.Send:
		if string(got) != string(payload) {
			t.Fatalf("payload = %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("ready client did not receive broadcast")
	}

	select {
	case <-notReady.Send:
		t.Fatal("not-ready client should not receive broadcast")
	default:
	}
}

func TestHub_BroadcastDocumentExcept(t *testing.T) {
	hub := NewHub()
	docID := uuid.New()
	a := newTestClient(t, hub, docID, uuid.New(), uuid.New())
	b := newTestClient(t, hub, docID, uuid.New(), uuid.New())
	hub.Register(a)
	hub.Register(b)

	payload := []byte(`{"type":"except"}`)
	hub.BroadcastDocumentExcept(docID, a, payload)

	select {
	case <-a.Send:
		t.Fatal("excluded client should not receive")
	default:
	}
	select {
	case got := <-b.Send:
		if string(got) != string(payload) {
			t.Fatalf("payload = %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("other client did not receive broadcast")
	}
}

func TestHub_BroadcastWorkspace(t *testing.T) {
	hub := NewHub()
	wsID := uuid.New()
	client := newTestClient(t, hub, uuid.Nil, wsID, uuid.New())
	hub.Register(client)

	payload := []byte(`{"type":"workspace"}`)
	hub.BroadcastWorkspace(wsID, payload)

	select {
	case got := <-client.Send:
		if string(got) != string(payload) {
			t.Fatalf("payload = %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("workspace client did not receive broadcast")
	}
}

func TestHub_DisconnectDocument(t *testing.T) {
	hub := NewHub()
	docID := uuid.New()
	client := newTestClient(t, hub, docID, uuid.New(), uuid.New())
	hub.Register(client)

	now := testNow()
	payload, err := MarshalEvent(EventError, ErrorData{Code: ReasonDocumentDeleted, Message: "gone"}, now)
	if err != nil {
		t.Fatalf("MarshalEvent: %v", err)
	}
	hub.DisconnectDocument(docID, payload)

	select {
	case got := <-client.Send:
		if string(got) != string(payload) {
			t.Fatalf("payload = %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("client did not receive disconnect payload")
	}
}

func TestHub_DisconnectDocumentNilPayload(t *testing.T) {
	hub := NewHub()
	docID := uuid.New()
	client := newTestClient(t, hub, docID, uuid.New(), uuid.New())
	hub.Register(client)

	hub.DisconnectDocument(docID, nil)

	select {
	case <-client.Send:
		t.Fatal("nil payload should not send")
	default:
	}
}

func TestHub_DisconnectWorkspace(t *testing.T) {
	hub := NewHub()
	wsID := uuid.New()
	client := newTestClient(t, hub, uuid.Nil, wsID, uuid.New())
	hub.Register(client)

	payload := []byte(`{"type":"disconnect"}`)
	hub.DisconnectWorkspace(wsID, payload)

	select {
	case got := <-client.Send:
		if string(got) != string(payload) {
			t.Fatalf("payload = %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("client did not receive disconnect payload")
	}
}

func TestHub_DisconnectUser(t *testing.T) {
	hub := NewHub()
	userID := uuid.New()
	docClient := newTestClient(t, hub, uuid.New(), uuid.New(), userID)
	hub.Register(docClient)

	payload := []byte(`{"type":"user-disconnect"}`)
	hub.DisconnectUser(userID, payload)

	select {
	case got := <-docClient.Send:
		if string(got) != string(payload) {
			t.Fatalf("payload = %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("client did not receive disconnect payload")
	}
}

func TestHub_DisconnectUserInWorkspace(t *testing.T) {
	hub := NewHub()
	wsID := uuid.New()
	userID := uuid.New()
	inWS := newTestClient(t, hub, uuid.New(), wsID, userID)
	otherWS := newTestClient(t, hub, uuid.New(), uuid.New(), userID)
	hub.Register(inWS)
	hub.Register(otherWS)

	payload := []byte(`{"type":"member-removed"}`)
	hub.DisconnectUserInWorkspace(wsID, userID, payload)

	select {
	case got := <-inWS.Send:
		if string(got) != string(payload) {
			t.Fatalf("payload = %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("in-workspace client did not receive disconnect")
	}

	select {
	case <-otherWS.Send:
		t.Fatal("other workspace client should not receive")
	default:
	}
}

func TestHub_DisconnectWorkspaceDocumentAndWatcher(t *testing.T) {
	hub := NewHub()
	wsID := uuid.New()
	watcher := newTestClient(t, hub, uuid.Nil, wsID, uuid.New())
	docEditor := newTestClient(t, hub, uuid.New(), wsID, uuid.New())
	otherWS := newTestClient(t, hub, uuid.New(), uuid.New(), uuid.New())
	hub.Register(watcher)
	hub.Register(docEditor)
	hub.Register(otherWS)

	payload := []byte(`{"type":"ws-deleted"}`)
	hub.DisconnectWorkspace(wsID, payload)

	for _, c := range []*Client{watcher, docEditor} {
		select {
		case got := <-c.Send:
			if string(got) != string(payload) {
				t.Fatalf("payload = %q", got)
			}
		case <-time.After(time.Second):
			t.Fatal("client did not receive disconnect payload")
		}
	}

	select {
	case <-otherWS.Send:
		t.Fatal("other workspace client should not receive")
	default:
	}
}

func TestHub_DisconnectUserSkipsNotReady(t *testing.T) {
	hub := NewHub()
	userID := uuid.New()
	ready := newTestClient(t, hub, uuid.New(), uuid.New(), userID)
	notReady := newTestClient(t, hub, uuid.New(), uuid.New(), userID)
	notReady.Ready.Store(false)
	hub.Register(ready)
	hub.Register(notReady)

	hub.DisconnectUser(userID, []byte(`{"type":"x"}`))

	select {
	case <-ready.Send:
	default:
		t.Fatal("ready client should receive disconnect payload")
	}
	select {
	case <-notReady.Send:
		t.Fatal("not-ready client should not receive")
	default:
	}
}

func TestHub_DisconnectUserInWorkspaceSkipsNotReady(t *testing.T) {
	hub := NewHub()
	wsID := uuid.New()
	userID := uuid.New()
	notReady := newTestClient(t, hub, uuid.New(), wsID, userID)
	notReady.Ready.Store(false)
	hub.Register(notReady)

	hub.DisconnectUserInWorkspace(wsID, userID, []byte(`{"type":"x"}`))
	select {
	case <-notReady.Send:
		t.Fatal("not-ready client should not receive")
	default:
	}
}

func TestHub_DisconnectWorkspaceSkipsNotReady(t *testing.T) {
	hub := NewHub()
	wsID := uuid.New()
	notReady := newTestClient(t, hub, uuid.Nil, wsID, uuid.New())
	notReady.Ready.Store(false)
	hub.Register(notReady)

	hub.DisconnectWorkspace(wsID, []byte(`{"type":"x"}`))
	select {
	case <-notReady.Send:
		t.Fatal("not-ready client should not receive")
	default:
	}
}

func TestHub_DisconnectWorkspaceNilPayload(t *testing.T) {
	hub := NewHub()
	wsID := uuid.New()
	client := newTestClient(t, hub, uuid.Nil, wsID, uuid.New())
	hub.Register(client)
	hub.DisconnectWorkspace(wsID, nil)
	select {
	case <-client.Send:
		t.Fatal("nil payload should not send")
	default:
	}
}

func TestHub_DisconnectUserNilPayload(t *testing.T) {
	hub := NewHub()
	userID := uuid.New()
	client := newTestClient(t, hub, uuid.New(), uuid.New(), userID)
	hub.Register(client)
	hub.DisconnectUser(userID, nil)
	select {
	case <-client.Send:
		t.Fatal("nil payload should not send")
	default:
	}
}

func TestHub_DisconnectUserInWorkspaceNilPayload(t *testing.T) {
	hub := NewHub()
	wsID := uuid.New()
	userID := uuid.New()
	client := newTestClient(t, hub, uuid.New(), wsID, userID)
	hub.Register(client)
	hub.DisconnectUserInWorkspace(wsID, userID, nil)
	select {
	case <-client.Send:
		t.Fatal("nil payload should not send")
	default:
	}
}
