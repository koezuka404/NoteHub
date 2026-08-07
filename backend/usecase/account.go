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

const deletedAccountPasswordHash = "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro4llC/.og/at2.uheWG/igi"

type IAccountWebSocketNotifier interface {
	NotifyAccountSuspended(userID uuid.UUID, suspendedAt string) error
	NotifyAccountDeleted(userID uuid.UUID, deletedAt string) error
	NotifyWorkspaceHostSuspended(workspaceID, hostUserID uuid.UUID, suspendedAt string) error
	NotifyWorkspaceHostDeleted(workspaceID, hostUserID uuid.UUID, deletedAt string) error
}

type IAccountUsecase interface {
	SuspendAccount(ctx context.Context, input SuspendAccountInput) (*SuspendAccountOutput, error)
	ReactivateAccount(ctx context.Context, input ReactivateAccountInput) (*ReactivateAccountOutput, error)
	DeleteAccount(ctx context.Context, input DeleteAccountInput) (*DeleteAccountOutput, error)
}

type AccountUseCase struct {
	users         repository.UserRepository
	refreshTokens repository.RefreshTokenRepository
	members       repository.WorkspaceMemberRepository
	workspaces    repository.WorkspaceRepository
	auditLogs     repository.AuditLogRepository
	transactions  repository.TransactionManager
	access        IAccessCheck
	notifier      IAccountWebSocketNotifier
	now           func() time.Time
}

func NewAccountUseCase(
	users repository.UserRepository,
	refreshTokens repository.RefreshTokenRepository,
	members repository.WorkspaceMemberRepository,
	workspaces repository.WorkspaceRepository,
	auditLogs repository.AuditLogRepository,
	transactions repository.TransactionManager,
	access IAccessCheck,
	notifier IAccountWebSocketNotifier,
) *AccountUseCase {
	return &AccountUseCase{
		users:         users,
		refreshTokens: refreshTokens,
		members:       members,
		workspaces:    workspaces,
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

func (uc *AccountUseCase) disconnectHostedWorkspaces(
	ctx context.Context,
	hostUserID uuid.UUID,
	notify func(workspaceID uuid.UUID) error,
) error {
	if uc.workspaces == nil || uc.notifier == nil {
		return nil
	}
	workspaces, err := uc.workspaces.FindByHostID(ctx, hostUserID)
	if err != nil {
		return fmt.Errorf("find hosted workspaces: %w", err)
	}
	for _, workspace := range workspaces {
		if err := notify(workspace.ID); err != nil {
			return err
		}
	}
	return nil
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
		if err := uc.disconnectHostedWorkspaces(ctx, output.UserID, func(workspaceID uuid.UUID) error {
			return uc.notifier.NotifyWorkspaceHostSuspended(workspaceID, output.UserID, output.SuspendedAt)
		}); err != nil {
			return nil, err
		}
	}
	return output, nil
}

type ReactivateAccountInput struct {
	UserID       uuid.UUID
	WorkspaceID  uuid.UUID
	TargetUserID uuid.UUID
	IPAddress    string
}

type ReactivateAccountOutput struct {
	UserID uuid.UUID
	Status string
}

func (uc *AccountUseCase) ReactivateAccount(ctx context.Context, input ReactivateAccountInput) (*ReactivateAccountOutput, error) {
	if err := uc.requireHost(ctx, input.UserID, input.WorkspaceID); err != nil {
		return nil, err
	}
	if input.TargetUserID == input.UserID {
		return nil, ErrCannotReactivateSelf
	}

	_, found, err := uc.members.FindByWorkspaceAndUser(ctx, input.WorkspaceID, input.TargetUserID)
	if err != nil {
		return nil, fmt.Errorf("find target workspace member: %w", err)
	}
	if !found {
		return nil, ErrMemberNotFound
	}

	now := uc.currentTime()
	var output *ReactivateAccountOutput
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
		if err := target.Reactivate(now); err != nil {
			if errors.Is(err, entity.ErrUserNotSuspended) {
				return ErrAccountNotSuspended
			}
			return fmt.Errorf("reactivate user: %w", err)
		}
		if err := uc.users.Update(txCtx, target); err != nil {
			return fmt.Errorf("update reactivated user: %w", err)
		}

		audit, err := entity.NewAuditLog(&input.UserID, "ACCOUNT_REACTIVATED", "user", &target.ID, nil, now)
		if err != nil {
			return fmt.Errorf("create audit log entity: %w", err)
		}
		audit.IPAddress = input.IPAddress
		if err := uc.auditLogs.Create(txCtx, &audit); err != nil {
			return fmt.Errorf("save audit log: %w", err)
		}

		output = &ReactivateAccountOutput{
			UserID: target.ID,
			Status: string(target.Status),
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return output, nil
}

type DeleteAccountInput struct {
	UserID       uuid.UUID
	WorkspaceID  uuid.UUID
	TargetUserID uuid.UUID
	IPAddress    string
}

type DeleteAccountOutput struct {
	UserID    uuid.UUID
	Status    string
	DeletedAt string
}

func deletedUserEmail(userID uuid.UUID) string {
	return fmt.Sprintf("deleted+%s@notehub.invalid", userID.String())
}

func (uc *AccountUseCase) DeleteAccount(ctx context.Context, input DeleteAccountInput) (*DeleteAccountOutput, error) {
	if err := uc.requireHost(ctx, input.UserID, input.WorkspaceID); err != nil {
		return nil, err
	}
	if input.TargetUserID == input.UserID {
		return nil, ErrCannotDeleteSelf
	}

	_, found, err := uc.members.FindByWorkspaceAndUser(ctx, input.WorkspaceID, input.TargetUserID)
	if err != nil {
		return nil, fmt.Errorf("find target workspace member: %w", err)
	}
	if !found {
		return nil, ErrMemberNotFound
	}

	now := uc.currentTime()
	var output *DeleteAccountOutput
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
		if err := target.LogicalDelete(deletedUserEmail(target.ID), deletedAccountPasswordHash, now); err != nil {
			if errors.Is(err, entity.ErrUserNotSuspended) {
				return ErrAccountNotSuspended
			}
			return fmt.Errorf("logical delete user: %w", err)
		}
		if err := uc.users.Update(txCtx, target); err != nil {
			return fmt.Errorf("update deleted user: %w", err)
		}
		if err := uc.refreshTokens.RevokeAllByUserID(txCtx, input.TargetUserID, now); err != nil {
			return fmt.Errorf("revoke refresh tokens: %w", err)
		}

		audit, err := entity.NewAuditLog(&input.UserID, "ACCOUNT_DELETED", "user", &target.ID, nil, now)
		if err != nil {
			return fmt.Errorf("create audit log entity: %w", err)
		}
		audit.IPAddress = input.IPAddress
		if err := uc.auditLogs.Create(txCtx, &audit); err != nil {
			return fmt.Errorf("save audit log: %w", err)
		}

		output = &DeleteAccountOutput{
			UserID:    target.ID,
			Status:    string(target.Status),
			DeletedAt: target.DeletedAt.Format(timeFormat),
		}
		return nil
	}); err != nil {
		return nil, err
	}

	if uc.notifier != nil {
		if err := uc.notifier.NotifyAccountDeleted(output.UserID, output.DeletedAt); err != nil {
			return nil, fmt.Errorf("notify account deleted: %w", err)
		}
		if err := uc.disconnectHostedWorkspaces(ctx, output.UserID, func(workspaceID uuid.UUID) error {
			return uc.notifier.NotifyWorkspaceHostDeleted(workspaceID, output.UserID, output.DeletedAt)
		}); err != nil {
			return nil, err
		}
	}
	return output, nil
}
