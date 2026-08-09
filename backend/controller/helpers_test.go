package controller

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/config"
	"github.com/koezuka404/notehub/entity"
	"github.com/koezuka404/notehub/usecase"
)

func TestParseSameSite(t *testing.T) {
	if got := parseSameSite("Strict"); got != http.SameSiteStrictMode {
		t.Fatalf("Strict = %v", got)
	}
	if got := parseSameSite("None"); got != http.SameSiteNoneMode {
		t.Fatalf("None = %v", got)
	}
	if got := parseSameSite("Lax"); got != http.SameSiteLaxMode {
		t.Fatalf("default = %v", got)
	}
}

func TestAuthenticatedUserID(t *testing.T) {
	ctx, rec := newEchoContext(t, http.MethodGet, "/", nil)
	if _, err := authenticatedUserID(ctx); !errors.Is(err, errResponseSent) {
		t.Fatalf("expected errResponseSent, got %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}

	userID := uuid.New()
	ctx, rec = newEchoContext(t, http.MethodGet, "/", nil)
	setAuthenticatedUser(ctx, userID)
	got, err := authenticatedUserID(ctx)
	if err != nil || got != userID {
		t.Fatalf("authenticatedUserID() = %v, %v", got, err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status change: %d", rec.Code)
	}
}

func TestParseWorkspaceIDParam(t *testing.T) {
	ctx, rec := newEchoContextWithParams(t, http.MethodGet, "/", map[string]string{"workspaceId": "bad"}, nil)
	if _, err := parseWorkspaceIDParam(ctx); !errors.Is(err, errResponseSent) {
		t.Fatalf("expected errResponseSent, got %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}

	id := uuid.New()
	ctx, _ = newEchoContextWithParams(t, http.MethodGet, "/", map[string]string{"workspaceId": id.String()}, nil)
	got, err := parseWorkspaceIDParam(ctx)
	if err != nil || got != id {
		t.Fatalf("parseWorkspaceIDParam() = %v, %v", got, err)
	}
}

func TestParseDocumentIDParam(t *testing.T) {
	ctx, rec := newEchoContextWithParams(t, http.MethodGet, "/", map[string]string{"documentId": "bad"}, nil)
	if _, err := parseDocumentIDParam(ctx); !errors.Is(err, errResponseSent) {
		t.Fatalf("expected errResponseSent, got %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}

	id := uuid.New()
	ctx, _ = newEchoContextWithParams(t, http.MethodGet, "/", map[string]string{"documentId": id.String()}, nil)
	got, err := parseDocumentIDParam(ctx)
	if err != nil || got != id {
		t.Fatalf("parseDocumentIDParam() = %v, %v", got, err)
	}
}

func TestParseUserIDParam(t *testing.T) {
	ctx, rec := newEchoContextWithParams(t, http.MethodGet, "/", map[string]string{"userId": "bad"}, nil)
	if _, err := parseUserIDParam(ctx); !errors.Is(err, errResponseSent) {
		t.Fatalf("expected errResponseSent, got %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}

	id := uuid.New()
	ctx, _ = newEchoContextWithParams(t, http.MethodGet, "/", map[string]string{"userId": id.String()}, nil)
	got, err := parseUserIDParam(ctx)
	if err != nil || got != id {
		t.Fatalf("parseUserIDParam() = %v, %v", got, err)
	}
}

func TestParseVersionIDParam(t *testing.T) {
	ctx, rec := newEchoContextWithParams(t, http.MethodGet, "/", map[string]string{"versionId": "bad"}, nil)
	if _, err := parseVersionIDParam(ctx); !errors.Is(err, errResponseSent) {
		t.Fatalf("expected errResponseSent, got %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}

	id := uuid.New()
	ctx, _ = newEchoContextWithParams(t, http.MethodGet, "/", map[string]string{"versionId": id.String()}, nil)
	got, err := parseVersionIDParam(ctx)
	if err != nil || got != id {
		t.Fatalf("parseVersionIDParam() = %v, %v", got, err)
	}
}

func TestHandleAuthUseCaseError(t *testing.T) {
	cases := []struct {
		err      error
		status   int
		code     string
	}{
		{usecase.ErrValidation, 400, "VALIDATION_ERROR"},
		{usecase.ErrPasswordInvalid, 400, "PASSWORD_INVALID"},
		{usecase.ErrEmailAlreadyExists, 409, "EMAIL_ALREADY_EXISTS"},
		{usecase.ErrInvalidCredentials, 401, "INVALID_CREDENTIALS"},
		{usecase.ErrLoginTemporarilyLocked, 429, "LOGIN_RATE_LIMITED"},
		{usecase.ErrAccountSuspended, 401, "INVALID_CREDENTIALS"},
		{usecase.ErrAccountDeleted, 401, "INVALID_CREDENTIALS"},
		{usecase.ErrRefreshTokenRequired, 401, "REFRESH_TOKEN_REQUIRED"},
		{usecase.ErrRefreshTokenExpired, 401, "REFRESH_TOKEN_EXPIRED"},
		{usecase.ErrRefreshTokenInvalid, 401, "REFRESH_TOKEN_INVALID"},
		{usecase.ErrRefreshTokenRevoked, 401, "REFRESH_TOKEN_REVOKED"},
		{usecase.ErrRefreshTokenReused, 401, "REFRESH_TOKEN_REVOKED"},
		{usecase.ErrTokenOwnerMismatch, 403, "TOKEN_OWNER_MISMATCH"},
		{usecase.ErrCSRFTokenInvalid, 403, "CSRF_TOKEN_INVALID"},
		{usecase.ErrAuthServiceUnavailable, 503, "AUTH_SERVICE_UNAVAILABLE"},
		{usecase.ErrAccessTokenInvalid, 401, "ACCESS_TOKEN_INVALID"},
		{errors.New("unknown"), 500, "INTERNAL_ERROR"},
	}
	for _, tc := range cases {
		ctx, rec := newEchoContext(t, http.MethodGet, "/", nil)
		if err := handleAuthUseCaseError(ctx, tc.err); err != nil {
			t.Fatalf("handleAuthUseCaseError(%v): %v", tc.err, err)
		}
		if rec.Code != tc.status {
			t.Fatalf("status for %v = %d, want %d", tc.err, rec.Code, tc.status)
		}
		payload := decodeErrorResponse(t, rec)
		if payload.Error.Code != tc.code {
			t.Fatalf("code for %v = %q", tc.err, payload.Error.Code)
		}
	}
}

func TestHandleAccountUseCaseError(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{usecase.ErrCannotDeleteSelf, http.StatusConflict, "CANNOT_DELETE_SELF"},
		{usecase.ErrCannotReactivateSelf, http.StatusConflict, "CANNOT_REACTIVATE_SELF"},
		{usecase.ErrAccountNotSuspended, http.StatusConflict, "ACCOUNT_NOT_SUSPENDED"},
		{usecase.ErrCannotSuspendSelf, http.StatusConflict, "CANNOT_SUSPEND_SELF"},
		{usecase.ErrCannotSuspendHost, http.StatusConflict, "CANNOT_SUSPEND_HOST"},
		{usecase.ErrAccountAlreadySuspended, http.StatusConflict, "ACCOUNT_ALREADY_SUSPENDED"},
		{usecase.ErrAccountDeleted, http.StatusConflict, "ACCOUNT_DELETED"},
		{usecase.ErrTargetUserNotFound, http.StatusNotFound, "TARGET_USER_NOT_FOUND"},
		{usecase.ErrMemberNotFound, http.StatusNotFound, "MEMBER_NOT_FOUND"},
		{usecase.ErrWorkspaceNotFound, http.StatusNotFound, "WORKSPACE_NOT_FOUND"},
	}
	for _, tc := range cases {
		ctx, rec := newEchoContext(t, http.MethodGet, "/", nil)
		_ = handleAccountUseCaseError(ctx, tc.err)
		if rec.Code != tc.status {
			t.Fatalf("status for %v = %d", tc.err, rec.Code)
		}
		payload := decodeErrorResponse(t, rec)
		if payload.Error.Code != tc.code {
			t.Fatalf("code for %v = %q", tc.err, payload.Error.Code)
		}
	}
}

func TestHandleWorkspaceUseCaseError(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{usecase.ErrValidation, http.StatusBadRequest, "VALIDATION_ERROR"},
		{usecase.ErrAccountUnavailable, http.StatusForbidden, "ACCOUNT_UNAVAILABLE"},
		{usecase.ErrAccountSuspended, http.StatusForbidden, "ACCOUNT_UNAVAILABLE"},
		{usecase.ErrAccountDeleted, http.StatusForbidden, "ACCOUNT_UNAVAILABLE"},
		{usecase.ErrWorkspaceNotFound, http.StatusNotFound, "WORKSPACE_NOT_FOUND"},
		{usecase.ErrWorkspaceAlreadyDeleted, http.StatusConflict, "WORKSPACE_ALREADY_DELETED"},
		{usecase.ErrWorkspaceAccessDenied, http.StatusForbidden, "WORKSPACE_ACCESS_DENIED"},
		{usecase.ErrWorkspacePermissionDenied, http.StatusForbidden, "WORKSPACE_PERMISSION_DENIED"},
		{usecase.ErrHostPermissionRequired, http.StatusForbidden, "HOST_PERMISSION_REQUIRED"},
		{usecase.ErrWorkspaceHostSuspended, http.StatusLocked, "WORKSPACE_HOST_SUSPENDED"},
		{usecase.ErrWorkspaceHostDeleted, http.StatusLocked, "WORKSPACE_HOST_DELETED"},
		{errors.New("unknown"), http.StatusInternalServerError, "INTERNAL_ERROR"},
	}
	for _, tc := range cases {
		ctx, rec := newEchoContext(t, http.MethodGet, "/", nil)
		_ = handleWorkspaceUseCaseError(ctx, tc.err)
		if rec.Code != tc.status {
			t.Fatalf("status for %v = %d", tc.err, rec.Code)
		}
		payload := decodeErrorResponse(t, rec)
		if payload.Error.Code != tc.code {
			t.Fatalf("code for %v = %q", tc.err, payload.Error.Code)
		}
	}
}

func TestHandleDocumentUseCaseError(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{usecase.ErrValidation, http.StatusBadRequest, "VALIDATION_ERROR"},
		{usecase.ErrDocumentNotFound, http.StatusNotFound, "DOCUMENT_NOT_FOUND"},
		{usecase.ErrDocumentDeleted, http.StatusNotFound, "DOCUMENT_DELETED"},
		{usecase.ErrDocumentContentTooLarge, http.StatusBadRequest, "DOCUMENT_CONTENT_TOO_LARGE"},
		{usecase.ErrWorkspaceNotFound, http.StatusNotFound, "WORKSPACE_NOT_FOUND"},
	}
	for _, tc := range cases {
		ctx, rec := newEchoContext(t, http.MethodGet, "/", nil)
		_ = handleDocumentUseCaseError(ctx, tc.err)
		if rec.Code != tc.status {
			t.Fatalf("status for %v = %d", tc.err, rec.Code)
		}
		payload := decodeErrorResponse(t, rec)
		if payload.Error.Code != tc.code {
			t.Fatalf("code for %v = %q", tc.err, payload.Error.Code)
		}
	}
}

func TestHandleMemberUseCaseError(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{usecase.ErrValidation, http.StatusBadRequest, "VALIDATION_ERROR"},
		{usecase.ErrUserNotFound, http.StatusNotFound, "USER_NOT_FOUND"},
		{usecase.ErrTargetUserNotFound, http.StatusNotFound, "TARGET_USER_NOT_FOUND"},
		{usecase.ErrTargetAccountUnavailable, http.StatusConflict, "TARGET_ACCOUNT_UNAVAILABLE"},
		{usecase.ErrCannotAddSelf, http.StatusConflict, "CANNOT_ADD_SELF"},
		{usecase.ErrMemberAlreadyExists, http.StatusConflict, "MEMBER_ALREADY_EXISTS"},
		{usecase.ErrMemberNotFound, http.StatusNotFound, "MEMBER_NOT_FOUND"},
		{usecase.ErrCannotRemoveHost, http.StatusConflict, "CANNOT_REMOVE_HOST"},
		{usecase.ErrWorkspaceAccessDenied, http.StatusForbidden, "WORKSPACE_ACCESS_DENIED"},
	}
	for _, tc := range cases {
		ctx, rec := newEchoContext(t, http.MethodGet, "/", nil)
		_ = handleMemberUseCaseError(ctx, tc.err)
		if rec.Code != tc.status {
			t.Fatalf("status for %v = %d", tc.err, rec.Code)
		}
		payload := decodeErrorResponse(t, rec)
		if payload.Error.Code != tc.code {
			t.Fatalf("code for %v = %q", tc.err, payload.Error.Code)
		}
	}
}

func TestHandleVersionUseCaseError(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{usecase.ErrVersionNotFound, http.StatusNotFound, "VERSION_NOT_FOUND"},
		{entity.ErrDocumentConflict, http.StatusConflict, "DOCUMENT_CONFLICT"},
		{usecase.ErrDocumentNotFound, http.StatusNotFound, "DOCUMENT_NOT_FOUND"},
	}
	for _, tc := range cases {
		ctx, rec := newEchoContext(t, http.MethodGet, "/", nil)
		_ = handleVersionUseCaseError(ctx, tc.err)
		if rec.Code != tc.status {
			t.Fatalf("status for %v = %d", tc.err, rec.Code)
		}
		payload := decodeErrorResponse(t, rec)
		if payload.Error.Code != tc.code {
			t.Fatalf("code for %v = %q", tc.err, payload.Error.Code)
		}
	}
}

func TestHandleWebSocketUseCaseError(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{usecase.ErrWebSocketConnectionLimitExceeded, http.StatusConflict, "WEBSOCKET_CONNECTION_LIMIT_EXCEEDED"},
		{usecase.ErrValidation, http.StatusBadRequest, "VALIDATION_ERROR"},
		{usecase.ErrDocumentNotFound, http.StatusNotFound, "DOCUMENT_NOT_FOUND"},
		{usecase.ErrDocumentDeleted, http.StatusNotFound, "DOCUMENT_DELETED"},
		{usecase.ErrWorkspaceNotFound, http.StatusNotFound, "WORKSPACE_NOT_FOUND"},
	}
	for _, tc := range cases {
		ctx, rec := newEchoContext(t, http.MethodGet, "/", nil)
		_ = handleWebSocketUseCaseError(ctx, tc.err)
		if rec.Code != tc.status {
			t.Fatalf("status for %v = %d", tc.err, rec.Code)
		}
		payload := decodeErrorResponse(t, rec)
		if payload.Error.Code != tc.code {
			t.Fatalf("code for %v = %q", tc.err, payload.Error.Code)
		}
	}
}

func TestWriteAuthErrorAndWriteWorkspaceError(t *testing.T) {
	ctx, rec := newEchoContext(t, http.MethodGet, "/", nil)
	_ = writeAuthError(ctx, http.StatusBadRequest, "CODE", "message")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("writeAuthError status = %d", rec.Code)
	}
	ctx, rec = newEchoContext(t, http.MethodGet, "/", nil)
	_ = writeWorkspaceError(ctx, http.StatusForbidden, "CODE", "message")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("writeWorkspaceError status = %d", rec.Code)
	}
}

func TestWriteWebSocketHTTPError(t *testing.T) {
	ctx, rec := newEchoContext(t, http.MethodGet, "/", nil)
	_ = writeWebSocketHTTPError(ctx, http.StatusUnauthorized, "CODE", "message")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestToEditorEventData(t *testing.T) {
	userID := uuid.New()
	out := toEditorEventData([]usecase.DocumentEditorInfo{{UserID: userID, Name: "Alice"}})
	if len(out) != 1 || out[0].UserID != userID.String() || out[0].Name != "Alice" {
		t.Fatalf("toEditorEventData() = %+v", out)
	}
}

func TestWebSocketControllerConfigFromApp(t *testing.T) {
	cfg := WebSocketControllerConfigFromApp(&config.Config{
		Environment:             config.EnvironmentDevelopment,
		AllowedOrigins:          nil,
		RedisOperationTimeout:   3 * time.Second,
	})
	if len(cfg.AllowedOrigins) != 2 || cfg.Production {
		t.Fatalf("dev config = %+v", cfg)
	}
	if cfg.OperationTimeout != 3*time.Second {
		t.Fatalf("timeout = %v", cfg.OperationTimeout)
	}

	prod := WebSocketControllerConfigFromApp(&config.Config{
		Environment:           config.EnvironmentProduction,
		AllowedOrigins:        []string{"https://app.example.com"},
		RedisOperationTimeout: 0,
	})
	if !prod.Production || len(prod.AllowedOrigins) != 1 {
		t.Fatalf("prod config = %+v", prod)
	}
}
