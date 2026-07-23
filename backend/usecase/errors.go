package usecase

import "errors"

var (
	ErrValidation             = errors.New("validation error")
	ErrPasswordInvalid        = errors.New("password invalid")
	ErrEmailAlreadyExists     = errors.New("email already exists")
	ErrInvalidCredentials     = errors.New("invalid credentials")
	ErrAccountSuspended       = errors.New("account suspended")
	ErrAccountDeleted         = errors.New("account deleted")
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
)
