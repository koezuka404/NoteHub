package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
)

func newWebSocketUseCase(
	docs *stubDocumentRepo,
	users *stubUserRepoFull,
	cache IDocumentCache,
	sessions *stubWebSocketSessions,
	editors *stubDocumentEditors,
	access *stubAccessCheck,
) *DocumentWebSocketUseCase {
	uc := NewDocumentWebSocketUseCase(docs, users, cache, sessions, editors, access, 2, 16*time.Minute)
	uc.now = usecaseTestNow
	return uc
}

func wsFixture() (userID, workspaceID uuid.UUID, doc entity.Document, users *stubUserRepoFull, access *stubAccessCheck) {
	userID = uuid.New()
	workspaceID = uuid.New()
	doc = activeDocument(workspaceID, userID)
	users = newStubUserRepoFull(map[uuid.UUID]*entity.User{
		userID: {ID: userID, Name: "Alice", Status: entity.UserStatusActive},
	})
	access = docAccess(workspaceID)
	return userID, workspaceID, doc, users, access
}

func TestPrepareConnection_Success(t *testing.T) {
	userID, _, doc, users, access := wsFixture()
	docs := &stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}}
	cache := &stubDocumentCache{
		content: map[uuid.UUID]DocumentContentState{
			doc.ID: {Content: "live", UpdatedBy: userID, UpdatedAt: usecaseTestNow()},
		},
		hasState: true,
	}
	uc := newWebSocketUseCase(docs, users, cache, &stubWebSocketSessions{}, &stubDocumentEditors{}, access)

	out, err := uc.PrepareConnection(context.Background(), PrepareWebSocketConnectionInput{
		UserID: userID, DocumentID: doc.ID,
	})
	if err != nil {
		t.Fatalf("PrepareConnection: %v", err)
	}
	if out.Content != "live" {
		t.Fatalf("content = %q", out.Content)
	}
}

func TestPrepareConnection_ValidationError(t *testing.T) {
	uc := newWebSocketUseCase(&stubDocumentRepo{}, &stubUserRepoFull{}, nil, &stubWebSocketSessions{}, &stubDocumentEditors{}, docAccess(uuid.New()))
	_, err := uc.PrepareConnection(context.Background(), PrepareWebSocketConnectionInput{})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestPrepareWorkspaceConnection_Success(t *testing.T) {
	userID, workspaceID, _, users, access := wsFixture()
	uc := newWebSocketUseCase(&stubDocumentRepo{}, users, nil, &stubWebSocketSessions{}, &stubDocumentEditors{}, access)

	if err := uc.PrepareWorkspaceConnection(context.Background(), PrepareWorkspaceConnectionInput{
		UserID: userID, WorkspaceID: workspaceID,
	}); err != nil {
		t.Fatalf("PrepareWorkspaceConnection: %v", err)
	}
}

func TestPrepareWorkspaceConnection_AccessDenied(t *testing.T) {
	access := &stubAccessCheck{err: ErrWorkspaceAccessDenied}
	uc := newWebSocketUseCase(&stubDocumentRepo{}, &stubUserRepoFull{}, nil, &stubWebSocketSessions{}, &stubDocumentEditors{}, access)

	err := uc.PrepareWorkspaceConnection(context.Background(), PrepareWorkspaceConnectionInput{
		UserID: uuid.New(), WorkspaceID: uuid.New(),
	})
	if !errors.Is(err, ErrWorkspaceAccessDenied) {
		t.Fatalf("expected access denied, got %v", err)
	}
}

func TestRegisterConnection_Success(t *testing.T) {
	userID, _, doc, users, access := wsFixture()
	docs := &stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}}
	sessions := &stubWebSocketSessions{added: true, count: 1}
	editors := &stubDocumentEditors{joined: true, editors: []DocumentEditorInfo{{UserID: userID, Name: "Alice"}}}
	uc := newWebSocketUseCase(docs, users, &stubDocumentCache{}, sessions, editors, access)

	out, err := uc.RegisterConnection(context.Background(), RegisterWebSocketConnectionInput{
		UserID: userID, DocumentID: doc.ID, ConnectionID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("RegisterConnection: %v", err)
	}
	if !out.EditorJoined {
		t.Fatal("expected editor joined")
	}
}

func TestRegisterConnection_LimitExceeded(t *testing.T) {
	userID, _, doc, users, access := wsFixture()
	docs := &stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}}
	sessions := &stubWebSocketSessions{added: false}
	uc := newWebSocketUseCase(docs, users, &stubDocumentCache{}, sessions, &stubDocumentEditors{}, access)

	_, err := uc.RegisterConnection(context.Background(), RegisterWebSocketConnectionInput{
		UserID: userID, DocumentID: doc.ID, ConnectionID: uuid.New(),
	})
	if !errors.Is(err, ErrWebSocketConnectionLimitExceeded) {
		t.Fatalf("expected limit exceeded, got %v", err)
	}
}

func TestRegisterConnection_EditorAddFailureRollback(t *testing.T) {
	userID, _, doc, users, access := wsFixture()
	docs := &stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}}
	sessions := &stubWebSocketSessions{added: true}
	editors := &stubDocumentEditors{addErr: fmt.Errorf("editor store down")}
	uc := newWebSocketUseCase(docs, users, &stubDocumentCache{}, sessions, editors, access)

	_, err := uc.RegisterConnection(context.Background(), RegisterWebSocketConnectionInput{
		UserID: userID, DocumentID: doc.ID, ConnectionID: uuid.New(),
	})
	if err == nil {
		t.Fatal("expected editor add error")
	}
}

func TestUnregisterConnection_RemainingConnections(t *testing.T) {
	userID, _, doc, users, _ := wsFixture()
	sessions := &stubWebSocketSessions{remaining: 2}
	editors := &stubDocumentEditors{editors: []DocumentEditorInfo{{UserID: userID, Name: "Alice"}}}
	uc := newWebSocketUseCase(&stubDocumentRepo{}, users, nil, sessions, editors, docAccess(doc.WorkspaceID))

	out, err := uc.UnregisterConnection(context.Background(), UnregisterWebSocketConnectionInput{
		UserID: userID, DocumentID: doc.ID, ConnectionID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("UnregisterConnection: %v", err)
	}
	if out.EditorLeft {
		t.Fatal("expected editor not removed when connections remain")
	}
	if len(out.Editors) != 1 {
		t.Fatalf("editors = %+v", out.Editors)
	}
}

func TestUnregisterConnection_LastConnection(t *testing.T) {
	userID, _, doc, users, _ := wsFixture()
	sessions := &stubWebSocketSessions{remaining: 0}
	editors := &stubDocumentEditors{left: true, editors: []DocumentEditorInfo{}}
	uc := newWebSocketUseCase(&stubDocumentRepo{}, users, nil, sessions, editors, docAccess(doc.WorkspaceID))

	out, err := uc.UnregisterConnection(context.Background(), UnregisterWebSocketConnectionInput{
		UserID: userID, DocumentID: doc.ID, ConnectionID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("UnregisterConnection: %v", err)
	}
	if !out.EditorLeft || out.LeftEditor.Name != "Alice" {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestApplyDocumentEdit_Success(t *testing.T) {
	userID, _, doc, users, access := wsFixture()
	docs := &stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}}
	cache := &stubDocumentCache{}
	uc := newWebSocketUseCase(docs, users, cache, &stubWebSocketSessions{}, &stubDocumentEditors{}, access)

	out, err := uc.ApplyDocumentEdit(context.Background(), ApplyDocumentEditInput{
		UserID: userID, DocumentID: doc.ID, Content: "edited live",
	})
	if err != nil {
		t.Fatalf("ApplyDocumentEdit: %v", err)
	}
	if out.Content != "edited live" {
		t.Fatalf("content = %q", out.Content)
	}
}

func TestApplyDocumentEdit_ContentTooLarge(t *testing.T) {
	userID, _, doc, users, access := wsFixture()
	docs := &stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}}
	uc := newWebSocketUseCase(docs, users, &stubDocumentCache{}, &stubWebSocketSessions{}, &stubDocumentEditors{}, access)

	large := strings.Repeat("a", maxDocumentContentBytes+1)
	_, err := uc.ApplyDocumentEdit(context.Background(), ApplyDocumentEditInput{
		UserID: userID, DocumentID: doc.ID, Content: large,
	})
	if !errors.Is(err, ErrDocumentContentTooLarge) {
		t.Fatalf("expected content too large, got %v", err)
	}
}

func TestApplyDocumentEdit_NoCache(t *testing.T) {
	userID, _, doc, users, access := wsFixture()
	docs := &stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}}
	var cache IDocumentCache
	uc := newWebSocketUseCase(docs, users, cache, &stubWebSocketSessions{}, &stubDocumentEditors{}, access)

	_, err := uc.ApplyDocumentEdit(context.Background(), ApplyDocumentEditInput{
		UserID: userID, DocumentID: doc.ID, Content: "x",
	})
	if err == nil {
		t.Fatal("expected cache not configured error")
	}
}
