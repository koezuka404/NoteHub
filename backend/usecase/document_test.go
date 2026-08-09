package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
)

func docAccess(workspaceID uuid.UUID) *stubAccessCheck {
	return &stubAccessCheck{result: &WorkspaceAccessResult{
		Workspace: &entity.Workspace{ID: workspaceID},
		Member:    &entity.WorkspaceMember{Role: entity.WorkspaceRoleMember},
	}}
}

func newDocumentUseCase(docs *stubDocumentRepo, access *stubAccessCheck, cache *stubDocumentCache, flush *stubDocumentFlush, notifier *stubDocumentNotifier) *DocumentUseCase {
	uc := NewDocumentUseCase(docs, &mockAuditLogRepo{}, &mockTransactionManager{}, access, cache, flush, notifier)
	uc.now = usecaseTestNow
	return uc
}

func activeDocument(workspaceID, userID uuid.UUID) entity.Document {
	now := usecaseTestNow()
	doc, _ := entity.NewDocument(workspaceID, userID, "Title", "content", now)
	return doc
}

func TestValidTitle(t *testing.T) {
	if err := validTitle("  My Doc  "); err != nil {
		t.Fatalf("valid title: %v", err)
	}
	if err := validTitle(""); !errors.Is(err, ErrValidation) {
		t.Fatalf("empty title: %v", err)
	}
	if err := validTitle(string(make([]rune, maxDocumentTitleLength+1))); !errors.Is(err, ErrValidation) {
		t.Fatalf("long title: %v", err)
	}
}

func TestValidDocumentContent(t *testing.T) {
	if err := validDocumentContent("hello"); err != nil {
		t.Fatalf("valid content: %v", err)
	}
	large := strings.Repeat("a", maxDocumentContentBytes+1)
	if !errors.Is(validDocumentContent(large), ErrDocumentContentTooLarge) {
		t.Fatal("expected content too large")
	}
}

func TestCreateDocument_Success(t *testing.T) {
	userID := uuid.New()
	workspaceID := uuid.New()
	docs := &stubDocumentRepo{}
	notifier := &stubDocumentNotifier{}
	uc := newDocumentUseCase(docs, docAccess(workspaceID), nil, nil, notifier)

	out, err := uc.CreateDocument(context.Background(), CreateDocumentInput{
		UserID: userID, WorkspaceID: workspaceID, Title: "  Doc  ", Content: "hello",
		IPAddress: "127.0.0.1", UserAgent: "test",
	})
	if err != nil {
		t.Fatalf("CreateDocument: %v", err)
	}
	if out.Title != "Doc" || out.Content != "hello" {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestCreateDocument_ValidationErrors(t *testing.T) {
	uc := newDocumentUseCase(&stubDocumentRepo{}, docAccess(uuid.New()), nil, nil, nil)

	_, err := uc.CreateDocument(context.Background(), CreateDocumentInput{
		UserID: uuid.New(), WorkspaceID: uuid.New(), Title: "",
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("title validation: %v", err)
	}

	large := strings.Repeat("a", maxDocumentContentBytes+1)
	_, err = uc.CreateDocument(context.Background(), CreateDocumentInput{
		UserID: uuid.New(), WorkspaceID: uuid.New(), Title: "Doc", Content: large,
	})
	if !errors.Is(err, ErrDocumentContentTooLarge) {
		t.Fatalf("content validation: %v", err)
	}
}

func TestCreateDocument_AccessDenied(t *testing.T) {
	access := &stubAccessCheck{err: ErrWorkspaceAccessDenied}
	uc := newDocumentUseCase(&stubDocumentRepo{}, access, nil, nil, nil)

	_, err := uc.CreateDocument(context.Background(), CreateDocumentInput{
		UserID: uuid.New(), WorkspaceID: uuid.New(), Title: "Doc",
	})
	if !errors.Is(err, ErrWorkspaceAccessDenied) {
		t.Fatalf("expected access denied, got %v", err)
	}
}

func TestCreateDocument_NotifierError(t *testing.T) {
	workspaceID := uuid.New()
	notifier := &stubDocumentNotifier{err: fmt.Errorf("notify failed")}
	uc := newDocumentUseCase(&stubDocumentRepo{}, docAccess(workspaceID), nil, nil, notifier)

	_, err := uc.CreateDocument(context.Background(), CreateDocumentInput{
		UserID: uuid.New(), WorkspaceID: workspaceID, Title: "Doc",
	})
	if err == nil {
		t.Fatal("expected notifier error")
	}
}

func TestGetDocument_FromCache(t *testing.T) {
	userID := uuid.New()
	workspaceID := uuid.New()
	doc := activeDocument(workspaceID, userID)
	docs := &stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}}
	cache := &stubDocumentCache{
		content: map[uuid.UUID]DocumentContentState{
			doc.ID: {Content: "cached", UpdatedBy: userID, UpdatedAt: usecaseTestNow()},
		},
		hasState: true,
	}
	uc := newDocumentUseCase(docs, docAccess(workspaceID), cache, nil, nil)

	out, err := uc.GetDocument(context.Background(), GetDocumentInput{UserID: userID, DocumentID: doc.ID})
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	if out.Content != "cached" {
		t.Fatalf("content = %q", out.Content)
	}
}

func TestGetDocument_FromDB(t *testing.T) {
	userID := uuid.New()
	workspaceID := uuid.New()
	doc := activeDocument(workspaceID, userID)
	docs := &stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}}
	uc := newDocumentUseCase(docs, docAccess(workspaceID), &stubDocumentCache{}, nil, nil)

	out, err := uc.GetDocument(context.Background(), GetDocumentInput{UserID: userID, DocumentID: doc.ID})
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	if out.Content != "content" {
		t.Fatalf("content = %q", out.Content)
	}
}

func TestGetDocument_NotFound(t *testing.T) {
	uc := newDocumentUseCase(&stubDocumentRepo{}, docAccess(uuid.New()), nil, nil, nil)
	_, err := uc.GetDocument(context.Background(), GetDocumentInput{UserID: uuid.New(), DocumentID: uuid.New()})
	if !errors.Is(err, ErrDocumentNotFound) {
		t.Fatalf("expected ErrDocumentNotFound, got %v", err)
	}
}

func TestGetDocument_Deleted(t *testing.T) {
	userID := uuid.New()
	workspaceID := uuid.New()
	doc := activeDocument(workspaceID, userID)
	now := usecaseTestNow()
	doc.DeletedAt = &now
	docs := &stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}}
	uc := newDocumentUseCase(docs, docAccess(workspaceID), nil, nil, nil)

	_, err := uc.GetDocument(context.Background(), GetDocumentInput{UserID: userID, DocumentID: doc.ID})
	if !errors.Is(err, ErrDocumentDeleted) {
		t.Fatalf("expected ErrDocumentDeleted, got %v", err)
	}
}

func TestListDocuments_Success(t *testing.T) {
	userID := uuid.New()
	workspaceID := uuid.New()
	doc := activeDocument(workspaceID, userID)
	docs := &stubDocumentRepo{byWorkspace: map[uuid.UUID][]entity.Document{workspaceID: {doc}}}
	uc := newDocumentUseCase(docs, docAccess(workspaceID), nil, nil, nil)

	items, err := uc.ListDocuments(context.Background(), ListDocumentsInput{UserID: userID, WorkspaceID: workspaceID})
	if err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 document, got %d", len(items))
	}
}

func TestUpdateDocument_Success(t *testing.T) {
	userID := uuid.New()
	workspaceID := uuid.New()
	doc := activeDocument(workspaceID, userID)
	docs := &stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}}
	uc := newDocumentUseCase(docs, docAccess(workspaceID), nil, nil, &stubDocumentNotifier{})

	out, err := uc.UpdateDocument(context.Background(), UpdateDocumentInput{
		UserID: userID, DocumentID: doc.ID, Title: "  Renamed  ",
	})
	if err != nil {
		t.Fatalf("UpdateDocument: %v", err)
	}
	if out.Title != "Renamed" {
		t.Fatalf("title = %q", out.Title)
	}
}

func TestDeleteDocument_Success(t *testing.T) {
	userID := uuid.New()
	workspaceID := uuid.New()
	doc := activeDocument(workspaceID, userID)
	docs := &stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}}
	cache := &stubDocumentCache{}
	uc := newDocumentUseCase(docs, docAccess(workspaceID), cache, &stubDocumentFlush{}, &stubDocumentNotifier{})

	out, err := uc.DeleteDocument(context.Background(), DeleteDocumentInput{
		UserID: userID, DocumentID: doc.ID,
	})
	if err != nil {
		t.Fatalf("DeleteDocument: %v", err)
	}
	if out.DocumentID != doc.ID {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestDeleteDocument_FlushError(t *testing.T) {
	userID := uuid.New()
	workspaceID := uuid.New()
	doc := activeDocument(workspaceID, userID)
	docs := &stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}}
	flush := &stubDocumentFlush{err: fmt.Errorf("flush failed")}
	uc := newDocumentUseCase(docs, docAccess(workspaceID), nil, flush, nil)

	_, err := uc.DeleteDocument(context.Background(), DeleteDocumentInput{UserID: userID, DocumentID: doc.ID})
	if err == nil {
		t.Fatal("expected flush error")
	}
}

func TestDeleteDocument_CacheClearError(t *testing.T) {
	userID := uuid.New()
	workspaceID := uuid.New()
	doc := activeDocument(workspaceID, userID)
	docs := &stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}}
	cache := &stubDocumentCache{setErr: fmt.Errorf("clear failed")}
	uc := newDocumentUseCase(docs, docAccess(workspaceID), cache, &stubDocumentFlush{}, &stubDocumentNotifier{})

	_, err := uc.DeleteDocument(context.Background(), DeleteDocumentInput{UserID: userID, DocumentID: doc.ID})
	if err == nil {
		t.Fatal("expected cache clear error")
	}
}
