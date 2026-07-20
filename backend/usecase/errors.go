package usecase

import "errors"

var (
	ErrValidation           = errors.New("validation error")
	ErrEmailAlreadyExists   = errors.New("email already exists")
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrAccountSuspended     = errors.New("account suspended")
	ErrAccountDeleted       = errors.New("account deleted")
	ErrRefreshTokenRequired = errors.New("refresh token is required")
	ErrRefreshTokenInvalid  = errors.New("refresh token is invalid")
	ErrRefreshTokenExpired  = errors.New("refresh token is expired")
	ErrRefreshTokenRevoked  = errors.New("refresh token is revoked")
	ErrRefreshTokenReused   = errors.New("refresh token was reused")
)
