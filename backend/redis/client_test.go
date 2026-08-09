package redis

import (
	"context"
	"testing"
	"time"
)

func TestNewClientWithTimeout_Validation(t *testing.T) {
	if _, err := NewClientWithTimeout("", time.Second); err == nil {
		t.Fatal("expected empty url error")
	}
	if _, err := NewClientWithTimeout("redis://127.0.0.1:6379/0", 0); err == nil {
		t.Fatal("expected timeout error")
	}
	if _, err := NewClientWithTimeout("not-a-redis-url", time.Second); err == nil {
		t.Fatal("expected parse url error")
	}
}

func TestNewClient(t *testing.T) {
	_, mr := newTestClient(t)
	client, err := NewClient("redis://" + mr.Addr() + "/0")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_ = client.Close()
}

func TestClient_PingAndClose(t *testing.T) {
	client, mr := newTestClient(t)
	ctx := context.Background()
	if err := client.Ping(ctx); err != nil {
		t.Fatalf("Ping: %v", err)
	}

	deadlineCtx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	if err := client.Ping(deadlineCtx); err != nil {
		t.Fatalf("Ping with deadline: %v", err)
	}

	mr.Close()
	if err := client.Ping(ctx); err == nil {
		t.Fatal("expected ping error after miniredis closed")
	}
}
