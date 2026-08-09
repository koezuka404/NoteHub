package controller

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/dto"
	"github.com/koezuka404/notehub/entity"
	"github.com/koezuka404/notehub/usecase"
)

func TestMemberController_Search(t *testing.T) {
	userID, wsID, targetID := uuid.New(), uuid.New(), uuid.New()
	ctrl := NewMemberController(&mockMemberUsecase{
		searchOut: &usecase.SearchUserOutput{
			ID: targetID, Email: "bob@example.com", Name: "Bob", Status: entity.UserStatusActive,
		},
	})
	ctx, rec := newEchoContextWithParams(t, http.MethodGet, "/workspaces/:workspaceId/members/search?email=bob@example.com",
		map[string]string{"workspaceId": wsID.String()}, nil)
	setAuthenticatedUser(ctx, userID)
	if err := ctrl.Search(ctx); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	ctrl = NewMemberController(&mockMemberUsecase{searchErr: usecase.ErrUserNotFound})
	ctx, rec = newEchoContextWithParams(t, http.MethodGet, "/workspaces/:workspaceId/members/search",
		map[string]string{"workspaceId": wsID.String()}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Search(ctx)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("error status = %d", rec.Code)
	}
}

func TestMemberController_List(t *testing.T) {
	userID, wsID, memberID := uuid.New(), uuid.New(), uuid.New()
	ctrl := NewMemberController(&mockMemberUsecase{
		listOut: []usecase.MemberListItem{{
			UserID: memberID, Name: "Bob", Email: "bob@example.com",
			Status: "active", Role: "member", JoinedAt: testFixedTime.Format("2006-01-02T15:04:05Z07:00"),
		}},
	})
	ctx, rec := newEchoContextWithParams(t, http.MethodGet, "/workspaces/:workspaceId/members",
		map[string]string{"workspaceId": wsID.String()}, nil)
	setAuthenticatedUser(ctx, userID)
	if err := ctrl.List(ctx); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	ctrl = NewMemberController(&mockMemberUsecase{listErr: usecase.ErrMemberNotFound})
	ctx, rec = newEchoContextWithParams(t, http.MethodGet, "/workspaces/:workspaceId/members",
		map[string]string{"workspaceId": wsID.String()}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.List(ctx)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("error status = %d", rec.Code)
	}
}

func TestMemberController_Add(t *testing.T) {
	userID, wsID, targetID := uuid.New(), uuid.New(), uuid.New()
	ctrl := NewMemberController(&mockMemberUsecase{
		addOut: &usecase.AddMemberOutput{
			UserID: targetID, Name: "Bob", Email: "bob@example.com",
			Role: "member", JoinedAt: testFixedTime.Format("2006-01-02T15:04:05Z07:00"),
		},
	})
	ctx, rec := newEchoContextWithParams(t, http.MethodPost, "/workspaces/:workspaceId/members",
		map[string]string{"workspaceId": wsID.String()}, dto.AddMemberRequest{UserID: targetID.String()})
	setAuthenticatedUser(ctx, userID)
	if err := ctrl.Add(ctx); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodPost, "/workspaces/:workspaceId/members",
		map[string]string{"workspaceId": wsID.String()}, dto.AddMemberRequest{UserID: "bad"})
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Add(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid user id status = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodPost, "/workspaces/:workspaceId/members",
		map[string]string{"workspaceId": wsID.String()}, "{bad")
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Add(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bind status = %d", rec.Code)
	}

	ctrl = NewMemberController(&mockMemberUsecase{addErr: usecase.ErrCannotAddSelf})
	ctx, rec = newEchoContextWithParams(t, http.MethodPost, "/workspaces/:workspaceId/members",
		map[string]string{"workspaceId": wsID.String()}, dto.AddMemberRequest{UserID: targetID.String()})
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Add(ctx)
	if rec.Code != http.StatusConflict {
		t.Fatalf("error status = %d", rec.Code)
	}
}

func TestMemberController_Remove(t *testing.T) {
	userID, wsID, targetID := uuid.New(), uuid.New(), uuid.New()
	ctrl := NewMemberController(&mockMemberUsecase{})
	ctx, rec := newEchoContextWithParams(t, http.MethodDelete, "/workspaces/:workspaceId/members/:userId",
		map[string]string{"workspaceId": wsID.String(), "userId": targetID.String()}, nil)
	setAuthenticatedUser(ctx, userID)
	if err := ctrl.Remove(ctx); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	ctrl = NewMemberController(&mockMemberUsecase{removeErr: usecase.ErrCannotRemoveHost})
	ctx, rec = newEchoContextWithParams(t, http.MethodDelete, "/workspaces/:workspaceId/members/:userId",
		map[string]string{"workspaceId": wsID.String(), "userId": targetID.String()}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Remove(ctx)
	if rec.Code != http.StatusConflict {
		t.Fatalf("error status = %d", rec.Code)
	}
}
