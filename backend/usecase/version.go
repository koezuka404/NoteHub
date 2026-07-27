package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
)

type IVersionUsecase interface {
	ListVersions(ctx context.Context, input ListVersionsInput) ([]VersionListItem, error)
	GetVersion(ctx context.Context, input GetVersionInput) (*GetVersionOutput, error)
	CreateVersion(ctx context.Context, input CreateVersionInput) error
	RestoreVersion(ctx context.Context, input RestoreVersionInput) (*RestoreVersionOutput, error)
}

type IVersionRepository interface {
	Create(ctx context.Context, version *entity.DocumentVersion) error
	FindByID(ctx context.Context, versionID uuid.UUID) (*entity.DocumentVersion, bool, error)
	FindByDocumentID(ctx context.Context, documentID uuid.UUID) ([]entity.DocumentVersion, error)
}

type VersionUseCase struct {
	docs         IDocumentRepository
	versions     IVersionRepository
	transactions ITransactionManager
	access       IAccessCheck
	cache        IDocumentCache
	now          func() time.Time
}

func NewVersionUseCase(
	docs IDocumentRepository,
	versions IVersionRepository,
	transactions ITransactionManager,
	access IAccessCheck,
	cache IDocumentCache,
) *VersionUseCase {
	return &VersionUseCase{
		docs:         docs,
		versions:     versions,
		transactions: transactions,
		access:       access,
		cache:        cache,
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
		if content, ok, err := uc.cache.GetContent(ctx, doc.ID); err != nil {
			return "", err
		} else if ok {
			return content, nil
		}
	}
	return doc.Content, nil
}
