package controller

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/dto"
	"github.com/koezuka404/notehub/entity"
	"github.com/koezuka404/notehub/usecase"
	appws "github.com/koezuka404/notehub/websocket"
	"github.com/labstack/echo/v4"
)

func TestHandlers_UnauthenticatedEarlyReturn(t *testing.T) {
	wsID, docID, targetID := uuid.New(), uuid.New(), uuid.New()

	account := NewAccountController(&mockAccountUsecase{})
	ctx, rec := newEchoContextWithParams(t, http.MethodPost, "/", map[string]string{
		"workspaceId": wsID.String(), "userId": targetID.String(),
	}, nil)
	_ = account.Reactivate(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("reactivate unauth = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodDelete, "/", map[string]string{
		"workspaceId": wsID.String(), "userId": targetID.String(),
	}, nil)
	_ = account.Delete(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("delete unauth = %d", rec.Code)
	}

	workspace := NewWorkspaceController(&mockWorkspaceUsecase{})
	ctx, rec = newEchoContext(t, http.MethodGet, "/workspaces", nil)
	_ = workspace.List(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("workspace list unauth = %d", rec.Code)
	}

	document := NewDocumentController(&mockDocumentUsecase{})
	ctx, rec = newEchoContextWithParams(t, http.MethodPost, "/", map[string]string{"workspaceId": wsID.String()}, dto.CreateDocumentRequest{Title: "Doc"})
	_ = document.Create(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("document create unauth = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodGet, "/", map[string]string{"workspaceId": wsID.String()}, nil)
	_ = document.List(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("document list unauth = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodGet, "/", map[string]string{"documentId": docID.String()}, nil)
	_ = document.Get(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("document get unauth = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodPatch, "/", map[string]string{"documentId": docID.String()}, dto.UpdateDocumentRequest{Title: "x"})
	_ = document.Update(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("document update unauth = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodDelete, "/", map[string]string{"documentId": docID.String()}, nil)
	_ = document.Delete(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("document delete unauth = %d", rec.Code)
	}

	member := NewMemberController(&mockMemberUsecase{})
	ctx, rec = newEchoContextWithParams(t, http.MethodGet, "/", map[string]string{"workspaceId": wsID.String()}, nil)
	_ = member.Search(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("member search unauth = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodGet, "/", map[string]string{"workspaceId": wsID.String()}, nil)
	_ = member.List(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("member list unauth = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodPost, "/", map[string]string{"workspaceId": wsID.String()}, dto.AddMemberRequest{UserID: targetID.String()})
	_ = member.Add(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("member add unauth = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodDelete, "/", map[string]string{
		"workspaceId": wsID.String(), "userId": targetID.String(),
	}, nil)
	_ = member.Remove(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("member remove unauth = %d", rec.Code)
	}

	version := NewVersionController(&mockVersionUsecase{})
	ctx, rec = newEchoContextWithParams(t, http.MethodGet, "/", map[string]string{"documentId": docID.String()}, nil)
	_ = version.List(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("version list unauth = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodGet, "/", map[string]string{
		"documentId": docID.String(), "versionId": uuid.NewString(),
	}, nil)
	_ = version.Get(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("version get unauth = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodPost, "/", map[string]string{"documentId": docID.String()}, nil)
	_ = version.Save(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("version save unauth = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodPost, "/", map[string]string{
		"documentId": docID.String(), "versionId": uuid.NewString(),
	}, nil)
	_ = version.Restore(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("version restore unauth = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodGet, "/", map[string]string{"workspaceId": wsID.String()}, nil)
	_ = workspace.Get(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("workspace get unauth = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodPatch, "/", map[string]string{"workspaceId": wsID.String()}, dto.UpdateWorkspaceRequest{Name: "x"})
	_ = workspace.Update(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("workspace update unauth = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodDelete, "/", map[string]string{"workspaceId": wsID.String()}, dto.DeleteWorkspaceRequest{Reason: "x"})
	_ = workspace.Delete(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("workspace delete unauth = %d", rec.Code)
	}
}

func TestAccountController_ReactivateInvalidUserParam(t *testing.T) {
	userID, workspaceID := uuid.New(), uuid.New()
	ctrl := NewAccountController(&mockAccountUsecase{})
	ctx, rec := newEchoContextWithParams(t, http.MethodPost, "/", map[string]string{
		"workspaceId": workspaceID.String(), "userId": "bad",
	}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Reactivate(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestAccountController_DeleteInvalidWorkspaceParam(t *testing.T) {
	userID, targetID := uuid.New(), uuid.New()
	ctrl := NewAccountController(&mockAccountUsecase{})
	ctx, rec := newEchoContextWithParams(t, http.MethodDelete, "/", map[string]string{
		"workspaceId": "bad", "userId": targetID.String(),
	}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Delete(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestWebSocketController_handleClientMessageInvalidEditPayload(t *testing.T) {
	ctrl := &WebSocketController{now: func() time.Time { return testFixedTime }}
	client := &appws.Client{Send: make(chan []byte, 1)}
	ctrl.handleClientMessage(client, []byte(`{"type":"document_edit","data":[]}`))
	select {
	case payload := <-client.Send:
		var envelope appws.Envelope
		_ = json.Unmarshal(payload, &envelope)
		var data appws.ErrorData
		_ = json.Unmarshal(envelope.Data, &data)
		if data.Code != "INVALID_REQUEST" {
			t.Fatalf("code = %q", data.Code)
		}
	default:
		t.Fatal("expected error event")
	}
}

func TestHandlers_InvalidParamAfterAuth(t *testing.T) {
	userID, wsID, docID, targetID, versionID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()

	document := NewDocumentController(&mockDocumentUsecase{})
	ctx, rec := newEchoContextWithParams(t, http.MethodPatch, "/", map[string]string{"documentId": "bad"}, dto.UpdateDocumentRequest{Title: "x"})
	setAuthenticatedUser(ctx, userID)
	_ = document.Update(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("document update bad param = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodDelete, "/", map[string]string{"documentId": "bad"}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = document.Delete(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("document delete bad param = %d", rec.Code)
	}

	member := NewMemberController(&mockMemberUsecase{})
	ctx, rec = newEchoContextWithParams(t, http.MethodGet, "/", map[string]string{"workspaceId": "bad"}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = member.List(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("member list bad workspace = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodPost, "/", map[string]string{"workspaceId": "bad"}, dto.AddMemberRequest{UserID: targetID.String()})
	setAuthenticatedUser(ctx, userID)
	_ = member.Add(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("member add bad workspace = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodDelete, "/", map[string]string{"workspaceId": "bad", "userId": targetID.String()}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = member.Remove(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("member remove bad workspace = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodDelete, "/", map[string]string{"workspaceId": wsID.String(), "userId": "bad"}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = member.Remove(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("member remove bad user = %d", rec.Code)
	}

	version := NewVersionController(&mockVersionUsecase{})
	ctx, rec = newEchoContextWithParams(t, http.MethodGet, "/", map[string]string{"documentId": "bad", "versionId": versionID.String()}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = version.Get(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("version get bad document = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodPost, "/", map[string]string{"documentId": "bad"}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = version.Save(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("version save bad document = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodPost, "/", map[string]string{"documentId": "bad", "versionId": versionID.String()}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = version.Restore(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("version restore bad document = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodPost, "/", map[string]string{"documentId": docID.String(), "versionId": "bad"}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = version.Restore(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("version restore bad version = %d", rec.Code)
	}

	workspace := NewWorkspaceController(&mockWorkspaceUsecase{})
	ctx, rec = newEchoContextWithParams(t, http.MethodPatch, "/", map[string]string{"workspaceId": "bad"}, dto.UpdateWorkspaceRequest{Name: "x"})
	setAuthenticatedUser(ctx, userID)
	_ = workspace.Update(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("workspace update bad param = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodDelete, "/", map[string]string{"workspaceId": "bad"}, dto.DeleteWorkspaceRequest{Reason: "x"})
	setAuthenticatedUser(ctx, userID)
	_ = workspace.Delete(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("workspace delete bad param = %d", rec.Code)
	}
}

func TestAuthController_RegisterUseCaseError(t *testing.T) {
	ctrl := NewAuthController(&mockAuthUsecase{registerErr: usecase.ErrEmailAlreadyExists}, defaultAuthCookies())
	ctx, rec := newEchoContext(t, http.MethodPost, "/register", dto.RegisterRequest{
		Name: "Alice", Email: "alice@example.com", Password: "Pass1234",
	})
	_ = ctrl.Register(ctx)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestAuthController_LoginUseCaseError(t *testing.T) {
	ctrl := NewAuthController(&mockAuthUsecase{loginErr: usecase.ErrInvalidCredentials}, defaultAuthCookies())
	ctx, rec := newEchoContext(t, http.MethodPost, "/login", dto.LoginRequest{
		Email: "alice@example.com", Password: "Pass1234",
	})
	_ = ctrl.Login(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestAuthController_RefreshEmptyCookieValue(t *testing.T) {
	ctrl := NewAuthController(&mockAuthUsecase{}, defaultAuthCookies())
	ctx, rec := newEchoContext(t, http.MethodPost, "/refresh", nil)
	ctx.Request().AddCookie(&http.Cookie{Name: "refresh_token", Value: ""})
	_ = ctrl.Refresh(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestAccountController_ReactivateAndDeleteParamErrors(t *testing.T) {
	userID, workspaceID, targetID := uuid.New(), uuid.New(), uuid.New()
	ctrl := NewAccountController(&mockAccountUsecase{
		reactivateErr: usecase.ErrAccountNotSuspended,
		deleteErr:     usecase.ErrAccountDeleted,
	})

	ctx, rec := newEchoContextWithParams(t, http.MethodPost, "/", map[string]string{
		"workspaceId": "bad", "userId": targetID.String(),
	}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Reactivate(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("reactivate bad workspace = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodDelete, "/", map[string]string{
		"workspaceId": workspaceID.String(), "userId": "bad",
	}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Delete(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("delete user param = %d", rec.Code)
	}

	ctrl = NewAccountController(&mockAccountUsecase{deleteErr: usecase.ErrTargetUserNotFound})
	ctx, rec = newEchoContextWithParams(t, http.MethodDelete, "/", map[string]string{
		"workspaceId": workspaceID.String(), "userId": targetID.String(),
	}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Delete(ctx)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("delete usecase error = %d", rec.Code)
	}
}

func TestWorkspaceController_UnauthenticatedAndParamErrors(t *testing.T) {
	wsID := uuid.New()
	userID := uuid.New()

	ctrl := NewWorkspaceController(&mockWorkspaceUsecase{getErr: usecase.ErrWorkspaceNotFound})
	ctx, rec := newEchoContextWithParams(t, http.MethodGet, "/", map[string]string{"workspaceId": "bad"}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Get(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("get after bad param = %d", rec.Code)
	}

	ctrl = NewWorkspaceController(&mockWorkspaceUsecase{updateErr: usecase.ErrValidation, deleteErr: usecase.ErrValidation})
	ctx, rec = newEchoContextWithParams(t, http.MethodPatch, "/", map[string]string{"workspaceId": wsID.String()}, dto.UpdateWorkspaceRequest{Name: "x"})
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Update(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("update usecase error = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodDelete, "/", map[string]string{"workspaceId": wsID.String()}, dto.DeleteWorkspaceRequest{Reason: "x"})
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Delete(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("delete usecase error = %d", rec.Code)
	}
}

func TestDocumentController_UnauthenticatedAndParamErrors(t *testing.T) {
	wsID, docID := uuid.New(), uuid.New()
	userID := uuid.New()

	ctrl := NewDocumentController(&mockDocumentUsecase{listErr: usecase.ErrValidation})
	ctx, rec := newEchoContextWithParams(t, http.MethodGet, "/", map[string]string{"workspaceId": "bad"}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.List(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("list after bad workspace param = %d", rec.Code)
	}

	ctrl = NewDocumentController(&mockDocumentUsecase{createErr: usecase.ErrValidation})
	ctx, rec = newEchoContextWithParams(t, http.MethodPost, "/", map[string]string{"workspaceId": wsID.String()}, dto.CreateDocumentRequest{Title: "Doc"})
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Create(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("create usecase error = %d", rec.Code)
	}

	ctrl = NewDocumentController(&mockDocumentUsecase{updateErr: usecase.ErrDocumentNotFound, deleteErr: usecase.ErrDocumentDeleted})
	ctx, rec = newEchoContextWithParams(t, http.MethodPatch, "/", map[string]string{"documentId": docID.String()}, dto.UpdateDocumentRequest{Title: "x"})
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Update(ctx)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("update usecase error = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodDelete, "/", map[string]string{"documentId": docID.String()}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Delete(ctx)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("delete usecase error = %d", rec.Code)
	}
}

func TestMemberController_UnauthenticatedAndParamErrors(t *testing.T) {
	wsID, targetID := uuid.New(), uuid.New()
	userID := uuid.New()

	ctrl := NewMemberController(&mockMemberUsecase{searchErr: usecase.ErrValidation})
	ctx, rec := newEchoContextWithParams(t, http.MethodGet, "/", map[string]string{"workspaceId": "bad"}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Search(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("search after bad workspace param = %d", rec.Code)
	}

	ctrl = NewMemberController(&mockMemberUsecase{listErr: usecase.ErrMemberNotFound, addErr: usecase.ErrMemberAlreadyExists, removeErr: usecase.ErrCannotRemoveHost})
	ctx, rec = newEchoContextWithParams(t, http.MethodGet, "/", map[string]string{"workspaceId": wsID.String()}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.List(ctx)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("list usecase error = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodPost, "/", map[string]string{"workspaceId": wsID.String()}, dto.AddMemberRequest{UserID: targetID.String()})
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Add(ctx)
	if rec.Code != http.StatusConflict {
		t.Fatalf("add usecase error = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodDelete, "/", map[string]string{
		"workspaceId": wsID.String(), "userId": targetID.String(),
	}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Remove(ctx)
	if rec.Code != http.StatusConflict {
		t.Fatalf("remove usecase error = %d", rec.Code)
	}
}

func TestVersionController_UnauthenticatedAndParamErrors(t *testing.T) {
	docID, versionID := uuid.New(), uuid.New()
	userID := uuid.New()

	ctrl := NewVersionController(&mockVersionUsecase{listErr: usecase.ErrDocumentNotFound})
	ctx, rec := newEchoContextWithParams(t, http.MethodGet, "/", map[string]string{"documentId": "bad"}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.List(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("list after bad document param = %d", rec.Code)
	}

	ctrl = NewVersionController(&mockVersionUsecase{
		getErr: usecase.ErrVersionNotFound, saveErr: entity.ErrDocumentConflict, restoreErr: usecase.ErrDocumentDeleted,
	})
	ctx, rec = newEchoContextWithParams(t, http.MethodGet, "/", map[string]string{
		"documentId": docID.String(), "versionId": versionID.String(),
	}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Get(ctx)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("get usecase error = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodPost, "/", map[string]string{"documentId": docID.String()}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Save(ctx)
	if rec.Code != http.StatusConflict {
		t.Fatalf("save usecase error = %d", rec.Code)
	}

	ctx, rec = newEchoContextWithParams(t, http.MethodPost, "/", map[string]string{
		"documentId": docID.String(), "versionId": versionID.String(),
	}, nil)
	setAuthenticatedUser(ctx, userID)
	_ = ctrl.Restore(ctx)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("restore usecase error = %d", rec.Code)
	}
}

func TestWebSocketController_HandleDocumentInvalidDocumentID(t *testing.T) {
	ctrl, _ := newWebSocketController(t, &mockWSUsecase{}, WebSocketControllerConfig{})
	ctx, rec := newEchoContextWithParams(t, http.MethodGet, "/", map[string]string{"documentId": "bad"}, nil)
	_ = ctrl.HandleDocument(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestWebSocketController_HandleDocumentAuthenticateFailure(t *testing.T) {
	ctrl, _ := newWebSocketController(t, &mockWSUsecase{}, WebSocketControllerConfig{})
	ctx, rec := newEchoContextWithParams(t, http.MethodGet, "/", map[string]string{"documentId": uuid.NewString()}, nil)
	_ = ctrl.HandleDocument(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestWebSocketController_HandleWorkspaceInvalidWorkspaceID(t *testing.T) {
	ctrl, _ := newWebSocketController(t, &mockWSUsecase{}, WebSocketControllerConfig{})
	ctx, rec := newEchoContextWithParams(t, http.MethodGet, "/", map[string]string{"workspaceId": "bad"}, nil)
	_ = ctrl.HandleWorkspace(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestWebSocketController_HandleWorkspaceAuthenticateFailure(t *testing.T) {
	ctrl, _ := newWebSocketController(t, &mockWSUsecase{}, WebSocketControllerConfig{})
	ctx, rec := newEchoContextWithParams(t, http.MethodGet, "/", map[string]string{"workspaceId": uuid.NewString()}, nil)
	_ = ctrl.HandleWorkspace(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestWebSocketController_HandleWorkspaceUpgradeFailure(t *testing.T) {
	wsID := uuid.New()
	ctrl, _ := newWebSocketController(t, &mockWSUsecase{}, WebSocketControllerConfig{})
	ctx, _ := newEchoContextWithParams(t, http.MethodGet, "/", map[string]string{"workspaceId": wsID.String()}, nil)
	ctx.Request().Header.Set(echo.HeaderAuthorization, "Bearer token")
	if err := ctrl.HandleWorkspace(ctx); err == nil {
		t.Fatal("expected upgrade error")
	}
}

func TestWebSocketController_handleClientMessageBroadcast(t *testing.T) {
	docID := uuid.New()
	userID := uuid.New()
	otherUserID := uuid.New()
	hub := appws.NewHub()
	ctrl := &WebSocketController{
		wsUseCase: &mockWSUsecase{
			applyEditOut: &usecase.ApplyDocumentEditOutput{
				DocumentID: docID, Content: "updated", UpdatedBy: userID, UpdatedAt: testFixedTime.Format("2006-01-02T15:04:05Z07:00"),
			},
		},
		hub: hub,
		now: func() time.Time { return testFixedTime },
	}
	editor := &appws.Client{
		ConnectionID: uuid.New(), UserID: userID, DocumentID: docID,
		Hub: hub, Send: make(chan []byte, 4),
	}
	watcher := &appws.Client{
		ConnectionID: uuid.New(), UserID: otherUserID, DocumentID: docID,
		Hub: hub, Send: make(chan []byte, 4),
	}
	watcher.Ready.Store(true)
	hub.Register(watcher)

	validEdit, err := json.Marshal(appws.ClientMessage{
		Type: appws.EventDocumentEdit,
		Data: mustJSON(t, appws.DocumentEditData{Content: "updated"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	ctrl.handleClientMessage(editor, validEdit)

	select {
	case <-watcher.Send:
	default:
		t.Fatal("expected broadcast to other client")
	}
}

func TestWebSocketController_HandleDocumentProductionBypassViaTLS(t *testing.T) {
	docID := uuid.New()
	ctrl, _ := newWebSocketController(t, &mockWSUsecase{
		prepareOut:    &usecase.PrepareWebSocketConnectionOutput{DocumentID: docID, WorkspaceID: uuid.New()},
		registerOut:   &usecase.RegisterWebSocketConnectionOutput{},
		unregisterOut: &usecase.UnregisterWebSocketConnectionOutput{},
		onUnregister:  func() {},
	}, WebSocketControllerConfig{Production: true})
	ctx, _ := newEchoContextWithParams(t, http.MethodGet, "/", map[string]string{"documentId": docID.String()}, nil)
	ctx.Request().Header.Set(echo.HeaderAuthorization, "Bearer token")
	ctx.Request().TLS = &tls.ConnectionState{}
	_ = ctrl.HandleDocument(ctx)
}

func TestWebSocketController_HandleWorkspaceProductionBypassViaScheme(t *testing.T) {
	wsID := uuid.New()
	ctrl, _ := newWebSocketController(t, &mockWSUsecase{}, WebSocketControllerConfig{Production: true})
	ctx, rec := newEchoContextWithParams(t, http.MethodGet, "/", map[string]string{"workspaceId": wsID.String()}, nil)
	ctx.Request().Header.Set(echo.HeaderAuthorization, "Bearer token")
	ctx.Request().Header.Set("X-Forwarded-Proto", "https")
	_ = ctrl.HandleWorkspace(ctx)
	if rec.Code == http.StatusForbidden {
		t.Fatalf("expected TLS guard bypass, status = %d", rec.Code)
	}
}

func TestWebSocketController_sendInitialEventsWithoutEditorJoined(t *testing.T) {
	docID, wsID, userID := uuid.New(), uuid.New(), uuid.New()
	hub := appws.NewHub()
	ctrl := &WebSocketController{hub: hub, now: func() time.Time { return testFixedTime }}
	client := &appws.Client{
		ConnectionID: uuid.New(), UserID: userID, DocumentID: docID, WorkspaceID: wsID,
		Hub: hub, Send: make(chan []byte, 8),
	}
	ctrl.sendInitialEvents(client, &usecase.PrepareWebSocketConnectionOutput{
		DocumentID: docID, WorkspaceID: wsID, Content: "hello", UpdatedAt: testFixedTime.Format(time.RFC3339),
	}, &usecase.RegisterWebSocketConnectionOutput{
		Editors: []usecase.DocumentEditorInfo{{UserID: userID, Name: "Alice"}},
	})
	if len(client.Send) != 3 {
		t.Fatalf("expected 3 events, got %d", len(client.Send))
	}
}
