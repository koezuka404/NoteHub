package redis

import (
	"context"
	"testing"
	"time"
)

func TestLockStore(t *testing.T) {
	client, _ := newTestClient(t)
	store := NewLockStore(client)
	ctx := context.Background()
	key := DocumentAutosaveLockKey("doc-1")

	if ok, err := store.TryLock(ctx, "", time.Second); err == nil || ok {
		t.Fatal("expected invalid key error")
	}
	if ok, err := store.TryLock(ctx, key, 0); err == nil || ok {
		t.Fatal("expected invalid ttl error")
	}

	ok, err := store.TryLock(ctx, key, time.Minute)
	if err != nil || !ok {
		t.Fatalf("TryLock() = %v, %v", ok, err)
	}
	ok, err = store.TryLock(ctx, key, time.Minute)
	if err != nil || ok {
		t.Fatalf("second TryLock() = %v, %v", ok, err)
	}

	if err := store.Unlock(ctx, ""); err != nil {
		t.Fatalf("Unlock empty key: %v", err)
	}
	if err := store.Unlock(ctx, key); err != nil {
		t.Fatalf("Unlock: %v", err)
	}
}

func TestLockStore_Errors(t *testing.T) {
	client, mr := newTestClient(t)
	store := NewLockStore(client)
	ctx := context.Background()
	mr.Close()

	if _, err := store.TryLock(ctx, "lock:test", time.Minute); err == nil {
		t.Fatal("expected try lock error")
	}
	if err := store.Unlock(ctx, "lock:test"); err == nil {
		t.Fatal("expected unlock error")
	}
}

func TestDocumentAutosaveLockKey(t *testing.T) {
	if got := DocumentAutosaveLockKey("abc"); got != "lock:document:abc:autosave" {
		t.Fatalf("key = %q", got)
	}
}
