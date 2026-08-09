package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/usecase"
	goredis "github.com/redis/go-redis/v9"
)

type DocumentEditorsStore struct {
	client      *goredis.Client
	withTimeout func(context.Context) (context.Context, context.CancelFunc)
	keyTTL      time.Duration
}

func NewDocumentEditorsStore(client *Client, keyTTL time.Duration) *DocumentEditorsStore {
	if keyTTL <= 0 {
		keyTTL = 16 * time.Minute
	}
	return &DocumentEditorsStore{client: client.client, withTimeout: client.withTimeout, keyTTL: keyTTL}
}

func (s *DocumentEditorsStore) List(ctx context.Context, documentID uuid.UUID) ([]usecase.DocumentEditorInfo, error) {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	values, err := s.client.HGetAll(ctx, editorsKey(documentID)).Result()
	if err != nil {
		return nil, fmt.Errorf("get document editors: %w", err)
	}
	editors := make([]usecase.DocumentEditorInfo, 0, len(values))
	for userIDRaw, name := range values {
		userID, err := uuid.Parse(userIDRaw)
		if err != nil {
			continue
		}
		editors = append(editors, usecase.DocumentEditorInfo{UserID: userID, Name: name})
	}
	return editors, nil
}

func (s *DocumentEditorsStore) Add(ctx context.Context, documentID uuid.UUID, editor usecase.DocumentEditorInfo) (bool, []usecase.DocumentEditorInfo, error) {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	key := editorsKey(documentID)
	added, err := s.client.HSet(ctx, key, editor.UserID.String(), editor.Name).Result()
	if err != nil {
		return false, nil, fmt.Errorf("add document editor: %w", err)
	}
	if err := expireEditorsKeyFn(s.client, ctx, key, s.keyTTL); err != nil {
		return false, nil, fmt.Errorf("refresh document editors ttl: %w", err)
	}

	editors, err := listDocumentEditorsFn(s, ctx, documentID)
	if err != nil {
		return false, nil, err
	}
	return added > 0, editors, nil
}

func (s *DocumentEditorsStore) Remove(ctx context.Context, documentID, userID uuid.UUID) (bool, []usecase.DocumentEditorInfo, error) {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	key := editorsKey(documentID)
	removed, err := s.client.HDel(ctx, key, userID.String()).Result()
	if err != nil {
		return false, nil, fmt.Errorf("remove document editor: %w", err)
	}
	if removed == 0 {
		editors, err := listDocumentEditorsFn(s, ctx, documentID)
		return false, editors, err
	}

	remaining, err := countEditorHashFieldsFn(s.client, ctx, key)
	if err != nil {
		return false, nil, fmt.Errorf("count document editors: %w", err)
	}
	if remaining == 0 {
		if err := deleteEditorsKeyFn(s.client, ctx, key); err != nil {
			return false, nil, fmt.Errorf("clear document editors: %w", err)
		}
		return true, nil, nil
	}
	if err := expireEditorsKeyFn(s.client, ctx, key, s.keyTTL); err != nil {
		return false, nil, fmt.Errorf("refresh document editors ttl: %w", err)
	}
	editors, err := listDocumentEditorsFn(s, ctx, documentID)
	if err != nil {
		return true, nil, err
	}
	return true, editors, nil
}

func (s *DocumentEditorsStore) RefreshTTL(ctx context.Context, documentID uuid.UUID) error {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	key := editorsKey(documentID)
	n, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("check document editors key: %w", err)
	}
	if n == 0 {
		return nil
	}
	if err := expireEditorsKeyFn(s.client, ctx, key, s.keyTTL); err != nil {
		return fmt.Errorf("refresh document editors ttl: %w", err)
	}
	return nil
}
