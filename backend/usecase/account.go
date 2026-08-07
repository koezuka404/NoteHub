package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
	"github.com/koezuka404/notehub/repository"
)

type IAccountWebSocketNotifier interface {
	NotifyAccountSuspended(userID uuid.UUID, suspendedAt string) error
}

type IAccountUsecase interface {
	SuspendAccount(ctx context.Context, input SuspendAccountInput) (*SuspendAccountOutput, error)
}

type AccountUseCase struct {
	users        repository.UserRepository
	refreshTokens repository.RefreshTokenRepository
	members      repository.WorkspaceMemberRepository
	auditLogs    repository.AuditLogRepository
	transactions repository.TransactionManager
	access       IAccessCheck
	notifier     IAccountWebSocketNotifier
	now          func() time.Time
}

func NewAccountUseCase(
	users repository.UserRepository,
	refreshTokens repository.RefreshTokenRepository,
	members repository.WorkspaceMemberRepository,
	auditLogs repository.AuditLogRepository,
	transactions repository.TransactionManager,
	access IAccessCheck,
	notifier IAccountWebSocketNotifier,
) *AccountUseCase {
	return &AccountUseCase{
		users:         users,
		refreshTokens: refreshTokens,
		members:       members,
		auditLogs:     auditLogs,
		transactions:  transactions,
		access:        access,
		notifier:      notifier,
		now:           time.Now,
	}
}

type SuspendAccountInput struct {
	UserID       uuid.UUID
	WorkspaceID  uuid.UUID
	TargetUserID uuid.UUID
	IPAddress    string
}

type SuspendAccountOutput struct {
	UserID      uuid.UUID
	Status      string
	SuspendedAt string
}

func (uc *AccountUseCase) currentTime() time.Time {
	return uc.now().UTC()
}

func (uc *AccountUseCase) requireHost(ctx context.Context, userID, workspaceID uuid.UUID) error {
	role := entity.WorkspaceRoleHost
	_, err := uc.access.CheckWorkspaceAccess(ctx, CheckWorkspaceAccessInput{
		UserID: userID, WorkspaceID: workspaceID, RequiredRole: &role,
	})
	return err
}

func (uc *AccountUseCase) SuspendAccount(ctx context.Context, input SuspendAccountInput) (*SuspendAccountOutput, error) {
	if err := uc.requireHost(ctx, input.UserID, input.WorkspaceID); err != nil {
		return nil, err
	}
	if input.TargetUserID == input.UserID {
		return nil, ErrCannotSuspendSelf
	}

	targetMember, found, err := uc.members.FindByWorkspaceAndUser(ctx, input.WorkspaceID, input.TargetUserID)
	if err != nil {
		return nil, fmt.Errorf("find target workspace member: %w", err)
	}
	if !found {
		return nil, ErrMemberNotFound
	}
	if targetMember.IsHost() {
		return nil, ErrCannotSuspendHost
	}

	now := uc.currentTime()
	var output *SuspendAccountOutput
	if err := uc.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		target, found, err := uc.users.FindByIDForUpdate(txCtx, input.TargetUserID)
		if err != nil {
			return fmt.Errorf("find target user for update: %w", err)
		}
		if !found {
			return ErrTargetUserNotFound
		}
		if target.IsDeleted() {
			return ErrAccountDeleted
		}
		if err := target.Suspend(now); err != nil {
			if errors.Is(err, entity.ErrInvalidStateTransition) {
				return ErrAccountAlreadySuspended
			}
			return fmt.Errorf("suspend user: %w", err)
		}
		if err := uc.users.Update(txCtx, target); err != nil {
			return fmt.Errorf("update suspended user: %w", err)
		}
		if err := uc.refreshTokens.RevokeAllByUserID(txCtx, input.TargetUserID, now); err != nil {
			return fmt.Errorf("revoke refresh tokens: %w", err)
		}

		audit, err := entity.NewAuditLog(&input.UserID, "ACCOUNT_SUSPENDED", "user", &target.ID, nil, now)
		if err != nil {
			return fmt.Errorf("create audit log entity: %w", err)
		}
		audit.IPAddress = input.IPAddress
		if err := uc.auditLogs.Create(txCtx, &audit); err != nil {
			return fmt.Errorf("save audit log: %w", err)
		}

		output = &SuspendAccountOutput{
			UserID:      target.ID,
			Status:      string(target.Status),
			SuspendedAt: target.SuspendedAt.Format(timeFormat),
		}
		return nil
	}); err != nil {
		return nil, err
	}

	if uc.notifier != nil {
		if err := uc.notifier.NotifyAccountSuspended(output.UserID, output.SuspendedAt); err != nil {
			return nil, fmt.Errorf("notify account suspended: %w", err)
		}
	}
	return output, nil
}
