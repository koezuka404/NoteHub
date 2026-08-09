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

type mapMemberRepo struct {
	members   map[string]*entity.WorkspaceMember
	createErr error
	findErr   error
}

func memberKey(workspaceID, userID uuid.UUID) string {
	return workspaceID.String() + ":" + userID.String()
}

func (m *mapMemberRepo) Create(_ context.Context, member *entity.WorkspaceMember) error {
	if m.createErr != nil {
		return m.createErr
	}
	if m.members == nil {
		m.members = map[string]*entity.WorkspaceMember{}
	}
	copy := *member
	m.members[memberKey(member.WorkspaceID, member.UserID)] = &copy
	return nil
}

func (m *mapMemberRepo) FindByWorkspaceAndUser(_ context.Context, workspaceID, userID uuid.UUID) (*entity.WorkspaceMember, bool, error) {
	if m.findErr != nil {
		return nil, false, m.findErr
	}
	member, ok := m.members[memberKey(workspaceID, userID)]
	if !ok {
		return nil, false, nil
	}
	copy := *member
	return &copy, true, nil
}

func (m *mapMemberRepo) FindByWorkspaceID(_ context.Context, workspaceID uuid.UUID) ([]entity.WorkspaceMember, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	var out []entity.WorkspaceMember
	for _, member := range m.members {
		if member.WorkspaceID == workspaceID {
			out = append(out, *member)
		}
	}
	return out, nil
}

func (m *mapMemberRepo) Exists(_ context.Context, workspaceID, userID uuid.UUID) (bool, error) {
	_, ok := m.members[memberKey(workspaceID, userID)]
	return ok, nil
}

func (m *mapMemberRepo) Delete(context.Context, uuid.UUID, uuid.UUID) error { return nil }

type workspaceListRepo struct {
	stubWorkspaceRepoFull
	memberWorkspaces map[uuid.UUID][]uuid.UUID
}

func (s *workspaceListRepo) FindByUserID(_ context.Context, userID uuid.UUID) ([]entity.Workspace, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	var out []entity.Workspace
	if ids, ok := s.memberWorkspaces[userID]; ok {
		for _, id := range ids {
			if ws, ok := s.byID[id]; ok {
				out = append(out, *ws)
			}
		}
	}
	return out, nil
}

func newWorkspaceUseCase(
	users *stubUserRepoFull,
	workspaces repository.WorkspaceRepository,
	members *mapMemberRepo,
	flush *stubDocumentFlush,
	notifier *stubDocumentNotifier,
) *WorkspaceUseCase {
	uc := NewWorkspaceUseCase(users, workspaces, members, &mockAuditLogRepo{}, &mockTransactionManager{}, flush, notifier)
	uc.now = usecaseTestNow
	return uc
}

func setupWorkspaceFixture() (hostID, workspaceID uuid.UUID, users *stubUserRepoFull, workspaces *stubWorkspaceRepoFull, members *mapMemberRepo) {
	hostID = uuid.New()
	workspaceID = uuid.New()
	now := usecaseTestNow()
	ws := entity.Workspace{
		ID: workspaceID, Name: "Team", HostID: hostID, CreatedAt: now, UpdatedAt: now,
	}
	hostMember, _ := entity.NewWorkspaceMember(workspaceID, hostID, entity.WorkspaceRoleHost, now)
	users = newStubUserRepoFull(map[uuid.UUID]*entity.User{
		hostID: {ID: hostID, Name: "Host", Email: "host@example.com", Status: entity.UserStatusActive, CreatedAt: now, UpdatedAt: now},
	})
	workspaces = &stubWorkspaceRepoFull{byID: map[uuid.UUID]*entity.Workspace{workspaceID: &ws}}
	members = &mapMemberRepo{members: map[string]*entity.WorkspaceMember{
		memberKey(workspaceID, hostID): &hostMember,
	}}
	return hostID, workspaceID, users, workspaces, members
}

func TestValidateWorkspaceName(t *testing.T) {
	if err := validateWorkspaceName("  My Workspace  "); err != nil {
		t.Fatalf("valid name: %v", err)
	}
	if err := validateWorkspaceName(""); !errors.Is(err, ErrValidation) {
		t.Fatalf("empty name: %v", err)
	}
	if err := validateWorkspaceName(string(make([]rune, maxWorkspaceNameLength+1))); !errors.Is(err, ErrValidation) {
		t.Fatalf("long name: %v", err)
	}
}

func TestValidateDeleteReason(t *testing.T) {
	if err := validateDeleteReason("reason"); err != nil {
		t.Fatalf("valid reason: %v", err)
	}
	if err := validateDeleteReason(""); !errors.Is(err, ErrValidation) {
		t.Fatalf("empty reason: %v", err)
	}
}

func TestWorkspaceAvailability(t *testing.T) {
	active := &entity.User{Status: entity.UserStatusActive}
	if ok, reason := workspaceAvailability(active); !ok || reason != unavailableReasonNone {
		t.Fatalf("active host: ok=%v reason=%q", ok, reason)
	}
	suspended := &entity.User{Status: entity.UserStatusSuspended}
	if ok, reason := workspaceAvailability(suspended); ok || reason != unavailableHostSuspended {
		t.Fatalf("suspended host: ok=%v reason=%q", ok, reason)
	}
	deleted := &entity.User{Status: entity.UserStatusDeleted}
	if ok, reason := workspaceAvailability(deleted); ok || reason != unavailableHostDeleted {
		t.Fatalf("deleted host: ok=%v reason=%q", ok, reason)
	}
}

func TestCheckWorkspaceHostStatus(t *testing.T) {
	active := &entity.User{Status: entity.UserStatusActive}
	if err := checkWorkspaceHostStatus(active); err != nil {
		t.Fatalf("active: %v", err)
	}
	suspended := &entity.User{Status: entity.UserStatusSuspended}
	if !errors.Is(checkWorkspaceHostStatus(suspended), ErrWorkspaceHostSuspended) {
		t.Fatalf("expected host suspended error")
	}
	deleted := &entity.User{Status: entity.UserStatusDeleted}
	if !errors.Is(checkWorkspaceHostStatus(deleted), ErrWorkspaceHostDeleted) {
		t.Fatalf("expected host deleted error")
	}
}

func TestCreateWorkspace_Success(t *testing.T) {
	userID := uuid.New()
	now := usecaseTestNow()
	users := newStubUserRepoFull(map[uuid.UUID]*entity.User{
		userID: {ID: userID, Name: "Alice", Email: "alice@example.com", Status: entity.UserStatusActive, CreatedAt: now, UpdatedAt: now},
	})
	uc := newWorkspaceUseCase(users, &stubWorkspaceRepoFull{}, &mapMemberRepo{}, &stubDocumentFlush{}, &stubDocumentNotifier{})

	out, err := uc.CreateWorkspace(context.Background(), CreateWorkspaceInput{
		UserID: userID, Name: "  New Team  ", IPAddress: "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("CreateWorkspace: %v", err)
	}
	if out.Name != "New Team" || out.HostID != userID {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestCreateWorkspace_ValidationError(t *testing.T) {
	uc := newWorkspaceUseCase(&stubUserRepoFull{}, &stubWorkspaceRepoFull{}, &mapMemberRepo{}, nil, nil)
	_, err := uc.CreateWorkspace(context.Background(), CreateWorkspaceInput{UserID: uuid.New(), Name: ""})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestCreateWorkspace_UserSuspended(t *testing.T) {
	userID := uuid.New()
	users := newStubUserRepoFull(map[uuid.UUID]*entity.User{
		userID: {ID: userID, Status: entity.UserStatusSuspended},
	})
	uc := newWorkspaceUseCase(users, &stubWorkspaceRepoFull{}, &mapMemberRepo{}, nil, nil)
	_, err := uc.CreateWorkspace(context.Background(), CreateWorkspaceInput{UserID: userID, Name: "Team"})
	if !errors.Is(err, ErrAccountSuspended) {
		t.Fatalf("expected ErrAccountSuspended, got %v", err)
	}
}

func TestGetWorkspace_Success(t *testing.T) {
	hostID, workspaceID, users, workspaces, members := setupWorkspaceFixture()
	uc := newWorkspaceUseCase(users, workspaces, members, nil, nil)

	out, err := uc.GetWorkspace(context.Background(), GetWorkspaceInput{UserID: hostID, WorkspaceID: workspaceID})
	if err != nil {
		t.Fatalf("GetWorkspace: %v", err)
	}
	if out.Name != "Team" || out.Role != string(entity.WorkspaceRoleHost) {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestCheckWorkspaceAccess_NotFound(t *testing.T) {
	hostID, _, users, workspaces, members := setupWorkspaceFixture()
	uc := newWorkspaceUseCase(users, workspaces, members, nil, nil)

	_, err := uc.CheckWorkspaceAccess(context.Background(), CheckWorkspaceAccessInput{
		UserID: hostID, WorkspaceID: uuid.New(),
	})
	if !errors.Is(err, ErrWorkspaceNotFound) {
		t.Fatalf("expected ErrWorkspaceNotFound, got %v", err)
	}
}

func TestCheckWorkspaceAccess_DeletedWorkspace(t *testing.T) {
	hostID, workspaceID, users, workspaces, members := setupWorkspaceFixture()
	now := usecaseTestNow()
	deletedAt := now
	workspaces.byID[workspaceID].DeletedAt = &deletedAt
	uc := newWorkspaceUseCase(users, workspaces, members, nil, nil)

	_, err := uc.CheckWorkspaceAccess(context.Background(), CheckWorkspaceAccessInput{
		UserID: hostID, WorkspaceID: workspaceID,
	})
	if !errors.Is(err, ErrWorkspaceAlreadyDeleted) {
		t.Fatalf("expected ErrWorkspaceAlreadyDeleted, got %v", err)
	}
}

func TestCheckWorkspaceAccess_HostSuspended(t *testing.T) {
	hostID, workspaceID, users, workspaces, members := setupWorkspaceFixture()
	users.users[hostID].Status = entity.UserStatusSuspended
	memberID := uuid.New()
	now := usecaseTestNow()
	member, _ := entity.NewWorkspaceMember(workspaceID, memberID, entity.WorkspaceRoleMember, now)
	members.members[memberKey(workspaceID, memberID)] = &member
	users.users[memberID] = &entity.User{ID: memberID, Status: entity.UserStatusActive}
	uc := newWorkspaceUseCase(users, workspaces, members, nil, nil)

	_, err := uc.CheckWorkspaceAccess(context.Background(), CheckWorkspaceAccessInput{
		UserID: memberID, WorkspaceID: workspaceID,
	})
	if !errors.Is(err, ErrWorkspaceHostSuspended) {
		t.Fatalf("expected ErrWorkspaceHostSuspended, got %v", err)
	}
}

func TestCheckWorkspaceAccess_AccessDenied(t *testing.T) {
	_, workspaceID, users, workspaces, members := setupWorkspaceFixture()
	otherID := uuid.New()
	users.users[otherID] = &entity.User{ID: otherID, Status: entity.UserStatusActive}
	uc := newWorkspaceUseCase(users, workspaces, members, nil, nil)

	_, err := uc.CheckWorkspaceAccess(context.Background(), CheckWorkspaceAccessInput{
		UserID: otherID, WorkspaceID: workspaceID,
	})
	if !errors.Is(err, ErrWorkspaceAccessDenied) {
		t.Fatalf("expected ErrWorkspaceAccessDenied, got %v", err)
	}
}

func TestCheckWorkspaceAccess_HostRequired(t *testing.T) {
	_, workspaceID, users, workspaces, members := setupWorkspaceFixture()
	memberID := uuid.New()
	now := usecaseTestNow()
	member, _ := entity.NewWorkspaceMember(workspaceID, memberID, entity.WorkspaceRoleMember, now)
	members.members[memberKey(workspaceID, memberID)] = &member
	users.users[memberID] = &entity.User{ID: memberID, Status: entity.UserStatusActive}
	uc := newWorkspaceUseCase(users, workspaces, members, nil, nil)

	hostRole := entity.WorkspaceRoleHost
	_, err := uc.CheckWorkspaceAccess(context.Background(), CheckWorkspaceAccessInput{
		UserID: memberID, WorkspaceID: workspaceID, RequiredRole: &hostRole,
	})
	if !errors.Is(err, ErrHostPermissionRequired) {
		t.Fatalf("expected ErrHostPermissionRequired, got %v", err)
	}
}

func TestListWorkspaces_Success(t *testing.T) {
	hostID, workspaceID, users, workspaces, members := setupWorkspaceFixture()
	workspaces.byID[workspaceID].HostID = hostID
	uc := newWorkspaceUseCase(users, workspaces, members, nil, nil)

	items, err := uc.ListWorkspaces(context.Background(), ListWorkspacesInput{UserID: hostID})
	if err != nil {
		t.Fatalf("ListWorkspaces: %v", err)
	}
	if len(items) != 1 || !items[0].IsAvailable {
		t.Fatalf("unexpected items: %+v", items)
	}
}

func TestListWorkspaces_HostDeleted(t *testing.T) {
	hostID, workspaceID, users, _, members := setupWorkspaceFixture()
	now := usecaseTestNow()
	deletedAt := now
	users.users[hostID].Status = entity.UserStatusDeleted
	users.users[hostID].DeletedAt = &deletedAt
	memberID := uuid.New()
	member, _ := entity.NewWorkspaceMember(workspaceID, memberID, entity.WorkspaceRoleMember, now)
	members.members[memberKey(workspaceID, memberID)] = &member
	users.users[memberID] = &entity.User{ID: memberID, Status: entity.UserStatusActive}
	workspaces := &workspaceListRepo{
		stubWorkspaceRepoFull: stubWorkspaceRepoFull{
			byID: map[uuid.UUID]*entity.Workspace{
				workspaceID: {ID: workspaceID, Name: "Team", HostID: hostID, CreatedAt: now, UpdatedAt: now},
			},
		},
		memberWorkspaces: map[uuid.UUID][]uuid.UUID{memberID: {workspaceID}},
	}
	uc := newWorkspaceUseCase(users, workspaces, members, nil, nil)

	items, err := uc.ListWorkspaces(context.Background(), ListWorkspacesInput{UserID: memberID})
	if err != nil {
		t.Fatalf("ListWorkspaces: %v", err)
	}
	if len(items) != 1 || items[0].IsAvailable || items[0].UnavailableReason != unavailableHostDeleted {
		t.Fatalf("unexpected availability: %+v", items[0])
	}
}

func TestUpdateWorkspace_Success(t *testing.T) {
	hostID, workspaceID, users, workspaces, members := setupWorkspaceFixture()
	uc := newWorkspaceUseCase(users, workspaces, members, nil, nil)

	out, err := uc.UpdateWorkspace(context.Background(), UpdateWorkspaceInput{
		UserID: hostID, WorkspaceID: workspaceID, Name: "  Renamed  ", IPAddress: "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("UpdateWorkspace: %v", err)
	}
	if out.Name != "Renamed" {
		t.Fatalf("name = %q", out.Name)
	}
}

func TestUpdateWorkspace_NotHost(t *testing.T) {
	_, workspaceID, users, workspaces, members := setupWorkspaceFixture()
	memberID := uuid.New()
	now := usecaseTestNow()
	member, _ := entity.NewWorkspaceMember(workspaceID, memberID, entity.WorkspaceRoleMember, now)
	members.members[memberKey(workspaceID, memberID)] = &member
	users.users[memberID] = &entity.User{ID: memberID, Status: entity.UserStatusActive}
	uc := newWorkspaceUseCase(users, workspaces, members, nil, nil)

	_, err := uc.UpdateWorkspace(context.Background(), UpdateWorkspaceInput{
		UserID: memberID, WorkspaceID: workspaceID, Name: "Renamed",
	})
	if !errors.Is(err, ErrHostPermissionRequired) {
		t.Fatalf("expected ErrHostPermissionRequired, got %v", err)
	}
}

func TestDeleteWorkspace_Success(t *testing.T) {
	hostID, workspaceID, users, workspaces, members := setupWorkspaceFixture()
	notifier := &stubDocumentNotifier{}
	uc := newWorkspaceUseCase(users, workspaces, members, &stubDocumentFlush{}, notifier)

	out, err := uc.DeleteWorkspace(context.Background(), DeleteWorkspaceInput{
		UserID: hostID, WorkspaceID: workspaceID, Reason: "no longer needed", IPAddress: "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("DeleteWorkspace: %v", err)
	}
	if out.WorkspaceID != workspaceID {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestDeleteWorkspace_FlushError(t *testing.T) {
	hostID, workspaceID, users, workspaces, members := setupWorkspaceFixture()
	flush := &stubDocumentFlush{err: fmt.Errorf("flush failed")}
	uc := newWorkspaceUseCase(users, workspaces, members, flush, nil)

	_, err := uc.DeleteWorkspace(context.Background(), DeleteWorkspaceInput{
		UserID: hostID, WorkspaceID: workspaceID, Reason: "cleanup",
	})
	if err == nil {
		t.Fatal("expected flush error")
	}
}

func TestDeleteWorkspace_NotifierError(t *testing.T) {
	hostID, workspaceID, users, workspaces, members := setupWorkspaceFixture()
	notifier := &stubDocumentNotifier{err: fmt.Errorf("notify failed")}
	uc := newWorkspaceUseCase(users, workspaces, members, &stubDocumentFlush{}, notifier)

	_, err := uc.DeleteWorkspace(context.Background(), DeleteWorkspaceInput{
		UserID: hostID, WorkspaceID: workspaceID, Reason: "cleanup",
	})
	if err == nil {
		t.Fatal("expected notifier error")
	}
}

func TestWorkspaceDeleteMetadata(t *testing.T) {
	raw, err := workspaceDeleteMetadata("test reason")
	if err != nil {
		t.Fatalf("workspaceDeleteMetadata: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("expected metadata")
	}
}
