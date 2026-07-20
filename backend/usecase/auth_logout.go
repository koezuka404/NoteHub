package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
)

type LogoutInput struct {
	UserID         uuid.UUID
	AccessTokenJTI uuid.UUID
	AccessTokenExp time.Time
	RefreshToken   string
	IPAddress      string
	UserAgent      string
	CSRFValidated  bool
}

type LogoutOutput struct{}

func (uc *AuthUseCase) Logout(ctx context.Context, input LogoutInput) (*LogoutOutput, error) {
	if input.UserID == uuid.Nil || input.AccessTokenJTI == uuid.Nil || input.AccessTokenExp.IsZero() {
		return nil, ErrAccessTokenInvalid
	}
	if !input.CSRFValidated {
		return nil, ErrCSRFTokenInvalid
	}

	now := uc.now().UTC()
	rawRefreshToken := strings.TrimSpace(input.RefreshToken)

	err := uc.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		if rawRefreshToken != "" {
			token, found, err := uc.refreshTokens.FindByHashForUpdate(txCtx, uc.tokenHashes.Hash(rawRefreshToken))
			if err != nil {
				return fmt.Errorf("find refresh token for logout: %w", err)
			}
			if found {
				if token.UserID != input.UserID {
					return ErrTokenOwnerMismatch
				}
				if token.Status == entity.RefreshTokenStatusActive || token.Status == entity.RefreshTokenStatusRotated {
					if err := token.Revoke(now); err != nil {
						return fmt.Errorf("revoke refresh token: %w", err)
					}
					if err := uc.refreshTokens.Update(txCtx, token); err != nil {
						return fmt.Errorf("save revoked refresh token: %w", err)
					}
				}
			}
		}

		if uc.auditLogs != nil {
			audit, err := entity.NewAuditLog(&input.UserID, "LOGOUT", "user", &input.UserID, nil, now)
			if err != nil {
				return fmt.Errorf("create logout audit log: %w", err)
			}
			audit.IPAddress = input.IPAddress
			audit.UserAgent = input.UserAgent
			if err := uc.auditLogs.Create(txCtx, &audit); err != nil {
				return fmt.Errorf("save logout audit log: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrTokenOwnerMismatch) {
			return nil, err
		}
		return nil, fmt.Errorf("logout transaction: %w", err)
	}

	ttl := input.AccessTokenExp.Sub(now)
	if ttl > 0 {
		if err := uc.revokedAccess.Revoke(ctx, input.AccessTokenJTI, ttl); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrAuthServiceUnavailable, err)
		}
	}
	return &LogoutOutput{}, nil
}
