package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
)

type LoginInput struct {
	Email, Password, IPAddress, UserAgent string
}

type LoginUserOutput struct {
	ID          uuid.UUID
	Name, Email string
	Status      entity.UserStatus
}

type LoginOutput struct {
	User         LoginUserOutput
	AccessToken  string
	RefreshToken string
	CSRFToken    string
	TokenType    string
	ExpiresAt    string
}

func (uc *AuthUseCase) Login(ctx context.Context, input LoginInput) (*LoginOutput, error) {
	if !validateLoginInput(input) {
		return nil, ErrValidation
	}
	user, found, err := uc.users.FindByEmail(ctx, normalizeEmail(input.Email))
	if err != nil {
		return nil, fmt.Errorf("find user by email: %w", err)
	}
	if !found {
		return nil, ErrInvalidCredentials
	}
	if user.IsDeleted() {
		return nil, ErrAccountDeleted
	}
	if user.IsSuspended() {
		return nil, ErrAccountSuspended
	}
	if !user.CanAuthenticate() {
		return nil, ErrInvalidCredentials
	}
	if err := uc.passwords.Compare(user.PasswordHash, input.Password); err != nil {
		return nil, ErrInvalidCredentials
	}

	now := uc.now().UTC()
	accessToken, expiresAt, err := uc.accessTokens.GenerateAccessToken(user.ID, user.AuthVersion, now)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}
	rawRefresh, err := uc.randomTokens.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}
	refresh, err := entity.NewRefreshToken(user.ID, uc.tokenHashes.Hash(rawRefresh), uuid.New(), now.Add(uc.refreshTokenTTL), now)
	if err != nil {
		return nil, fmt.Errorf("create refresh token entity: %w", err)
	}
	refresh.IPAddress = input.IPAddress
	refresh.UserAgent = input.UserAgent
	if err := uc.refreshTokens.Create(ctx, &refresh); err != nil {
		return nil, fmt.Errorf("save refresh token: %w", err)
	}
	csrfToken, err := uc.randomTokens.GenerateCSRFToken()
	if err != nil {
		return nil, fmt.Errorf("generate csrf token: %w", err)
	}

	return &LoginOutput{
		User:        LoginUserOutput{ID: user.ID, Name: user.Name, Email: user.Email, Status: user.Status},
		AccessToken: accessToken, RefreshToken: rawRefresh, CSRFToken: csrfToken, TokenType: "Bearer", ExpiresAt: expiresAt.Format(timeFormat),
	}, nil
}
