package redis

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/koezuka404/notehub/usecase"
)

func TestSetContent_ErrorPaths(t *testing.T) {
	docID := uuid.New()
	ctx := context.Background()

	t.Run("autosave set error", func(t *testing.T) {
		calls := 0
		store := newDocumentCacheStore(&mockDocumentCacheCommands{
			setErrFn: func() error {
				calls++
				if calls == 2 {
					return errors.New("autosave set failed")
				}
				return nil
			},
		}, nil)
		if err := store.SetContent(ctx, docID, "x", uuid.New(), time.Now()); err == nil {
			t.Fatal("expected autosave set error")
		}
	})

	t.Run("increment error", func(t *testing.T) {
		store := newDocumentCacheStore(&mockDocumentCacheCommands{incrErr: errors.New("incr failed")}, nil)
		if err := store.SetContent(ctx, docID, "x", uuid.New(), time.Now()); err == nil {
			t.Fatal("expected increment error")
		}
	})

	t.Run("content encode error", func(t *testing.T) {
		orig := jsonMarshalDocumentCacheFn
		calls := 0
		jsonMarshalDocumentCacheFn = func(v any) ([]byte, error) {
			calls++
			if calls == 1 {
				return nil, errors.New("content encode failed")
			}
			return orig(v)
		}
		t.Cleanup(func() { jsonMarshalDocumentCacheFn = orig })

		client, _ := newTestClient(t)
		store := NewDocumentCacheStore(client)
		if err := store.SetContent(ctx, docID, "x", uuid.New(), time.Now()); err == nil {
			t.Fatal("expected content encode error")
		}
	})

	t.Run("autosave encode error", func(t *testing.T) {
		orig := jsonMarshalDocumentCacheFn
		calls := 0
		jsonMarshalDocumentCacheFn = func(v any) ([]byte, error) {
			calls++
			if calls == 2 {
				return nil, errors.New("autosave encode failed")
			}
			return orig(v)
		}
		t.Cleanup(func() { jsonMarshalDocumentCacheFn = orig })

		client, _ := newTestClient(t)
		store := NewDocumentCacheStore(client)
		if err := store.SetContent(ctx, docID, "x", uuid.New(), time.Now()); err == nil {
			t.Fatal("expected autosave encode error")
		}
	})
}

func TestIsDirty_GetError(t *testing.T) {
	store := newDocumentCacheStore(&mockDocumentCacheCommands{getErr: errors.New("get failed")}, nil)
	if _, err := store.IsDirty(context.Background(), uuid.New()); err == nil {
		t.Fatal("expected is dirty get error")
	}
}

func TestListDirtyDocumentIDs_ScanError(t *testing.T) {
	client, mr := newTestClient(t)
	store := NewDocumentCacheStore(client)
	mr.Close()
	if _, err := store.ListDirtyDocumentIDs(context.Background()); err == nil {
		t.Fatal("expected scan error")
	}
}

func TestDocumentEditorsStore_AddExpireError(t *testing.T) {
	orig := expireEditorsKeyFn
	expireEditorsKeyFn = func(*goredis.Client, context.Context, string, time.Duration) error {
		return errors.New("expire failed")
	}
	t.Cleanup(func() { expireEditorsKeyFn = orig })

	client, _ := newTestClient(t)
	store := NewDocumentEditorsStore(client, time.Minute)
	if _, _, err := store.Add(context.Background(), uuid.New(), usecase.DocumentEditorInfo{UserID: uuid.New(), Name: "A"}); err == nil {
		t.Fatal("expected add expire error")
	}
}

func TestDocumentEditorsStore_RemoveErrors(t *testing.T) {
	client, _ := newTestClient(t)
	store := NewDocumentEditorsStore(client, time.Minute)
	ctx := context.Background()

	t.Run("expire error", func(t *testing.T) {
		docID := uuid.New()
		userA := uuid.New()
		userB := uuid.New()
		if _, _, err := store.Add(ctx, docID, usecase.DocumentEditorInfo{UserID: userA, Name: "A"}); err != nil {
			t.Fatalf("Add A: %v", err)
		}
		if _, _, err := store.Add(ctx, docID, usecase.DocumentEditorInfo{UserID: userB, Name: "B"}); err != nil {
			t.Fatalf("Add B: %v", err)
		}
		orig := expireEditorsKeyFn
		expireEditorsKeyFn = func(*goredis.Client, context.Context, string, time.Duration) error {
			return errors.New("expire failed")
		}
		t.Cleanup(func() { expireEditorsKeyFn = orig })
		if _, _, err := store.Remove(ctx, docID, userA); err == nil {
			t.Fatal("expected remove expire error")
		}
	})

	t.Run("list error after remove", func(t *testing.T) {
		docID := uuid.New()
		userA := uuid.New()
		userB := uuid.New()
		if _, _, err := store.Add(ctx, docID, usecase.DocumentEditorInfo{UserID: userA, Name: "A"}); err != nil {
			t.Fatalf("Add A: %v", err)
		}
		if _, _, err := store.Add(ctx, docID, usecase.DocumentEditorInfo{UserID: userB, Name: "B"}); err != nil {
			t.Fatalf("Add B: %v", err)
		}
		orig := listDocumentEditorsFn
		listDocumentEditorsFn = func(*DocumentEditorsStore, context.Context, uuid.UUID) ([]usecase.DocumentEditorInfo, error) {
			return nil, errors.New("list failed")
		}
		t.Cleanup(func() { listDocumentEditorsFn = orig })
		if _, _, err := store.Remove(ctx, docID, userB); err == nil {
			t.Fatal("expected list error after remove")
		}
	})

	t.Run("count error", func(t *testing.T) {
		docID := uuid.New()
		userA := uuid.New()
		userB := uuid.New()
		if _, _, err := store.Add(ctx, docID, usecase.DocumentEditorInfo{UserID: userA, Name: "A"}); err != nil {
			t.Fatalf("Add A: %v", err)
		}
		if _, _, err := store.Add(ctx, docID, usecase.DocumentEditorInfo{UserID: userB, Name: "B"}); err != nil {
			t.Fatalf("Add B: %v", err)
		}
		orig := countEditorHashFieldsFn
		countEditorHashFieldsFn = func(*goredis.Client, context.Context, string) (int64, error) {
			return 0, errors.New("count failed")
		}
		t.Cleanup(func() { countEditorHashFieldsFn = orig })
		if _, _, err := store.Remove(ctx, docID, userA); err == nil {
			t.Fatal("expected count error")
		}
	})

	t.Run("delete error", func(t *testing.T) {
		singleDoc := uuid.New()
		singleUser := uuid.New()
		if _, _, err := store.Add(ctx, singleDoc, usecase.DocumentEditorInfo{UserID: singleUser, Name: "Only"}); err != nil {
			t.Fatalf("Add: %v", err)
		}
		orig := countEditorHashFieldsFn
		origDel := deleteEditorsKeyFn
		countEditorHashFieldsFn = func(*goredis.Client, context.Context, string) (int64, error) {
			return 0, nil
		}
		deleteEditorsKeyFn = func(*goredis.Client, context.Context, string) error {
			return errors.New("delete failed")
		}
		t.Cleanup(func() {
			countEditorHashFieldsFn = orig
			deleteEditorsKeyFn = origDel
		})
		if _, _, err := store.Remove(ctx, singleDoc, singleUser); err == nil {
			t.Fatal("expected delete error")
		}
	})
}

func TestDocumentEditorsStore_AddListError(t *testing.T) {
	orig := listDocumentEditorsFn
	listDocumentEditorsFn = func(*DocumentEditorsStore, context.Context, uuid.UUID) ([]usecase.DocumentEditorInfo, error) {
		return nil, errors.New("list failed")
	}
	t.Cleanup(func() { listDocumentEditorsFn = orig })

	client, _ := newTestClient(t)
	store := NewDocumentEditorsStore(client, time.Minute)
	if _, _, err := store.Add(context.Background(), uuid.New(), usecase.DocumentEditorInfo{UserID: uuid.New(), Name: "A"}); err == nil {
		t.Fatal("expected add list error")
	}
}

func TestDocumentEditorsStore_RemoveMissing(t *testing.T) {
	client, _ := newTestClient(t)
	store := NewDocumentEditorsStore(client, time.Minute)
	docID := uuid.New()
	removed, editors, err := store.Remove(context.Background(), docID, uuid.New())
	if err != nil || removed || len(editors) != 0 {
		t.Fatalf("Remove missing = %v, %v, %v", removed, editors, err)
	}
}

func TestIsDirty_NotDirty(t *testing.T) {
	raw, _ := json.Marshal(autosaveCacheValue{Dirty: false})
	store := newDocumentCacheStore(&mockDocumentCacheCommands{getVal: string(raw)}, nil)
	dirty, err := store.IsDirty(context.Background(), uuid.New())
	if err != nil || dirty {
		t.Fatalf("IsDirty() = %v, %v", dirty, err)
	}
}

func TestIsDirty_MissingKey(t *testing.T) {
	client, _ := newTestClient(t)
	store := NewDocumentCacheStore(client)
	dirty, err := store.IsDirty(context.Background(), uuid.New())
	if err != nil || dirty {
		t.Fatalf("IsDirty() = %v, %v", dirty, err)
	}
}

func TestListDirtyDocumentIDs_SkipsInvalidKeys(t *testing.T) {
	client, mr := newTestClient(t)
	store := NewDocumentCacheStore(client)
	ctx := context.Background()

	mr.Set("document:not-a-uuid:autosave", `{"dirty":true}`)

	docBadJSON := uuid.New()
	mr.Set(autosaveKey(docBadJSON), "not-json")

	docBadTime := uuid.New()
	mr.Set(autosaveKey(docBadTime), `{"document_id":"`+docBadTime.String()+`","updated_by":"`+uuid.New().String()+`","updated_at":"bad","dirty":true}`)

	docRecent := uuid.New()
	mr.Set(autosaveKey(docRecent), `{"document_id":"`+docRecent.String()+`","updated_by":"`+uuid.New().String()+`","updated_at":"`+time.Now().UTC().Format(time.RFC3339)+`","dirty":true}`)

	docDirty := uuid.New()
	mr.Set(autosaveKey(docDirty), `{"document_id":"`+docDirty.String()+`","updated_by":"`+uuid.New().String()+`","updated_at":"`+time.Now().UTC().Add(-2*time.Hour).Format(time.RFC3339)+`","dirty":true}`)

	docNilGet := uuid.New()
	mr.Set(autosaveKey(docNilGet), `{"document_id":"`+docNilGet.String()+`","updated_by":"`+uuid.New().String()+`","updated_at":"`+time.Now().UTC().Format(time.RFC3339)+`","dirty":true}`)
	store.commands = &nilOnGetDocumentCacheCommands{
		inner:   store.commands,
		nilKeys: map[string]struct{}{autosaveKey(docNilGet): {}},
	}

	ids, err := store.ListDirtyDocumentIDs(ctx)
	if err != nil {
		t.Fatalf("ListDirtyDocumentIDs: %v", err)
	}
	if len(ids) != 3 {
		t.Fatalf("ids = %v, want 3 dirty documents", ids)
	}

	ids, err = store.ListIdleDirtyDocumentIDs(ctx, time.Hour)
	if err != nil {
		t.Fatalf("ListIdleDirtyDocumentIDs: %v", err)
	}
	if len(ids) != 1 || ids[0] != docDirty {
		t.Fatalf("idle ids = %v, want [%v]", ids, docDirty)
	}
}

func TestListIdleDirtyDocumentIDs_ZeroIdle(t *testing.T) {
	client, mr := newTestClient(t)
	store := NewDocumentCacheStore(client)
	docID := uuid.New()
	mr.Set(autosaveKey(docID), `{"document_id":"`+docID.String()+`","updated_by":"`+uuid.New().String()+`","updated_at":"`+time.Now().UTC().Format(time.RFC3339)+`","dirty":true}`)

	ids, err := store.ListIdleDirtyDocumentIDs(context.Background(), 0)
	if err != nil {
		t.Fatalf("ListIdleDirtyDocumentIDs: %v", err)
	}
	if len(ids) != 1 || ids[0] != docID {
		t.Fatalf("ids = %v", ids)
	}
}

func TestAppendUniqueDirtyDocumentID_SkipsDuplicate(t *testing.T) {
	docID := uuid.New()
	seen := make(map[uuid.UUID]struct{})
	ids := appendUniqueDirtyDocumentIDFn(nil, seen, docID)
	ids = appendUniqueDirtyDocumentIDFn(ids, seen, docID)
	if len(ids) != 1 {
		t.Fatalf("ids = %v", ids)
	}
}

type nilOnGetDocumentCacheCommands struct {
	inner   IDocumentCacheCommands
	nilKeys map[string]struct{}
}

func (d *nilOnGetDocumentCacheCommands) Get(ctx context.Context, key string) *goredis.StringCmd {
	if _, ok := d.nilKeys[key]; ok {
		cmd := goredis.NewStringCmd(ctx, key)
		cmd.SetErr(goredis.Nil)
		return cmd
	}
	return d.inner.Get(ctx, key)
}

func (d *nilOnGetDocumentCacheCommands) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *goredis.StatusCmd {
	return d.inner.Set(ctx, key, value, expiration)
}

func (d *nilOnGetDocumentCacheCommands) Del(ctx context.Context, keys ...string) *goredis.IntCmd {
	return d.inner.Del(ctx, keys...)
}

func (d *nilOnGetDocumentCacheCommands) Incr(ctx context.Context, key string) *goredis.IntCmd {
	return d.inner.Incr(ctx, key)
}

func TestListDirtyDocumentIDs_SkipsNonDirty(t *testing.T) {
	client, mr := newTestClient(t)
	store := NewDocumentCacheStore(client)
	docID := uuid.New()
	mr.Set(autosaveKey(docID), `{"document_id":"`+docID.String()+`","updated_by":"`+uuid.New().String()+`","updated_at":"`+time.Now().UTC().Format(time.RFC3339)+`","dirty":false}`)

	ids, err := store.ListDirtyDocumentIDs(context.Background())
	if err != nil {
		t.Fatalf("ListDirtyDocumentIDs: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("ids = %v", ids)
	}
}

func TestDocumentEditorsStore_RefreshTTLExistsError(t *testing.T) {
	client, _ := newTestClient(t)
	store := NewDocumentEditorsStore(client, time.Minute)
	client.client.Close()
	if err := store.RefreshTTL(context.Background(), uuid.New()); err == nil {
		t.Fatal("expected exists error")
	}
}

func TestCleanupPattern_HookErrors(t *testing.T) {
	t.Run("ttl read error", func(t *testing.T) {
		client, mr := newTestClient(t)
		store := NewCleanupStore(client)
		mr.Set("lock:document:hook:autosave", "1")

		orig := readCleanupKeyTTLFn
		readCleanupKeyTTLFn = func(_ *goredis.Client, _ context.Context, _ string) (time.Duration, error) {
			return 0, errors.New("ttl failed")
		}
		t.Cleanup(func() { readCleanupKeyTTLFn = orig })
		if _, err := store.CleanupEphemeralKeys(context.Background()); err == nil {
			t.Fatal("expected ttl read error")
		}
	})

	t.Run("delete error", func(t *testing.T) {
		client, mr := newTestClient(t)
		store := NewCleanupStore(client)
		mr.Set("lock:document:del:autosave", "1")

		orig := readCleanupKeyTTLFn
		origDel := deleteCleanupKeyFn
		readCleanupKeyTTLFn = func(_ *goredis.Client, _ context.Context, _ string) (time.Duration, error) {
			return -1, nil
		}
		deleteCleanupKeyFn = func(_ *goredis.Client, _ context.Context, _ string) error {
			return errors.New("delete failed")
		}
		t.Cleanup(func() {
			readCleanupKeyTTLFn = orig
			deleteCleanupKeyFn = origDel
		})
		if _, err := store.CleanupEphemeralKeys(context.Background()); err == nil {
			t.Fatal("expected delete error")
		}
	})

	t.Run("scard error", func(t *testing.T) {
		client, mr := newTestClient(t)
		store := NewCleanupStore(client)
		wsKey := websocketUserConnectionsKey(uuid.New(), uuid.New())
		mr.SAdd(wsKey, "conn")
		mr.SetTTL(wsKey, time.Minute)

		orig := countCleanupSetMembersFn
		countCleanupSetMembersFn = func(_ *goredis.Client, _ context.Context, _ string) (int64, error) {
			return 0, errors.New("scard failed")
		}
		t.Cleanup(func() { countCleanupSetMembersFn = orig })
		if _, err := store.CleanupEphemeralKeys(context.Background()); err == nil {
			t.Fatal("expected scard error")
		}
	})

	t.Run("empty websocket delete error", func(t *testing.T) {
		client, mr := newTestClient(t)
		store := NewCleanupStore(client)
		wsKey := websocketUserConnectionsKey(uuid.New(), uuid.New())
		mr.SAdd(wsKey, "conn")
		mr.SetTTL(wsKey, time.Minute)

		origCount := countCleanupSetMembersFn
		origDel := deleteCleanupKeyFn
		countCleanupSetMembersFn = func(_ *goredis.Client, _ context.Context, _ string) (int64, error) {
			return 0, nil
		}
		deleteCleanupKeyFn = func(_ *goredis.Client, _ context.Context, _ string) error {
			return errors.New("delete failed")
		}
		t.Cleanup(func() {
			countCleanupSetMembersFn = origCount
			deleteCleanupKeyFn = origDel
		})
		if _, err := store.CleanupEphemeralKeys(context.Background()); err == nil {
			t.Fatal("expected empty websocket delete error")
		}
	})

	t.Run("ttl missing key", func(t *testing.T) {
		client, mr := newTestClient(t)
		store := NewCleanupStore(client)
		mr.Set("lock:document:missing:autosave", "1")

		orig := readCleanupKeyTTLFn
		readCleanupKeyTTLFn = func(_ *goredis.Client, _ context.Context, _ string) (time.Duration, error) {
			return -2, nil
		}
		t.Cleanup(func() { readCleanupKeyTTLFn = orig })
		if _, err := store.CleanupEphemeralKeys(context.Background()); err != nil {
			t.Fatalf("CleanupEphemeralKeys: %v", err)
		}
	})

	t.Run("skip non websocket ttl key", func(t *testing.T) {
		client, mr := newTestClient(t)
		store := NewCleanupStore(client)
		mr.Set("rate_limit:token_bucket:test", "1")
		mr.SetTTL("rate_limit:token_bucket:test", time.Minute)

		if _, err := store.CleanupEphemeralKeys(context.Background()); err != nil {
			t.Fatalf("CleanupEphemeralKeys: %v", err)
		}
	})
}

func TestCleanupPattern_TTLAndScanErrors(t *testing.T) {
	client, mr := newTestClient(t)
	store := NewCleanupStore(client)
	mr.Set("lock:document:err:autosave", "1")
	mr.Close()
	if _, err := store.CleanupEphemeralKeys(context.Background()); err == nil {
		t.Fatal("expected cleanup scan error")
	}
}
func TestMarkClean_GetError(t *testing.T) {
	store := newDocumentCacheStore(&mockDocumentCacheCommands{getErr: errors.New("get failed")}, nil)
	if err := store.MarkClean(context.Background(), uuid.New()); err == nil {
		t.Fatal("expected get error")
	}
}

func TestListDirtyDocumentIDs_FiltersRecentAndErrors(t *testing.T) {
	client, mr := newTestClient(t)
	store := NewDocumentCacheStore(client)
	ctx := context.Background()

	oldDoc := uuid.New()
	newDoc := uuid.New()
	oldAt := time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)
	newAt := time.Now().UTC().Format(time.RFC3339)
	mr.Set(autosaveKey(oldDoc), `{"document_id":"`+oldDoc.String()+`","updated_by":"`+uuid.New().String()+`","updated_at":"`+oldAt+`","dirty":true}`)
	mr.Set(autosaveKey(newDoc), `{"document_id":"`+newDoc.String()+`","updated_by":"`+uuid.New().String()+`","updated_at":"`+newAt+`","dirty":true}`)

	ids, err := store.ListIdleDirtyDocumentIDs(ctx, time.Hour)
	if err != nil {
		t.Fatalf("ListIdleDirtyDocumentIDs: %v", err)
	}
	if len(ids) != 1 || ids[0] != oldDoc {
		t.Fatalf("ids = %v", ids)
	}

	store.commands = &mockDocumentCacheCommands{getErr: errors.New("get failed")}
	if _, err := store.listDirtyDocumentIDs(ctx, 0); err == nil {
		t.Fatal("expected list dirty get error")
	}
}

func TestCleanupPattern_Branches(t *testing.T) {
	client, mr := newTestClient(t)
	store := NewCleanupStore(client)
	ctx := context.Background()

	mr.Set("lock:document:1:autosave", "1")
	wsKey := websocketUserConnectionsKey(uuid.New(), uuid.New())
	mr.SAdd(wsKey, "conn-1")
	mr.SetTTL(wsKey, time.Minute)

	removed, err := store.CleanupEphemeralKeys(ctx)
	if err != nil {
		t.Fatalf("CleanupEphemeralKeys: %v", err)
	}
	if removed == 0 {
		t.Fatal("expected removed persistent keys")
	}
	if !mr.Exists(wsKey) {
		t.Fatal("expected websocket key with members to remain")
	}
}

func TestCleanupPattern_EmptyWebSocketSet(t *testing.T) {
	client, mr := newTestClient(t)
	store := NewCleanupStore(client)
	wsKey := websocketUserConnectionsKey(uuid.New(), uuid.New())
	mr.SAdd(wsKey, "conn")
	mr.SetTTL(wsKey, time.Minute)

	origCount := countCleanupSetMembersFn
	origDel := deleteCleanupKeyFn
	countCleanupSetMembersFn = func(_ *goredis.Client, _ context.Context, _ string) (int64, error) {
		return 0, nil
	}
	t.Cleanup(func() { countCleanupSetMembersFn = origCount })

	removed, err := store.CleanupEphemeralKeys(context.Background())
	deleteCleanupKeyFn = origDel
	if err != nil {
		t.Fatalf("CleanupEphemeralKeys: %v", err)
	}
	if removed == 0 {
		t.Fatal("expected empty websocket set removed")
	}
}

func TestDocumentEditorsStore_RemoveRemainingEditors(t *testing.T) {
	client, _ := newTestClient(t)
	store := NewDocumentEditorsStore(client, time.Minute)
	ctx := context.Background()
	docID := uuid.New()
	userA := uuid.New()
	userB := uuid.New()

	_, _, err := store.Add(ctx, docID, usecase.DocumentEditorInfo{UserID: userA, Name: "A"})
	if err != nil {
		t.Fatalf("Add A: %v", err)
	}
	_, _, err = store.Add(ctx, docID, usecase.DocumentEditorInfo{UserID: userB, Name: "B"})
	if err != nil {
		t.Fatalf("Add B: %v", err)
	}

	removed, editors, err := store.Remove(ctx, docID, userA)
	if err != nil || !removed || len(editors) != 1 {
		t.Fatalf("Remove A = %v, %v, %v", removed, editors, err)
	}
}

func TestDocumentEditorsStore_RefreshTTLExpireError(t *testing.T) {
	client, _ := newTestClient(t)
	store := NewDocumentEditorsStore(client, time.Minute)
	docID := uuid.New()
	if _, _, err := store.Add(context.Background(), docID, usecase.DocumentEditorInfo{UserID: uuid.New(), Name: "A"}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	orig := expireEditorsKeyFn
	expireEditorsKeyFn = func(*goredis.Client, context.Context, string, time.Duration) error {
		return errors.New("expire failed")
	}
	t.Cleanup(func() { expireEditorsKeyFn = orig })

	if err := store.RefreshTTL(context.Background(), docID); err == nil {
		t.Fatal("expected refresh ttl expire error")
	}
}

func TestLoginFailureStore_RecordFailureError(t *testing.T) {
	client, _ := newTestClient(t)
	store := NewLoginFailureStore(client, 3, time.Minute, 5*time.Minute)
	client.client.Close()
	if _, _, err := store.RecordFailure(context.Background(), "user@example.com"); err == nil {
		t.Fatal("expected record failure error")
	}
}

func TestTokenBucketStore_ResultParseErrors(t *testing.T) {
	orig := runTokenBucketScriptFn
	t.Cleanup(func() { runTokenBucketScriptFn = orig })

	client, _ := newTestClient(t)
	store := NewTokenBucketStore(client)

	runTokenBucketScriptFn = func(context.Context, *TokenBucketStore, string, int, float64, time.Time) ([]any, error) {
		return []any{"bad", int64(0)}, nil
	}
	if _, _, err := store.Allow(context.Background(), "k", 2, 1, time.Now()); err == nil {
		t.Fatal("expected allowed parse error")
	}

	runTokenBucketScriptFn = func(context.Context, *TokenBucketStore, string, int, float64, time.Time) ([]any, error) {
		return []any{int64(0), "bad"}, nil
	}
	if _, _, err := store.Allow(context.Background(), "k", 2, 1, time.Now()); err == nil {
		t.Fatal("expected retry parse error")
	}
	runTokenBucketScriptFn = func(context.Context, *TokenBucketStore, string, int, float64, time.Time) ([]any, error) {
		return nil, errors.New("script failed")
	}
	if _, _, err := store.Allow(context.Background(), "k", 2, 1, time.Now()); err == nil {
		t.Fatal("expected script error")
	}
}

func TestWebSocketSessionStore_RemoveConnectionErrors(t *testing.T) {
	client, _ := newTestClient(t)
	store := NewWebSocketSessionStore(client)
	client.client.Close()
	if _, err := store.RemoveConnection(context.Background(), uuid.New(), uuid.New(), uuid.New()); err == nil {
		t.Fatal("expected remove connection error")
	}
}

func TestWebSocketSessionStore_RemoveConnectionCountError(t *testing.T) {
	client, _ := newTestClient(t)
	store := NewWebSocketSessionStore(client)
	docID := uuid.New()
	userID := uuid.New()
	connID := uuid.New()

	_, _, err := store.TryAddConnection(context.Background(), docID, userID, connID, 2, time.Minute)
	if err != nil {
		t.Fatalf("TryAddConnection: %v", err)
	}

	orig := countWebSocketConnectionsFn
	countWebSocketConnectionsFn = func(*goredis.Client, context.Context, string) (int64, error) {
		return 0, errors.New("scard failed")
	}
	t.Cleanup(func() { countWebSocketConnectionsFn = orig })

	if _, err := store.RemoveConnection(context.Background(), docID, userID, connID); err == nil {
		t.Fatal("expected count error after remove")
	}
}
