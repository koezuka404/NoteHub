package controller

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
	"github.com/koezuka404/notehub/usecase"
)

func TestVersionController_List(t *testing.T) {
	userID, docID, versionID := uuid.New(), uuid.New(), uuid.New()
	ctrl := NewVersionController(&mockVersionUsecase{
		listOut: []usecase.VersionListItem{{
			ID: versionID, Type: "manual", CreatedBy: userID,
			CreatedAt: testFixedTime.Format("2006-01-02T15:04:05Z07:00"),
		}},
	})
	ctx, rec := newEchoContextWithParams(t, http.MethodGet, "/documents/:documentId/versions",
		map[string]string{"documentId": docID.String()}, nil)
	setAuthenticatedUser(ctx, userID)
	if err := ctrl.List(ctx); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	ctrl = NewVersionController(&mockVersionUsecase{listErr: usecase.ErrDocumentNotFound})
	ctx, rec = newEchoContextWithParams(t, http.MethodGet, "/documents/:documentId/versions",
		map[string]string{"documentId": docID.String()}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.List(ctx)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("error status = %d", rec.Code)
	}
}

func TestVersionController_Get(t *testing.T) {
	userID, docID, versionID := uuid.New(), uuid.New(), uuid.New()
	ctrl := NewVersionController(&mockVersionUsecase{
		getOut: &usecase.GetVersionOutput{
			ID: versionID, DocumentID: docID, Content: "hello", Type: "manual",
			CreatedBy: userID, CreatedAt: testFixedTime.Format("2006-01-02T15:04:05Z07:00"),
		},
	})
	ctx, rec := newEchoContextWithParams(t, http.MethodGet, "/documents/:documentId/versions/:versionId",
		map[string]string{"documentId": docID.String(), "versionId": versionID.String()}, nil)
	setAuthenticatedUser(ctx, userID)
	if err := ctrl.Get(ctx); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodGet, "/documents/:documentId/versions/:versionId",
		map[string]string{"documentId": docID.String(), "versionId": "bad"}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Get(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("version param status = %d", rec.Code)
	}

	ctrl = NewVersionController(&mockVersionUsecase{getErr: usecase.ErrVersionNotFound})
	ctx, rec = newEchoContextWithParams(t, http.MethodGet, "/documents/:documentId/versions/:versionId",
		map[string]string{"documentId": docID.String(), "versionId": versionID.String()}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Get(ctx)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("error status = %d", rec.Code)
	}
}

func TestVersionController_Save(t *testing.T) {
	userID, docID, versionID := uuid.New(), uuid.New(), uuid.New()
	ctrl := NewVersionController(&mockVersionUsecase{
		saveOut: &usecase.SaveManualVersionOutput{
			ID: versionID, CreatedAt: testFixedTime.Format("2006-01-02T15:04:05Z07:00"),
		},
	})
	ctx, rec := newEchoContextWithParams(t, http.MethodPost, "/documents/:documentId/versions",
		map[string]string{"documentId": docID.String()}, nil)
	setAuthenticatedUser(ctx, userID)
	if err := ctrl.Save(ctx); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d", rec.Code)
	}

	ctrl = NewVersionController(&mockVersionUsecase{saveErr: entity.ErrDocumentConflict})
	ctx, rec = newEchoContextWithParams(t, http.MethodPost, "/documents/:documentId/versions",
		map[string]string{"documentId": docID.String()}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Save(ctx)
	if rec.Code != http.StatusConflict {
		t.Fatalf("error status = %d", rec.Code)
	}
}

func TestVersionController_Restore(t *testing.T) {
	userID, docID, versionID := uuid.New(), uuid.New(), uuid.New()
	ctrl := NewVersionController(&mockVersionUsecase{
		restoreOut: &usecase.RestoreVersionOutput{
			DocumentID: docID, VersionID: versionID, RestoredAt: testFixedTime.Format("2006-01-02T15:04:05Z07:00"),
		},
	})
	ctx, rec := newEchoContextWithParams(t, http.MethodPost, "/documents/:documentId/versions/:versionId/restore",
		map[string]string{"documentId": docID.String(), "versionId": versionID.String()}, nil)
	setAuthenticatedUser(ctx, userID)
	if err := ctrl.Restore(ctx); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	ctrl = NewVersionController(&mockVersionUsecase{restoreErr: usecase.ErrDocumentDeleted})
	ctx, rec = newEchoContextWithParams(t, http.MethodPost, "/documents/:documentId/versions/:versionId/restore",
		map[string]string{"documentId": docID.String(), "versionId": versionID.String()}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Restore(ctx)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("error status = %d", rec.Code)
	}
}
