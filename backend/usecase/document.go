package usecase

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
	"github.com/koezuka404/notehub/repository"
	"strings"
	"time"
	"unicode/utf8"
)

// document_helpers.go

const maxDocumentTitleLength = 100
const maxDocumentContentBytes = 512 * 1024

type IDocumentUsecase interface {
	CreateDocument(ctx context.Context, input CreateDocumentInput) (*CreateDocumentOutput, error)
	ListDocuments(ctx context.Context, input ListDocumentsInput) ([]DocumentListItem, error)
	GetDocument(ctx context.Context, input GetDocumentInput) (*GetDocumentOutput, error)
	UpdateDocument(ctx context.Context, input UpdateDocumentInput) (*UpdateDocumentOutput, error)
	DeleteDocument(ctx context.Context, input DeleteDocumentInput) (*DeleteDocumentOutput, error)
}

// document_cache.go

type DocumentContentState struct {
	Content   string
	UpdatedBy uuid.UUID
	UpdatedAt time.Time
}

type IDocumentCache interface {
	GetContentState(ctx context.Context, documentID uuid.UUID) (DocumentContentState, bool, error)
	SetContent(ctx context.Context, documentID uuid.UUID, content string, updatedBy uuid.UUID, updatedAt time.Time) error
	IsDirty(ctx context.Context, documentID uuid.UUID) (bool, error)
	MarkClean(ctx context.Context, documentID uuid.UUID) error
	GetRevision(ctx context.Context, documentID uuid.UUID) (uint64, error)
	ListDirtyDocumentIDs(ctx context.Context) ([]uuid.UUID, error)
	ListIdleDirtyDocumentIDs(ctx context.Context, idle time.Duration) ([]uuid.UUID, error)
	Clear(ctx context.Context, documentID uuid.UUID) error
}

type IAutoSaveLockStore interface {
	TryLock(ctx context.Context, key string, ttl time.Duration) (bool, error)
	Unlock(ctx context.Context, key string) error
}

// document_flush.go

type IDocumentFlushService interface {
	FlushDocument(ctx context.Context, documentID uuid.UUID) error
	FlushAllDirty(ctx context.Context) error
	FlushWorkspaceDocuments(ctx context.Context, workspaceID uuid.UUID) error
}

// document_websocket_notifier.go

type IDocumentWebSocketNotifier interface {
	NotifyDocumentRestored(documentID uuid.UUID, content string, sourceVersionID uuid.UUID) error
	NotifyDocumentDeleted(documentID, deletedBy uuid.UUID, deletedAt string) error
	NotifyWorkspaceDeleted(workspaceID, deletedBy uuid.UUID, deletedAt string) error
}

type DocumentUseCase struct {
	docs         repository.DocumentRepository
	auditLogs    repository.AuditLogRepository
	transactions repository.TransactionManager
	access       IAccessCheck
	cache        IDocumentCache
	flush        IDocumentFlushService
	notifier     IDocumentWebSocketNotifier
	now          func() time.Time
}

func NewDocumentUseCase(
	docs repository.DocumentRepository,
	auditLogs repository.AuditLogRepository,
	transactions repository.TransactionManager,
	access IAccessCheck,
	cache IDocumentCache,
	flush IDocumentFlushService,
	notifier IDocumentWebSocketNotifier,
) *DocumentUseCase {
	return &DocumentUseCase{
		docs:         docs,
		auditLogs:    auditLogs,
		transactions: transactions,
		access:       access,
		cache:        cache,
		flush:        flush,
		notifier:     notifier,
		now:          time.Now,
	}
}

func normalizeTitle(title string) string {
	return strings.TrimSpace(title)
}

func validTitle(title string) error {
	title = normalizeTitle(title)
	if title == "" {
		return ErrValidation
	}
	if utf8.RuneCountInString(title) > maxDocumentTitleLength {
		return ErrValidation
	}
	return nil
}

func validDocumentContent(content string) error {
	if len([]byte(content)) > maxDocumentContentBytes {
		return ErrDocumentContentTooLarge
	}
	return nil
}

func (uc *DocumentUseCase) currentTime() time.Time {
	return uc.now().UTC()
}

func (uc *DocumentUseCase) requireMember(ctx context.Context, userID, workspaceID uuid.UUID) error {
	_, err := uc.access.CheckWorkspaceAccess(ctx, CheckWorkspaceAccessInput{
		UserID: userID, WorkspaceID: workspaceID,
	})
	return err
}

func (uc *DocumentUseCase) loadDoc(ctx context.Context, documentID uuid.UUID) (*entity.Document, error) {
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

func (uc *DocumentUseCase) authorizeDoc(ctx context.Context, userID uuid.UUID, doc *entity.Document) error {
	return uc.requireMember(ctx, userID, doc.WorkspaceID)
}

func (uc *DocumentUseCase) docContent(ctx context.Context, doc *entity.Document) (string, error) {
	if uc.cache != nil {
		if state, ok, err := uc.cache.GetContentState(ctx, doc.ID); err != nil {
			return "", err
		} else if ok {
			return state.Content, nil
		}
	}
	return doc.Content, nil
}

// document_create.go

type CreateDocumentInput struct {
	UserID      uuid.UUID
	WorkspaceID uuid.UUID
	Title       string
	Content     string
	IPAddress   string
}

type CreateDocumentOutput struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Title       string
	Content     string
	CreatedBy   uuid.UUID
	CreatedAt   string
}

func (uc *DocumentUseCase) CreateDocument(ctx context.Context, input CreateDocumentInput) (*CreateDocumentOutput, error) {
	if err := validTitle(input.Title); err != nil {
		return nil, err
	}
	if err := validDocumentContent(input.Content); err != nil {
		return nil, err
	}
	if err := uc.requireMember(ctx, input.UserID, input.WorkspaceID); err != nil {
		return nil, err
	}

	now := uc.currentTime()
	doc, err := entity.NewDocument(input.WorkspaceID, input.UserID, normalizeTitle(input.Title), input.Content, now)
	if err != nil {
		return nil, fmt.Errorf("create document entity: %w", err)
	}

	if err := uc.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.docs.Create(txCtx, &doc); err != nil {
			return fmt.Errorf("save document: %w", err)
		}
		audit, err := entity.NewAuditLog(&input.UserID, "DOCUMENT_CREATED", "document", &doc.ID, nil, now)
		if err != nil {
			return fmt.Errorf("create audit log entity: %w", err)
		}
		audit.IPAddress = input.IPAddress
		if err := uc.auditLogs.Create(txCtx, &audit); err != nil {
			return fmt.Errorf("save audit log: %w", err)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return &CreateDocumentOutput{
		ID:          doc.ID,
		WorkspaceID: doc.WorkspaceID,
		Title:       doc.Title,
		Content:     doc.Content,
		CreatedBy:   doc.CreatedBy,
		CreatedAt:   doc.CreatedAt.Format(timeFormat),
	}, nil
}

// document_get.go

type GetDocumentInput struct {
	UserID     uuid.UUID
	DocumentID uuid.UUID
}

type GetDocumentOutput struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Title       string
	Content     string
	UpdatedBy   uuid.UUID
	UpdatedAt   string
}

func (uc *DocumentUseCase) GetDocument(ctx context.Context, input GetDocumentInput) (*GetDocumentOutput, error) {
	doc, err := uc.loadDoc(ctx, input.DocumentID)
	if err != nil {
		return nil, err
	}
	if err := uc.authorizeDoc(ctx, input.UserID, doc); err != nil {
		return nil, err
	}

	content, err := uc.docContent(ctx, doc)
	if err != nil {
		return nil, fmt.Errorf("resolve document content: %w", err)
	}

	return &GetDocumentOutput{
		ID:          doc.ID,
		WorkspaceID: doc.WorkspaceID,
		Title:       doc.Title,
		Content:     content,
		UpdatedBy:   doc.UpdatedBy,
		UpdatedAt:   doc.UpdatedAt.Format(timeFormat),
	}, nil
}

// document_list.go

type ListDocumentsInput struct {
	UserID      uuid.UUID
	WorkspaceID uuid.UUID
}

type DocumentListItem struct {
	ID        uuid.UUID
	Title     string
	UpdatedBy uuid.UUID
	UpdatedAt string
}

func (uc *DocumentUseCase) ListDocuments(ctx context.Context, input ListDocumentsInput) ([]DocumentListItem, error) {
	if err := uc.requireMember(ctx, input.UserID, input.WorkspaceID); err != nil {
		return nil, err
	}

	docs, err := uc.docs.FindByWorkspaceID(ctx, input.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}

	items := make([]DocumentListItem, 0, len(docs))
	for _, doc := range docs {
		items = append(items, DocumentListItem{
			ID:        doc.ID,
			Title:     doc.Title,
			UpdatedBy: doc.UpdatedBy,
			UpdatedAt: doc.UpdatedAt.Format(timeFormat),
		})
	}
	return items, nil
}

// document_update.go

type UpdateDocumentInput struct {
	UserID     uuid.UUID
	DocumentID uuid.UUID
	Title      string
	IPAddress  string
}

type UpdateDocumentOutput struct {
	ID        uuid.UUID
	Title     string
	UpdatedAt string
}

func (uc *DocumentUseCase) UpdateDocument(ctx context.Context, input UpdateDocumentInput) (*UpdateDocumentOutput, error) {
	if err := validTitle(input.Title); err != nil {
		return nil, err
	}

	doc, err := uc.loadDoc(ctx, input.DocumentID)
	if err != nil {
		return nil, err
	}
	if err := uc.authorizeDoc(ctx, input.UserID, doc); err != nil {
		return nil, err
	}

	now := uc.currentTime()
	var output *UpdateDocumentOutput

	if err := uc.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		locked, found, err := uc.docs.FindByIDForUpdate(txCtx, input.DocumentID)
		if err != nil {
			return fmt.Errorf("find document for update: %w", err)
		}
		if !found {
			return ErrDocumentNotFound
		}
		if locked.IsDeleted() {
			return ErrDocumentDeleted
		}
		if err := locked.Rename(normalizeTitle(input.Title), input.UserID, now); err != nil {
			return err
		}
		if err := uc.docs.Update(txCtx, locked); err != nil {
			return fmt.Errorf("update document: %w", err)
		}

		audit, err := entity.NewAuditLog(&input.UserID, "DOCUMENT_UPDATED", "document", &locked.ID, nil, now)
		if err != nil {
			return fmt.Errorf("create audit log entity: %w", err)
		}
		audit.IPAddress = input.IPAddress
		if err := uc.auditLogs.Create(txCtx, &audit); err != nil {
			return fmt.Errorf("save audit log: %w", err)
		}

		output = &UpdateDocumentOutput{
			ID:        locked.ID,
			Title:     locked.Title,
			UpdatedAt: locked.UpdatedAt.Format(timeFormat),
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return output, nil
}

// document_delete.go

type DeleteDocumentInput struct {
	UserID     uuid.UUID
	DocumentID uuid.UUID
	IPAddress  string
}

type DeleteDocumentOutput struct {
	DocumentID uuid.UUID
	DeletedAt  string
}

func (uc *DocumentUseCase) DeleteDocument(ctx context.Context, input DeleteDocumentInput) (*DeleteDocumentOutput, error) {
	doc, err := uc.loadDoc(ctx, input.DocumentID)
	if err != nil {
		return nil, err
	}
	if err := uc.authorizeDoc(ctx, input.UserID, doc); err != nil {
		return nil, err
	}

	if uc.flush != nil {
		if err := uc.flush.FlushDocument(ctx, input.DocumentID); err != nil {
			return nil, fmt.Errorf("flush document before delete: %w", err)
		}
	}

	now := uc.currentTime()
	var output *DeleteDocumentOutput

	if err := uc.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		locked, found, err := uc.docs.FindByIDForUpdate(txCtx, input.DocumentID)
		if err != nil {
			return fmt.Errorf("find document for delete: %w", err)
		}
		if !found {
			return ErrDocumentNotFound
		}
		if locked.IsDeleted() {
			return ErrDocumentDeleted
		}
		if err := locked.LogicalDelete(input.UserID, now); err != nil {
			return err
		}
		if err := uc.docs.Update(txCtx, locked); err != nil {
			return fmt.Errorf("update deleted document: %w", err)
		}

		audit, err := entity.NewAuditLog(&input.UserID, "DOCUMENT_DELETED", "document", &locked.ID, nil, now)
		if err != nil {
			return fmt.Errorf("create audit log entity: %w", err)
		}
		audit.IPAddress = input.IPAddress
		if err := uc.auditLogs.Create(txCtx, &audit); err != nil {
			return fmt.Errorf("save audit log: %w", err)
		}

		output = &DeleteDocumentOutput{
			DocumentID: locked.ID,
			DeletedAt:  locked.DeletedAt.Format(timeFormat),
		}
		return nil
	}); err != nil {
		return nil, err
	}

	if uc.cache != nil {
		if err := uc.cache.Clear(ctx, input.DocumentID); err != nil {
			return nil, fmt.Errorf("clear document cache: %w", err)
		}
	}
	if uc.notifier != nil {
		if err := uc.notifier.NotifyDocumentDeleted(output.DocumentID, input.UserID, output.DeletedAt); err != nil {
			return nil, fmt.Errorf("notify document deleted: %w", err)
		}
	}
	return output, nil
}
