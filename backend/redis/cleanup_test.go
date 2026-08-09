package redis

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestCleanupStore(t *testing.T) {
	client, mr := newTestClient(t)
	store := NewCleanupStore(client)
	ctx := context.Background()

	mr.Set("lock:document:abc:autosave", "1")
	mr.Set("auth:login:failures:abc", "1")
	mr.Set("auth:login:locked:abc", "1")
	mr.Set("rate_limit:token_bucket:abc", "1")

	wsKey := websocketUserConnectionsKey(uuid.New(), uuid.New())
	mr.Set(wsKey, "1")
	mr.SAdd(wsKey)

	removed, err := store.CleanupEphemeralKeys(ctx)
	if err != nil {
		t.Fatalf("CleanupEphemeralKeys: %v", err)
	}
	if removed == 0 {
		t.Fatal("expected removed keys")
	}
}

func TestCleanupStore_WebSocketEmptySet(t *testing.T) {
	client, mr := newTestClient(t)
	store := NewCleanupStore(client)
	wsKey := websocketUserConnectionsKey(uuid.New(), uuid.New())
	mr.Set(wsKey, "1")

	removed, err := store.CleanupEphemeralKeys(context.Background())
	if err != nil {
		t.Fatalf("CleanupEphemeralKeys: %v", err)
	}
	if removed == 0 {
		t.Fatal("expected empty websocket key removed")
	}
}

func TestCleanupStore_Error(t *testing.T) {
	client, _ := newTestClient(t)
	store := NewCleanupStore(client)
	client.client.Close()
	if _, err := store.CleanupEphemeralKeys(context.Background()); err == nil {
		t.Fatal("expected cleanup error")
	}
}

func TestIsWebSocketConnectionsKey_PrefixOnly(t *testing.T) {
	if isWebSocketConnectionsKey("ws:document:abc") {
		t.Fatal("expected false without connections suffix")
	}
}
