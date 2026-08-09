package redis

import (
	"context"
	"testing"
	"time"
)

func TestLoginFailureStore(t *testing.T) {
	client, _ := newTestClient(t)
	store := NewLoginFailureStore(client, 3, time.Minute, 5*time.Minute)
	ctx := context.Background()
	email := " user@example.com "

	locked, retryAfter, err := store.IsLocked(ctx, email)
	if err != nil || locked || retryAfter != 0 {
		t.Fatalf("IsLocked() = %v, %v, %v", locked, retryAfter, err)
	}

	for i := 0; i < 2; i++ {
		locked, _, err = store.RecordFailure(ctx, email)
		if err != nil || locked {
			t.Fatalf("RecordFailure attempt %d: locked=%v err=%v", i+1, locked, err)
		}
	}
	locked, retryAfter, err = store.RecordFailure(ctx, email)
	if err != nil || !locked || retryAfter <= 0 {
		t.Fatalf("RecordFailure lock: locked=%v retryAfter=%v err=%v", locked, retryAfter, err)
	}

	locked, retryAfter, err = store.IsLocked(ctx, email)
	if err != nil || !locked || retryAfter <= 0 {
		t.Fatalf("IsLocked after lock: locked=%v retryAfter=%v err=%v", locked, retryAfter, err)
	}

	if err := store.Reset(ctx, email); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	locked, _, err = store.IsLocked(ctx, email)
	if err != nil || locked {
		t.Fatalf("IsLocked after reset: locked=%v err=%v", locked, err)
	}
}

func TestTokenBucketStore_Allow(t *testing.T) {
	client, _ := newTestClient(t)
	store := NewTokenBucketStore(client)
	ctx := context.Background()
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)

	allowed, retryAfter, err := store.Allow(ctx, "user:1", 2, 1, now)
	if err != nil || !allowed || retryAfter != 0 {
		t.Fatalf("first Allow() = %v, %v, %v", allowed, retryAfter, err)
	}

	allowed, retryAfter, err = store.Allow(ctx, "user:1", 2, 1, now)
	if err != nil || !allowed {
		t.Fatalf("second Allow() = %v, %v, %v", allowed, retryAfter, err)
	}

	allowed, retryAfter, err = store.Allow(ctx, "user:1", 2, 1, now)
	if err != nil || allowed || retryAfter <= 0 {
		t.Fatalf("third Allow() = %v, %v, %v", allowed, retryAfter, err)
	}
}

func TestTokenBucketStore_InvalidConfig(t *testing.T) {
	client, _ := newTestClient(t)
	store := NewTokenBucketStore(client)
	if _, _, err := store.Allow(context.Background(), "k", 0, 1, time.Now()); err == nil {
		t.Fatal("expected invalid capacity error")
	}
	if _, _, err := store.Allow(context.Background(), "k", 1, 0, time.Now()); err == nil {
		t.Fatal("expected invalid refill error")
	}
}

func TestHashKeyAndInt64Value(t *testing.T) {
	if hashKey("a") == hashKey("a") {
		if hashKey("a") == hashKey("b") {
			t.Fatal("expected different hash keys")
		}
	}

	if v, err := int64Value(int64(7)); err != nil || v != 7 {
		t.Fatalf("int64Value int64 = %v, %v", v, err)
	}
	if v, err := int64Value("9"); err != nil || v != 9 {
		t.Fatalf("int64Value string = %v, %v", v, err)
	}
	if _, err := int64Value("bad"); err == nil {
		t.Fatal("expected parse error")
	}
	if _, err := int64Value(3.14); err == nil {
		t.Fatal("expected type error")
	}
}

func TestTokenBucketStore_UnexpectedResult(t *testing.T) {
	orig := runTokenBucketScriptFn
	runTokenBucketScriptFn = func(context.Context, *TokenBucketStore, string, int, float64, time.Time) ([]any, error) {
		return []any{int64(1)}, nil
	}
	t.Cleanup(func() { runTokenBucketScriptFn = orig })

	client, _ := newTestClient(t)
	store := NewTokenBucketStore(client)
	if _, _, err := store.Allow(context.Background(), "k", 2, 1, time.Now()); err == nil {
		t.Fatal("expected unexpected result error")
	}
}

func TestLoginFailureStore_ReadError(t *testing.T) {
	client, _ := newTestClient(t)
	store := NewLoginFailureStore(client, 3, time.Minute, 5*time.Minute)
	client.client.Close()
	if _, _, err := store.IsLocked(context.Background(), "user@example.com"); err == nil {
		t.Fatal("expected read login lock error")
	}
}

func TestLoginFailureStore_ResetError(t *testing.T) {
	client, _ := newTestClient(t)
	store := NewLoginFailureStore(client, 3, time.Minute, 5*time.Minute)
	client.client.Close()
	if err := store.Reset(context.Background(), "user@example.com"); err == nil {
		t.Fatal("expected reset error")
	}
}
