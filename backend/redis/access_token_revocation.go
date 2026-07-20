package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

const accessTokenRevocationPrefix = "auth:access_token:revoked:"

type AccessTokenRevocationStore struct {
	client *Client
}

func NewAccessTokenRevocationStore(client *Client) *AccessTokenRevocationStore {
	return &AccessTokenRevocationStore{client: client}
}

func (s *AccessTokenRevocationStore) Revoke(ctx context.Context, jti uuid.UUID, ttl time.Duration) error {
	if jti == uuid.Nil || ttl <= 0 {
		return nil
	}
	if err := s.client.client.Set(ctx, accessTokenRevocationPrefix+jti.String(), "1", ttl).Err(); err != nil {
		return fmt.Errorf("store revoked access token: %w", err)
	}
	return nil
}

func (s *AccessTokenRevocationStore) IsRevoked(ctx context.Context, jti uuid.UUID) (bool, error) {
	if jti == uuid.Nil {
		return false, nil
	}
	_, err := s.client.client.Get(ctx, accessTokenRevocationPrefix+jti.String()).Result()
	if err == nil {
		return true, nil
	}
	if err == goredis.Nil {
		return false, nil
	}
	return false, fmt.Errorf("read revoked access token: %w", err)
}
