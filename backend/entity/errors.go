package entity

import "errors"

var (
	ErrInvalidStateTransition = errors.New("invalid state transition")
	ErrUserNotActive          = errors.New("user is not active")
	ErrUserNotSuspended       = errors.New("user is not suspended")
	ErrUserDeleted            = errors.New("user is deleted")
	ErrWorkspaceDeleted       = errors.New("workspace is deleted")
	ErrDocumentDeleted        = errors.New("document is deleted")
	ErrDocumentConflict       = errors.New("document revision conflict")
	ErrInvalidRole            = errors.New("invalid workspace role")
	ErrInvalidVersionType     = errors.New("invalid document version type")
	ErrRefreshTokenInactive   = errors.New("refresh token is not active")
	ErrRefreshTokenExpired    = errors.New("refresh token is expired")
	ErrReplacementTokenID     = errors.New("replacement token id is required")
)
