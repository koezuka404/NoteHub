package controller

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/dto"
	"github.com/koezuka404/notehub/usecase"
)

func TestDocumentController_Create(t *testing.T) {
	userID, wsID, docID := uuid.New(), uuid.New(), uuid.New()
	ctrl := NewDocumentController(&mockDocumentUsecase{
		createOut: &usecase.CreateDocumentOutput{
			ID: docID, WorkspaceID: wsID, Title: "Doc", Content: "hello",
			CreatedBy: userID, CreatedAt: testFixedTime.Format("2006-01-02T15:04:05Z07:00"),
		},
	})
	ctx, rec := newEchoContextWithParams(t, http.MethodPost, "/workspaces/:workspaceId/documents",
		map[string]string{"workspaceId": wsID.String()}, dto.CreateDocumentRequest{Title: "Doc", Content: "hello"})
	setAuthenticatedUser(ctx, userID)
	if err := ctrl.Create(ctx); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodPost, "/workspaces/:workspaceId/documents",
		map[string]string{"workspaceId": "bad"}, dto.CreateDocumentRequest{Title: "Doc"})
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Create(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("workspace param status = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodPost, "/workspaces/:workspaceId/documents",
		map[string]string{"workspaceId": wsID.String()}, "{bad")
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Create(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bind status = %d", rec.Code)
	}

	ctrl = NewDocumentController(&mockDocumentUsecase{createErr: usecase.ErrDocumentContentTooLarge})
	ctx, rec = newEchoContextWithParams(t, http.MethodPost, "/workspaces/:workspaceId/documents",
		map[string]string{"workspaceId": wsID.String()}, dto.CreateDocumentRequest{Title: "Doc"})
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Create(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("error status = %d", rec.Code)
	}
}

func TestDocumentController_List(t *testing.T) {
	userID, wsID, docID := uuid.New(), uuid.New(), uuid.New()
	ctrl := NewDocumentController(&mockDocumentUsecase{
		listOut: []usecase.DocumentListItem{{
			ID: docID, Title: "Doc", UpdatedBy: userID, UpdatedAt: testFixedTime.Format("2006-01-02T15:04:05Z07:00"),
		}},
	})
	ctx, rec := newEchoContextWithParams(t, http.MethodGet, "/workspaces/:workspaceId/documents",
		map[string]string{"workspaceId": wsID.String()}, nil)
	setAuthenticatedUser(ctx, userID)
	if err := ctrl.List(ctx); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	ctrl = NewDocumentController(&mockDocumentUsecase{listErr: usecase.ErrWorkspaceAccessDenied})
	ctx, rec = newEchoContextWithParams(t, http.MethodGet, "/workspaces/:workspaceId/documents",
		map[string]string{"workspaceId": wsID.String()}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.List(ctx)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("error status = %d", rec.Code)
	}
}

func TestDocumentController_Get(t *testing.T) {
	userID, wsID, docID := uuid.New(), uuid.New(), uuid.New()
	ctrl := NewDocumentController(&mockDocumentUsecase{
		getOut: &usecase.GetDocumentOutput{
			ID: docID, WorkspaceID: wsID, Title: "Doc", Content: "hello",
			UpdatedBy: userID, UpdatedAt: testFixedTime.Format("2006-01-02T15:04:05Z07:00"),
		},
	})
	ctx, rec := newEchoContextWithParams(t, http.MethodGet, "/documents/:documentId",
		map[string]string{"documentId": docID.String()}, nil)
	setAuthenticatedUser(ctx, userID)
	if err := ctrl.Get(ctx); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodGet, "/documents/:documentId",
		map[string]string{"documentId": "bad"}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Get(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("param status = %d", rec.Code)
	}

	ctrl = NewDocumentController(&mockDocumentUsecase{getErr: usecase.ErrDocumentNotFound})
	ctx, rec = newEchoContextWithParams(t, http.MethodGet, "/documents/:documentId",
		map[string]string{"documentId": docID.String()}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Get(ctx)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("error status = %d", rec.Code)
	}
}

func TestDocumentController_Update(t *testing.T) {
	userID, docID := uuid.New(), uuid.New()
	ctrl := NewDocumentController(&mockDocumentUsecase{
		updateOut: &usecase.UpdateDocumentOutput{
			ID: docID, Title: "Updated", UpdatedAt: testFixedTime.Format("2006-01-02T15:04:05Z07:00"),
		},
	})
	ctx, rec := newEchoContextWithParams(t, http.MethodPatch, "/documents/:documentId",
		map[string]string{"documentId": docID.String()}, dto.UpdateDocumentRequest{Title: "Updated"})
	setAuthenticatedUser(ctx, userID)
	if err := ctrl.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodPatch, "/documents/:documentId",
		map[string]string{"documentId": docID.String()}, "{bad")
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Update(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bind status = %d", rec.Code)
	}

	ctrl = NewDocumentController(&mockDocumentUsecase{updateErr: usecase.ErrDocumentDeleted})
	ctx, rec = newEchoContextWithParams(t, http.MethodPatch, "/documents/:documentId",
		map[string]string{"documentId": docID.String()}, dto.UpdateDocumentRequest{Title: "Updated"})
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Update(ctx)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("error status = %d", rec.Code)
	}
}

func TestDocumentController_Delete(t *testing.T) {
	userID, docID := uuid.New(), uuid.New()
	ctrl := NewDocumentController(&mockDocumentUsecase{
		deleteOut: &usecase.DeleteDocumentOutput{
			DocumentID: docID, DeletedAt: testFixedTime.Format("2006-01-02T15:04:05Z07:00"),
		},
	})
	ctx, rec := newEchoContextWithParams(t, http.MethodDelete, "/documents/:documentId",
		map[string]string{"documentId": docID.String()}, nil)
	setAuthenticatedUser(ctx, userID)
	if err := ctrl.Delete(ctx); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	ctrl = NewDocumentController(&mockDocumentUsecase{deleteErr: usecase.ErrValidation})
	ctx, rec = newEchoContextWithParams(t, http.MethodDelete, "/documents/:documentId",
		map[string]string{"documentId": docID.String()}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Delete(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("error status = %d", rec.Code)
	}
}
