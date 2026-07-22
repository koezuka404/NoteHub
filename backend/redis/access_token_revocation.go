package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

const accessTokenRevocationPrefix = "auth:access_token:revoked:"

type revocationCommands interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *goredis.StatusCmd
	Get(ctx context.Context, key string) *goredis.StringCmd
}

type AccessTokenRevocationStore struct {
	commands    revocationCommands
	withTimeout func(context.Context) (context.Context, context.CancelFunc)
}

func NewAccessTokenRevocationStore(client *Client) *AccessTokenRevocationStore {
	return &AccessTokenRevocationStore{commands: client.client, withTimeout: client.withTimeout}
}

func (s *AccessTokenRevocationStore) Revoke(ctx context.Context, jti uuid.UUID, ttl time.Duration) error {
	if jti == uuid.Nil || ttl <= 0 {
		return nil
	}
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()
	if err := s.commands.Set(ctx, accessTokenRevocationPrefix+jti.String(), "1", ttl).Err(); err != nil {
		return fmt.Errorf("store revoked access token: %w", err)
	}
	return nil
}

func (s *AccessTokenRevocationStore) IsRevoked(ctx context.Context, jti uuid.UUID) (bool, error) {
	if jti == uuid.Nil {
		return false, nil
	}
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()
	_, err := s.commands.Get(ctx, accessTokenRevocationPrefix+jti.String()).Result()
	if err == nil {
		return true, nil
	}
	if err == goredis.Nil {
		return false, nil
	}
	return false, fmt.Errorf("read revoked access token: %w", err)
}
