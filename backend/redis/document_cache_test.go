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

func TestDocumentCacheStore(t *testing.T) {
	client, _ := newTestClient(t)
	store := NewDocumentCacheStore(client)
	ctx := context.Background()
	docID := uuid.New()
	userID := uuid.New()

	_, found, err := store.GetContentState(ctx, docID)
	if err != nil || found {
		t.Fatalf("missing content state = %v, %v", found, err)
	}

	updatedAt := time.Now().UTC().Add(-2 * time.Hour)

	if err := store.SetContent(ctx, docID, "hello", userID, updatedAt); err != nil {
		t.Fatalf("SetContent: %v", err)
	}

	state, found, err := store.GetContentState(ctx, docID)
	if err != nil || !found || state.Content != "hello" || state.UpdatedBy != userID {
		t.Fatalf("GetContentState() = %+v, %v, %v", state, found, err)
	}

	dirty, err := store.IsDirty(ctx, docID)
	if err != nil || !dirty {
		t.Fatalf("IsDirty() = %v, %v", dirty, err)
	}

	revision, err := store.GetRevision(ctx, docID)
	if err != nil || revision != 1 {
		t.Fatalf("GetRevision() = %v, %v", revision, err)
	}

	ids, err := store.ListDirtyDocumentIDs(ctx)
	if err != nil || len(ids) != 1 {
		t.Fatalf("ListDirtyDocumentIDs() = %v, %v", ids, err)
	}

	idleIDs, err := store.ListIdleDirtyDocumentIDs(ctx, time.Hour)
	if err != nil || len(idleIDs) != 1 {
		t.Fatalf("ListIdleDirtyDocumentIDs idle = %v, %v", idleIDs, err)
	}
	idleIDs, err = store.ListIdleDirtyDocumentIDs(ctx, 0)
	if err != nil || len(idleIDs) != 1 {
		t.Fatalf("ListIdleDirtyDocumentIDs zero idle = %v, %v", idleIDs, err)
	}

	if err := store.MarkClean(ctx, docID); err != nil {
		t.Fatalf("MarkClean: %v", err)
	}
	dirty, err = store.IsDirty(ctx, docID)
	if err != nil || dirty {
		t.Fatalf("IsDirty after clean = %v, %v", dirty, err)
	}

	if err := store.Clear(ctx, docID); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	revision, err = store.GetRevision(ctx, docID)
	if err != nil || revision != 0 {
		t.Fatalf("GetRevision after clear = %v, %v", revision, err)
	}
}

func storeGetContentExists(store *DocumentCacheStore, ctx context.Context, docID uuid.UUID) (bool, error) {
	_, found, err := store.GetContentState(ctx, docID)
	return found, err
}

func TestDocumentCacheStore_MarkCleanMissing(t *testing.T) {
	client, _ := newTestClient(t)
	store := NewDocumentCacheStore(client)
	if err := store.MarkClean(context.Background(), uuid.New()); err != nil {
		t.Fatalf("MarkClean missing: %v", err)
	}
}

func TestDocumentCacheStore_GetRevisionNegative(t *testing.T) {
	client, _ := newTestClient(t)
	store := NewDocumentCacheStore(client)
	docID := uuid.New()
	ctx := context.Background()
	if err := store.commands.Set(ctx, revisionKey(docID), "-5", 0).Err(); err != nil {
		t.Fatalf("Set revision: %v", err)
	}
	revision, err := store.GetRevision(ctx, docID)
	if err != nil || revision != 0 {
		t.Fatalf("GetRevision() = %v, %v", revision, err)
	}
}

func TestParseDocumentIDFromAutosaveKey(t *testing.T) {
	docID := uuid.New()
	key := autosaveKey(docID)
	parsed, err := parseDocumentIDFromAutosaveKey(key)
	if err != nil || parsed != docID {
		t.Fatalf("parseDocumentIDFromAutosaveKey() = %v, %v", parsed, err)
	}
	if _, err := parseDocumentIDFromAutosaveKey("bad-key"); err == nil {
		t.Fatal("expected invalid autosave key error")
	}
}

type mockDocumentCacheCommands struct {
	getVal  string
	getErr  error
	setErr  error
	setErrFn func() error
	delErr  error
	incrErr error
}

func (m *mockDocumentCacheCommands) Get(_ context.Context, key string) *goredis.StringCmd {
	cmd := goredis.NewStringCmd(context.Background(), key)
	if m.getErr != nil {
		cmd.SetErr(m.getErr)
		return cmd
	}
	cmd.SetVal(m.getVal)
	return cmd
}

func (m *mockDocumentCacheCommands) Set(ctx context.Context, _ string, _ interface{}, _ time.Duration) *goredis.StatusCmd {
	cmd := goredis.NewStatusCmd(ctx)
	if m.setErrFn != nil {
		if err := m.setErrFn(); err != nil {
			cmd.SetErr(err)
			return cmd
		}
	}
	if m.setErr != nil {
		cmd.SetErr(m.setErr)
	}
	return cmd
}

func (m *mockDocumentCacheCommands) Del(context.Context, ...string) *goredis.IntCmd {
	cmd := goredis.NewIntCmd(context.Background())
	if m.delErr != nil {
		cmd.SetErr(m.delErr)
	}
	return cmd
}

func (m *mockDocumentCacheCommands) Incr(context.Context, string) *goredis.IntCmd {
	cmd := goredis.NewIntCmd(context.Background())
	if m.incrErr != nil {
		cmd.SetErr(m.incrErr)
	}
	return cmd
}

func newDocumentCacheStore(commands IDocumentCacheCommands, scanClient *goredis.Client) *DocumentCacheStore {
	return &DocumentCacheStore{
		commands: commands,
		client:   scanClient,
		withTimeout: func(ctx context.Context) (context.Context, context.CancelFunc) {
			return context.WithCancel(ctx)
		},
	}
}

func TestDocumentCacheStore_GetContentStateErrors(t *testing.T) {
	docID := uuid.New()
	ctx := context.Background()

	t.Run("get error", func(t *testing.T) {
		store := newDocumentCacheStore(&mockDocumentCacheCommands{getErr: errors.New("get failed")}, nil)
		if _, _, err := store.GetContentState(ctx, docID); err == nil {
			t.Fatal("expected get error")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		store := newDocumentCacheStore(&mockDocumentCacheCommands{getVal: "not-json"}, nil)
		if _, _, err := store.GetContentState(ctx, docID); err == nil {
			t.Fatal("expected decode error")
		}
	})

	t.Run("invalid updated by", func(t *testing.T) {
		raw, _ := json.Marshal(contentCacheValue{Content: "x", UpdatedBy: "bad", UpdatedAt: time.Now().UTC().Format(time.RFC3339)})
		store := newDocumentCacheStore(&mockDocumentCacheCommands{getVal: string(raw)}, nil)
		if _, _, err := store.GetContentState(ctx, docID); err == nil {
			t.Fatal("expected updated_by error")
		}
	})

	t.Run("invalid updated at", func(t *testing.T) {
		raw, _ := json.Marshal(contentCacheValue{Content: "x", UpdatedBy: uuid.New().String(), UpdatedAt: "bad"})
		store := newDocumentCacheStore(&mockDocumentCacheCommands{getVal: string(raw)}, nil)
		if _, _, err := store.GetContentState(ctx, docID); err == nil {
			t.Fatal("expected updated_at error")
		}
	})
}

func TestDocumentCacheStore_IsDirtyAndMarkCleanErrors(t *testing.T) {
	docID := uuid.New()
	ctx := context.Background()

	store := newDocumentCacheStore(&mockDocumentCacheCommands{getErr: errors.New("get failed")}, nil)
	if _, err := store.IsDirty(ctx, docID); err == nil {
		t.Fatal("expected is dirty error")
	}

	raw, _ := json.Marshal(autosaveCacheValue{Dirty: true})
	store = newDocumentCacheStore(&mockDocumentCacheCommands{getVal: "not-json"}, nil)
	if _, err := store.IsDirty(ctx, docID); err == nil {
		t.Fatal("expected decode autosave error")
	}
	store = newDocumentCacheStore(&mockDocumentCacheCommands{getVal: "not-json"}, nil)
	if err := store.MarkClean(ctx, docID); err == nil {
		t.Fatal("expected mark clean decode error")
	}

	store = newDocumentCacheStore(&mockDocumentCacheCommands{
		getVal:  string(raw),
		setErr:  errors.New("set failed"),
	}, nil)
	if err := store.MarkClean(ctx, docID); err == nil {
		t.Fatal("expected mark clean set error")
	}
}

func TestDocumentCacheStore_SetContentAndClearErrors(t *testing.T) {
	docID := uuid.New()
	ctx := context.Background()

	store := newDocumentCacheStore(&mockDocumentCacheCommands{setErr: errors.New("set failed")}, nil)
	if err := store.SetContent(ctx, docID, "x", uuid.New(), time.Now()); err == nil {
		t.Fatal("expected set content error")
	}

	client, _ := newTestClient(t)
	store = NewDocumentCacheStore(client)
	store.commands = &mockDocumentCacheCommands{setErr: errors.New("set failed")}
	if err := store.SetContent(ctx, docID, "x", uuid.New(), time.Now()); err == nil {
		t.Fatal("expected set content cache error")
	}

	store = newDocumentCacheStore(&mockDocumentCacheCommands{delErr: errors.New("del failed")}, client.client)
	if err := store.Clear(ctx, docID); err == nil {
		t.Fatal("expected clear error")
	}
}

func TestDocumentCacheStore_ListDirtySkipsInvalidEntries(t *testing.T) {
	client, mr := newTestClient(t)
	store := NewDocumentCacheStore(client)
	ctx := context.Background()
	docID := uuid.New()

	mr.Set(autosaveKey(docID), `{"document_id":"`+docID.String()+`","updated_by":"`+uuid.New().String()+`","updated_at":"bad","dirty":true}`)
	mr.Set("document:not-a-uuid:autosave", `{"dirty":true}`)

	ids, err := store.ListIdleDirtyDocumentIDs(ctx, time.Minute)
	if err != nil {
		t.Fatalf("ListIdleDirtyDocumentIDs: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("ids = %v", ids)
	}
}

func TestDocumentCacheStore_GetRevisionError(t *testing.T) {
	store := newDocumentCacheStore(&mockDocumentCacheCommands{getErr: errors.New("get failed")}, nil)
	if _, err := store.GetRevision(context.Background(), uuid.New()); err == nil {
		t.Fatal("expected revision error")
	}
}

func TestDocumentCacheStore_SetContentEncodeError(t *testing.T) {
	orig := jsonMarshalDocumentCacheFn
	jsonMarshalDocumentCacheFn = func(any) ([]byte, error) { return nil, errors.New("marshal failed") }
	t.Cleanup(func() { jsonMarshalDocumentCacheFn = orig })

	client, _ := newTestClient(t)
	store := NewDocumentCacheStore(client)
	if err := store.SetContent(context.Background(), uuid.New(), "x", uuid.New(), time.Now()); err == nil {
		t.Fatal("expected encode error")
	}
}

func TestDocumentCacheStore_MarkCleanEncodeError(t *testing.T) {
	orig := jsonMarshalDocumentCacheFn
	jsonMarshalDocumentCacheFn = func(any) ([]byte, error) { return nil, errors.New("marshal failed") }
	t.Cleanup(func() { jsonMarshalDocumentCacheFn = orig })

	raw, _ := json.Marshal(autosaveCacheValue{Dirty: true})
	store := newDocumentCacheStore(&mockDocumentCacheCommands{getVal: string(raw)}, nil)
	if err := store.MarkClean(context.Background(), uuid.New()); err == nil {
		t.Fatal("expected encode error")
	}
}

// ensure usecase type is referenced
var _ = usecase.DocumentContentState{}
