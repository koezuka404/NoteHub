package redis

import (
	"context"
	"fmt"

	goredis "github.com/redis/go-redis/v9"
)

var cleanupKeyPatterns = []string{
	"lock:document:*",
	"ws:document:*:connections",
	"auth:login:failures:*",
	"auth:login:locked:*",
	"rate_limit:token_bucket:*",
}

type CleanupStore struct {
	client      *goredis.Client
	withTimeout func(context.Context) (context.Context, context.CancelFunc)
}

func NewCleanupStore(client *Client) *CleanupStore {
	return &CleanupStore{client: client.client, withTimeout: client.withTimeout}
}

func (s *CleanupStore) CleanupEphemeralKeys(ctx context.Context) (int, error) {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	removed := 0
	for _, pattern := range cleanupKeyPatterns {
		count, err := s.cleanupPattern(ctx, pattern)
		if err != nil {
			return removed, err
		}
		removed += count
	}
	return removed, nil
}

func (s *CleanupStore) cleanupPattern(ctx context.Context, pattern string) (int, error) {
	removed := 0
	iter := s.client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		ttl, err := readCleanupKeyTTLFn(s.client, ctx, key)
		if err != nil {
			return removed, fmt.Errorf("read ttl for %s: %w", key, err)
		}
		if ttl == -1 {
			if err := deleteCleanupKeyFn(s.client, ctx, key); err != nil {
				return removed, fmt.Errorf("delete key without ttl %s: %w", key, err)
			}
			removed++
			continue
		}
		if ttl == -2 {
			continue
		}
		if !isWebSocketConnectionsKey(key) {
			continue
		}
		count, err := countCleanupSetMembersFn(s.client, ctx, key)
		if err != nil {
			return removed, fmt.Errorf("count websocket connections for %s: %w", key, err)
		}
		if count == 0 {
			if err := deleteCleanupKeyFn(s.client, ctx, key); err != nil {
				return removed, fmt.Errorf("delete empty websocket key %s: %w", key, err)
			}
			removed++
		}
	}
	if err := iter.Err(); err != nil {
		return removed, fmt.Errorf("scan %s: %w", pattern, err)
	}
	return removed, nil
}

func isWebSocketConnectionsKey(key string) bool {
	return len(key) > len(websocketConnectionKeyPrefix) &&
		key[:len(websocketConnectionKeyPrefix)] == websocketConnectionKeyPrefix &&
		len(key) >= len(":connections") &&
		key[len(key)-len(":connections"):] == ":connections"
}
