package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

type LockStore struct {
	client      *goredis.Client
	withTimeout func(context.Context) (context.Context, context.CancelFunc)
}

func NewLockStore(client *Client) *LockStore {
	return &LockStore{client: client.client, withTimeout: client.withTimeout}
}

func (s *LockStore) TryLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	if key == "" || ttl <= 0 {
		return false, fmt.Errorf("lock key and ttl are required")
	}
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	ok, err := s.client.SetNX(ctx, key, "1", ttl).Result()
	if err != nil {
		return false, fmt.Errorf("acquire lock: %w", err)
	}
	return ok, nil
}

func (s *LockStore) Unlock(ctx context.Context, key string) error {
	if key == "" {
		return nil
	}
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	if err := s.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("release lock: %w", err)
	}
	return nil
}

func DocumentAutosaveLockKey(documentID string) string {
	return "lock:document:" + documentID + ":autosave"
}
