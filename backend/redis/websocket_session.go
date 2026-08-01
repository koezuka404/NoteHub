package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

const websocketConnectionKeyPrefix = "ws:document:"

var tryAddWebSocketConnectionScript = goredis.NewScript(`
local count = redis.call('SCARD', KEYS[1])
if count >= tonumber(ARGV[2]) then
  return {0, count}
end
redis.call('SADD', KEYS[1], ARGV[1])
local ttl = tonumber(ARGV[3])
if ttl > 0 then
  redis.call('EXPIRE', KEYS[1], ttl)
end
count = redis.call('SCARD', KEYS[1])
return {1, count}
`)

type WebSocketSessionStore struct {
	client      *goredis.Client
	withTimeout func(context.Context) (context.Context, context.CancelFunc)
}

func NewWebSocketSessionStore(client *Client) *WebSocketSessionStore {
	return &WebSocketSessionStore{client: client.client, withTimeout: client.withTimeout}
}

func (s *WebSocketSessionStore) TryAddConnection(
	ctx context.Context,
	documentID, userID, connectionID uuid.UUID,
	maxConnections int,
	ttl time.Duration,
) (count int, added bool, err error) {
	if maxConnections < 1 {
		return 0, false, fmt.Errorf("max connections must be at least 1")
	}
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	ttlSeconds := int64(ttl.Seconds())
	if ttlSeconds < 1 {
		ttlSeconds = 1
	}

	key := websocketUserConnectionsKey(documentID, userID)
	result, err := tryAddWebSocketConnectionScript.Run(
		ctx,
		s.client,
		[]string{key},
		connectionID.String(),
		maxConnections,
		ttlSeconds,
	).Int64Slice()
	if err != nil {
		return 0, false, fmt.Errorf("try add websocket connection: %w", err)
	}
	if len(result) != 2 {
		return 0, false, fmt.Errorf("unexpected try add websocket connection result")
	}
	return int(result[1]), result[0] == 1, nil
}

func (s *WebSocketSessionStore) RemoveConnection(ctx context.Context, documentID, userID, connectionID uuid.UUID) (int, error) {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	key := websocketUserConnectionsKey(documentID, userID)
	if err := s.client.SRem(ctx, key, connectionID.String()).Err(); err != nil {
		return 0, fmt.Errorf("remove websocket connection: %w", err)
	}
	count, err := s.client.SCard(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("count websocket connections after remove: %w", err)
	}
	return int(count), nil
}

func websocketUserConnectionsKey(documentID, userID uuid.UUID) string {
	return websocketConnectionKeyPrefix + documentID.String() + ":user:" + userID.String() + ":connections"
}
