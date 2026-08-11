package usecase

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
	"github.com/koezuka404/notehub/repository"
	"time"
)

type IVersionUsecase interface {
	ListVersions(ctx context.Context, input ListVersionsInput) ([]VersionListItem, error)
	GetVersion(ctx context.Context, input GetVersionInput) (*GetVersionOutput, error)
	CreateVersion(ctx context.Context, input CreateVersionInput) error
	SaveManualVersion(ctx context.Context, input SaveManualVersionInput) (*SaveManualVersionOutput, error)
	RestoreVersion(ctx context.Context, input RestoreVersionInput) (*RestoreVersionOutput, error)
}

type VersionUseCase struct {
	docs         repository.DocumentRepository
	versions     repository.DocumentVersionRepository
	transactions repository.TransactionManager
	access       IAccessCheck
	cache        IDocumentCache
	notifier     IDocumentWebSocketNotifier
	auditLogs    repository.AuditLogRepository
	now          func() time.Time
}

func NewVersionUseCase(
	docs repository.DocumentRepository,
	versions repository.DocumentVersionRepository,
	transactions repository.TransactionManager,
	access IAccessCheck,
	cache IDocumentCache,
	notifier IDocumentWebSocketNotifier,
	auditLogs repository.AuditLogRepository,
) *VersionUseCase {
	return &VersionUseCase{
		docs:         docs,
		versions:     versions,
		transactions: transactions,
		access:       access,
		cache:        cache,
		notifier:     notifier,
		auditLogs:    auditLogs,
		now:          time.Now,
	}
}

func (uc *VersionUseCase) currentTime() time.Time {
	return uc.now().UTC()
}

func (uc *VersionUseCase) loadDoc(ctx context.Context, documentID uuid.UUID) (*entity.Document, error) {
	doc, found, err := uc.docs.FindByID(ctx, documentID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrDocumentNotFound
	}
	if doc.IsDeleted() {
		return nil, ErrDocumentDeleted
	}
	return doc, nil
}

func (uc *VersionUseCase) authorizeDoc(ctx context.Context, userID uuid.UUID, doc *entity.Document) error {
	_, err := uc.access.CheckWorkspaceAccess(ctx, CheckWorkspaceAccessInput{
		UserID: userID, WorkspaceID: doc.WorkspaceID,
	})
	return err
}

func (uc *VersionUseCase) currentContent(ctx context.Context, doc *entity.Document) (string, error) {
	if uc.cache != nil {
		if state, ok, err := uc.cache.GetContentState(ctx, doc.ID); err != nil {
			return "", err
		} else if ok {
			return state.Content, nil
		}
	}
	return doc.Content, nil
}


type CreateVersionInput struct {
	DocumentID  uuid.UUID
	Content     string
	CreatedBy   uuid.UUID
	VersionType entity.DocumentVersionType
}

func (uc *VersionUseCase) CreateVersion(ctx context.Context, input CreateVersionInput) error {
	doc, err := uc.loadDoc(ctx, input.DocumentID)
	if err != nil {
		return err
	}

	now := uc.currentTime()
	version, err := newDocumentVersionFn(*doc, input.Content, input.VersionType, input.CreatedBy, nil, now)
	if err != nil {
		return fmt.Errorf("create document version entity: %w", err)
	}
	if err := uc.versions.Create(ctx, &version); err != nil {
		return fmt.Errorf("save document version: %w", err)
	}
	return nil
}

type SaveManualVersionInput struct {
	UserID     uuid.UUID
	DocumentID uuid.UUID
}

type SaveManualVersionOutput struct {
	ID        uuid.UUID
	CreatedAt string
}

func (uc *VersionUseCase) SaveManualVersion(ctx context.Context, input SaveManualVersionInput) (*SaveManualVersionOutput, error) {
	doc, err := uc.loadDoc(ctx, input.DocumentID)
	if err != nil {
		return nil, err
	}
	if err := uc.authorizeDoc(ctx, input.UserID, doc); err != nil {
		return nil, err
	}

	content, err := uc.currentContent(ctx, doc)
	if err != nil {
		return nil, fmt.Errorf("load current document content: %w", err)
	}
	if err := validDocumentContent(content); err != nil {
		return nil, err
	}

	now := uc.currentTime()
	var output *SaveManualVersionOutput

	if err := uc.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		locked, found, err := uc.docs.FindByIDForUpdate(txCtx, input.DocumentID)
		if err != nil {
			return fmt.Errorf("find document for manual save: %w", err)
		}
		if !found {
			return ErrDocumentNotFound
		}
		if locked.IsDeleted() {
			return ErrDocumentDeleted
		}

		if locked.Content != content {
			if err := replaceDocumentContentFn(locked, content, input.UserID, locked.Revision, now); err != nil {
				return err
			}
			if err := uc.docs.Update(txCtx, locked); err != nil {
				return fmt.Errorf("update document content: %w", err)
			}
		}

		version, err := newDocumentVersionFn(*locked, content, entity.DocumentVersionManualSave, input.UserID, nil, now)
		if err != nil {
			return fmt.Errorf("create manual save version entity: %w", err)
		}
		if err := uc.versions.Create(txCtx, &version); err != nil {
			return fmt.Errorf("save manual save version: %w", err)
		}

		output = &SaveManualVersionOutput{
			ID:        version.ID,
			CreatedAt: version.CreatedAt.Format(timeFormat),
		}
		return nil
	}); err != nil {
		return nil, err
	}

	if uc.cache != nil {
		if err := uc.cache.SetContent(ctx, input.DocumentID, content, input.UserID, now); err != nil {
			return nil, fmt.Errorf("update document cache after manual save: %w", err)
		}
		if err := uc.cache.MarkClean(ctx, input.DocumentID); err != nil {
			return nil, fmt.Errorf("mark document clean after manual save: %w", err)
		}
	}

	return output, nil
}


type GetVersionInput struct {
	UserID     uuid.UUID
	DocumentID uuid.UUID
	VersionID  uuid.UUID
}

type GetVersionOutput struct {
	ID         uuid.UUID
	DocumentID uuid.UUID
	Content    string
	Type       string
	CreatedBy  uuid.UUID
	CreatedAt  string
}

func (uc *VersionUseCase) GetVersion(ctx context.Context, input GetVersionInput) (*GetVersionOutput, error) {
	doc, err := uc.loadDoc(ctx, input.DocumentID)
	if err != nil {
		return nil, err
	}
	if err := uc.authorizeDoc(ctx, input.UserID, doc); err != nil {
		return nil, err
	}

	version, found, err := uc.versions.FindByID(ctx, input.VersionID)
	if err != nil {
		return nil, fmt.Errorf("find document version: %w", err)
	}
	if !found || version.DocumentID != input.DocumentID {
		return nil, ErrVersionNotFound
	}

	return &GetVersionOutput{
		ID:         version.ID,
		DocumentID: version.DocumentID,
		Content:    version.Content,
		Type:       string(version.Type),
		CreatedBy:  version.CreatedBy,
		CreatedAt:  version.CreatedAt.Format(timeFormat),
	}, nil
}


type ListVersionsInput struct {
	UserID     uuid.UUID
	DocumentID uuid.UUID
}

type VersionListItem struct {
	ID        uuid.UUID
	Type      string
	CreatedBy uuid.UUID
	CreatedAt string
}

func (uc *VersionUseCase) ListVersions(ctx context.Context, input ListVersionsInput) ([]VersionListItem, error) {
	doc, err := uc.loadDoc(ctx, input.DocumentID)
	if err != nil {
		return nil, err
	}
	if err := uc.authorizeDoc(ctx, input.UserID, doc); err != nil {
		return nil, err
	}

	versions, err := uc.versions.FindByDocumentID(ctx, input.DocumentID)
	if err != nil {
		return nil, fmt.Errorf("list document versions: %w", err)
	}

	items := make([]VersionListItem, 0, len(versions))
	for _, v := range versions {
		items = append(items, VersionListItem{
			ID:        v.ID,
			Type:      string(v.Type),
			CreatedBy: v.CreatedBy,
			CreatedAt: v.CreatedAt.Format(timeFormat),
		})
	}
	return items, nil
}


type RestoreVersionInput struct {
	UserID     uuid.UUID
	DocumentID uuid.UUID
	VersionID  uuid.UUID
}

type RestoreVersionOutput struct {
	DocumentID uuid.UUID
	VersionID  uuid.UUID
	RestoredAt string
}

func (uc *VersionUseCase) RestoreVersion(ctx context.Context, input RestoreVersionInput) (*RestoreVersionOutput, error) {
	doc, err := uc.loadDoc(ctx, input.DocumentID)
	if err != nil {
		return nil, err
	}
	if err := uc.authorizeDoc(ctx, input.UserID, doc); err != nil {
		return nil, err
	}

	target, found, err := uc.versions.FindByID(ctx, input.VersionID)
	if err != nil {
		return nil, fmt.Errorf("find restore version: %w", err)
	}
	if !found || target.DocumentID != input.DocumentID {
		return nil, ErrVersionNotFound
	}

	currentContent, err := uc.currentContent(ctx, doc)
	if err != nil {
		return nil, fmt.Errorf("load current document content: %w", err)
	}

	now := uc.currentTime()
	sourceID := target.ID
	var output *RestoreVersionOutput

	if err := uc.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		locked, found, err := uc.docs.FindByIDForUpdate(txCtx, input.DocumentID)
		if err != nil {
			return fmt.Errorf("find document for restore: %w", err)
		}
		if !found {
			return ErrDocumentNotFound
		}
		if locked.IsDeleted() {
			return ErrDocumentDeleted
		}

		before, err := newDocumentVersionFn(*locked, currentContent, entity.DocumentVersionBeforeRestore, input.UserID, nil, now)
		if err != nil {
			return fmt.Errorf("create before_restore version: %w", err)
		}
		if err := uc.versions.Create(txCtx, &before); err != nil {
			return fmt.Errorf("save before_restore version: %w", err)
		}

		if err := replaceDocumentContentFn(locked, target.Content, input.UserID, locked.Revision, now); err != nil {
			return err
		}
		if err := uc.docs.Update(txCtx, locked); err != nil {
			return fmt.Errorf("update restored document: %w", err)
		}

		restored, err := newDocumentVersionFn(*locked, target.Content, entity.DocumentVersionRestore, input.UserID, &sourceID, now)
		if err != nil {
			return fmt.Errorf("create restore version: %w", err)
		}
		if err := uc.versions.Create(txCtx, &restored); err != nil {
			return fmt.Errorf("save restore version: %w", err)
		}

		if uc.auditLogs != nil {
			docID := locked.ID
			audit, err := newAuditLogFn(
				&input.UserID,
				"DOCUMENT_RESTORED",
				"document",
				&docID,
				map[string]string{"source_version_id": sourceID.String()},
				now,
			)
			if err != nil {
				return fmt.Errorf("create document restored audit log entity: %w", err)
			}
			audit.WorkspaceID = &locked.WorkspaceID
			entity.ApplyDocumentAuditContext(&audit, locked.WorkspaceID, "", "")
			if err := uc.auditLogs.Create(txCtx, &audit); err != nil {
				return fmt.Errorf("save document restored audit log: %w", err)
			}
		}

		output = &RestoreVersionOutput{
			DocumentID: locked.ID,
			VersionID:  restored.ID,
			RestoredAt: now.Format(timeFormat),
		}
		return nil
	}); err != nil {
		return nil, err
	}

	if uc.cache != nil {
		if err := uc.cache.SetContent(ctx, output.DocumentID, target.Content, input.UserID, now); err != nil {
			return nil, fmt.Errorf("update document cache after restore: %w", err)
		}
	}
	if uc.notifier != nil {
		if err := uc.notifier.NotifyDocumentRestored(output.DocumentID, target.Content, target.ID); err != nil {
			return nil, fmt.Errorf("notify document restored: %w", err)
		}
	}
	return output, nil
}
