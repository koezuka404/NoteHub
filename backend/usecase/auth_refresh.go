package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/koezuka404/notehub/entity"
)

type RefreshInput struct {
	RefreshToken string
	IPAddress    string
	UserAgent    string
}

type RefreshOutput struct {
	AccessToken  string
	RefreshToken string
	CSRFToken    string
	TokenType    string
	ExpiresAt    string
}

func (uc *AuthUseCase) Refresh(ctx context.Context, input RefreshInput) (*RefreshOutput, error) {
	if strings.TrimSpace(input.RefreshToken) == "" {
		return nil, ErrRefreshTokenRequired
	}
	now := uc.now().UTC()
	var output *RefreshOutput
	var reused bool

	err := uc.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		current, found, err := uc.refreshTokens.FindByHashForUpdate(txCtx, uc.tokenHashes.Hash(input.RefreshToken))
		if err != nil {
			return fmt.Errorf("find refresh token: %w", err)
		}
		if !found {
			return ErrRefreshTokenInvalid
		}
		if current.IsExpired(now) {
			return ErrRefreshTokenExpired
		}
		if current.Status != entity.RefreshTokenStatusActive || current.RotatedAt != nil || current.RevokedAt != nil {
			if err := uc.refreshTokens.RevokeFamily(txCtx, current.FamilyID, now); err != nil {
				return fmt.Errorf("revoke reused token family: %w", err)
			}
			if err := uc.users.IncrementAuthVersion(txCtx, current.UserID, now); err != nil {
				return fmt.Errorf("increment auth version: %w", err)
			}
			reused = true
			return nil
		}

		user, found, err := uc.users.FindByID(txCtx, current.UserID)
		if err != nil {
			return fmt.Errorf("find refresh token user: %w", err)
		}
		if !found {
			return ErrRefreshTokenInvalid
		}
		if user.IsDeleted() {
			return ErrAccountDeleted
		}
		if user.IsSuspended() {
			return ErrAccountSuspended
		}
		if !user.CanAuthenticate() {
			return ErrRefreshTokenRevoked
		}

		rawNext, err := uc.randomTokens.GenerateRefreshToken()
		if err != nil {
			return fmt.Errorf("generate next refresh token: %w", err)
		}
		next, err := entity.NewRefreshToken(user.ID, uc.tokenHashes.Hash(rawNext), current.FamilyID, now.Add(uc.refreshTokenTTL), now)
		if err != nil {
			return fmt.Errorf("create next refresh token: %w", err)
		}
		next.IPAddress = input.IPAddress
		next.UserAgent = input.UserAgent
		if err := uc.refreshTokens.Create(txCtx, &next); err != nil {
			return fmt.Errorf("save next refresh token: %w", err)
		}
		if err := current.Rotate(next.ID, now); err != nil {
			return ErrRefreshTokenRevoked
		}
		if err := uc.refreshTokens.Update(txCtx, current); err != nil {
			return fmt.Errorf("rotate current refresh token: %w", err)
		}
		access, expiresAt, err := uc.accessTokens.GenerateAccessToken(user.ID, user.AuthVersion, now)
		if err != nil {
			return fmt.Errorf("generate access token: %w", err)
		}
		csrfToken, err := uc.randomTokens.GenerateCSRFToken()
		if err != nil {
			return fmt.Errorf("generate csrf token: %w", err)
		}
		output = &RefreshOutput{AccessToken: access, RefreshToken: rawNext, CSRFToken: csrfToken, TokenType: "Bearer", ExpiresAt: expiresAt.Format(timeFormat)}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if reused {
		return nil, ErrRefreshTokenReused
	}
	if output == nil {
		return nil, fmt.Errorf("refresh token transaction completed without output")
	}
	return output, nil
}
