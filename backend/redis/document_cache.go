package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/usecase"
	goredis "github.com/redis/go-redis/v9"
)

const (
	documentContentKeyPrefix     = "document:"
	documentContentKeySuffix     = ":content"
	documentAutosaveKeySuffix    = ":autosave"
	documentRevisionKeySuffix    = ":revision"
	documentEditorsKeySuffix     = ":editors"
	documentConnectionsKeySuffix = ":connections"
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
	Incr(ctx context.Context, key string) *goredis.IntCmd
}

type DocumentCacheStore struct {
	commands    IDocumentCacheCommands
	client      *goredis.Client
	withTimeout func(context.Context) (context.Context, context.CancelFunc)
}

func NewDocumentCacheStore(client *Client) *DocumentCacheStore {
	return &DocumentCacheStore{
		commands:    client.client,
		client:      client.client,
		withTimeout: client.withTimeout,
	}
}

func (s *DocumentCacheStore) GetContentState(ctx context.Context, documentID uuid.UUID) (usecase.DocumentContentState, bool, error) {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	raw, err := s.commands.Get(ctx, contentKey(documentID)).Result()
	if err == goredis.Nil {
		return usecase.DocumentContentState{}, false, nil
	}
	if err != nil {
		return usecase.DocumentContentState{}, false, fmt.Errorf("get document content cache: %w", err)
	}

	var value contentCacheValue
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return usecase.DocumentContentState{}, false, fmt.Errorf("decode document content cache: %w", err)
	}
	updatedBy, err := uuid.Parse(value.UpdatedBy)
	if err != nil {
		return usecase.DocumentContentState{}, false, fmt.Errorf("decode document content cache updated_by: %w", err)
	}
	updatedAt, err := time.Parse(time.RFC3339, value.UpdatedAt)
	if err != nil {
		return usecase.DocumentContentState{}, false, fmt.Errorf("decode document content cache updated_at: %w", err)
	}
	return usecase.DocumentContentState{
		Content:   value.Content,
		UpdatedBy: updatedBy,
		UpdatedAt: updatedAt.UTC(),
	}, true, nil
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
	if err := s.commands.Incr(ctx, revisionKey(documentID)).Err(); err != nil {
		return fmt.Errorf("increment document revision: %w", err)
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

func (s *DocumentCacheStore) GetRevision(ctx context.Context, documentID uuid.UUID) (uint64, error) {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	value, err := s.commands.Get(ctx, revisionKey(documentID)).Int64()
	if err == goredis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get document revision: %w", err)
	}
	if value < 0 {
		return 0, nil
	}
	return uint64(value), nil
}

func (s *DocumentCacheStore) ListDirtyDocumentIDs(ctx context.Context) ([]uuid.UUID, error) {
	return s.listDirtyDocumentIDs(ctx, 0)
}

func (s *DocumentCacheStore) ListIdleDirtyDocumentIDs(ctx context.Context, idle time.Duration) ([]uuid.UUID, error) {
	if idle <= 0 {
		return s.ListDirtyDocumentIDs(ctx)
	}
	return s.listDirtyDocumentIDs(ctx, idle)
}

func (s *DocumentCacheStore) listDirtyDocumentIDs(ctx context.Context, idle time.Duration) ([]uuid.UUID, error) {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	pattern := documentContentKeyPrefix + "*" + documentAutosaveKeySuffix
	var ids []uuid.UUID
	seen := make(map[uuid.UUID]struct{})
	now := time.Now().UTC()

	iter := s.client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		documentID, err := parseDocumentIDFromAutosaveKey(key)
		if err != nil {
			continue
		}
		raw, err := s.commands.Get(ctx, key).Result()
		if err == goredis.Nil {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("get document autosave cache: %w", err)
		}
		var value autosaveCacheValue
		if err := json.Unmarshal([]byte(raw), &value); err != nil {
			continue
		}
		if !value.Dirty {
			continue
		}
		if idle > 0 {
			updatedAt, err := time.Parse(time.RFC3339, value.UpdatedAt)
			if err != nil {
				continue
			}
			if now.Sub(updatedAt.UTC()) < idle {
				continue
			}
		}
		if _, ok := seen[documentID]; ok {
			continue
		}
		seen[documentID] = struct{}{}
		ids = append(ids, documentID)
	}
	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("scan dirty document ids: %w", err)
	}
	return ids, nil
}

func (s *DocumentCacheStore) Clear(ctx context.Context, documentID uuid.UUID) error {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	if err := s.commands.Del(ctx,
		contentKey(documentID),
		autosaveKey(documentID),
		revisionKey(documentID),
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

func revisionKey(documentID uuid.UUID) string {
	return documentContentKeyPrefix + documentID.String() + documentRevisionKeySuffix
}

func parseDocumentIDFromAutosaveKey(key string) (uuid.UUID, error) {
	prefix := documentContentKeyPrefix
	suffix := documentAutosaveKeySuffix
	if !strings.HasPrefix(key, prefix) || !strings.HasSuffix(key, suffix) {
		return uuid.Nil, fmt.Errorf("invalid autosave key")
	}
	raw := strings.TrimSuffix(strings.TrimPrefix(key, prefix), suffix)
	return uuid.Parse(raw)
}

func editorsKey(documentID uuid.UUID) string {
	return documentContentKeyPrefix + documentID.String() + documentEditorsKeySuffix
}

func connectionsKey(documentID uuid.UUID) string {
	return documentContentKeyPrefix + documentID.String() + documentConnectionsKeySuffix
}
