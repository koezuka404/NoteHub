package redis

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/usecase"
	goredis "github.com/redis/go-redis/v9"
)

var runTokenBucketScriptFn func(
	ctx context.Context,
	store *TokenBucketStore,
	key string,
	capacity int,
	refillPerSecond float64,
	now time.Time,
) ([]any, error)

var runTryAddWebSocketConnectionScriptFn func(
	ctx context.Context,
	store *WebSocketSessionStore,
	documentID, userID, connectionID uuid.UUID,
	maxConnections int,
	ttl time.Duration,
) ([]int64, error)

var expireEditorsKeyFn = func(client *goredis.Client, ctx context.Context, key string, ttl time.Duration) error {
	return client.Expire(ctx, key, ttl).Err()
}

var readCleanupKeyTTLFn = func(client *goredis.Client, ctx context.Context, key string) (time.Duration, error) {
	return client.TTL(ctx, key).Result()
}

var deleteCleanupKeyFn = func(client *goredis.Client, ctx context.Context, key string) error {
	return client.Del(ctx, key).Err()
}

var countCleanupSetMembersFn = func(client *goredis.Client, ctx context.Context, key string) (int64, error) {
	return client.SCard(ctx, key).Result()
}

var listDocumentEditorsFn = func(s *DocumentEditorsStore, ctx context.Context, documentID uuid.UUID) ([]usecase.DocumentEditorInfo, error) {
	return s.List(ctx, documentID)
}

var countEditorHashFieldsFn = func(client *goredis.Client, ctx context.Context, key string) (int64, error) {
	return client.HLen(ctx, key).Result()
}

var deleteEditorsKeyFn = func(client *goredis.Client, ctx context.Context, key string) error {
	return client.Del(ctx, key).Err()
}

var countWebSocketConnectionsFn = func(client *goredis.Client, ctx context.Context, key string) (int64, error) {
	return client.SCard(ctx, key).Result()
}

var appendUniqueDirtyDocumentIDFn = func(ids []uuid.UUID, seen map[uuid.UUID]struct{}, documentID uuid.UUID) []uuid.UUID {
	if _, ok := seen[documentID]; ok {
		return ids
	}
	seen[documentID] = struct{}{}
	return append(ids, documentID)
}
