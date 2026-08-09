package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
	"gorm.io/gorm"
)

func newMemberUseCase(access *stubAccessCheck, users *stubUserRepoFull, members *stubMemberRepoFull, notifier *stubMemberNotifier) *MemberUseCase {
	uc := NewMemberUseCase(users, members, &mockAuditLogRepo{}, &mockTransactionManager{}, access, notifier)
	uc.now = usecaseTestNow
	return uc
}

func memberFixture() (hostID, workspaceID, targetID uuid.UUID, access *stubAccessCheck) {
	hostID = uuid.New()
	workspaceID = uuid.New()
	targetID = uuid.New()
	now := usecaseTestNow()
	access = &stubAccessCheck{result: &WorkspaceAccessResult{
		Member:    &entity.WorkspaceMember{Role: entity.WorkspaceRoleHost, UserID: hostID},
		Workspace: &entity.Workspace{ID: workspaceID, HostID: hostID, CreatedAt: now, UpdatedAt: now},
		Host:      &entity.User{ID: hostID, Status: entity.UserStatusActive},
	}}
	return hostID, workspaceID, targetID, access
}

func TestValidEmail(t *testing.T) {
	email, err := validEmail("  Alice@Example.com ")
	if err != nil || email != "alice@example.com" {
		t.Fatalf("valid email: %q %v", email, err)
	}
	if _, err := validEmail("bad"); !errors.Is(err, ErrValidation) {
		t.Fatalf("invalid email: %v", err)
	}
}

func TestListUser_Deleted(t *testing.T) {
	deletedAt := usecaseTestNow()
	user := &entity.User{Status: entity.UserStatusDeleted, DeletedAt: &deletedAt}
	name, email, status := listUser(user)
	if name != deletedName || email != "" || status != string(entity.UserStatusDeleted) {
		t.Fatalf("deleted user listing: name=%q email=%q status=%q", name, email, status)
	}
}

func TestSearchUser_Success(t *testing.T) {
	hostID, workspaceID, targetID, access := memberFixture()
	users := &stubUserRepoFull{
		findByEmailFound: true,
		findByEmailUser:  &entity.User{ID: targetID, Name: "Bob", Email: "bob@example.com", Status: entity.UserStatusActive},
	}
	members := &stubMemberRepoFull{exists: false}
	uc := newMemberUseCase(access, users, members, nil)

	out, err := uc.SearchUser(context.Background(), SearchUserInput{
		UserID: hostID, WorkspaceID: workspaceID, Email: "bob@example.com",
	})
	if err != nil {
		t.Fatalf("SearchUser: %v", err)
	}
	if out.ID != targetID {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestSearchUser_NotHost(t *testing.T) {
	hostID, workspaceID, _, _ := memberFixture()
	access := &stubAccessCheck{err: ErrHostPermissionRequired}
	uc := newMemberUseCase(access, &stubUserRepoFull{}, &stubMemberRepoFull{}, nil)

	_, err := uc.SearchUser(context.Background(), SearchUserInput{
		UserID: hostID, WorkspaceID: workspaceID, Email: "bob@example.com",
	})
	if !errors.Is(err, ErrHostPermissionRequired) {
		t.Fatalf("expected ErrHostPermissionRequired, got %v", err)
	}
}

func TestSearchUser_InvalidEmail(t *testing.T) {
	hostID, workspaceID, _, access := memberFixture()
	uc := newMemberUseCase(access, &stubUserRepoFull{}, &stubMemberRepoFull{}, nil)

	_, err := uc.SearchUser(context.Background(), SearchUserInput{
		UserID: hostID, WorkspaceID: workspaceID, Email: "bad",
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestSearchUser_NotFound(t *testing.T) {
	hostID, workspaceID, _, access := memberFixture()
	uc := newMemberUseCase(access, &stubUserRepoFull{}, &stubMemberRepoFull{}, nil)

	_, err := uc.SearchUser(context.Background(), SearchUserInput{
		UserID: hostID, WorkspaceID: workspaceID, Email: "missing@example.com",
	})
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestSearchUser_Self(t *testing.T) {
	hostID, workspaceID, _, access := memberFixture()
	users := &stubUserRepoFull{
		findByEmailFound: true,
		findByEmailUser:  &entity.User{ID: hostID, Email: "host@example.com", Status: entity.UserStatusActive},
	}
	uc := newMemberUseCase(access, users, &stubMemberRepoFull{}, nil)

	_, err := uc.SearchUser(context.Background(), SearchUserInput{
		UserID: hostID, WorkspaceID: workspaceID, Email: "host@example.com",
	})
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound for self, got %v", err)
	}
}

func TestSearchUser_AlreadyMember(t *testing.T) {
	hostID, workspaceID, targetID, access := memberFixture()
	users := &stubUserRepoFull{
		findByEmailFound: true,
		findByEmailUser:  &entity.User{ID: targetID, Email: "bob@example.com", Status: entity.UserStatusActive},
	}
	members := &stubMemberRepoFull{exists: true}
	uc := newMemberUseCase(access, users, members, nil)

	_, err := uc.SearchUser(context.Background(), SearchUserInput{
		UserID: hostID, WorkspaceID: workspaceID, Email: "bob@example.com",
	})
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound for existing member, got %v", err)
	}
}

func TestSearchUser_ExistsError(t *testing.T) {
	hostID, workspaceID, targetID, access := memberFixture()
	users := &stubUserRepoFull{
		findByEmailFound: true,
		findByEmailUser:  &entity.User{ID: targetID, Email: "bob@example.com", Status: entity.UserStatusActive},
	}
	members := &stubMemberRepoFull{mockMemberRepo: mockMemberRepo{err: fmt.Errorf("db error")}}
	uc := newMemberUseCase(access, users, members, nil)

	_, err := uc.SearchUser(context.Background(), SearchUserInput{
		UserID: hostID, WorkspaceID: workspaceID, Email: "bob@example.com",
	})
	if err == nil {
		t.Fatal("expected exists error")
	}
}

func TestListMembers_Success(t *testing.T) {
	hostID, workspaceID, targetID, access := memberFixture()
	now := usecaseTestNow()
	hostMember, _ := entity.NewWorkspaceMember(workspaceID, hostID, entity.WorkspaceRoleHost, now)
	targetMember, _ := entity.NewWorkspaceMember(workspaceID, targetID, entity.WorkspaceRoleMember, now)
	members := &stubMemberRepoFull{list: []entity.WorkspaceMember{hostMember, targetMember}}
	users := newStubUserRepoFull(map[uuid.UUID]*entity.User{
		hostID:   {ID: hostID, Name: "Host", Email: "host@example.com", Status: entity.UserStatusActive},
		targetID: {ID: targetID, Name: "Bob", Email: "bob@example.com", Status: entity.UserStatusActive},
	})
	uc := newMemberUseCase(access, users, members, nil)

	items, err := uc.ListMembers(context.Background(), ListMembersInput{UserID: hostID, WorkspaceID: workspaceID})
	if err != nil {
		t.Fatalf("ListMembers: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 members, got %d", len(items))
	}
}

func TestListMembers_NotMember(t *testing.T) {
	access := &stubAccessCheck{err: ErrWorkspaceAccessDenied}
	uc := newMemberUseCase(access, &stubUserRepoFull{}, &stubMemberRepoFull{}, nil)

	_, err := uc.ListMembers(context.Background(), ListMembersInput{UserID: uuid.New(), WorkspaceID: uuid.New()})
	if !errors.Is(err, ErrWorkspaceAccessDenied) {
		t.Fatalf("expected ErrWorkspaceAccessDenied, got %v", err)
	}
}

func TestListMembers_ListError(t *testing.T) {
	hostID, workspaceID, _, access := memberFixture()
	members := &stubMemberRepoFull{listErr: fmt.Errorf("list failed")}
	uc := newMemberUseCase(access, &stubUserRepoFull{}, members, nil)

	_, err := uc.ListMembers(context.Background(), ListMembersInput{UserID: hostID, WorkspaceID: workspaceID})
	if err == nil {
		t.Fatal("expected list error")
	}
}

func TestAddMember_Success(t *testing.T) {
	hostID, workspaceID, targetID, access := memberFixture()
	users := newStubUserRepoFull(map[uuid.UUID]*entity.User{
		targetID: {ID: targetID, Name: "Bob", Email: "bob@example.com", Status: entity.UserStatusActive},
	})
	members := &stubMemberRepoFull{}
	uc := newMemberUseCase(access, users, members, nil)

	out, err := uc.AddMember(context.Background(), AddMemberInput{
		UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID, IPAddress: "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("AddMember: %v", err)
	}
	if out.Email != "bob@example.com" {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestAddMember_CannotAddSelf(t *testing.T) {
	hostID, workspaceID, _, access := memberFixture()
	uc := newMemberUseCase(access, &stubUserRepoFull{}, &stubMemberRepoFull{}, nil)

	_, err := uc.AddMember(context.Background(), AddMemberInput{
		UserID: hostID, WorkspaceID: workspaceID, TargetUserID: hostID,
	})
	if !errors.Is(err, ErrCannotAddSelf) {
		t.Fatalf("expected ErrCannotAddSelf, got %v", err)
	}
}

func TestAddMember_TargetNotFound(t *testing.T) {
	hostID, workspaceID, targetID, access := memberFixture()
	uc := newMemberUseCase(access, &stubUserRepoFull{}, &stubMemberRepoFull{}, nil)

	_, err := uc.AddMember(context.Background(), AddMemberInput{
		UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID,
	})
	if !errors.Is(err, ErrTargetUserNotFound) {
		t.Fatalf("expected ErrTargetUserNotFound, got %v", err)
	}
}

func TestAddMember_TargetUnavailable(t *testing.T) {
	hostID, workspaceID, targetID, access := memberFixture()
	users := newStubUserRepoFull(map[uuid.UUID]*entity.User{
		targetID: {ID: targetID, Status: entity.UserStatusSuspended},
	})
	uc := newMemberUseCase(access, users, &stubMemberRepoFull{}, nil)

	_, err := uc.AddMember(context.Background(), AddMemberInput{
		UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID,
	})
	if !errors.Is(err, ErrTargetAccountUnavailable) {
		t.Fatalf("expected ErrTargetAccountUnavailable, got %v", err)
	}
}

func TestAddMember_AlreadyExists(t *testing.T) {
	hostID, workspaceID, targetID, access := memberFixture()
	users := newStubUserRepoFull(map[uuid.UUID]*entity.User{
		targetID: {ID: targetID, Status: entity.UserStatusActive},
	})
	members := &stubMemberRepoFull{exists: true}
	uc := newMemberUseCase(access, users, members, nil)

	_, err := uc.AddMember(context.Background(), AddMemberInput{
		UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID,
	})
	if !errors.Is(err, ErrMemberAlreadyExists) {
		t.Fatalf("expected ErrMemberAlreadyExists, got %v", err)
	}
}

func TestAddMember_DuplicateKey(t *testing.T) {
	hostID, workspaceID, targetID, access := memberFixture()
	users := newStubUserRepoFull(map[uuid.UUID]*entity.User{
		targetID: {ID: targetID, Name: "Bob", Email: "bob@example.com", Status: entity.UserStatusActive},
	})
	members := &stubMemberRepoFull{createErr: gorm.ErrDuplicatedKey}
	uc := newMemberUseCase(access, users, members, nil)

	_, err := uc.AddMember(context.Background(), AddMemberInput{
		UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID,
	})
	if !errors.Is(err, ErrMemberAlreadyExists) {
		t.Fatalf("expected ErrMemberAlreadyExists, got %v", err)
	}
}

func TestRemoveMember_Success(t *testing.T) {
	hostID, workspaceID, targetID, access := memberFixture()
	now := usecaseTestNow()
	targetMember, _ := entity.NewWorkspaceMember(workspaceID, targetID, entity.WorkspaceRoleMember, now)
	members := &stubMemberRepoFull{
		mockMemberRepo: mockMemberRepo{found: true, member: &targetMember},
	}
	notifier := &stubMemberNotifier{}
	uc := newMemberUseCase(access, &stubUserRepoFull{}, members, notifier)

	if err := uc.RemoveMember(context.Background(), RemoveMemberInput{
		UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID, IPAddress: "127.0.0.1",
	}); err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}
}

func TestRemoveMember_CannotRemoveHost(t *testing.T) {
	hostID, workspaceID, targetID, access := memberFixture()
	members := &stubMemberRepoFull{
		mockMemberRepo: mockMemberRepo{found: true, member: &entity.WorkspaceMember{Role: entity.WorkspaceRoleHost}},
	}
	uc := newMemberUseCase(access, &stubUserRepoFull{}, members, nil)

	err := uc.RemoveMember(context.Background(), RemoveMemberInput{
		UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID,
	})
	if !errors.Is(err, ErrCannotRemoveHost) {
		t.Fatalf("expected ErrCannotRemoveHost, got %v", err)
	}
}

func TestRemoveMember_NotFound(t *testing.T) {
	hostID, workspaceID, targetID, access := memberFixture()
	uc := newMemberUseCase(access, &stubUserRepoFull{}, &stubMemberRepoFull{}, nil)

	err := uc.RemoveMember(context.Background(), RemoveMemberInput{
		UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID,
	})
	if !errors.Is(err, ErrMemberNotFound) {
		t.Fatalf("expected ErrMemberNotFound, got %v", err)
	}
}

func TestRemoveMember_NotifierError(t *testing.T) {
	hostID, workspaceID, targetID, access := memberFixture()
	now := usecaseTestNow()
	targetMember, _ := entity.NewWorkspaceMember(workspaceID, targetID, entity.WorkspaceRoleMember, now)
	members := &stubMemberRepoFull{
		mockMemberRepo: mockMemberRepo{found: true, member: &targetMember},
	}
	notifier := &stubMemberNotifier{err: fmt.Errorf("notify failed")}
	uc := newMemberUseCase(access, &stubUserRepoFull{}, members, notifier)

	err := uc.RemoveMember(context.Background(), RemoveMemberInput{
		UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID,
	})
	if err == nil {
		t.Fatal("expected notifier error")
	}
}
