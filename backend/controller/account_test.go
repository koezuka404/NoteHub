package controller

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/usecase"
)

func TestAccountController_Suspend(t *testing.T) {
	userID, workspaceID, targetID := uuid.New(), uuid.New(), uuid.New()
	ctrl := NewAccountController(&mockAccountUsecase{
		suspendOut: &usecase.SuspendAccountOutput{
			UserID: targetID, Status: "suspended", SuspendedAt: testFixedTime.Format(time.RFC3339),
		},
	})

	ctx, rec := newEchoContextWithParams(t, http.MethodPost, "/workspaces/:workspaceId/users/:userId/suspend",
		map[string]string{"workspaceId": workspaceID.String(), "userId": targetID.String()}, nil)
	setAuthenticatedUser(ctx, userID)
	if err := ctrl.Suspend(ctx); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	ctrl = NewAccountController(&mockAccountUsecase{suspendErr: usecase.ErrCannotSuspendSelf})
	ctx, rec = newEchoContextWithParams(t, http.MethodPost, "/workspaces/:workspaceId/users/:userId/suspend",
		map[string]string{"workspaceId": workspaceID.String(), "userId": targetID.String()}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Suspend(ctx)
	if rec.Code != http.StatusConflict {
		t.Fatalf("error status = %d", rec.Code)
	}

	ctx, rec = newEchoContext(t, http.MethodPost, "/", nil)
	_ = ctrl.Suspend(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauth status = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodPost, "/workspaces/:workspaceId/users/:userId/suspend",
		map[string]string{"workspaceId": "bad", "userId": targetID.String()}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Suspend(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("workspace param status = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodPost, "/workspaces/:workspaceId/users/:userId/suspend",
		map[string]string{"workspaceId": workspaceID.String(), "userId": "bad"}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Suspend(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("user param status = %d", rec.Code)
	}
}

func TestAccountController_Reactivate(t *testing.T) {
	userID, workspaceID, targetID := uuid.New(), uuid.New(), uuid.New()
	ctrl := NewAccountController(&mockAccountUsecase{
		reactivateOut: &usecase.ReactivateAccountOutput{UserID: targetID, Status: "active"},
	})
	ctx, rec := newEchoContextWithParams(t, http.MethodPost, "/workspaces/:workspaceId/users/:userId/reactivate",
		map[string]string{"workspaceId": workspaceID.String(), "userId": targetID.String()}, nil)
	setAuthenticatedUser(ctx, userID)
	if err := ctrl.Reactivate(ctx); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	ctrl = NewAccountController(&mockAccountUsecase{reactivateErr: usecase.ErrCannotReactivateSelf})
	ctx, rec = newEchoContextWithParams(t, http.MethodPost, "/workspaces/:workspaceId/users/:userId/reactivate",
		map[string]string{"workspaceId": workspaceID.String(), "userId": targetID.String()}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Reactivate(ctx)
	if rec.Code != http.StatusConflict {
		t.Fatalf("error status = %d", rec.Code)
	}
}

func TestAccountController_Delete(t *testing.T) {
	userID, workspaceID, targetID := uuid.New(), uuid.New(), uuid.New()
	ctrl := NewAccountController(&mockAccountUsecase{
		deleteOut: &usecase.DeleteAccountOutput{
			UserID: targetID, Status: "deleted", DeletedAt: testFixedTime.Format(time.RFC3339),
		},
	})
	ctx, rec := newEchoContextWithParams(t, http.MethodDelete, "/workspaces/:workspaceId/users/:userId",
		map[string]string{"workspaceId": workspaceID.String(), "userId": targetID.String()}, nil)
	setAuthenticatedUser(ctx, userID)
	if err := ctrl.Delete(ctx); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	ctrl = NewAccountController(&mockAccountUsecase{deleteErr: usecase.ErrCannotDeleteSelf})
	ctx, rec = newEchoContextWithParams(t, http.MethodDelete, "/workspaces/:workspaceId/users/:userId",
		map[string]string{"workspaceId": workspaceID.String(), "userId": targetID.String()}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Delete(ctx)
	if rec.Code != http.StatusConflict {
		t.Fatalf("error status = %d", rec.Code)
	}
}
