package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	gorillaws "github.com/gorilla/websocket"
	"github.com/koezuka404/notehub/entity"
	infrcrypto "github.com/koezuka404/notehub/usecase/crypto"
	"github.com/koezuka404/notehub/usecase"
	appws "github.com/koezuka404/notehub/websocket"
	"github.com/labstack/echo/v4"
)

func newWebSocketController(t *testing.T, ws *mockWSUsecase, cfg WebSocketControllerConfig) (*WebSocketController, uuid.UUID) {
	t.Helper()
	userID := uuid.New()
	user := &entity.User{ID: userID, Status: entity.UserStatusActive, AuthVersion: 1}
	claims := infrcrypto.AccessTokenClaims{UserID: userID, JTI: uuid.New(), AuthVersion: 1}
	ctrl := NewWebSocketController(
		ws,
		&mockTokenValidator{claims: claims},
		&mockUserFinder{user: user, found: true},
		&mockRevocationChecker{},
		appws.NewHub(),
		cfg,
	)
	ctrl.now = func() time.Time { return testFixedTime }
	return ctrl, userID
}

func TestNewWebSocketController_TrimsOrigins(t *testing.T) {
	ctrl := NewWebSocketController(&mockWSUsecase{}, &mockTokenValidator{}, &mockUserFinder{}, &mockRevocationChecker{}, appws.NewHub(), WebSocketControllerConfig{
		AllowedOrigins: []string{" http://localhost:5173 ", ""},
	})
	if _, ok := ctrl.allowedOrigins["http://localhost:5173"]; !ok {
		t.Fatal("expected trimmed origin")
	}
	if len(ctrl.allowedOrigins) != 1 {
		t.Fatalf("origins = %v", ctrl.allowedOrigins)
	}
}

func TestWebSocketController_backgroundContext(t *testing.T) {
	ctrl := &WebSocketController{opTimeout: 0}
	ctx, cancel := ctrl.backgroundContext()
	if ctx == nil {
		t.Fatal("expected context")
	}
	cancel()

	ctrl.opTimeout = 2 * time.Second
	ctx, cancel = ctrl.backgroundContext()
	deadline, ok := ctx.Deadline()
	cancel()
	if !ok || time.Until(deadline) > 2*time.Second {
		t.Fatalf("deadline = %v ok=%v", deadline, ok)
	}
}

func TestWebSocketController_checkOrigin(t *testing.T) {
	ctrl := &WebSocketController{}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if !ctrl.checkOrigin(req) {
		t.Fatal("empty origin should pass")
	}

	ctrl.allowedOrigins = map[string]struct{}{}
	req.Header.Set("Origin", "http://any.example")
	if !ctrl.checkOrigin(req) {
		t.Fatal("empty allowlist should pass")
	}

	ctrl.allowedOrigins = map[string]struct{}{"http://allowed.example": {}}
	req.Header.Set("Origin", "http://allowed.example")
	if !ctrl.checkOrigin(req) {
		t.Fatal("allowed origin should pass")
	}
	req.Header.Set("Origin", "http://denied.example")
	if ctrl.checkOrigin(req) {
		t.Fatal("denied origin should fail")
	}
}

func TestWebSocketController_authenticate(t *testing.T) {
	userID := uuid.New()
	validUser := &entity.User{ID: userID, Status: entity.UserStatusActive, AuthVersion: 1}
	validClaims := infrcrypto.AccessTokenClaims{UserID: userID, JTI: uuid.New(), AuthVersion: 1}

	makeCtrl := func(tokens *mockTokenValidator, users *mockUserFinder, revoked *mockRevocationChecker) *WebSocketController {
		ctrl := NewWebSocketController(&mockWSUsecase{}, tokens, users, revoked, appws.NewHub(), WebSocketControllerConfig{})
		ctrl.now = func() time.Time { return testFixedTime }
		return ctrl
	}

	t.Run("query token", func(t *testing.T) {
		ctrl := makeCtrl(&mockTokenValidator{claims: validClaims}, &mockUserFinder{user: validUser, found: true}, &mockRevocationChecker{})
		ctx, _ := newEchoContext(t, http.MethodGet, "/?access_token=token", nil)
		got, err := ctrl.authenticate(ctx)
		if err != nil || got != userID {
			t.Fatalf("authenticate() = %v, %v", got, err)
		}
	})

	t.Run("bearer token", func(t *testing.T) {
		ctrl := makeCtrl(&mockTokenValidator{claims: validClaims}, &mockUserFinder{user: validUser, found: true}, &mockRevocationChecker{})
		ctx, _ := newEchoContext(t, http.MethodGet, "/", nil)
		ctx.Request().Header.Set(echo.HeaderAuthorization, "Bearer token")
		got, err := ctrl.authenticate(ctx)
		if err != nil || got != userID {
			t.Fatalf("authenticate() = %v, %v", got, err)
		}
	})

	t.Run("missing token", func(t *testing.T) {
		ctrl := makeCtrl(&mockTokenValidator{}, &mockUserFinder{}, &mockRevocationChecker{})
		ctx, rec := newEchoContext(t, http.MethodGet, "/", nil)
		_, _ = ctrl.authenticate(ctx)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d", rec.Code)
		}
		payload := decodeErrorResponse(t, rec)
		if payload.Error.Code != "ACCESS_TOKEN_REQUIRED" {
			t.Fatalf("code = %q", payload.Error.Code)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		ctrl := makeCtrl(&mockTokenValidator{err: jwt.ErrTokenExpired}, &mockUserFinder{}, &mockRevocationChecker{})
		ctx, rec := newEchoContext(t, http.MethodGet, "/?access_token=token", nil)
		_, _ = ctrl.authenticate(ctx)
		payload := decodeErrorResponse(t, rec)
		if payload.Error.Code != "ACCESS_TOKEN_EXPIRED" {
			t.Fatalf("code = %q", payload.Error.Code)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		ctrl := makeCtrl(&mockTokenValidator{err: errors.New("bad")}, &mockUserFinder{}, &mockRevocationChecker{})
		ctx, rec := newEchoContext(t, http.MethodGet, "/?access_token=token", nil)
		_, _ = ctrl.authenticate(ctx)
		payload := decodeErrorResponse(t, rec)
		if payload.Error.Code != "ACCESS_TOKEN_INVALID" {
			t.Fatalf("code = %q", payload.Error.Code)
		}
	})

	t.Run("revocation check error", func(t *testing.T) {
		ctrl := makeCtrl(&mockTokenValidator{claims: validClaims}, &mockUserFinder{}, &mockRevocationChecker{err: errors.New("redis")})
		ctx, rec := newEchoContext(t, http.MethodGet, "/?access_token=token", nil)
		_, _ = ctrl.authenticate(ctx)
		payload := decodeErrorResponse(t, rec)
		if payload.Error.Code != "AUTH_SERVICE_UNAVAILABLE" {
			t.Fatalf("code = %q", payload.Error.Code)
		}
	})

	t.Run("revoked token", func(t *testing.T) {
		ctrl := makeCtrl(&mockTokenValidator{claims: validClaims}, &mockUserFinder{}, &mockRevocationChecker{revoked: true})
		ctx, rec := newEchoContext(t, http.MethodGet, "/?access_token=token", nil)
		_, _ = ctrl.authenticate(ctx)
		payload := decodeErrorResponse(t, rec)
		if payload.Error.Code != "ACCESS_TOKEN_REVOKED" {
			t.Fatalf("code = %q", payload.Error.Code)
		}
	})

	t.Run("user lookup error", func(t *testing.T) {
		ctrl := makeCtrl(&mockTokenValidator{claims: validClaims}, &mockUserFinder{err: errors.New("db")}, &mockRevocationChecker{})
		ctx, rec := newEchoContext(t, http.MethodGet, "/?access_token=token", nil)
		_, _ = ctrl.authenticate(ctx)
		payload := decodeErrorResponse(t, rec)
		if payload.Error.Code != "DATABASE_ERROR" {
			t.Fatalf("code = %q", payload.Error.Code)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		ctrl := makeCtrl(&mockTokenValidator{claims: validClaims}, &mockUserFinder{found: false}, &mockRevocationChecker{})
		ctx, rec := newEchoContext(t, http.MethodGet, "/?access_token=token", nil)
		_, _ = ctrl.authenticate(ctx)
		payload := decodeErrorResponse(t, rec)
		if payload.Error.Code != "ACCESS_TOKEN_INVALID" {
			t.Fatalf("code = %q", payload.Error.Code)
		}
	})

	t.Run("user cannot authenticate", func(t *testing.T) {
		suspended := &entity.User{ID: userID, Status: entity.UserStatusSuspended, AuthVersion: 1}
		ctrl := makeCtrl(&mockTokenValidator{claims: validClaims}, &mockUserFinder{user: suspended, found: true}, &mockRevocationChecker{})
		ctx, rec := newEchoContext(t, http.MethodGet, "/?access_token=token", nil)
		_, _ = ctrl.authenticate(ctx)
		payload := decodeErrorResponse(t, rec)
		if payload.Error.Code != "ACCESS_TOKEN_INVALID" {
			t.Fatalf("code = %q", payload.Error.Code)
		}
	})

	t.Run("auth version mismatch", func(t *testing.T) {
		ctrl := makeCtrl(&mockTokenValidator{claims: validClaims}, &mockUserFinder{user: &entity.User{ID: userID, Status: entity.UserStatusActive, AuthVersion: 2}, found: true}, &mockRevocationChecker{})
		ctx, rec := newEchoContext(t, http.MethodGet, "/?access_token=token", nil)
		_, _ = ctrl.authenticate(ctx)
		payload := decodeErrorResponse(t, rec)
		if payload.Error.Code != "ACCESS_TOKEN_REVOKED" {
			t.Fatalf("code = %q", payload.Error.Code)
		}
	})
}

func TestWebSocketController_sendUseCaseError(t *testing.T) {
	ctrl := &WebSocketController{now: func() time.Time { return testFixedTime }}
	client := &appws.Client{Send: make(chan []byte, 4)}

	errorsCases := []struct {
		err  error
		code string
	}{
		{usecase.ErrValidation, "VALIDATION_ERROR"},
		{usecase.ErrDocumentNotFound, "DOCUMENT_NOT_FOUND"},
		{usecase.ErrDocumentDeleted, "DOCUMENT_DELETED"},
		{usecase.ErrDocumentContentTooLarge, "DOCUMENT_CONTENT_TOO_LARGE"},
		{usecase.ErrWorkspaceNotFound, "WORKSPACE_NOT_FOUND"},
		{usecase.ErrWorkspaceAccessDenied, "WORKSPACE_ACCESS_DENIED"},
		{usecase.ErrWorkspaceHostSuspended, "WORKSPACE_HOST_SUSPENDED"},
		{usecase.ErrWorkspaceHostDeleted, "WORKSPACE_HOST_DELETED"},
		{errors.New("unknown"), "INTERNAL_ERROR"},
	}
	for _, tc := range errorsCases {
		client.Send = make(chan []byte, 1)
		ctrl.sendUseCaseError(client, tc.err)
		payload := <-client.Send
		var envelope appws.Envelope
		if err := json.Unmarshal(payload, &envelope); err != nil {
			t.Fatal(err)
		}
		var data appws.ErrorData
		if err := json.Unmarshal(envelope.Data, &data); err != nil {
			t.Fatal(err)
		}
		if data.Code != tc.code {
			t.Fatalf("code for %v = %q", tc.err, data.Code)
		}
	}
}

func TestWebSocketController_handleClientMessage(t *testing.T) {
	docID := uuid.New()
	userID := uuid.New()
	ctrl := &WebSocketController{
		wsUseCase: &mockWSUsecase{
			applyEditOut: &usecase.ApplyDocumentEditOutput{
				DocumentID: docID, Content: "updated", UpdatedBy: userID, UpdatedAt: testFixedTime.Format(time.RFC3339),
			},
		},
		hub: appws.NewHub(),
		now: func() time.Time { return testFixedTime },
	}
	client := &appws.Client{
		ConnectionID: uuid.New(), UserID: userID, DocumentID: docID,
		Hub: ctrl.hub, Send: make(chan []byte, 4),
	}

	ctrl.handleClientMessage(client, []byte("{bad"))
	select {
	case payload := <-client.Send:
		var envelope appws.Envelope
		_ = json.Unmarshal(payload, &envelope)
		var data appws.ErrorData
		_ = json.Unmarshal(envelope.Data, &data)
		if data.Code != "INVALID_REQUEST" {
			t.Fatalf("invalid json code = %q", data.Code)
		}
	default:
		t.Fatal("expected error event")
	}

	editMsg, _ := json.Marshal(appws.ClientMessage{Type: appws.EventDocumentEdit, Data: json.RawMessage(`{"content":`)})
	ctrl.handleClientMessage(client, editMsg)
	select {
	case payload := <-client.Send:
		var envelope appws.Envelope
		_ = json.Unmarshal(payload, &envelope)
		var data appws.ErrorData
		_ = json.Unmarshal(envelope.Data, &data)
		if data.Code != "INVALID_REQUEST" {
			t.Fatalf("invalid edit code = %q", data.Code)
		}
	default:
		t.Fatal("expected invalid edit error")
	}

	validEdit, _ := json.Marshal(appws.ClientMessage{
		Type: appws.EventDocumentEdit,
		Data: mustJSON(t, appws.DocumentEditData{Content: "updated"}),
	})
	ctrl.handleClientMessage(client, validEdit)

	unsupported, _ := json.Marshal(appws.ClientMessage{Type: "unknown", Data: json.RawMessage(`{}`)})
	ctrl.handleClientMessage(client, unsupported)
	select {
	case payload := <-client.Send:
		var envelope appws.Envelope
		_ = json.Unmarshal(payload, &envelope)
		var data appws.ErrorData
		_ = json.Unmarshal(envelope.Data, &data)
		if data.Code != "UNSUPPORTED_EVENT" {
			t.Fatalf("unsupported code = %q", data.Code)
		}
	default:
		t.Fatal("expected unsupported event error")
	}

	ctrl.wsUseCase = &mockWSUsecase{applyEditErr: usecase.ErrDocumentNotFound}
	ctrl.handleClientMessage(client, validEdit)
	select {
	case payload := <-client.Send:
		var envelope appws.Envelope
		_ = json.Unmarshal(payload, &envelope)
		var data appws.ErrorData
		_ = json.Unmarshal(envelope.Data, &data)
		if data.Code != "DOCUMENT_NOT_FOUND" {
			t.Fatalf("usecase error code = %q", data.Code)
		}
	default:
		t.Fatal("expected usecase error event")
	}

	origMarshal := wsMarshalEvent
	wsMarshalEvent = func(string, any, time.Time) ([]byte, error) { return nil, errors.New("marshal") }
	t.Cleanup(func() { wsMarshalEvent = origMarshal })
	ctrl.wsUseCase = &mockWSUsecase{
		applyEditOut: &usecase.ApplyDocumentEditOutput{
			DocumentID: docID, Content: "updated", UpdatedBy: userID, UpdatedAt: testFixedTime.Format(time.RFC3339),
		},
	}
	ctrl.handleClientMessage(client, validEdit)
}

func TestWebSocketController_sendInitialEvents(t *testing.T) {
	docID, wsID, userID := uuid.New(), uuid.New(), uuid.New()
	hub := appws.NewHub()
	ctrl := &WebSocketController{hub: hub, now: func() time.Time { return testFixedTime }}
	client := &appws.Client{
		ConnectionID: uuid.New(), UserID: userID, DocumentID: docID, WorkspaceID: wsID,
		Hub: hub, Send: make(chan []byte, 8),
	}
	prepared := &usecase.PrepareWebSocketConnectionOutput{
		DocumentID: docID, WorkspaceID: wsID, Content: "hello", UpdatedAt: testFixedTime.Format(time.RFC3339),
	}
	registered := &usecase.RegisterWebSocketConnectionOutput{
		Editors:      []usecase.DocumentEditorInfo{{UserID: userID, Name: "Alice"}},
		EditorJoined: true,
		JoinedEditor: usecase.DocumentEditorInfo{UserID: userID, Name: "Alice"},
	}
	ctrl.sendInitialEvents(client, prepared, registered)
	if len(client.Send) != 3 {
		t.Fatalf("expected 3 direct events, got %d", len(client.Send))
	}

	origMarshal := wsMarshalEvent
	wsMarshalEvent = func(eventType string, _ any, _ time.Time) ([]byte, error) {
		if eventType == appws.EventEditorJoined {
			return nil, errors.New("marshal")
		}
		return origMarshal(eventType, appws.EditorEventData{UserID: userID.String(), Name: "Alice"}, testFixedTime)
	}
	t.Cleanup(func() { wsMarshalEvent = origMarshal })
	ctrl.sendInitialEvents(client, prepared, registered)
}

func TestWebSocketController_cleanupClient(t *testing.T) {
	docID, userID := uuid.New(), uuid.New()
	ctrl := &WebSocketController{
		wsUseCase: &mockWSUsecase{unregisterErr: errors.New("fail")},
		now:       func() time.Time { return testFixedTime },
	}
	client := &appws.Client{ConnectionID: uuid.New(), UserID: userID, DocumentID: docID}
	ctrl.cleanupClient(client)

	ctrl.wsUseCase = &mockWSUsecase{
		unregisterOut: &usecase.UnregisterWebSocketConnectionOutput{
			EditorLeft: true,
			LeftEditor: usecase.DocumentEditorInfo{UserID: userID, Name: "Alice"},
		},
	}
	ctrl.hub = appws.NewHub()
	client.Hub = ctrl.hub
	ctrl.cleanupClient(client)

	origMarshal := wsMarshalEvent
	wsMarshalEvent = func(string, any, time.Time) ([]byte, error) { return nil, errors.New("marshal") }
	t.Cleanup(func() { wsMarshalEvent = origMarshal })
	ctrl.cleanupClient(client)
}

func TestWebSocketController_sendEventMarshalError(t *testing.T) {
	ctrl := &WebSocketController{now: func() time.Time { return testFixedTime }}
	client := &appws.Client{Send: make(chan []byte, 1)}
	origMarshal := wsMarshalEvent
	wsMarshalEvent = func(string, any, time.Time) ([]byte, error) { return nil, errors.New("marshal") }
	t.Cleanup(func() { wsMarshalEvent = origMarshal })
	ctrl.sendEvent(client, appws.EventConnected, appws.ConnectedData{ConnectionID: uuid.NewString()}, testFixedTime)
	if len(client.Send) != 0 {
		t.Fatal("expected no message on marshal error")
	}
}

func TestWebSocketController_authenticateBearerHeaderOnly(t *testing.T) {
	userID := uuid.New()
	validUser := &entity.User{ID: userID, Status: entity.UserStatusActive, AuthVersion: 1}
	validClaims := infrcrypto.AccessTokenClaims{UserID: userID, JTI: uuid.New(), AuthVersion: 1}
	ctrl := NewWebSocketController(&mockWSUsecase{}, &mockTokenValidator{claims: validClaims}, &mockUserFinder{user: validUser, found: true}, &mockRevocationChecker{}, appws.NewHub(), WebSocketControllerConfig{})
	ctrl.now = func() time.Time { return testFixedTime }
	ctx, _ := newEchoContext(t, http.MethodGet, "/", nil)
	ctx.Request().Header.Set(echo.HeaderAuthorization, "Bearer header-token")
	got, err := ctrl.authenticate(ctx)
	if err != nil || got != userID {
		t.Fatalf("authenticate() = %v, %v", got, err)
	}
}

func TestWebSocketController_HandleDocument_ProductionTLS(t *testing.T) {
	ctrl, _ := newWebSocketController(t, &mockWSUsecase{}, WebSocketControllerConfig{Production: true})
	ctx, rec := newEchoContextWithParams(t, http.MethodGet, "/documents/:documentId",
		map[string]string{"documentId": uuid.NewString()}, nil)
	ctx.Request().Header.Set(echo.HeaderAuthorization, "Bearer token")
	err := ctrl.HandleDocument(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestWebSocketController_HandleWorkspace_ProductionTLS(t *testing.T) {
	ctrl, _ := newWebSocketController(t, &mockWSUsecase{}, WebSocketControllerConfig{Production: true})
	ctx, rec := newEchoContextWithParams(t, http.MethodGet, "/workspaces/:workspaceId/ws",
		map[string]string{"workspaceId": uuid.NewString()}, nil)
	ctx.Request().Header.Set(echo.HeaderAuthorization, "Bearer token")
	err := ctrl.HandleWorkspace(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestWebSocketController_HandleDocument_RegisterConnectionError(t *testing.T) {
	docID := uuid.New()
	ctrl, _ := newWebSocketController(t, &mockWSUsecase{
		prepareOut:  &usecase.PrepareWebSocketConnectionOutput{DocumentID: docID, WorkspaceID: uuid.New()},
		registerErr: usecase.ErrWebSocketConnectionLimitExceeded,
	}, WebSocketControllerConfig{})
	ctx, rec := newEchoContextWithParams(t, http.MethodGet, "/documents/:documentId",
		map[string]string{"documentId": docID.String()}, nil)
	ctx.Request().Header.Set(echo.HeaderAuthorization, "Bearer token")
	err := ctrl.HandleDocument(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestWebSocketController_HandleDocument_UpgradeFailure(t *testing.T) {
	docID := uuid.New()
	unregisterCalled := false
	mock := &mockWSUsecase{
		prepareOut:    &usecase.PrepareWebSocketConnectionOutput{DocumentID: docID, WorkspaceID: uuid.New()},
		registerOut:   &usecase.RegisterWebSocketConnectionOutput{},
		unregisterOut: &usecase.UnregisterWebSocketConnectionOutput{},
		onUnregister:  func() { unregisterCalled = true },
	}
	ctrl, _ := newWebSocketController(t, mock, WebSocketControllerConfig{})
	ctx, _ := newEchoContextWithParams(t, http.MethodGet, "/documents/:documentId",
		map[string]string{"documentId": docID.String()}, nil)
	ctx.Request().Header.Set(echo.HeaderAuthorization, "Bearer token")
	if err := ctrl.HandleDocument(ctx); err == nil {
		t.Fatal("expected upgrade error")
	}
	if !unregisterCalled {
		t.Fatal("expected unregister on upgrade failure")
	}
}

func TestWebSocketController_HandleDocument_Success(t *testing.T) {
	docID := uuid.New()
	wsID := uuid.New()
	ctrl, _ := newWebSocketController(t, &mockWSUsecase{
		prepareOut: &usecase.PrepareWebSocketConnectionOutput{
			DocumentID: docID, WorkspaceID: wsID, Content: "hello", UpdatedAt: testFixedTime.Format(time.RFC3339),
		},
		registerOut: &usecase.RegisterWebSocketConnectionOutput{
			Editors: []usecase.DocumentEditorInfo{{UserID: uuid.New(), Name: "Alice"}},
		},
		unregisterOut: &usecase.UnregisterWebSocketConnectionOutput{},
	}, WebSocketControllerConfig{AllowedOrigins: []string{"http://127.0.0.1"}})

	e := echo.New()
	e.GET("/ws/documents/:documentId", ctrl.HandleDocument)
	server := httptest.NewServer(e)
	t.Cleanup(server.Close)

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/documents/" + docID.String() + "?access_token=token"
	header := http.Header{}
	header.Set("Origin", "http://127.0.0.1")
	conn, resp, err := gorillaws.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var envelope appws.Envelope
	if err := json.Unmarshal(msg, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Type != appws.EventConnected {
		t.Fatalf("type = %q", envelope.Type)
	}
}

func TestWebSocketController_HandleWorkspace_Success(t *testing.T) {
	wsID := uuid.New()
	ctrl, _ := newWebSocketController(t, &mockWSUsecase{}, WebSocketControllerConfig{AllowedOrigins: []string{"http://127.0.0.1"}})

	e := echo.New()
	e.GET("/ws/workspaces/:workspaceId", ctrl.HandleWorkspace)
	server := httptest.NewServer(e)
	t.Cleanup(server.Close)

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/workspaces/" + wsID.String() + "?access_token=token"
	header := http.Header{}
	header.Set("Origin", "http://127.0.0.1")
	conn, resp, err := gorillaws.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	_ = conn.WriteMessage(gorillaws.TextMessage, []byte(`{"type":"ping","data":{}}`))
	time.Sleep(50 * time.Millisecond)
}

func TestWebSocketController_HandleDocument_PrepareError(t *testing.T) {
	docID := uuid.New()
	ctrl, _ := newWebSocketController(t, &mockWSUsecase{prepareErr: usecase.ErrDocumentNotFound}, WebSocketControllerConfig{})
	ctx, rec := newEchoContextWithParams(t, http.MethodGet, "/documents/:documentId",
		map[string]string{"documentId": docID.String()}, nil)
	ctx.Request().Header.Set(echo.HeaderAuthorization, "Bearer token")
	_ = ctrl.HandleDocument(ctx)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestWebSocketController_HandleWorkspace_PrepareError(t *testing.T) {
	wsID := uuid.New()
	ctrl, _ := newWebSocketController(t, &mockWSUsecase{prepareWSErr: usecase.ErrWorkspaceAccessDenied}, WebSocketControllerConfig{})
	ctx, rec := newEchoContextWithParams(t, http.MethodGet, "/workspaces/:workspaceId/ws",
		map[string]string{"workspaceId": wsID.String()}, nil)
	ctx.Request().Header.Set(echo.HeaderAuthorization, "Bearer token")
	_ = ctrl.HandleWorkspace(ctx)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rec.Code)
	}
}

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
