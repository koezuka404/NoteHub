package controller

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/dto"
	"github.com/koezuka404/notehub/usecase"
)

func TestWorkspaceController_Create(t *testing.T) {
	userID := uuid.New()
	workspaceID := uuid.New()
	ctrl := NewWorkspaceController(&mockWorkspaceUsecase{
		createOut: &usecase.CreateWorkspaceOutput{
			ID: workspaceID, Name: "Team", HostID: userID, CreatedAt: testFixedTime.Format("2006-01-02T15:04:05Z07:00"),
		},
	})
	ctx, rec := newEchoContext(t, http.MethodPost, "/workspaces", dto.CreateWorkspaceRequest{Name: "Team"})
	setAuthenticatedUser(ctx, userID)
	if err := ctrl.Create(ctx); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d", rec.Code)
	}

	ctrl = NewWorkspaceController(&mockWorkspaceUsecase{createErr: usecase.ErrValidation})
	ctx, rec = newEchoContext(t, http.MethodPost, "/workspaces", dto.CreateWorkspaceRequest{Name: "Team"})
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Create(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("usecase error status = %d", rec.Code)
	}

	ctx, rec = newEchoContext(t, http.MethodPost, "/workspaces", nil)
	_ = ctrl.Create(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauth status = %d", rec.Code)
	}

	ctx, rec = newEchoContext(t, http.MethodPost, "/workspaces", "{bad")
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Create(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bind status = %d", rec.Code)
	}
}

func TestWorkspaceController_List(t *testing.T) {
	userID := uuid.New()
	wsID := uuid.New()
	ctrl := NewWorkspaceController(&mockWorkspaceUsecase{
		listOut: []usecase.WorkspaceListItem{{
			ID: wsID, Name: "Team", HostID: userID, HostName: "Alice", Role: "host",
			IsAvailable: true, UpdatedAt: testFixedTime.Format("2006-01-02T15:04:05Z07:00"),
		}},
	})
	ctx, rec := newEchoContext(t, http.MethodGet, "/workspaces", nil)
	setAuthenticatedUser(ctx, userID)
	if err := ctrl.List(ctx); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	ctrl = NewWorkspaceController(&mockWorkspaceUsecase{listErr: usecase.ErrAccountUnavailable})
	ctx, rec = newEchoContext(t, http.MethodGet, "/workspaces", nil)
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.List(ctx)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("error status = %d", rec.Code)
	}
}

func TestWorkspaceController_Get(t *testing.T) {
	userID, wsID := uuid.New(), uuid.New()
	ctrl := NewWorkspaceController(&mockWorkspaceUsecase{
		getOut: &usecase.GetWorkspaceOutput{
			ID: wsID, Name: "Team", Role: "host",
			Host: usecase.WorkspaceHostOutput{ID: userID, Name: "Alice", Status: "active"},
			CreatedAt: testFixedTime.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt: testFixedTime.Format("2006-01-02T15:04:05Z07:00"),
		},
	})
	ctx, rec := newEchoContextWithParams(t, http.MethodGet, "/workspaces/:workspaceId",
		map[string]string{"workspaceId": wsID.String()}, nil)
	setAuthenticatedUser(ctx, userID)
	if err := ctrl.Get(ctx); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	ctrl = NewWorkspaceController(&mockWorkspaceUsecase{getErr: usecase.ErrWorkspaceNotFound})
	ctx, rec = newEchoContextWithParams(t, http.MethodGet, "/workspaces/:workspaceId",
		map[string]string{"workspaceId": wsID.String()}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Get(ctx)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("error status = %d", rec.Code)
	}
}

func TestWorkspaceController_Update(t *testing.T) {
	userID, wsID := uuid.New(), uuid.New()
	ctrl := NewWorkspaceController(&mockWorkspaceUsecase{
		updateOut: &usecase.UpdateWorkspaceOutput{
			ID: wsID, Name: "Updated", UpdatedAt: testFixedTime.Format("2006-01-02T15:04:05Z07:00"),
		},
	})
	ctx, rec := newEchoContextWithParams(t, http.MethodPatch, "/workspaces/:workspaceId",
		map[string]string{"workspaceId": wsID.String()}, dto.UpdateWorkspaceRequest{Name: "Updated"})
	setAuthenticatedUser(ctx, userID)
	if err := ctrl.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodPatch, "/workspaces/:workspaceId",
		map[string]string{"workspaceId": wsID.String()}, "{bad")
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Update(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bind status = %d", rec.Code)
	}

	ctrl = NewWorkspaceController(&mockWorkspaceUsecase{updateErr: usecase.ErrHostPermissionRequired})
	ctx, rec = newEchoContextWithParams(t, http.MethodPatch, "/workspaces/:workspaceId",
		map[string]string{"workspaceId": wsID.String()}, dto.UpdateWorkspaceRequest{Name: "Updated"})
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Update(ctx)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("error status = %d", rec.Code)
	}
}

func TestWorkspaceController_Delete(t *testing.T) {
	userID, wsID := uuid.New(), uuid.New()
	ctrl := NewWorkspaceController(&mockWorkspaceUsecase{
		deleteOut: &usecase.DeleteWorkspaceOutput{
			WorkspaceID: wsID, DeletedAt: testFixedTime.Format("2006-01-02T15:04:05Z07:00"),
		},
	})
	ctx, rec := newEchoContextWithParams(t, http.MethodDelete, "/workspaces/:workspaceId",
		map[string]string{"workspaceId": wsID.String()}, dto.DeleteWorkspaceRequest{Reason: "cleanup"})
	setAuthenticatedUser(ctx, userID)
	if err := ctrl.Delete(ctx); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodDelete, "/workspaces/:workspaceId",
		map[string]string{"workspaceId": wsID.String()}, "{bad")
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Delete(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bind status = %d", rec.Code)
	}

	ctrl = NewWorkspaceController(&mockWorkspaceUsecase{deleteErr: usecase.ErrWorkspaceAlreadyDeleted})
	ctx, rec = newEchoContextWithParams(t, http.MethodDelete, "/workspaces/:workspaceId",
		map[string]string{"workspaceId": wsID.String()}, dto.DeleteWorkspaceRequest{Reason: "cleanup"})
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Delete(ctx)
	if rec.Code != http.StatusConflict {
		t.Fatalf("error status = %d", rec.Code)
	}
}
