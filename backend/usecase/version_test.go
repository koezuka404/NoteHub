package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
)

func newVersionUseCase(docs *stubDocumentRepo, versions *stubVersionRepo, access *stubAccessCheck, cache *stubDocumentCache, notifier *stubDocumentNotifier) *VersionUseCase {
	uc := NewVersionUseCase(docs, versions, &mockTransactionManager{}, access, cache, notifier, &mockAuditLogRepo{})
	uc.now = usecaseTestNow
	return uc
}

func versionFixture() (userID, workspaceID uuid.UUID, doc entity.Document) {
	userID = uuid.New()
	workspaceID = uuid.New()
	doc = activeDocument(workspaceID, userID)
	return userID, workspaceID, doc
}

func TestCreateVersion_Success(t *testing.T) {
	_, workspaceID, doc := versionFixture()
	docs := &stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}}
	versions := &stubVersionRepo{}
	uc := newVersionUseCase(docs, versions, docAccess(workspaceID), nil, nil)

	err := uc.CreateVersion(context.Background(), CreateVersionInput{
		DocumentID: doc.ID, Content: "v1", CreatedBy: doc.CreatedBy, VersionType: entity.DocumentVersionAutoSave,
	})
	if err != nil {
		t.Fatalf("CreateVersion: %v", err)
	}
}

func TestCreateVersion_DocumentNotFound(t *testing.T) {
	uc := newVersionUseCase(&stubDocumentRepo{}, &stubVersionRepo{}, docAccess(uuid.New()), nil, nil)
	err := uc.CreateVersion(context.Background(), CreateVersionInput{DocumentID: uuid.New(), Content: "x", CreatedBy: uuid.New(), VersionType: entity.DocumentVersionAutoSave})
	if !errors.Is(err, ErrDocumentNotFound) {
		t.Fatalf("expected ErrDocumentNotFound, got %v", err)
	}
}

func TestSaveManualVersion_Success(t *testing.T) {
	userID, workspaceID, doc := versionFixture()
	docs := &stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}}
	versions := &stubVersionRepo{}
	cache := &stubDocumentCache{
		content: map[uuid.UUID]DocumentContentState{
			doc.ID: {Content: "edited", UpdatedBy: userID, UpdatedAt: usecaseTestNow()},
		},
		hasState: true,
	}
	uc := newVersionUseCase(docs, versions, docAccess(workspaceID), cache, nil)

	out, err := uc.SaveManualVersion(context.Background(), SaveManualVersionInput{UserID: userID, DocumentID: doc.ID})
	if err != nil {
		t.Fatalf("SaveManualVersion: %v", err)
	}
	if out.ID == uuid.Nil {
		t.Fatal("expected version id")
	}
}

func TestSaveManualVersion_AccessDenied(t *testing.T) {
	userID, _, doc := versionFixture()
	access := &stubAccessCheck{err: ErrWorkspaceAccessDenied}
	uc := newVersionUseCase(&stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}}, &stubVersionRepo{}, access, nil, nil)

	_, err := uc.SaveManualVersion(context.Background(), SaveManualVersionInput{UserID: userID, DocumentID: doc.ID})
	if !errors.Is(err, ErrWorkspaceAccessDenied) {
		t.Fatalf("expected access denied, got %v", err)
	}
}

func TestGetVersion_Success(t *testing.T) {
	userID, workspaceID, doc := versionFixture()
	now := usecaseTestNow()
	version, _ := entity.NewDocumentVersion(doc, "snapshot", entity.DocumentVersionManualSave, userID, nil, now)
	versions := &stubVersionRepo{byID: map[uuid.UUID]*entity.DocumentVersion{version.ID: &version}}
	docs := &stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}}
	uc := newVersionUseCase(docs, versions, docAccess(workspaceID), nil, nil)

	out, err := uc.GetVersion(context.Background(), GetVersionInput{
		UserID: userID, DocumentID: doc.ID, VersionID: version.ID,
	})
	if err != nil {
		t.Fatalf("GetVersion: %v", err)
	}
	if out.Content != "snapshot" {
		t.Fatalf("content = %q", out.Content)
	}
}

func TestGetVersion_NotFound(t *testing.T) {
	userID, workspaceID, doc := versionFixture()
	docs := &stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}}
	uc := newVersionUseCase(docs, &stubVersionRepo{}, docAccess(workspaceID), nil, nil)

	_, err := uc.GetVersion(context.Background(), GetVersionInput{
		UserID: userID, DocumentID: doc.ID, VersionID: uuid.New(),
	})
	if !errors.Is(err, ErrVersionNotFound) {
		t.Fatalf("expected ErrVersionNotFound, got %v", err)
	}
}

func TestListVersions_Success(t *testing.T) {
	userID, workspaceID, doc := versionFixture()
	now := usecaseTestNow()
	v1, _ := entity.NewDocumentVersion(doc, "a", entity.DocumentVersionAutoSave, userID, nil, now)
	v2, _ := entity.NewDocumentVersion(doc, "b", entity.DocumentVersionManualSave, userID, nil, now.Add(time.Minute))
	versions := &stubVersionRepo{byID: map[uuid.UUID]*entity.DocumentVersion{v1.ID: &v1, v2.ID: &v2}}
	docs := &stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}}
	uc := newVersionUseCase(docs, versions, docAccess(workspaceID), nil, nil)

	items, err := uc.ListVersions(context.Background(), ListVersionsInput{UserID: userID, DocumentID: doc.ID})
	if err != nil {
		t.Fatalf("ListVersions: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(items))
	}
}

func TestRestoreVersion_Success(t *testing.T) {
	userID, workspaceID, doc := versionFixture()
	now := usecaseTestNow()
	target, _ := entity.NewDocumentVersion(doc, "restored content", entity.DocumentVersionManualSave, userID, nil, now)
	versions := &stubVersionRepo{byID: map[uuid.UUID]*entity.DocumentVersion{target.ID: &target}}
	docs := &stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}}
	cache := &stubDocumentCache{}
	notifier := &stubDocumentNotifier{}
	uc := newVersionUseCase(docs, versions, docAccess(workspaceID), cache, notifier)

	out, err := uc.RestoreVersion(context.Background(), RestoreVersionInput{
		UserID: userID, DocumentID: doc.ID, VersionID: target.ID,
	})
	if err != nil {
		t.Fatalf("RestoreVersion: %v", err)
	}
	if out.DocumentID != doc.ID {
		t.Fatalf("unexpected output: %+v", out)
	}
	updated := docs.byID[doc.ID]
	if updated.Content != "restored content" {
		t.Fatalf("content = %q", updated.Content)
	}
}

func TestRestoreVersion_NotFound(t *testing.T) {
	userID, workspaceID, doc := versionFixture()
	docs := &stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}}
	uc := newVersionUseCase(docs, &stubVersionRepo{}, docAccess(workspaceID), nil, nil)

	_, err := uc.RestoreVersion(context.Background(), RestoreVersionInput{
		UserID: userID, DocumentID: doc.ID, VersionID: uuid.New(),
	})
	if !errors.Is(err, ErrVersionNotFound) {
		t.Fatalf("expected ErrVersionNotFound, got %v", err)
	}
}

func TestRestoreVersion_NotifierError(t *testing.T) {
	userID, workspaceID, doc := versionFixture()
	now := usecaseTestNow()
	target, _ := entity.NewDocumentVersion(doc, "restored", entity.DocumentVersionManualSave, userID, nil, now)
	versions := &stubVersionRepo{byID: map[uuid.UUID]*entity.DocumentVersion{target.ID: &target}}
	docs := &stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}}
	notifier := &stubDocumentNotifier{err: fmt.Errorf("notify failed")}
	uc := newVersionUseCase(docs, versions, docAccess(workspaceID), &stubDocumentCache{}, notifier)

	_, err := uc.RestoreVersion(context.Background(), RestoreVersionInput{
		UserID: userID, DocumentID: doc.ID, VersionID: target.ID,
	})
	if err == nil {
		t.Fatal("expected notifier error")
	}
}
