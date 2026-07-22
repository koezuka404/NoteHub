package redis

import (
	"context"
	"fmt"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const defaultOperationTimeout = 2 * time.Second

type Client struct {
	client           *goredis.Client
	operationTimeout time.Duration
}

func NewClient(rawURL string) (*Client, error) {
	return NewClientWithTimeout(rawURL, defaultOperationTimeout)
}

func NewClientWithTimeout(rawURL string, operationTimeout time.Duration) (*Client, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, fmt.Errorf("REDIS_URL is required")
	}
	if operationTimeout <= 0 {
		return nil, fmt.Errorf("redis operation timeout must be greater than 0")
	}
	options, err := goredis.ParseURL(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	return &Client{client: goredis.NewClient(options), operationTimeout: operationTimeout}, nil
}

func (c *Client) Ping(ctx context.Context) error {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	if err := c.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping: %w", err)
	}
	return nil
}

func (c *Client) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, c.operationTimeout)
}

func (c *Client) Close() error { return c.client.Close() }
