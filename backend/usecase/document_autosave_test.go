package usecase

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
)

type revisionChangingCache struct {
	stubDocumentCache
	revisionCalls int
}

func (c *revisionChangingCache) GetRevision(_ context.Context, documentID uuid.UUID) (uint64, error) {
	if c.getErr != nil {
		return 0, c.getErr
	}
	c.revisionCalls++
	if c.revisionCalls == 1 {
		if c.revisions != nil {
			return c.revisions[documentID], nil
		}
		return 1, nil
	}
	if c.revisions != nil {
		return c.revisions[documentID] + 1, nil
	}
	return 2, nil
}

func newAutoSaveUseCaseWithCache(docs *stubDocumentRepo, versions *stubVersionRepo, cache IDocumentCache, locks *stubAutoSaveLock) *DocumentAutoSaveUseCase {
	uc := NewDocumentAutoSaveUseCase(docs, versions, cache, locks, &mockTransactionManager{}, 30*time.Second, 60*time.Second)
	uc.now = usecaseTestNow
	return uc
}

func autosaveFixture() (docID uuid.UUID, docs *stubDocumentRepo, cache *stubDocumentCache) {
	userID := uuid.New()
	workspaceID := uuid.New()
	doc := activeDocument(workspaceID, userID)
	docID = doc.ID
	docs = &stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}}
	cache = &stubDocumentCache{
		content: map[uuid.UUID]DocumentContentState{
			doc.ID: {Content: "edited", UpdatedBy: userID, UpdatedAt: usecaseTestNow()},
		},
		dirty:     map[uuid.UUID]bool{doc.ID: true},
		revisions: map[uuid.UUID]uint64{doc.ID: 1},
	}
	return docID, docs, cache
}

func TestAutoSave_LockNotAcquired(t *testing.T) {
	docID, docs, cache := autosaveFixture()
	locks := &stubAutoSaveLock{locked: false}
	uc := newAutoSaveUseCase(docs, &stubVersionRepo{}, cache, locks)

	if err := uc.SaveDocument(context.Background(), docID); err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}
	if len(cache.dirty) != 0 && cache.dirty[docID] {
	} else if _, ok := cache.dirty[docID]; ok && !cache.dirty[docID] {
		t.Fatal("expected document to remain dirty when lock not acquired")
	}
}

func TestAutoSave_NotDirty(t *testing.T) {
	docID, docs, cache := autosaveFixture()
	cache.dirty[docID] = false
	uc := newAutoSaveUseCase(docs, &stubVersionRepo{}, cache, &stubAutoSaveLock{locked: true})

	if err := uc.SaveDocument(context.Background(), docID); err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}
}

func TestAutoSave_DocumentDeletedSwallowed(t *testing.T) {
	docID, docs, cache := autosaveFixture()
	now := usecaseTestNow()
	docs.byID[docID].DeletedAt = &now
	uc := newAutoSaveUseCase(docs, &stubVersionRepo{}, cache, &stubAutoSaveLock{locked: true})

	if err := uc.SaveDocument(context.Background(), docID); err != nil {
		t.Fatalf("expected deleted doc to be swallowed, got %v", err)
	}
}

func TestAutoSave_DocumentNotFoundSwallowed(t *testing.T) {
	docID, _, cache := autosaveFixture()
	uc := newAutoSaveUseCase(&stubDocumentRepo{}, &stubVersionRepo{}, cache, &stubAutoSaveLock{locked: true})

	if err := uc.SaveDocument(context.Background(), docID); err != nil {
		t.Fatalf("expected not found to be swallowed, got %v", err)
	}
}

func newAutoSaveUseCase(docs *stubDocumentRepo, versions *stubVersionRepo, cache *stubDocumentCache, locks *stubAutoSaveLock) *DocumentAutoSaveUseCase {
	return newAutoSaveUseCaseWithCache(docs, versions, cache, locks)
}

func TestAutoSave_RevisionMismatchSkipsMarkClean(t *testing.T) {
	docID, docs, baseCache := autosaveFixture()
	cache := &revisionChangingCache{stubDocumentCache: *baseCache}
	uc := newAutoSaveUseCaseWithCache(docs, &stubVersionRepo{}, cache, &stubAutoSaveLock{locked: true})

	if err := uc.SaveDocument(context.Background(), docID); err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}
	if !cache.dirty[docID] {
		t.Fatal("expected document to remain dirty after revision mismatch")
	}
}

func TestAutoSave_SuccessMarksClean(t *testing.T) {
	docID, docs, cache := autosaveFixture()
	versions := &stubVersionRepo{}
	uc := newAutoSaveUseCase(docs, versions, cache, &stubAutoSaveLock{locked: true})

	if err := uc.SaveDocument(context.Background(), docID); err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}
	if cache.dirty != nil {
		if dirty, ok := cache.dirty[docID]; ok && dirty {
			t.Fatal("expected document marked clean")
		}
	}
	updated := docs.byID[docID]
	if updated.Content != "edited" {
		t.Fatalf("content = %q", updated.Content)
	}
}

func TestAutoSave_RunOnce(t *testing.T) {
	docID, docs, cache := autosaveFixture()
	cache.idleIDs = []uuid.UUID{docID}
	uc := newAutoSaveUseCase(docs, &stubVersionRepo{}, cache, &stubAutoSaveLock{locked: true})

	if err := uc.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
}

func TestAutoSave_FlushDocument_NotDirty(t *testing.T) {
	docID, docs, cache := autosaveFixture()
	cache.dirty[docID] = false
	uc := newAutoSaveUseCase(docs, &stubVersionRepo{}, cache, &stubAutoSaveLock{locked: true})

	if err := uc.FlushDocument(context.Background(), docID); err != nil {
		t.Fatalf("FlushDocument: %v", err)
	}
}

func TestAutoSave_FlushAllDirty(t *testing.T) {
	docID, docs, cache := autosaveFixture()
	cache.dirtyIDs = []uuid.UUID{docID}
	uc := newAutoSaveUseCase(docs, &stubVersionRepo{}, cache, &stubAutoSaveLock{locked: true})

	if err := uc.FlushAllDirty(context.Background()); err != nil {
		t.Fatalf("FlushAllDirty: %v", err)
	}
}

func TestAutoSave_LockError(t *testing.T) {
	docID, docs, cache := autosaveFixture()
	locks := &stubAutoSaveLock{tryErr: fmt.Errorf("lock store down")}
	uc := newAutoSaveUseCase(docs, &stubVersionRepo{}, cache, locks)

	if err := uc.SaveDocument(context.Background(), docID); err == nil {
		t.Fatal("expected lock error")
	}
}

func TestAutoSave_IsDirtyError(t *testing.T) {
	docID, docs, cache := autosaveFixture()
	cache.getErr = fmt.Errorf("cache down")
	uc := newAutoSaveUseCase(docs, &stubVersionRepo{}, cache, &stubAutoSaveLock{locked: true})

	if err := uc.SaveDocument(context.Background(), docID); err == nil {
		t.Fatal("expected dirty check error")
	}
}

func TestAutoSave_CacheStateNotFound(t *testing.T) {
	docID, docs, cache := autosaveFixture()
	cache.content = nil
	cache.hasState = false
	uc := newAutoSaveUseCase(docs, &stubVersionRepo{}, cache, &stubAutoSaveLock{locked: true})

	if err := uc.SaveDocument(context.Background(), docID); err != nil {
		t.Fatalf("expected nil when cache state missing, got %v", err)
	}
}

func TestAutoSave_FlushWorkspaceDocuments(t *testing.T) {
	userID := uuid.New()
	workspaceID := uuid.New()
	doc := activeDocument(workspaceID, userID)
	docs := &stubDocumentRepo{
		byID:        map[uuid.UUID]*entity.Document{doc.ID: &doc},
		byWorkspace: map[uuid.UUID][]entity.Document{workspaceID: {doc}},
	}
	cache := &stubDocumentCache{
		content: map[uuid.UUID]DocumentContentState{
			doc.ID: {Content: "edited", UpdatedBy: userID, UpdatedAt: usecaseTestNow()},
		},
		dirty:     map[uuid.UUID]bool{doc.ID: true},
		revisions: map[uuid.UUID]uint64{doc.ID: 1},
	}
	uc := newAutoSaveUseCase(docs, &stubVersionRepo{}, cache, &stubAutoSaveLock{locked: true})

	if err := uc.FlushWorkspaceDocuments(context.Background(), workspaceID); err != nil {
		t.Fatalf("FlushWorkspaceDocuments: %v", err)
	}
}

func TestAutoSave_MarkCleanError(t *testing.T) {
	docID, docs, cache := autosaveFixture()
	cache.setErr = fmt.Errorf("mark clean failed")
	uc := newAutoSaveUseCase(docs, &stubVersionRepo{}, cache, &stubAutoSaveLock{locked: true})

	if err := uc.SaveDocument(context.Background(), docID); err == nil {
		t.Fatal("expected mark clean error")
	}
}
