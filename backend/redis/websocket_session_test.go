package redis

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestWebSocketSessionStore(t *testing.T) {
	client, _ := newTestClient(t)
	store := NewWebSocketSessionStore(client)
	ctx := context.Background()
	docID := uuid.New()
	userID := uuid.New()
	connA := uuid.New()
	connB := uuid.New()

	if _, _, err := store.TryAddConnection(ctx, docID, userID, connA, 0, time.Minute); err == nil {
		t.Fatal("expected max connections error")
	}

	count, added, err := store.TryAddConnection(ctx, docID, userID, connA, 1, 500*time.Millisecond)
	if err != nil || !added || count != 1 {
		t.Fatalf("TryAddConnection first = %d, %v, %v", count, added, err)
	}

	count, added, err = store.TryAddConnection(ctx, docID, userID, connB, 1, time.Minute)
	if err != nil || added || count != 1 {
		t.Fatalf("TryAddConnection full = %d, %v, %v", count, added, err)
	}

	count, err = store.RemoveConnection(ctx, docID, userID, connA)
	if err != nil || count != 0 {
		t.Fatalf("RemoveConnection = %d, %v", count, err)
	}
}

func TestWebSocketSessionStore_UnexpectedResult(t *testing.T) {
	orig := runTryAddWebSocketConnectionScriptFn
	runTryAddWebSocketConnectionScriptFn = func(context.Context, *WebSocketSessionStore, uuid.UUID, uuid.UUID, uuid.UUID, int, time.Duration) ([]int64, error) {
		return []int64{1}, nil
	}
	t.Cleanup(func() { runTryAddWebSocketConnectionScriptFn = orig })

	client, _ := newTestClient(t)
	store := NewWebSocketSessionStore(client)
	if _, _, err := store.TryAddConnection(context.Background(), uuid.New(), uuid.New(), uuid.New(), 2, time.Second); err == nil {
		t.Fatal("expected unexpected result error")
	}
}

func TestWebSocketSessionStore_Errors(t *testing.T) {
	client, _ := newTestClient(t)
	store := NewWebSocketSessionStore(client)
	client.client.Close()
	ctx := context.Background()
	docID := uuid.New()
	userID := uuid.New()
	connID := uuid.New()

	if _, _, err := store.TryAddConnection(ctx, docID, userID, connID, 2, time.Second); err == nil {
		t.Fatal("expected try add error")
	}
	if _, err := store.RemoveConnection(ctx, docID, userID, connID); err == nil {
		t.Fatal("expected remove error")
	}
}

func TestIsWebSocketConnectionsKey(t *testing.T) {
	key := websocketUserConnectionsKey(uuid.New(), uuid.New())
	if !isWebSocketConnectionsKey(key) {
		t.Fatalf("expected websocket key %q", key)
	}
	if isWebSocketConnectionsKey("other:key") {
		t.Fatal("expected non websocket key")
	}
}
