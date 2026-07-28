package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
)

type IDocumentUsecase interface {
	CreateDocument(ctx context.Context, input CreateDocumentInput) (*CreateDocumentOutput, error)
	ListDocuments(ctx context.Context, input ListDocumentsInput) ([]DocumentListItem, error)
	GetDocument(ctx context.Context, input GetDocumentInput) (*GetDocumentOutput, error)
	UpdateDocument(ctx context.Context, input UpdateDocumentInput) (*UpdateDocumentOutput, error)
	DeleteDocument(ctx context.Context, input DeleteDocumentInput) (*DeleteDocumentOutput, error)
}

type IDocumentRepository interface {
	Create(ctx context.Context, doc *entity.Document) error
	FindByID(ctx context.Context, documentID uuid.UUID) (*entity.Document, bool, error)
	FindByIDForUpdate(ctx context.Context, documentID uuid.UUID) (*entity.Document, bool, error)
	FindByWorkspaceID(ctx context.Context, workspaceID uuid.UUID) ([]entity.Document, error)
	Update(ctx context.Context, doc *entity.Document) error
}

type DocumentUseCase struct {
	docs         IDocumentRepository
	auditLogs    IAuditLogRepository
	transactions ITransactionManager
	access       IAccessCheck
	cache        IDocumentCache
	now          func() time.Time
}

func NewDocumentUseCase(
	docs IDocumentRepository,
	auditLogs IAuditLogRepository,
	transactions ITransactionManager,
	access IAccessCheck,
	cache IDocumentCache,
) *DocumentUseCase {
	return &DocumentUseCase{
		docs:         docs,
		auditLogs:    auditLogs,
		transactions: transactions,
		access:       access,
		cache:        cache,
		now:          time.Now,
	}
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
		if content, ok, err := uc.cache.GetContent(ctx, doc.ID); err != nil {
			return "", err
		} else if ok {
			return content, nil
		}
	}
	return doc.Content, nil
}
