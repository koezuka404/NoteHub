package usecase

import "errors"


const timeFormat = "2006-01-02T15:04:05Z07:00"

var (
	ErrValidation             = errors.New("validation error")
	ErrPasswordInvalid        = errors.New("password invalid")
	ErrEmailAlreadyExists     = errors.New("email already exists")
	ErrInvalidCredentials     = errors.New("invalid credentials")
	ErrAccountSuspended       = errors.New("account suspended")
	ErrAccountDeleted         = errors.New("account deleted")
	ErrAccountUnavailable     = errors.New("account unavailable")
	ErrRefreshTokenRequired   = errors.New("refresh token is required")
	ErrRefreshTokenInvalid    = errors.New("refresh token is invalid")
	ErrRefreshTokenExpired    = errors.New("refresh token is expired")
	ErrRefreshTokenRevoked    = errors.New("refresh token is revoked")
	ErrRefreshTokenReused     = errors.New("refresh token was reused")
	ErrAccessTokenInvalid     = errors.New("access token is invalid")
	ErrAccessTokenExpired     = errors.New("access token is expired")
	ErrAccessTokenRevoked     = errors.New("access token is revoked")
	ErrCSRFTokenInvalid       = errors.New("csrf token is invalid")
	ErrTokenOwnerMismatch     = errors.New("token owner mismatch")
	ErrAuthServiceUnavailable = errors.New("authentication service unavailable")
	ErrLoginTemporarilyLocked = errors.New("login temporarily locked")

	ErrWorkspaceNotFound           = errors.New("workspace not found")
	ErrWorkspaceAlreadyDeleted     = errors.New("workspace already deleted")
	ErrWorkspaceAccessDenied       = errors.New("workspace access denied")
	ErrWorkspacePermissionDenied   = errors.New("workspace permission denied")
	ErrHostPermissionRequired      = errors.New("host permission required")
	ErrWorkspaceHostSuspended      = errors.New("workspace host suspended")
	ErrWorkspaceHostDeleted        = errors.New("workspace host deleted")

	ErrUserNotFound              = errors.New("user not found")
	ErrTargetUserNotFound        = errors.New("target user not found")
	ErrTargetAccountUnavailable  = errors.New("target account unavailable")
	ErrCannotAddSelf             = errors.New("cannot add self")
	ErrMemberAlreadyExists       = errors.New("member already exists")
	ErrMemberNotFound            = errors.New("member not found")
	ErrCannotRemoveHost          = errors.New("cannot remove host")
	ErrCannotSuspendSelf         = errors.New("cannot suspend self")
	ErrCannotSuspendHost         = errors.New("cannot suspend host")
	ErrAccountAlreadySuspended   = errors.New("account already suspended")
	ErrAccountNotSuspended       = errors.New("account not suspended")
	ErrCannotReactivateSelf      = errors.New("cannot reactivate self")
	ErrCannotDeleteSelf          = errors.New("cannot delete self")

	ErrDocumentNotFound = errors.New("document not found")
	ErrDocumentDeleted  = errors.New("document deleted")
	ErrDocumentContentTooLarge = errors.New("document content too large")

	ErrVersionNotFound = errors.New("version not found")

	ErrWebSocketConnectionLimitExceeded = errors.New("websocket connection limit exceeded")
)
