package redis

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func newTestClient(t *testing.T) (*Client, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client, err := NewClientWithTimeout("redis://"+mr.Addr()+"/0", 2*time.Second)
	if err != nil {
		t.Fatalf("NewClientWithTimeout: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client, mr
}
