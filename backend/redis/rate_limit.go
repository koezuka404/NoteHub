package redis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const (
	loginFailurePrefix = "auth:login:failures:"
	loginLockPrefix    = "auth:login:locked:"
	rateLimitPrefix    = "rate_limit:token_bucket:"
)

var loginFailureScript = goredis.NewScript(`
local failure_key = KEYS[1]
local lock_key = KEYS[2]
local max_failures = tonumber(ARGV[1])
local failure_window_seconds = tonumber(ARGV[2])
local lock_seconds = tonumber(ARGV[3])

if redis.call("EXISTS", lock_key) == 1 then
  return redis.call("TTL", lock_key)
end

local failures = redis.call("INCR", failure_key)
if failures == 1 then
  redis.call("EXPIRE", failure_key, failure_window_seconds)
end

if failures >= max_failures then
  redis.call("SET", lock_key, "1", "EX", lock_seconds)
  redis.call("DEL", failure_key)
  return lock_seconds
end

return 0
`)

var tokenBucketScript = goredis.NewScript(`
local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local refill_per_second = tonumber(ARGV[2])
local now_ms = tonumber(ARGV[3])
local requested = tonumber(ARGV[4])
local ttl_ms = tonumber(ARGV[5])

local values = redis.call("HMGET", key, "tokens", "updated_at")
local tokens = tonumber(values[1])
local updated_at = tonumber(values[2])

if tokens == nil then
  tokens = capacity
  updated_at = now_ms
else
  local elapsed_seconds = math.max(0, now_ms - updated_at) / 1000
  tokens = math.min(capacity, tokens + elapsed_seconds * refill_per_second)
end

local allowed = 0
local retry_after_ms = 0
if tokens >= requested then
  tokens = tokens - requested
  allowed = 1
else
  retry_after_ms = math.ceil((requested - tokens) / refill_per_second * 1000)
end

redis.call("HSET", key, "tokens", tokens, "updated_at", now_ms)
redis.call("PEXPIRE", key, ttl_ms)
return {allowed, retry_after_ms}
`)

type LoginFailureStore struct {
	client        *Client
	maxFailures   int
	failureWindow time.Duration
	lockDuration  time.Duration
}

func NewLoginFailureStore(client *Client, maxFailures int, failureWindow, lockDuration time.Duration) *LoginFailureStore {
	return &LoginFailureStore{
		client: client, maxFailures: maxFailures,
		failureWindow: failureWindow, lockDuration: lockDuration,
	}
}

func (s *LoginFailureStore) IsLocked(ctx context.Context, email string) (bool, time.Duration, error) {
	key := loginLockPrefix + hashKey(strings.ToLower(strings.TrimSpace(email)))
	ctx, cancel := s.client.withTimeout(ctx)
	defer cancel()
	ttl, err := s.client.client.TTL(ctx, key).Result()
	if err != nil {
		return false, 0, fmt.Errorf("read login lock: %w", err)
	}
	if ttl > 0 {
		return true, ttl, nil
	}
	return false, 0, nil
}

func (s *LoginFailureStore) RecordFailure(ctx context.Context, email string) (bool, time.Duration, error) {
	hash := hashKey(strings.ToLower(strings.TrimSpace(email)))
	ctx, cancel := s.client.withTimeout(ctx)
	defer cancel()
	result, err := loginFailureScript.Run(ctx, s.client.client,
		[]string{loginFailurePrefix + hash, loginLockPrefix + hash},
		s.maxFailures,
		int64(s.failureWindow/time.Second),
		int64(s.lockDuration/time.Second),
	).Int64()
	if err != nil {
		return false, 0, fmt.Errorf("record login failure: %w", err)
	}
	if result > 0 {
		return true, time.Duration(result) * time.Second, nil
	}
	return false, 0, nil
}

func (s *LoginFailureStore) Reset(ctx context.Context, email string) error {
	hash := hashKey(strings.ToLower(strings.TrimSpace(email)))
	ctx, cancel := s.client.withTimeout(ctx)
	defer cancel()
	if err := s.client.client.Del(ctx, loginFailurePrefix+hash, loginLockPrefix+hash).Err(); err != nil {
		return fmt.Errorf("reset login failures: %w", err)
	}
	return nil
}

type TokenBucketStore struct {
	client *Client
}

func NewTokenBucketStore(client *Client) *TokenBucketStore {
	return &TokenBucketStore{client: client}
}

func (s *TokenBucketStore) Allow(ctx context.Context, key string, capacity int, refillPerSecond float64, now time.Time) (bool, time.Duration, error) {
	if capacity <= 0 || refillPerSecond <= 0 {
		return false, 0, fmt.Errorf("invalid token bucket configuration")
	}

	result, err := runTokenBucketScriptFn(ctx, s, key, capacity, refillPerSecond, now)
	if err != nil {
		return false, 0, fmt.Errorf("apply token bucket: %w", err)
	}
	if len(result) != 2 {
		return false, 0, fmt.Errorf("unexpected token bucket result")
	}
	allowed, err := int64Value(result[0])
	if err != nil {
		return false, 0, err
	}
	retryMS, err := int64Value(result[1])
	if err != nil {
		return false, 0, err
	}
	return allowed == 1, time.Duration(retryMS) * time.Millisecond, nil
}

func init() {
	runTokenBucketScriptFn = func(
		ctx context.Context,
		store *TokenBucketStore,
		key string,
		capacity int,
		refillPerSecond float64,
		now time.Time,
	) ([]any, error) {
		bucketTTL := time.Duration(math.Ceil(float64(capacity)/refillPerSecond*2)) * time.Second
		if bucketTTL < time.Minute {
			bucketTTL = time.Minute
		}
		redisKey := rateLimitPrefix + hashKey(key)
		ctx, cancel := store.client.withTimeout(ctx)
		defer cancel()
		return tokenBucketScript.Run(ctx, store.client.client, []string{redisKey},
			capacity,
			strconv.FormatFloat(refillPerSecond, 'f', -1, 64),
			now.UnixMilli(),
			1,
			bucketTTL.Milliseconds(),
		).Slice()
	}
}

func hashKey(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func int64Value(value any) (int64, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case string:
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("parse redis integer: %w", err)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("unexpected redis value type %T", value)
	}
}
