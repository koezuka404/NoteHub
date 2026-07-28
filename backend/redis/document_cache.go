package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

const (
	documentContentKeyPrefix     = "document:"
	documentContentKeySuffix     = ":content"
	documentAutosaveKeySuffix      = ":autosave"
	documentEditorsKeySuffix       = ":editors"
	documentConnectionsKeySuffix   = ":connections"
)

type contentCacheValue struct {
	Content   string `json:"content"`
	UpdatedBy string `json:"updated_by"`
	UpdatedAt string `json:"updated_at"`
}

type autosaveCacheValue struct {
	DocumentID string `json:"document_id"`
	UpdatedBy  string `json:"updated_by"`
	UpdatedAt  string `json:"updated_at"`
	Dirty      bool   `json:"dirty"`
}

type IDocumentCacheCommands interface {
	Get(ctx context.Context, key string) *goredis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *goredis.StatusCmd
	Del(ctx context.Context, keys ...string) *goredis.IntCmd
}

type DocumentCacheStore struct {
	commands    IDocumentCacheCommands
	withTimeout func(context.Context) (context.Context, context.CancelFunc)
}

func NewDocumentCacheStore(client *Client) *DocumentCacheStore {
	return &DocumentCacheStore{commands: client.client, withTimeout: client.withTimeout}
}

func (s *DocumentCacheStore) GetContent(ctx context.Context, documentID uuid.UUID) (string, bool, error) {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	raw, err := s.commands.Get(ctx, contentKey(documentID)).Result()
	if err == goredis.Nil {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("get document content cache: %w", err)
	}

	var value contentCacheValue
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return "", false, fmt.Errorf("decode document content cache: %w", err)
	}
	return value.Content, true, nil
}

func (s *DocumentCacheStore) SetContent(ctx context.Context, documentID uuid.UUID, content string, updatedBy uuid.UUID, updatedAt time.Time) error {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	at := updatedAt.UTC().Format(time.RFC3339)
	contentRaw, err := json.Marshal(contentCacheValue{
		Content:   content,
		UpdatedBy: updatedBy.String(),
		UpdatedAt: at,
	})
	if err != nil {
		return fmt.Errorf("encode document content cache: %w", err)
	}
	autosaveRaw, err := json.Marshal(autosaveCacheValue{
		DocumentID: documentID.String(),
		UpdatedBy:  updatedBy.String(),
		UpdatedAt:  at,
		Dirty:      true,
	})
	if err != nil {
		return fmt.Errorf("encode document autosave cache: %w", err)
	}

	if err := s.commands.Set(ctx, contentKey(documentID), contentRaw, 0).Err(); err != nil {
		return fmt.Errorf("set document content cache: %w", err)
	}
	if err := s.commands.Set(ctx, autosaveKey(documentID), autosaveRaw, 0).Err(); err != nil {
		return fmt.Errorf("set document autosave cache: %w", err)
	}
	return nil
}

func (s *DocumentCacheStore) IsDirty(ctx context.Context, documentID uuid.UUID) (bool, error) {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	raw, err := s.commands.Get(ctx, autosaveKey(documentID)).Result()
	if err == goredis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("get document autosave cache: %w", err)
	}

	var value autosaveCacheValue
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return false, fmt.Errorf("decode document autosave cache: %w", err)
	}
	return value.Dirty, nil
}

func (s *DocumentCacheStore) MarkClean(ctx context.Context, documentID uuid.UUID) error {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	raw, err := s.commands.Get(ctx, autosaveKey(documentID)).Result()
	if err == goredis.Nil {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get document autosave cache: %w", err)
	}

	var value autosaveCacheValue
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return fmt.Errorf("decode document autosave cache: %w", err)
	}
	value.Dirty = false
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode document autosave cache: %w", err)
	}
	if err := s.commands.Set(ctx, autosaveKey(documentID), encoded, 0).Err(); err != nil {
		return fmt.Errorf("update document autosave cache: %w", err)
	}
	return nil
}

func (s *DocumentCacheStore) Clear(ctx context.Context, documentID uuid.UUID) error {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	if err := s.commands.Del(ctx,
		contentKey(documentID),
		autosaveKey(documentID),
		editorsKey(documentID),
		connectionsKey(documentID),
	).Err(); err != nil {
		return fmt.Errorf("clear document cache keys: %w", err)
	}
	return nil
}

func contentKey(documentID uuid.UUID) string {
	return documentContentKeyPrefix + documentID.String() + documentContentKeySuffix
}

func autosaveKey(documentID uuid.UUID) string {
	return documentContentKeyPrefix + documentID.String() + documentAutosaveKeySuffix
}

func editorsKey(documentID uuid.UUID) string {
	return documentContentKeyPrefix + documentID.String() + documentEditorsKeySuffix
}

func connectionsKey(documentID uuid.UUID) string {
	return documentContentKeyPrefix + documentID.String() + documentConnectionsKeySuffix
}
