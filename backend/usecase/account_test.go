package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
)

type mockAccessCheck struct {
	err error
}

func (m *mockAccessCheck) CheckWorkspaceAccess(context.Context, CheckWorkspaceAccessInput) (*WorkspaceAccessResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &WorkspaceAccessResult{}, nil
}

type mockMemberRepo struct {
	member *entity.WorkspaceMember
	found  bool
	err    error
}

func (m *mockMemberRepo) Create(context.Context, *entity.WorkspaceMember) error { return nil }
func (m *mockMemberRepo) FindByWorkspaceAndUser(context.Context, uuid.UUID, uuid.UUID) (*entity.WorkspaceMember, bool, error) {
	if m.err != nil {
		return nil, false, m.err
	}
	return m.member, m.found, nil
}
func (m *mockMemberRepo) FindByWorkspaceID(context.Context, uuid.UUID) ([]entity.WorkspaceMember, error) {
	return nil, nil
}
func (m *mockMemberRepo) Exists(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return false, nil
}
func (m *mockMemberRepo) Delete(context.Context, uuid.UUID, uuid.UUID) error { return nil }

type mockAccountUserRepo struct {
	users            map[uuid.UUID]*entity.User
	updated          *entity.User
	updateErr        error
	findForUpdateErr error
}

func (m *mockAccountUserRepo) Create(context.Context, *entity.User) error { return nil }
func (m *mockAccountUserRepo) FindByID(context.Context, uuid.UUID) (*entity.User, bool, error) {
	return nil, false, nil
}
func (m *mockAccountUserRepo) FindByIDForUpdate(_ context.Context, id uuid.UUID) (*entity.User, bool, error) {
	if m.findForUpdateErr != nil {
		return nil, false, m.findForUpdateErr
	}
	user, ok := m.users[id]
	if !ok {
		return nil, false, nil
	}
	copy := *user
	return &copy, true, nil
}
func (m *mockAccountUserRepo) FindByEmail(context.Context, string) (*entity.User, bool, error) {
	return nil, false, nil
}
func (m *mockAccountUserRepo) ExistsByEmail(context.Context, string) (bool, error) { return false, nil }
func (m *mockAccountUserRepo) Update(_ context.Context, user *entity.User) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.updated = user
	if m.users != nil {
		copy := *user
		m.users[user.ID] = &copy
	}
	return nil
}
func (m *mockAccountUserRepo) IncrementAuthVersion(context.Context, uuid.UUID, time.Time) error {
	return nil
}

type trackingRefreshTokenRepo struct {
	mockRefreshTokenRepo
	revokeCalled  bool
	revokedUserID uuid.UUID
}

func (m *trackingRefreshTokenRepo) RevokeAllByUserID(_ context.Context, userID uuid.UUID, _ time.Time) error {
	m.revokeCalled = true
	m.revokedUserID = userID
	return nil
}

type mockAccountNotifier struct {
	suspendedUserID          uuid.UUID
	suspendedAt              string
	deletedUserID            uuid.UUID
	deletedAt                string
	hostSuspendedWorkspaceID uuid.UUID
	hostDeletedWorkspaceID   uuid.UUID
}

func (m *mockAccountNotifier) NotifyAccountSuspended(userID uuid.UUID, suspendedAt string) error {
	m.suspendedUserID = userID
	m.suspendedAt = suspendedAt
	return nil
}

func (m *mockAccountNotifier) NotifyAccountDeleted(userID uuid.UUID, deletedAt string) error {
	m.deletedUserID = userID
	m.deletedAt = deletedAt
	return nil
}

func (m *mockAccountNotifier) NotifyWorkspaceHostSuspended(workspaceID, hostUserID uuid.UUID, suspendedAt string) error {
	m.hostSuspendedWorkspaceID = workspaceID
	return nil
}

func (m *mockAccountNotifier) NotifyWorkspaceHostDeleted(workspaceID, hostUserID uuid.UUID, deletedAt string) error {
	m.hostDeletedWorkspaceID = workspaceID
	return nil
}

type mockWorkspaceRepo struct {
	byHostID []entity.Workspace
}

func (m *mockWorkspaceRepo) Create(context.Context, *entity.Workspace) error { return nil }
func (m *mockWorkspaceRepo) FindByID(context.Context, uuid.UUID) (*entity.Workspace, bool, error) {
	return nil, false, nil
}
func (m *mockWorkspaceRepo) FindByIDForUpdate(context.Context, uuid.UUID) (*entity.Workspace, bool, error) {
	return nil, false, nil
}
func (m *mockWorkspaceRepo) FindByHostID(context.Context, uuid.UUID) ([]entity.Workspace, error) {
	return m.byHostID, nil
}
func (m *mockWorkspaceRepo) FindByUserID(context.Context, uuid.UUID) ([]entity.Workspace, error) {
	return nil, nil
}
func (m *mockWorkspaceRepo) Update(context.Context, *entity.Workspace) error { return nil }

func accountTestNow() time.Time {
	return time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
}

func newAccountUseCase(
	access *mockAccessCheck,
	members *mockMemberRepo,
	users *mockAccountUserRepo,
	workspaces *mockWorkspaceRepo,
	refresh *trackingRefreshTokenRepo,
	notifier *mockAccountNotifier,
) *AccountUseCase {
	uc := NewAccountUseCase(
		users,
		refresh,
		members,
		workspaces,
		&mockAuditLogRepo{},
		&mockTransactionManager{},
		access,
		notifier,
	)
	uc.now = accountTestNow
	return uc
}

func testAccountIDs() (hostID, workspaceID, targetID uuid.UUID) {
	hostID = uuid.New()
	workspaceID = uuid.New()
	targetID = uuid.New()
	return hostID, workspaceID, targetID
}

func memberUser(id uuid.UUID, status entity.UserStatus) *entity.User {
	return &entity.User{
		ID:           id,
		Name:         "Bob",
		Email:        "bob@example.com",
		PasswordHash: "hash",
		Status:       status,
		AuthVersion:  1,
		CreatedAt:    accountTestNow(),
		UpdatedAt:    accountTestNow(),
	}
}

func TestSuspendAccount_CannotSuspendSelf(t *testing.T) {
	hostID, workspaceID, _ := testAccountIDs()
	uc := newAccountUseCase(
		&mockAccessCheck{},
		&mockMemberRepo{found: true, member: &entity.WorkspaceMember{Role: entity.WorkspaceRoleMember}},
		&mockAccountUserRepo{},
		&mockWorkspaceRepo{},
		&trackingRefreshTokenRepo{},
		&mockAccountNotifier{},
	)

	_, err := uc.SuspendAccount(context.Background(), SuspendAccountInput{
		UserID: hostID, WorkspaceID: workspaceID, TargetUserID: hostID,
	})
	if !errors.Is(err, ErrCannotSuspendSelf) {
		t.Fatalf("expected ErrCannotSuspendSelf, got %v", err)
	}
}

func TestSuspendAccount_CannotSuspendHost(t *testing.T) {
	hostID, workspaceID, targetID := testAccountIDs()
	uc := newAccountUseCase(
		&mockAccessCheck{},
		&mockMemberRepo{found: true, member: &entity.WorkspaceMember{Role: entity.WorkspaceRoleHost, UserID: targetID}},
		&mockAccountUserRepo{},
		&mockWorkspaceRepo{},
		&trackingRefreshTokenRepo{},
		&mockAccountNotifier{},
	)

	_, err := uc.SuspendAccount(context.Background(), SuspendAccountInput{
		UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID,
	})
	if !errors.Is(err, ErrCannotSuspendHost) {
		t.Fatalf("expected ErrCannotSuspendHost, got %v", err)
	}
}

func TestSuspendAccount_NotHost(t *testing.T) {
	hostID, workspaceID, targetID := testAccountIDs()
	uc := newAccountUseCase(
		&mockAccessCheck{err: ErrHostPermissionRequired},
		&mockMemberRepo{},
		&mockAccountUserRepo{},
		&mockWorkspaceRepo{},
		&trackingRefreshTokenRepo{},
		&mockAccountNotifier{},
	)

	_, err := uc.SuspendAccount(context.Background(), SuspendAccountInput{
		UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID,
	})
	if !errors.Is(err, ErrHostPermissionRequired) {
		t.Fatalf("expected ErrHostPermissionRequired, got %v", err)
	}
}

func TestSuspendAccount_AlreadySuspended(t *testing.T) {
	hostID, workspaceID, targetID := testAccountIDs()
	users := &mockAccountUserRepo{users: map[uuid.UUID]*entity.User{
		targetID: memberUser(targetID, entity.UserStatusSuspended),
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
	if !errors.Is(err, ErrAccountAlreadySuspended) {
		t.Fatalf("expected ErrAccountAlreadySuspended, got %v", err)
	}
}

func TestSuspendAccount_Success(t *testing.T) {
	hostID, workspaceID, targetID := testAccountIDs()
	users := &mockAccountUserRepo{users: map[uuid.UUID]*entity.User{
		targetID: memberUser(targetID, entity.UserStatusActive),
	}}
	refresh := &trackingRefreshTokenRepo{}
	notifier := &mockAccountNotifier{}
	uc := newAccountUseCase(
		&mockAccessCheck{},
		&mockMemberRepo{found: true, member: &entity.WorkspaceMember{Role: entity.WorkspaceRoleMember, UserID: targetID}},
		users,
		&mockWorkspaceRepo{},
		refresh,
		notifier,
	)

	out, err := uc.SuspendAccount(context.Background(), SuspendAccountInput{
		UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID, IPAddress: "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("suspend: %v", err)
	}
	if out.Status != string(entity.UserStatusSuspended) {
		t.Fatalf("status = %q", out.Status)
	}
	if users.updated == nil || users.updated.AuthVersion != 2 {
		t.Fatalf("expected auth_version 2, got %+v", users.updated)
	}
	if !refresh.revokeCalled || refresh.revokedUserID != targetID {
		t.Fatal("expected refresh tokens revoked")
	}
	if notifier.suspendedUserID != targetID {
		t.Fatal("expected ws notification")
	}
}

func TestSuspendAccount_DisconnectsHostedWorkspaces(t *testing.T) {
	hostID, workspaceID, targetID := testAccountIDs()
	hostedWorkspaceID := uuid.New()
	users := &mockAccountUserRepo{users: map[uuid.UUID]*entity.User{
		targetID: memberUser(targetID, entity.UserStatusActive),
	}}
	notifier := &mockAccountNotifier{}
	uc := newAccountUseCase(
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
	if err != nil {
		t.Fatalf("suspend: %v", err)
	}
	if notifier.hostSuspendedWorkspaceID != hostedWorkspaceID {
		t.Fatalf("expected hosted workspace disconnect, got %v", notifier.hostSuspendedWorkspaceID)
	}
}

func TestReactivateAccount_NotSuspended(t *testing.T) {
	hostID, workspaceID, targetID := testAccountIDs()
	users := &mockAccountUserRepo{users: map[uuid.UUID]*entity.User{
		targetID: memberUser(targetID, entity.UserStatusActive),
	}}
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
	if !errors.Is(err, ErrAccountNotSuspended) {
		t.Fatalf("expected ErrAccountNotSuspended, got %v", err)
	}
}

func TestReactivateAccount_Success(t *testing.T) {
	hostID, workspaceID, targetID := testAccountIDs()
	user := memberUser(targetID, entity.UserStatusActive)
	_ = user.Suspend(accountTestNow())
	users := &mockAccountUserRepo{users: map[uuid.UUID]*entity.User{targetID: user}}
	uc := newAccountUseCase(
		&mockAccessCheck{},
		&mockMemberRepo{found: true, member: &entity.WorkspaceMember{Role: entity.WorkspaceRoleMember, UserID: targetID}},
		users,
		&mockWorkspaceRepo{},
		&trackingRefreshTokenRepo{},
		&mockAccountNotifier{},
	)

	out, err := uc.ReactivateAccount(context.Background(), ReactivateAccountInput{
		UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID,
	})
	if err != nil {
		t.Fatalf("reactivate: %v", err)
	}
	if out.Status != string(entity.UserStatusActive) {
		t.Fatalf("status = %q", out.Status)
	}
	if users.updated == nil || users.updated.AuthVersion != 3 {
		t.Fatalf("expected auth_version 3, got %+v", users.updated)
	}
}

func TestDeleteAccount_NotSuspended(t *testing.T) {
	hostID, workspaceID, targetID := testAccountIDs()
	users := &mockAccountUserRepo{users: map[uuid.UUID]*entity.User{
		targetID: memberUser(targetID, entity.UserStatusActive),
	}}
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
	if !errors.Is(err, ErrAccountNotSuspended) {
		t.Fatalf("expected ErrAccountNotSuspended, got %v", err)
	}
}

func TestDeleteAccount_Success(t *testing.T) {
	hostID, workspaceID, targetID := testAccountIDs()
	user := memberUser(targetID, entity.UserStatusActive)
	_ = user.Suspend(accountTestNow())
	users := &mockAccountUserRepo{users: map[uuid.UUID]*entity.User{targetID: user}}
	refresh := &trackingRefreshTokenRepo{}
	notifier := &mockAccountNotifier{}
	uc := newAccountUseCase(
		&mockAccessCheck{},
		&mockMemberRepo{found: true, member: &entity.WorkspaceMember{Role: entity.WorkspaceRoleMember, UserID: targetID}},
		users,
		&mockWorkspaceRepo{},
		refresh,
		notifier,
	)

	out, err := uc.DeleteAccount(context.Background(), DeleteAccountInput{
		UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID,
	})
	if err != nil {
		t.Fatalf("delete account: %v", err)
	}
	if out.Status != string(entity.UserStatusDeleted) {
		t.Fatalf("status = %q", out.Status)
	}
	if users.updated == nil {
		t.Fatal("expected user update")
	}
	if users.updated.Email != deletedUserEmail(targetID) {
		t.Fatalf("email = %q", users.updated.Email)
	}
	if users.updated.PasswordHash != deletedAccountPasswordHash {
		t.Fatal("expected unusable password hash")
	}
	if !refresh.revokeCalled {
		t.Fatal("expected refresh tokens revoked")
	}
	if notifier.deletedUserID != targetID {
		t.Fatal("expected ws notification")
	}
}

func TestDeleteAccount_DisconnectsHostedWorkspaces(t *testing.T) {
	hostID, workspaceID, targetID := testAccountIDs()
	hostedWorkspaceID := uuid.New()
	user := memberUser(targetID, entity.UserStatusActive)
	_ = user.Suspend(accountTestNow())
	notifier := &mockAccountNotifier{}
	uc := newAccountUseCase(
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
	if err != nil {
		t.Fatalf("delete account: %v", err)
	}
	if notifier.hostDeletedWorkspaceID != hostedWorkspaceID {
		t.Fatalf("expected hosted workspace disconnect, got %v", notifier.hostDeletedWorkspaceID)
	}
}

func TestDeleteAccount_CannotDeleteSelf(t *testing.T) {
	hostID, workspaceID, _ := testAccountIDs()
	uc := newAccountUseCase(
		&mockAccessCheck{},
		&mockMemberRepo{found: true},
		&mockAccountUserRepo{},
		&mockWorkspaceRepo{},
		&trackingRefreshTokenRepo{},
		&mockAccountNotifier{},
	)

	_, err := uc.DeleteAccount(context.Background(), DeleteAccountInput{
		UserID: hostID, WorkspaceID: workspaceID, TargetUserID: hostID,
	})
	if !errors.Is(err, ErrCannotDeleteSelf) {
		t.Fatalf("expected ErrCannotDeleteSelf, got %v", err)
	}
}
