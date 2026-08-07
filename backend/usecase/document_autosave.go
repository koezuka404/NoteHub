package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
	"github.com/koezuka404/notehub/repository"
)

type DocumentAutoSaveUseCase struct {
	docs         repository.DocumentRepository
	versions     repository.DocumentVersionRepository
	cache        IDocumentCache
	locks        IAutoSaveLockStore
	transactions repository.TransactionManager
	lockTTL      time.Duration
	idleDuration time.Duration
	now          func() time.Time
}

func NewDocumentAutoSaveUseCase(
	docs repository.DocumentRepository,
	versions repository.DocumentVersionRepository,
	cache IDocumentCache,
	locks IAutoSaveLockStore,
	transactions repository.TransactionManager,
	lockTTL time.Duration,
	idleDuration time.Duration,
) *DocumentAutoSaveUseCase {
	if lockTTL <= 0 {
		lockTTL = 30 * time.Second
	}
	if idleDuration <= 0 {
		idleDuration = 60 * time.Second
	}
	return &DocumentAutoSaveUseCase{
		docs:         docs,
		versions:     versions,
		cache:        cache,
		locks:        locks,
		transactions: transactions,
		lockTTL:      lockTTL,
		idleDuration: idleDuration,
		now:          time.Now,
	}
}

var _ IDocumentFlushService = (*DocumentAutoSaveUseCase)(nil)

func (uc *DocumentAutoSaveUseCase) RunOnce(ctx context.Context) error {
	return uc.flushDirty(ctx, uc.idleDuration)
}

func (uc *DocumentAutoSaveUseCase) FlushAllDirty(ctx context.Context) error {
	return uc.flushDirty(ctx, 0)
}

func (uc *DocumentAutoSaveUseCase) flushDirty(ctx context.Context, idle time.Duration) error {
	var documentIDs []uuid.UUID
	var err error
	if idle > 0 {
		documentIDs, err = uc.cache.ListIdleDirtyDocumentIDs(ctx, idle)
	} else {
		documentIDs, err = uc.cache.ListDirtyDocumentIDs(ctx)
	}
	if err != nil {
		return fmt.Errorf("list dirty documents: %w", err)
	}
	for _, documentID := range documentIDs {
		if err := uc.SaveDocument(ctx, documentID); err != nil {
			log.Printf("autosave document %s: %v", documentID, err)
		}
	}
	return nil
}

func (uc *DocumentAutoSaveUseCase) FlushDocument(ctx context.Context, documentID uuid.UUID) error {
	dirty, err := uc.cache.IsDirty(ctx, documentID)
	if err != nil {
		return fmt.Errorf("check dirty flag: %w", err)
	}
	if !dirty {
		return nil
	}
	return uc.persistDocument(ctx, documentID, entity.DocumentVersionManualSave)
}

func (uc *DocumentAutoSaveUseCase) FlushWorkspaceDocuments(ctx context.Context, workspaceID uuid.UUID) error {
	docs, err := uc.docs.FindByWorkspaceID(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("list workspace documents: %w", err)
	}
	for _, doc := range docs {
		if err := uc.FlushDocument(ctx, doc.ID); err != nil {
			log.Printf("flush document %s before workspace delete: %v", doc.ID, err)
		}
	}
	return nil
}

func (uc *DocumentAutoSaveUseCase) SaveDocument(ctx context.Context, documentID uuid.UUID) error {
	return uc.persistDocument(ctx, documentID, entity.DocumentVersionAutoSave)
}

func (uc *DocumentAutoSaveUseCase) persistDocument(
	ctx context.Context,
	documentID uuid.UUID,
	versionType entity.DocumentVersionType,
) error {
	dirty, err := uc.cache.IsDirty(ctx, documentID)
	if err != nil {
		return fmt.Errorf("check dirty flag: %w", err)
	}
	if !dirty {
		return nil
	}

	lockKey := documentAutosaveLockKey(documentID)
	locked, err := uc.locks.TryLock(ctx, lockKey, uc.lockTTL)
	if err != nil {
		return fmt.Errorf("acquire autosave lock: %w", err)
	}
	if !locked {
		return nil
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := uc.locks.Unlock(unlockCtx, lockKey); err != nil {
			log.Printf("autosave unlock document %s: %v", documentID, err)
		}
	}()

	dirty, err = uc.cache.IsDirty(ctx, documentID)
	if err != nil {
		return fmt.Errorf("recheck dirty flag: %w", err)
	}
	if !dirty {
		return nil
	}

	startRevision, err := uc.cache.GetRevision(ctx, documentID)
	if err != nil {
		return fmt.Errorf("read start revision: %w", err)
	}

	state, found, err := uc.cache.GetContentState(ctx, documentID)
	if err != nil {
		return fmt.Errorf("load cached content: %w", err)
	}
	if !found {
		return nil
	}

	if err := uc.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		lockedDoc, docFound, err := uc.docs.FindByIDForUpdate(txCtx, documentID)
		if err != nil {
			return fmt.Errorf("find document for persist: %w", err)
		}
		if !docFound {
			return ErrDocumentNotFound
		}
		if lockedDoc.IsDeleted() {
			return ErrDocumentDeleted
		}
		if err := lockedDoc.ReplaceContent(state.Content, state.UpdatedBy, lockedDoc.Revision, state.UpdatedAt.UTC()); err != nil {
			return err
		}
		if err := uc.docs.Update(txCtx, lockedDoc); err != nil {
			return fmt.Errorf("update document content: %w", err)
		}

		version, err := entity.NewDocumentVersion(
			*lockedDoc,
			state.Content,
			versionType,
			state.UpdatedBy,
			nil,
			state.UpdatedAt.UTC(),
		)
		if err != nil {
			return fmt.Errorf("create document version entity: %w", err)
		}
		if err := uc.versions.Create(txCtx, &version); err != nil {
			return fmt.Errorf("save document version: %w", err)
		}
		return nil
	}); err != nil {
		if errors.Is(err, ErrDocumentNotFound) || errors.Is(err, ErrDocumentDeleted) {
			return nil
		}
		if errors.Is(err, entity.ErrDocumentConflict) {
			return fmt.Errorf("document revision conflict: %w", err)
		}
		return err
	}

	endRevision, err := uc.cache.GetRevision(ctx, documentID)
	if err != nil {
		return fmt.Errorf("read end revision: %w", err)
	}
	if endRevision != startRevision {
		return nil
	}
	if err := uc.cache.MarkClean(ctx, documentID); err != nil {
		return fmt.Errorf("mark document clean: %w", err)
	}
	return nil
}

func documentAutosaveLockKey(documentID uuid.UUID) string {
	return "lock:document:" + documentID.String() + ":autosave"
}
