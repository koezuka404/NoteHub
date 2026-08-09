package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
	"github.com/koezuka404/notehub/repository"
)

type errWorkspaceRepo struct {
	mockWorkspaceRepo
	findErr error
}

func (r *errWorkspaceRepo) FindByHostID(context.Context, uuid.UUID) ([]entity.Workspace, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	return r.mockWorkspaceRepo.FindByHostID(context.Background(), uuid.Nil)
}

type errAccountNotifier struct {
	mockAccountNotifier
	suspendErr       error
	deleteErr        error
	hostSuspendErr   error
	hostDeleteErr    error
}

func (n *errAccountNotifier) NotifyAccountSuspended(uuid.UUID, string) error {
	if n.suspendErr != nil {
		return n.suspendErr
	}
	return n.mockAccountNotifier.NotifyAccountSuspended(uuid.Nil, "")
}

func (n *errAccountNotifier) NotifyAccountDeleted(uuid.UUID, string) error {
	if n.deleteErr != nil {
		return n.deleteErr
	}
	return n.mockAccountNotifier.NotifyAccountDeleted(uuid.Nil, "")
}

func (n *errAccountNotifier) NotifyWorkspaceHostSuspended(uuid.UUID, uuid.UUID, string) error {
	if n.hostSuspendErr != nil {
		return n.hostSuspendErr
	}
	return n.mockAccountNotifier.NotifyWorkspaceHostSuspended(uuid.Nil, uuid.Nil, "")
}

func (n *errAccountNotifier) NotifyWorkspaceHostDeleted(uuid.UUID, uuid.UUID, string) error {
	if n.hostDeleteErr != nil {
		return n.hostDeleteErr
	}
	return n.mockAccountNotifier.NotifyWorkspaceHostDeleted(uuid.Nil, uuid.Nil, "")
}

func newExtendedAccountUseCase(
	access *mockAccessCheck,
	members *mockMemberRepo,
	users *mockAccountUserRepo,
	workspaces repository.WorkspaceRepository,
	refresh *trackingRefreshTokenRepo,
	notifier IAccountWebSocketNotifier,
) *AccountUseCase {
	uc := NewAccountUseCase(users, refresh, members, workspaces, &mockAuditLogRepo{}, &mockTransactionManager{}, access, notifier)
	uc.now = accountTestNow
	return uc
}

func TestSuspendAccount_MemberLookupError(t *testing.T) {
	hostID, workspaceID, targetID := testAccountIDs()
	members := &mockMemberRepo{err: fmt.Errorf("lookup failed")}
	uc := newAccountUseCase(&mockAccessCheck{}, members, &mockAccountUserRepo{}, &mockWorkspaceRepo{}, &trackingRefreshTokenRepo{}, &mockAccountNotifier{})

	_, err := uc.SuspendAccount(context.Background(), SuspendAccountInput{
		UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID,
	})
	if err == nil {
		t.Fatal("expected member lookup error")
	}
}

func TestSuspendAccount_MemberNotFound(t *testing.T) {
	hostID, workspaceID, targetID := testAccountIDs()
	uc := newAccountUseCase(&mockAccessCheck{}, &mockMemberRepo{}, &mockAccountUserRepo{}, &mockWorkspaceRepo{}, &trackingRefreshTokenRepo{}, &mockAccountNotifier{})

	_, err := uc.SuspendAccount(context.Background(), SuspendAccountInput{
		UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID,
	})
	if !errors.Is(err, ErrMemberNotFound) {
		t.Fatalf("expected ErrMemberNotFound, got %v", err)
	}
}

func TestSuspendAccount_TargetNotFound(t *testing.T) {
	hostID, workspaceID, targetID := testAccountIDs()
	uc := newAccountUseCase(
		&mockAccessCheck{},
		&mockMemberRepo{found: true, member: &entity.WorkspaceMember{Role: entity.WorkspaceRoleMember, UserID: targetID}},
		&mockAccountUserRepo{},
		&mockWorkspaceRepo{},
		&trackingRefreshTokenRepo{},
		&mockAccountNotifier{},
	)

	_, err := uc.SuspendAccount(context.Background(), SuspendAccountInput{
		UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID,
	})
	if !errors.Is(err, ErrTargetUserNotFound) {
		t.Fatalf("expected ErrTargetUserNotFound, got %v", err)
	}
}

func TestSuspendAccount_TargetDeleted(t *testing.T) {
	hostID, workspaceID, targetID := testAccountIDs()
	deletedAt := accountTestNow()
	users := &mockAccountUserRepo{users: map[uuid.UUID]*entity.User{
		targetID: {ID: targetID, Status: entity.UserStatusDeleted, DeletedAt: &deletedAt},
	}}
	uc := newAccountUseCase(
		&mockAccessCheck{},
		&mockMemberRepo{found: true, member: &entity.WorkspaceMember{Role: entity.WorkspaceRoleMember, UserID: targetID}},
		users,
		&mockWorkspaceRepo{},
		&trackingRefreshTokenRepo{},
		&mockAccountNotifier{},
	)

	_, err := uc.SuspendAccount(context.Background(), SuspendAccountInput{
		UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID,
	})
	if !errors.Is(err, ErrAccountDeleted) {
		t.Fatalf("expected ErrAccountDeleted, got %v", err)
	}
}

func TestSuspendAccount_NotifierError(t *testing.T) {
	hostID, workspaceID, targetID := testAccountIDs()
	users := &mockAccountUserRepo{users: map[uuid.UUID]*entity.User{
		targetID: memberUser(targetID, entity.UserStatusActive),
	}}
	notifier := &errAccountNotifier{suspendErr: fmt.Errorf("notify failed")}
	uc := newExtendedAccountUseCase(
		&mockAccessCheck{},
		&mockMemberRepo{found: true, member: &entity.WorkspaceMember{Role: entity.WorkspaceRoleMember, UserID: targetID}},
		users,
		&mockWorkspaceRepo{},
		&trackingRefreshTokenRepo{},
		notifier,
	)

	_, err := uc.SuspendAccount(context.Background(), SuspendAccountInput{
		UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID,
	})
	if err == nil {
		t.Fatal("expected notifier error")
	}
}

func TestSuspendAccount_DisconnectHostedWorkspacesFindError(t *testing.T) {
	hostID, workspaceID, targetID := testAccountIDs()
	users := &mockAccountUserRepo{users: map[uuid.UUID]*entity.User{
		targetID: memberUser(targetID, entity.UserStatusActive),
	}}
	workspaces := &errWorkspaceRepo{findErr: fmt.Errorf("find failed")}
	uc := newExtendedAccountUseCase(
		&mockAccessCheck{},
		&mockMemberRepo{found: true, member: &entity.WorkspaceMember{Role: entity.WorkspaceRoleMember, UserID: targetID}},
		users,
		workspaces,
		&trackingRefreshTokenRepo{},
		&mockAccountNotifier{},
	)

	_, err := uc.SuspendAccount(context.Background(), SuspendAccountInput{
		UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID,
	})
	if err == nil {
		t.Fatal("expected hosted workspace find error")
	}
}

func TestSuspendAccount_NilWorkspacesAndNotifier(t *testing.T) {
	hostID, workspaceID, targetID := testAccountIDs()
	users := &mockAccountUserRepo{users: map[uuid.UUID]*entity.User{
		targetID: memberUser(targetID, entity.UserStatusActive),
	}}
	uc := NewAccountUseCase(
		users,
		&trackingRefreshTokenRepo{},
		&mockMemberRepo{found: true, member: &entity.WorkspaceMember{Role: entity.WorkspaceRoleMember, UserID: targetID}},
		nil,
		&mockAuditLogRepo{},
		&mockTransactionManager{},
		&mockAccessCheck{},
		nil,
	)
	uc.now = accountTestNow

	out, err := uc.SuspendAccount(context.Background(), SuspendAccountInput{
		UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID,
	})
	if err != nil {
		t.Fatalf("SuspendAccount: %v", err)
	}
	if out.Status != string(entity.UserStatusSuspended) {
		t.Fatalf("status = %q", out.Status)
	}
}

func TestReactivateAccount_CannotReactivateSelf(t *testing.T) {
	hostID, workspaceID, _ := testAccountIDs()
	uc := newAccountUseCase(&mockAccessCheck{}, &mockMemberRepo{found: true}, &mockAccountUserRepo{}, &mockWorkspaceRepo{}, &trackingRefreshTokenRepo{}, &mockAccountNotifier{})

	_, err := uc.ReactivateAccount(context.Background(), ReactivateAccountInput{
		UserID: hostID, WorkspaceID: workspaceID, TargetUserID: hostID,
	})
	if !errors.Is(err, ErrCannotReactivateSelf) {
		t.Fatalf("expected ErrCannotReactivateSelf, got %v", err)
	}
}

func TestReactivateAccount_MemberLookupError(t *testing.T) {
	hostID, workspaceID, targetID := testAccountIDs()
	uc := newAccountUseCase(
		&mockAccessCheck{},
		&mockMemberRepo{err: fmt.Errorf("lookup failed")},
		&mockAccountUserRepo{},
		&mockWorkspaceRepo{},
		&trackingRefreshTokenRepo{},
		&mockAccountNotifier{},
	)

	_, err := uc.ReactivateAccount(context.Background(), ReactivateAccountInput{
		UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID,
	})
	if err == nil {
		t.Fatal("expected member lookup error")
	}
}

func TestReactivateAccount_TargetDeleted(t *testing.T) {
	hostID, workspaceID, targetID := testAccountIDs()
	deletedAt := accountTestNow()
	user := memberUser(targetID, entity.UserStatusActive)
	user.Status = entity.UserStatusDeleted
	user.DeletedAt = &deletedAt
	users := &mockAccountUserRepo{users: map[uuid.UUID]*entity.User{targetID: user}}
	uc := newAccountUseCase(
		&mockAccessCheck{},
		&mockMemberRepo{found: true, member: &entity.WorkspaceMember{Role: entity.WorkspaceRoleMember, UserID: targetID}},
		users,
		&mockWorkspaceRepo{},
		&trackingRefreshTokenRepo{},
		&mockAccountNotifier{},
	)

	_, err := uc.ReactivateAccount(context.Background(), ReactivateAccountInput{
		UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID,
	})
	if !errors.Is(err, ErrAccountDeleted) {
		t.Fatalf("expected ErrAccountDeleted, got %v", err)
	}
}

func TestDeleteAccount_AlreadyDeleted(t *testing.T) {
	hostID, workspaceID, targetID := testAccountIDs()
	deletedAt := accountTestNow()
	user := memberUser(targetID, entity.UserStatusActive)
	user.Status = entity.UserStatusDeleted
	user.DeletedAt = &deletedAt
	users := &mockAccountUserRepo{users: map[uuid.UUID]*entity.User{targetID: user}}
	uc := newAccountUseCase(
		&mockAccessCheck{},
		&mockMemberRepo{found: true, member: &entity.WorkspaceMember{Role: entity.WorkspaceRoleMember, UserID: targetID}},
		users,
		&mockWorkspaceRepo{},
		&trackingRefreshTokenRepo{},
		&mockAccountNotifier{},
	)

	_, err := uc.DeleteAccount(context.Background(), DeleteAccountInput{
		UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID,
	})
	if !errors.Is(err, ErrAccountDeleted) {
		t.Fatalf("expected ErrAccountDeleted, got %v", err)
	}
}

func TestDeleteAccount_NotifierError(t *testing.T) {
	hostID, workspaceID, targetID := testAccountIDs()
	user := memberUser(targetID, entity.UserStatusActive)
	_ = user.Suspend(accountTestNow())
	notifier := &errAccountNotifier{deleteErr: fmt.Errorf("notify failed")}
	uc := newExtendedAccountUseCase(
		&mockAccessCheck{},
		&mockMemberRepo{found: true, member: &entity.WorkspaceMember{Role: entity.WorkspaceRoleMember, UserID: targetID}},
		&mockAccountUserRepo{users: map[uuid.UUID]*entity.User{targetID: user}},
		&mockWorkspaceRepo{},
		&trackingRefreshTokenRepo{},
		notifier,
	)

	_, err := uc.DeleteAccount(context.Background(), DeleteAccountInput{
		UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID,
	})
	if err == nil {
		t.Fatal("expected notifier error")
	}
}

func TestDeleteAccount_HostDeletedNotifyError(t *testing.T) {
	hostID, workspaceID, targetID := testAccountIDs()
	hostedWorkspaceID := uuid.New()
	user := memberUser(targetID, entity.UserStatusActive)
	_ = user.Suspend(accountTestNow())
	notifier := &errAccountNotifier{hostDeleteErr: fmt.Errorf("host notify failed")}
	uc := newExtendedAccountUseCase(
		&mockAccessCheck{},
		&mockMemberRepo{found: true, member: &entity.WorkspaceMember{Role: entity.WorkspaceRoleMember, UserID: targetID}},
		&mockAccountUserRepo{users: map[uuid.UUID]*entity.User{targetID: user}},
		&mockWorkspaceRepo{byHostID: []entity.Workspace{{ID: hostedWorkspaceID, HostID: targetID}}},
		&trackingRefreshTokenRepo{},
		notifier,
	)

	_, err := uc.DeleteAccount(context.Background(), DeleteAccountInput{
		UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID,
	})
	if err == nil {
		t.Fatal("expected host deleted notify error")
	}
}

func TestDeleteAccount_NilNotifier(t *testing.T) {
	hostID, workspaceID, targetID := testAccountIDs()
	user := memberUser(targetID, entity.UserStatusActive)
	_ = user.Suspend(accountTestNow())
	uc := NewAccountUseCase(
		&mockAccountUserRepo{users: map[uuid.UUID]*entity.User{targetID: user}},
		&trackingRefreshTokenRepo{},
		&mockMemberRepo{found: true, member: &entity.WorkspaceMember{Role: entity.WorkspaceRoleMember, UserID: targetID}},
		nil,
		&mockAuditLogRepo{},
		&mockTransactionManager{},
		&mockAccessCheck{},
		nil,
	)
	uc.now = accountTestNow

	out, err := uc.DeleteAccount(context.Background(), DeleteAccountInput{
		UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID,
	})
	if err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}
	if out.Status != string(entity.UserStatusDeleted) {
		t.Fatalf("status = %q", out.Status)
	}
}

func TestSuspendAccount_HostSuspendedNotifyError(t *testing.T) {
	hostID, workspaceID, targetID := testAccountIDs()
	hostedWorkspaceID := uuid.New()
	users := &mockAccountUserRepo{users: map[uuid.UUID]*entity.User{
		targetID: memberUser(targetID, entity.UserStatusActive),
	}}
	notifier := &errAccountNotifier{hostSuspendErr: fmt.Errorf("host notify failed")}
	uc := newExtendedAccountUseCase(
		&mockAccessCheck{},
		&mockMemberRepo{found: true, member: &entity.WorkspaceMember{Role: entity.WorkspaceRoleMember, UserID: targetID}},
		users,
		&mockWorkspaceRepo{byHostID: []entity.Workspace{{ID: hostedWorkspaceID, HostID: targetID}}},
		&trackingRefreshTokenRepo{},
		notifier,
	)

	_, err := uc.SuspendAccount(context.Background(), SuspendAccountInput{
		UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID,
	})
	if err == nil {
		t.Fatal("expected host suspended notify error")
	}
}
