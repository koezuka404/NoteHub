package usecase

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
)

func hookAuditEntityError(t *testing.T) {
	t.Helper()
	orig := newAuditLogFn
	newAuditLogFn = func(*uuid.UUID, string, string, *uuid.UUID, any, time.Time) (entity.AuditLog, error) {
		return entity.AuditLog{}, fmt.Errorf("audit entity failed")
	}
	t.Cleanup(func() { newAuditLogFn = orig })
}

func TestCoverage_Hooks_AuditEntityErrors(t *testing.T) {
	ctx := context.Background()
	hookAuditEntityError(t)

	t.Run("account suspend", func(t *testing.T) {
		hostID, workspaceID, targetID, member, users := accountFixture()
		uc := NewAccountUseCase(users, &trackingRefreshTokenRepo{}, member, &mockWorkspaceRepo{}, &mockAuditLogRepo{}, &mockTransactionManager{}, &mockAccessCheck{}, nil)
		uc.now = accountTestNow
		_, err := uc.SuspendAccount(ctx, SuspendAccountInput{UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID})
		if err == nil || !strings.Contains(err.Error(), "create audit log entity") {
			t.Fatalf("expected audit entity error, got %v", err)
		}
	})

	t.Run("account reactivate", func(t *testing.T) {
		hostID, workspaceID, targetID, member, users := accountFixture()
		suspended := memberUser(targetID, entity.UserStatusActive)
		_ = suspended.Suspend(accountTestNow())
		users.users[targetID] = suspended
		uc := NewAccountUseCase(users, &trackingRefreshTokenRepo{}, member, &mockWorkspaceRepo{}, &mockAuditLogRepo{}, &mockTransactionManager{}, &mockAccessCheck{}, nil)
		uc.now = accountTestNow
		_, err := uc.ReactivateAccount(ctx, ReactivateAccountInput{UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID})
		if err == nil || !strings.Contains(err.Error(), "create audit log entity") {
			t.Fatalf("expected audit entity error, got %v", err)
		}
	})

	t.Run("account delete", func(t *testing.T) {
		hostID, workspaceID, targetID, member, users := accountFixture()
		suspended := memberUser(targetID, entity.UserStatusActive)
		_ = suspended.Suspend(accountTestNow())
		users.users[targetID] = suspended
		uc := NewAccountUseCase(users, &trackingRefreshTokenRepo{}, member, &mockWorkspaceRepo{}, &mockAuditLogRepo{}, &mockTransactionManager{}, &mockAccessCheck{}, nil)
		uc.now = accountTestNow
		_, err := uc.DeleteAccount(ctx, DeleteAccountInput{UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID})
		if err == nil || !strings.Contains(err.Error(), "create audit log entity") {
			t.Fatalf("expected audit entity error, got %v", err)
		}
	})

	t.Run("auth login success", func(t *testing.T) {
		user := &entity.User{ID: uuid.New(), Email: "user@example.com", PasswordHash: "hash", Status: entity.UserStatusActive, AuthVersion: 1}
		uc := NewAuthUseCase(&mockUserRepo{user: user, found: true}, &mockRefreshTokenRepo{}, &mockAuditLogRepo{}, &mockAuthService{}, &mockTransactionManager{}, time.Hour)
		uc.now = usecaseTestNow
		_, err := uc.Login(ctx, LoginInput{Email: "user@example.com", Password: "Pass1234", IPAddress: "127.0.0.1"})
		if err == nil || !strings.Contains(err.Error(), "create login success audit log entity") {
			t.Fatalf("expected login audit entity error, got %v", err)
		}
	})

	t.Run("auth register", func(t *testing.T) {
		uc := NewAuthUseCase(&stubUserRepoFull{}, &mockRefreshTokenRepo{}, &mockAuditLogRepo{}, &mockAuthService{}, &mockTransactionManager{}, time.Hour)
		uc.now = usecaseTestNow
		_, err := uc.Register(ctx, RegisterInput{Name: "Alice", Email: "new@example.com", Password: "Pass1234", IPAddress: "127.0.0.1"})
		if err == nil || !strings.Contains(err.Error(), "create audit log entity") {
			t.Fatalf("expected register audit entity error, got %v", err)
		}
	})

	t.Run("auth logout", func(t *testing.T) {
		now := usecaseTestNow()
		uc := NewAuthUseCase(&stubUserRepoFull{}, &trackingRefreshRepo{}, &mockAuditLogRepo{}, &mockAuthService{}, &mockTransactionManager{}, time.Hour)
		uc.now = usecaseTestNow
		_, err := uc.Logout(ctx, LogoutInput{UserID: uuid.New(), AccessTokenJTI: uuid.New(), AccessTokenExp: now.Add(time.Hour), CSRFValidated: true})
		if err == nil || !strings.Contains(err.Error(), "create logout audit log") {
			t.Fatalf("expected logout audit entity error, got %v", err)
		}
	})

	t.Run("document create", func(t *testing.T) {
		userID, workspaceID, _ := versionFixture()
		uc := NewDocumentUseCase(&stubDocumentRepo{}, &mockAuditLogRepo{}, &mockTransactionManager{}, docAccess(workspaceID), nil, nil, nil)
		uc.now = usecaseTestNow
		_, err := uc.CreateDocument(ctx, CreateDocumentInput{UserID: userID, WorkspaceID: workspaceID, Title: "Doc", Content: "body"})
		if err == nil || !strings.Contains(err.Error(), "create audit log entity") {
			t.Fatalf("expected create audit entity error, got %v", err)
		}
	})

	t.Run("document update", func(t *testing.T) {
		userID, workspaceID, doc := versionFixture()
		docs := &stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}}
		uc := NewDocumentUseCase(docs, &mockAuditLogRepo{}, &mockTransactionManager{}, docAccess(workspaceID), nil, nil, nil)
		uc.now = usecaseTestNow
		_, err := uc.UpdateDocument(ctx, UpdateDocumentInput{UserID: userID, DocumentID: doc.ID, Title: "Renamed"})
		if err == nil || !strings.Contains(err.Error(), "create audit log entity") {
			t.Fatalf("expected update audit entity error, got %v", err)
		}
	})

	t.Run("document delete", func(t *testing.T) {
		userID, workspaceID, doc := versionFixture()
		docs := &stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}}
		uc := NewDocumentUseCase(docs, &mockAuditLogRepo{}, &mockTransactionManager{}, docAccess(workspaceID), nil, nil, nil)
		uc.now = usecaseTestNow
		_, err := uc.DeleteDocument(ctx, DeleteDocumentInput{UserID: userID, DocumentID: doc.ID})
		if err == nil || !strings.Contains(err.Error(), "create audit log entity") {
			t.Fatalf("expected delete audit entity error, got %v", err)
		}
	})

	t.Run("workspace create", func(t *testing.T) {
		userID := uuid.New()
		users := newStubUserRepoFull(map[uuid.UUID]*entity.User{
			userID: {ID: userID, Status: entity.UserStatusActive, CreatedAt: usecaseTestNow(), UpdatedAt: usecaseTestNow()},
		})
		uc := NewWorkspaceUseCase(users, &stubWorkspaceRepoFull{}, &mapMemberRepo{}, &mockAuditLogRepo{}, &mockTransactionManager{}, nil, nil)
		uc.now = usecaseTestNow
		_, err := uc.CreateWorkspace(ctx, CreateWorkspaceInput{UserID: userID, Name: "Team"})
		if err == nil || !strings.Contains(err.Error(), "create audit log entity") {
			t.Fatalf("expected workspace create audit entity error, got %v", err)
		}
	})

	t.Run("workspace update", func(t *testing.T) {
		hostID, workspaceID, users, workspaces, members := setupWorkspaceFixture()
		uc := NewWorkspaceUseCase(users, workspaces, members, &mockAuditLogRepo{}, &mockTransactionManager{}, nil, nil)
		uc.now = usecaseTestNow
		_, err := uc.UpdateWorkspace(ctx, UpdateWorkspaceInput{UserID: hostID, WorkspaceID: workspaceID, Name: "New"})
		if err == nil || !strings.Contains(err.Error(), "create audit log entity") {
			t.Fatalf("expected workspace update audit entity error, got %v", err)
		}
	})

	t.Run("workspace delete", func(t *testing.T) {
		hostID, workspaceID, users, workspaces, members := setupWorkspaceFixture()
		uc := NewWorkspaceUseCase(users, workspaces, members, &mockAuditLogRepo{}, &mockTransactionManager{}, &stubDocumentFlush{}, nil)
		uc.now = usecaseTestNow
		_, err := uc.DeleteWorkspace(ctx, DeleteWorkspaceInput{UserID: hostID, WorkspaceID: workspaceID, Reason: "done"})
		if err == nil || !strings.Contains(err.Error(), "create audit log entity") {
			t.Fatalf("expected workspace delete audit entity error, got %v", err)
		}
	})

	t.Run("member add", func(t *testing.T) {
		hostID, workspaceID, targetID, access := memberFixture()
		target := uuid.New()
		users := newStubUserRepoFull(map[uuid.UUID]*entity.User{
			targetID: {ID: targetID, Status: entity.UserStatusActive, Email: "target@example.com"},
		})
		uc := NewMemberUseCase(users, &mapMemberRepo{}, &mockAuditLogRepo{}, &mockTransactionManager{}, access, nil)
		uc.now = usecaseTestNow
		_, err := uc.AddMember(ctx, AddMemberInput{UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID, IPAddress: "127.0.0.1"})
		_ = target
		if err == nil || !strings.Contains(err.Error(), "create audit log entity") {
			t.Fatalf("expected add member audit entity error, got %v", err)
		}
	})

	t.Run("member remove", func(t *testing.T) {
		hostID, workspaceID, targetID, access := memberFixture()
		now := usecaseTestNow()
		targetMember, _ := entity.NewWorkspaceMember(workspaceID, targetID, entity.WorkspaceRoleMember, now)
		members := &stubMemberRepoFull{mockMemberRepo: mockMemberRepo{found: true, member: &targetMember}}
		uc := NewMemberUseCase(&stubUserRepoFull{}, members, &mockAuditLogRepo{}, &mockTransactionManager{}, access, nil)
		uc.now = usecaseTestNow
		if err := uc.RemoveMember(ctx, RemoveMemberInput{UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID}); err == nil || !strings.Contains(err.Error(), "create audit log entity") {
			t.Fatalf("expected remove member audit entity error, got %v", err)
		}
	})

	t.Run("version restore", func(t *testing.T) {
		userID, workspaceID, doc := versionFixture()
		now := usecaseTestNow()
		target, _ := entity.NewDocumentVersion(doc, "restored", entity.DocumentVersionManualSave, userID, nil, now)
		docs := &stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}}
		versions := &stubVersionRepo{byID: map[uuid.UUID]*entity.DocumentVersion{target.ID: &target}}
		uc := NewVersionUseCase(docs, versions, &mockTransactionManager{}, docAccess(workspaceID), nil, nil, &mockAuditLogRepo{})
		uc.now = usecaseTestNow
		_, err := uc.RestoreVersion(ctx, RestoreVersionInput{UserID: userID, DocumentID: doc.ID, VersionID: target.ID})
		if err == nil || !strings.Contains(err.Error(), "create document restored audit log entity") {
			t.Fatalf("expected restore audit entity error, got %v", err)
		}
	})

	t.Run("batch backup", func(t *testing.T) {
		dir := t.TempDir()
		binDir := t.TempDir()
		scriptPath := filepath.Join(binDir, "pg_dump")
		script := "#!/bin/sh\nshift\nwhile [ $# -gt 0 ]; do case \"$1\" in -f) OUT=\"$2\"; shift 2;; *) shift;; esac; done\necho backup > \"$OUT\"\n"
		if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
		oldPath := os.Getenv("PATH")
		t.Cleanup(func() { _ = os.Setenv("PATH", oldPath) })
		_ = os.Setenv("PATH", binDir+string(os.PathListSeparator)+oldPath)

		uc := NewDatabaseBackupUseCase("postgres://x", dir, time.Hour, &mockAuditLogRepo{})
		uc.now = usecaseTestNow
		if err := uc.RunOnce(ctx); err == nil || !strings.Contains(err.Error(), "build backup audit log") {
			t.Fatalf("expected backup audit entity error, got %v", err)
		}
	})
}

func TestCoverage_Hooks_EntityConstructors(t *testing.T) {
	ctx := context.Background()
	entityErr := fmt.Errorf("entity failed")

	t.Run("newUserFn register", func(t *testing.T) {
		orig := newUserFn
		newUserFn = func(string, string, string, time.Time) (entity.User, error) {
			return entity.User{}, entityErr
		}
		t.Cleanup(func() { newUserFn = orig })

		uc := NewAuthUseCase(&stubUserRepoFull{}, &mockRefreshTokenRepo{}, &mockAuditLogRepo{}, &mockAuthService{}, &mockTransactionManager{}, time.Hour)
		uc.now = usecaseTestNow
		_, err := uc.Register(ctx, RegisterInput{Name: "Alice", Email: "new@example.com", Password: "Pass1234"})
		if err == nil || !strings.Contains(err.Error(), "create user entity") {
			t.Fatalf("expected user entity error, got %v", err)
		}
	})

	t.Run("newDocumentFn create", func(t *testing.T) {
		orig := newDocumentFn
		newDocumentFn = func(uuid.UUID, uuid.UUID, string, string, time.Time) (entity.Document, error) {
			return entity.Document{}, entityErr
		}
		t.Cleanup(func() { newDocumentFn = orig })

		userID, workspaceID, _ := versionFixture()
		uc := newDocumentUseCase(&stubDocumentRepo{}, docAccess(workspaceID), nil, nil, nil)
		_, err := uc.CreateDocument(ctx, CreateDocumentInput{UserID: userID, WorkspaceID: workspaceID, Title: "Doc", Content: "body"})
		if err == nil || !strings.Contains(err.Error(), "create document entity") {
			t.Fatalf("expected document entity error, got %v", err)
		}
	})

	t.Run("newWorkspaceFn create", func(t *testing.T) {
		orig := newWorkspaceFn
		newWorkspaceFn = func(uuid.UUID, string, time.Time) (entity.Workspace, error) {
			return entity.Workspace{}, entityErr
		}
		t.Cleanup(func() { newWorkspaceFn = orig })

		userID := uuid.New()
		users := newStubUserRepoFull(map[uuid.UUID]*entity.User{
			userID: {ID: userID, Status: entity.UserStatusActive, CreatedAt: usecaseTestNow(), UpdatedAt: usecaseTestNow()},
		})
		uc := newWorkspaceUseCase(users, &stubWorkspaceRepoFull{}, &mapMemberRepo{}, nil, nil)
		_, err := uc.CreateWorkspace(ctx, CreateWorkspaceInput{UserID: userID, Name: "Team"})
		if err == nil || !strings.Contains(err.Error(), "create workspace entity") {
			t.Fatalf("expected workspace entity error, got %v", err)
		}
	})

	t.Run("newWorkspaceMemberFn create workspace", func(t *testing.T) {
		orig := newWorkspaceMemberFn
		newWorkspaceMemberFn = func(uuid.UUID, uuid.UUID, entity.WorkspaceRole, time.Time) (entity.WorkspaceMember, error) {
			return entity.WorkspaceMember{}, entityErr
		}
		t.Cleanup(func() { newWorkspaceMemberFn = orig })

		userID := uuid.New()
		users := newStubUserRepoFull(map[uuid.UUID]*entity.User{
			userID: {ID: userID, Status: entity.UserStatusActive, CreatedAt: usecaseTestNow(), UpdatedAt: usecaseTestNow()},
		})
		uc := newWorkspaceUseCase(users, &stubWorkspaceRepoFull{}, &mapMemberRepo{}, nil, nil)
		_, err := uc.CreateWorkspace(ctx, CreateWorkspaceInput{UserID: userID, Name: "Team"})
		if err == nil || !strings.Contains(err.Error(), "create workspace host member") {
			t.Fatalf("expected workspace member entity error, got %v", err)
		}
	})

	t.Run("newWorkspaceMemberFn add member", func(t *testing.T) {
		orig := newWorkspaceMemberFn
		newWorkspaceMemberFn = func(uuid.UUID, uuid.UUID, entity.WorkspaceRole, time.Time) (entity.WorkspaceMember, error) {
			return entity.WorkspaceMember{}, entityErr
		}
		t.Cleanup(func() { newWorkspaceMemberFn = orig })

		hostID, workspaceID, targetID, access := memberFixture()
		users := newStubUserRepoFull(map[uuid.UUID]*entity.User{
			targetID: {ID: targetID, Status: entity.UserStatusActive, Email: "target@example.com"},
		})
		uc := NewMemberUseCase(users, &mapMemberRepo{}, &mockAuditLogRepo{}, &mockTransactionManager{}, access, nil)
		uc.now = usecaseTestNow
		_, err := uc.AddMember(ctx, AddMemberInput{UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID})
		if err == nil || !strings.Contains(err.Error(), "create workspace member entity") {
			t.Fatalf("expected add member entity error, got %v", err)
		}
	})

	t.Run("newDocumentVersionFn paths", func(t *testing.T) {
		userID, workspaceID, doc := versionFixture()
		docs := &stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}}
		orig := newDocumentVersionFn
		newDocumentVersionFn = func(entity.Document, string, entity.DocumentVersionType, uuid.UUID, *uuid.UUID, time.Time) (entity.DocumentVersion, error) {
			return entity.DocumentVersion{}, entityErr
		}
		t.Cleanup(func() { newDocumentVersionFn = orig })

		uc := newVersionUseCase(docs, &stubVersionRepo{}, docAccess(workspaceID), nil, nil)
		if err := uc.CreateVersion(ctx, CreateVersionInput{DocumentID: doc.ID, Content: "x", CreatedBy: userID, VersionType: entity.DocumentVersionAutoSave}); err == nil || !strings.Contains(err.Error(), "create document version entity") {
			t.Fatalf("create version entity: %v", err)
		}

		cache := &stubDocumentCache{content: map[uuid.UUID]DocumentContentState{doc.ID: {Content: "edited", UpdatedBy: userID, UpdatedAt: usecaseTestNow()}}, hasState: true}
		vuc := NewVersionUseCase(docs, &stubVersionRepo{}, &mockTransactionManager{}, docAccess(workspaceID), cache, nil, &mockAuditLogRepo{})
		vuc.now = usecaseTestNow
		if _, err := vuc.SaveManualVersion(ctx, SaveManualVersionInput{UserID: userID, DocumentID: doc.ID}); err == nil || !strings.Contains(err.Error(), "create manual save version entity") {
			t.Fatalf("manual save entity: %v", err)
		}

		now := usecaseTestNow()
		target, _ := entity.NewDocumentVersion(doc, "restored", entity.DocumentVersionManualSave, userID, nil, now)
		versions := &stubVersionRepo{byID: map[uuid.UUID]*entity.DocumentVersion{target.ID: &target}}
		ruc := NewVersionUseCase(docs, versions, &mockTransactionManager{}, docAccess(workspaceID), nil, nil, &mockAuditLogRepo{})
		ruc.now = usecaseTestNow
		if _, err := ruc.RestoreVersion(ctx, RestoreVersionInput{UserID: userID, DocumentID: doc.ID, VersionID: target.ID}); err == nil || !strings.Contains(err.Error(), "create before_restore version") {
			t.Fatalf("restore before entity: %v", err)
		}
	})

	t.Run("newRefreshTokenEntityFn refresh", func(t *testing.T) {
		now := usecaseTestNow()
		userID := uuid.New()
		token := activeRefreshToken(userID, now)
		orig := newRefreshTokenEntityFn
		newRefreshTokenEntityFn = func(uuid.UUID, string, uuid.UUID, time.Time, time.Time) (entity.RefreshToken, error) {
			return entity.RefreshToken{}, entityErr
		}
		t.Cleanup(func() { newRefreshTokenEntityFn = orig })

		users := newStubUserRepoFull(map[uuid.UUID]*entity.User{userID: {ID: userID, Status: entity.UserStatusActive, AuthVersion: 1}})
		refresh := &trackingRefreshRepo{stubRefreshRepoFull: stubRefreshRepoFull{byHash: map[string]*entity.RefreshToken{testTokenHash: &token}}}
		uc := newRefreshAuthUseCaseWithAudit(users, refresh, &mockAuthService{}, &mockAuditLogRepo{})
		_, err := uc.Refresh(ctx, RefreshInput{RefreshToken: "raw-token"})
		if err == nil || !strings.Contains(err.Error(), "create next refresh token") {
			t.Fatalf("expected refresh token entity error, got %v", err)
		}
	})
}

func TestCoverage_Hooks_AuthTokenOps(t *testing.T) {
	ctx := context.Background()

	t.Run("rotate refresh token error", func(t *testing.T) {
		now := usecaseTestNow()
		userID := uuid.New()
		token := activeRefreshToken(userID, now)
		orig := rotateRefreshTokenFn
		rotateRefreshTokenFn = func(*entity.RefreshToken, uuid.UUID, time.Time) error {
			return fmt.Errorf("rotate failed")
		}
		t.Cleanup(func() { rotateRefreshTokenFn = orig })

		users := newStubUserRepoFull(map[uuid.UUID]*entity.User{userID: {ID: userID, Status: entity.UserStatusActive, AuthVersion: 1}})
		refresh := &trackingRefreshRepo{stubRefreshRepoFull: stubRefreshRepoFull{byHash: map[string]*entity.RefreshToken{testTokenHash: &token}}}
		uc := newRefreshAuthUseCaseWithAudit(users, refresh, &mockAuthService{}, &mockAuditLogRepo{})
		_, err := uc.Refresh(ctx, RefreshInput{RefreshToken: "raw-token"})
		if !errors.Is(err, ErrRefreshTokenRevoked) {
			t.Fatalf("expected ErrRefreshTokenRevoked, got %v", err)
		}
	})

	t.Run("revoke refresh token error on logout", func(t *testing.T) {
		now := usecaseTestNow()
		userID := uuid.New()
		token := activeRefreshToken(userID, now)
		orig := revokeRefreshTokenFn
		revokeRefreshTokenFn = func(*entity.RefreshToken, time.Time) error {
			return fmt.Errorf("revoke failed")
		}
		t.Cleanup(func() { revokeRefreshTokenFn = orig })

		refresh := &trackingRefreshRepo{stubRefreshRepoFull: stubRefreshRepoFull{byHash: map[string]*entity.RefreshToken{testTokenHash: &token}}}
		uc := newRefreshAuthUseCaseWithAudit(newStubUserRepoFull(map[uuid.UUID]*entity.User{}), refresh, &mockAuthService{}, &mockAuditLogRepo{})
		_, err := uc.Logout(ctx, LogoutInput{
			UserID: userID, AccessTokenJTI: uuid.New(), AccessTokenExp: now.Add(time.Hour),
			RefreshToken: "raw-token", CSRFValidated: true,
		})
		if err == nil || !strings.Contains(err.Error(), "revoke refresh token") {
			t.Fatalf("expected revoke error, got %v", err)
		}
	})

	t.Run("refresh output nil", func(t *testing.T) {
		uc := NewAuthUseCase(&stubUserRepoFull{}, &mockRefreshTokenRepo{}, &mockAuditLogRepo{}, &mockAuthService{}, noopSuccessTransactionManager{}, time.Hour)
		uc.now = usecaseTestNow
		_, err := uc.Refresh(ctx, RefreshInput{RefreshToken: "token"})
		if err == nil || !strings.Contains(err.Error(), "refresh token transaction completed without output") {
			t.Fatalf("expected output nil error, got %v", err)
		}
	})

	t.Run("refresh reused audit entity error", func(t *testing.T) {
		now := usecaseTestNow()
		userID := uuid.New()
		reused := activeRefreshToken(userID, now)
		reused.Status = entity.RefreshTokenStatusRotated
		orig := newAuditLogFn
		newAuditLogFn = func(*uuid.UUID, string, string, *uuid.UUID, any, time.Time) (entity.AuditLog, error) {
			return entity.AuditLog{}, fmt.Errorf("audit entity failed")
		}
		t.Cleanup(func() { newAuditLogFn = orig })

		users := newStubUserRepoFull(map[uuid.UUID]*entity.User{userID: {ID: userID, Status: entity.UserStatusActive, AuthVersion: 1}})
		refresh := &trackingRefreshRepo{stubRefreshRepoFull: stubRefreshRepoFull{byHash: map[string]*entity.RefreshToken{testTokenHash: &reused}}}
		uc := newRefreshAuthUseCaseWithAudit(users, refresh, &mockAuthService{}, &mockAuditLogRepo{})
		_, err := uc.Refresh(ctx, RefreshInput{RefreshToken: "raw-token"})
		if err == nil || !strings.Contains(err.Error(), "create refresh token reused audit log entity") {
			t.Fatalf("expected reused audit entity error, got %v", err)
		}
	})
}

func TestCoverage_Hooks_AccountUserOps(t *testing.T) {
	ctx := context.Background()
	opErr := fmt.Errorf("user op failed")
	hostID, workspaceID, targetID, member, users := accountFixture()

	t.Run("suspend generic error", func(t *testing.T) {
		orig := suspendUserFn
		suspendUserFn = func(*entity.User, time.Time) error { return opErr }
		t.Cleanup(func() { suspendUserFn = orig })

		uc := NewAccountUseCase(users, &trackingRefreshTokenRepo{}, member, &mockWorkspaceRepo{}, &mockAuditLogRepo{}, &mockTransactionManager{}, &mockAccessCheck{}, nil)
		uc.now = accountTestNow
		_, err := uc.SuspendAccount(ctx, SuspendAccountInput{UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID})
		if err == nil || !strings.Contains(err.Error(), "suspend user") {
			t.Fatalf("expected suspend user error, got %v", err)
		}
	})

	t.Run("reactivate generic error", func(t *testing.T) {
		suspended := memberUser(targetID, entity.UserStatusActive)
		_ = suspended.Suspend(accountTestNow())
		testUsers := &mockAccountUserRepo{users: map[uuid.UUID]*entity.User{targetID: suspended}}
		orig := reactivateUserFn
		reactivateUserFn = func(*entity.User, time.Time) error { return opErr }
		t.Cleanup(func() { reactivateUserFn = orig })

		uc := NewAccountUseCase(testUsers, &trackingRefreshTokenRepo{}, member, &mockWorkspaceRepo{}, &mockAuditLogRepo{}, &mockTransactionManager{}, &mockAccessCheck{}, nil)
		uc.now = accountTestNow
		_, err := uc.ReactivateAccount(ctx, ReactivateAccountInput{UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID})
		if err == nil || !strings.Contains(err.Error(), "reactivate user") {
			t.Fatalf("expected reactivate user error, got %v", err)
		}
	})

	t.Run("delete generic error", func(t *testing.T) {
		suspended := memberUser(targetID, entity.UserStatusActive)
		_ = suspended.Suspend(accountTestNow())
		testUsers := &mockAccountUserRepo{users: map[uuid.UUID]*entity.User{targetID: suspended}}
		orig := logicalDeleteUserFn
		logicalDeleteUserFn = func(*entity.User, string, string, time.Time) error { return opErr }
		t.Cleanup(func() { logicalDeleteUserFn = orig })

		uc := NewAccountUseCase(testUsers, &trackingRefreshTokenRepo{}, member, &mockWorkspaceRepo{}, &mockAuditLogRepo{}, &mockTransactionManager{}, &mockAccessCheck{}, nil)
		uc.now = accountTestNow
		_, err := uc.DeleteAccount(ctx, DeleteAccountInput{UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID})
		if err == nil || !strings.Contains(err.Error(), "logical delete user") {
			t.Fatalf("expected logical delete user error, got %v", err)
		}
	})
}

func TestCoverage_Hooks_WorkspaceJSON(t *testing.T) {
	orig := jsonMarshalFn
	jsonMarshalFn = func(any) ([]byte, error) { return nil, fmt.Errorf("marshal failed") }
	t.Cleanup(func() { jsonMarshalFn = orig })

	hostID, workspaceID, users, workspaces, members := setupWorkspaceFixture()
	uc := NewWorkspaceUseCase(users, workspaces, members, &mockAuditLogRepo{}, &mockTransactionManager{}, &stubDocumentFlush{}, nil)
	uc.now = usecaseTestNow
	_, err := uc.DeleteWorkspace(context.Background(), DeleteWorkspaceInput{UserID: hostID, WorkspaceID: workspaceID, Reason: "done"})
	if err == nil || !strings.Contains(err.Error(), "marshal delete metadata") {
		t.Fatalf("expected marshal error, got %v", err)
	}
}

func TestCoverage_Hooks_LoginAuditSilentReturns(t *testing.T) {
	orig := newAuditLogFn
	newAuditLogFn = func(*uuid.UUID, string, string, *uuid.UUID, any, time.Time) (entity.AuditLog, error) {
		return entity.AuditLog{}, fmt.Errorf("audit entity failed")
	}
	t.Cleanup(func() { newAuditLogFn = orig })

	uc := NewAuthUseCase(&stubUserRepoFull{}, &mockRefreshTokenRepo{}, &mockAuditLogRepo{}, &mockAuthService{}, &mockTransactionManager{}, time.Hour)
	uc.now = usecaseTestNow
	uc.recordLoginRateLimitedAudit(context.Background(), nil, nil, LoginInput{IPAddress: "127.0.0.1"})
	uc.recordLoginFailedAudit(context.Background(), nil, nil, LoginInput{IPAddress: "127.0.0.1"}, "bad password")
}

func TestCoverage_Hooks_BatchPrune(t *testing.T) {
	ctx := context.Background()

	t.Run("prune error logged", func(t *testing.T) {
		dir := t.TempDir()
		binDir := t.TempDir()
		scriptPath := filepath.Join(binDir, "pg_dump")
		script := "#!/bin/sh\nshift\nwhile [ $# -gt 0 ]; do case \"$1\" in -f) OUT=\"$2\"; shift 2;; *) shift;; esac; done\necho backup > \"$OUT\"\n"
		if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
		oldPath := os.Getenv("PATH")
		t.Cleanup(func() { _ = os.Setenv("PATH", oldPath) })
		_ = os.Setenv("PATH", binDir+string(os.PathListSeparator)+oldPath)

		orig := backupPruneOldBackups
		backupPruneOldBackups = func(*DatabaseBackupUseCase, time.Time) (int, error) {
			return 0, fmt.Errorf("prune failed")
		}
		t.Cleanup(func() { backupPruneOldBackups = orig })

		uc := NewDatabaseBackupUseCase("postgres://x", dir, time.Hour, &mockAuditLogRepo{})
		uc.now = usecaseTestNow
		if err := uc.RunOnce(ctx); err != nil {
			t.Fatalf("RunOnce should continue after prune error: %v", err)
		}
	})

	t.Run("stat backup file error", func(t *testing.T) {
		dir := t.TempDir()
		origInfo := backupFileInfoFn
		backupFileInfoFn = func(os.DirEntry) (os.FileInfo, error) { return nil, fmt.Errorf("stat failed") }
		t.Cleanup(func() { backupFileInfoFn = origInfo })

		if err := os.WriteFile(filepath.Join(dir, "notehub-backup-old.sql"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		oldTime := usecaseTestNow().Add(-48 * time.Hour)
		if err := os.Chtimes(filepath.Join(dir, "notehub-backup-old.sql"), oldTime, oldTime); err != nil {
			t.Fatal(err)
		}
		uc := NewDatabaseBackupUseCase("postgres://x", dir, time.Hour, &mockAuditLogRepo{})
		_, err := uc.pruneOldBackups(usecaseTestNow())
		if err == nil || !strings.Contains(err.Error(), "stat backup file") {
			t.Fatalf("expected stat error, got %v", err)
		}
	})
}

func TestCoverage_Hooks_RemainingBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("register validation via Register", func(t *testing.T) {
		uc := NewAuthUseCase(&stubUserRepoFull{}, &mockRefreshTokenRepo{}, &mockAuditLogRepo{}, &mockAuthService{}, &mockTransactionManager{}, time.Hour)
		_, err := uc.Register(ctx, RegisterInput{Name: "", Email: "bad", Password: "short"})
		if err == nil {
			t.Fatal("expected validation error")
		}
	})

	t.Run("update document pre-tx errors", func(t *testing.T) {
		userID, workspaceID, doc := versionFixture()
		docs := &stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}, findErr: fmt.Errorf("find failed")}
		uc := newDocumentUseCase(docs, docAccess(workspaceID), nil, nil, nil)
		_, err := uc.UpdateDocument(ctx, UpdateDocumentInput{UserID: userID, DocumentID: doc.ID, Title: "Renamed"})
		if err == nil {
			t.Fatal("expected load error")
		}

		docs = &stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}}
		uc = newDocumentUseCase(docs, &stubAccessCheck{err: ErrWorkspaceAccessDenied}, nil, nil, nil)
		_, err = uc.UpdateDocument(ctx, UpdateDocumentInput{UserID: userID, DocumentID: doc.ID, Title: "Renamed"})
		if !errors.Is(err, ErrWorkspaceAccessDenied) {
			t.Fatalf("expected access denied, got %v", err)
		}

		uc = newDocumentUseCase(docs, docAccess(workspaceID), nil, nil, nil)
		_, err = uc.UpdateDocument(ctx, UpdateDocumentInput{UserID: uuid.Nil, DocumentID: doc.ID, Title: "Renamed"})
		if err == nil {
			t.Fatal("expected rename error")
		}
	})

	t.Run("delete document pre-tx and audit entity already covered", func(t *testing.T) {
		userID, _, doc := versionFixture()
		docs := &stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}}
		uc := newDocumentUseCase(docs, &stubAccessCheck{err: ErrWorkspaceAccessDenied}, nil, nil, nil)
		_, err := uc.DeleteDocument(ctx, DeleteDocumentInput{UserID: userID, DocumentID: doc.ID})
		if !errors.Is(err, ErrWorkspaceAccessDenied) {
			t.Fatalf("expected access denied, got %v", err)
		}
	})

	t.Run("version load and authorize errors", func(t *testing.T) {
		userID, workspaceID, doc := versionFixture()
		docs := &stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}, findErr: fmt.Errorf("find failed")}
		uc := newVersionUseCase(docs, &stubVersionRepo{}, docAccess(workspaceID), nil, nil)
		_, err := uc.GetVersion(ctx, GetVersionInput{UserID: userID, DocumentID: doc.ID, VersionID: uuid.New()})
		if err == nil {
			t.Fatal("expected get version load error")
		}

		docs = &stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}}
		uc = newVersionUseCase(docs, &stubVersionRepo{}, &stubAccessCheck{err: ErrWorkspaceAccessDenied}, nil, nil)
		_, err = uc.ListVersions(ctx, ListVersionsInput{UserID: userID, DocumentID: doc.ID})
		if !errors.Is(err, ErrWorkspaceAccessDenied) {
			t.Fatalf("expected list access denied, got %v", err)
		}

		now := usecaseTestNow()
		target, _ := entity.NewDocumentVersion(doc, "restored", entity.DocumentVersionManualSave, userID, nil, now)
		versions := &findVersionErrRepo{stubVersionRepo: stubVersionRepo{byID: map[uuid.UUID]*entity.DocumentVersion{target.ID: &target}}, findErr: fmt.Errorf("find failed")}
		uc = NewVersionUseCase(docs, versions, &mockTransactionManager{}, docAccess(workspaceID), nil, nil, &mockAuditLogRepo{})
		_, err = uc.RestoreVersion(ctx, RestoreVersionInput{UserID: userID, DocumentID: doc.ID, VersionID: target.ID})
		if err == nil || !strings.Contains(err.Error(), "find restore version") {
			t.Fatalf("expected find restore version error, got %v", err)
		}
	})

	t.Run("workspace access and validation", func(t *testing.T) {
		hostID, workspaceID, users, workspaces, members := setupWorkspaceFixture()
		uc := newWorkspaceUseCase(users, workspaces, members, nil, nil)
		_, err := uc.GetWorkspace(ctx, GetWorkspaceInput{UserID: hostID, WorkspaceID: uuid.New()})
		if !errors.Is(err, ErrWorkspaceNotFound) {
			t.Fatalf("expected not found, got %v", err)
		}

		suspendedID := uuid.New()
		users.users[suspendedID] = &entity.User{ID: suspendedID, Status: entity.UserStatusSuspended}
		_, err = uc.ListWorkspaces(ctx, ListWorkspacesInput{UserID: suspendedID})
		if !errors.Is(err, ErrAccountSuspended) {
			t.Fatalf("expected suspended, got %v", err)
		}

		_, err = uc.UpdateWorkspace(ctx, UpdateWorkspaceInput{UserID: hostID, WorkspaceID: workspaceID, Name: ""})
		if !errors.Is(err, ErrValidation) {
			t.Fatalf("expected validation, got %v", err)
		}

		_, err = uc.DeleteWorkspace(ctx, DeleteWorkspaceInput{UserID: hostID, WorkspaceID: workspaceID, Reason: ""})
		if !errors.Is(err, ErrValidation) {
			t.Fatalf("expected delete validation, got %v", err)
		}

		_, err = uc.CheckWorkspaceAccess(ctx, CheckWorkspaceAccessInput{UserID: suspendedID, WorkspaceID: workspaceID})
		if !errors.Is(err, ErrAccountSuspended) {
			t.Fatalf("expected check access suspended, got %v", err)
		}
	})

	t.Run("member require host", func(t *testing.T) {
		hostID, workspaceID, targetID, _ := memberFixture()
		uc := NewMemberUseCase(&stubUserRepoFull{}, &mapMemberRepo{}, &mockAuditLogRepo{}, &mockTransactionManager{}, &stubAccessCheck{err: ErrHostPermissionRequired}, nil)
		_, err := uc.AddMember(ctx, AddMemberInput{UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID})
		if !errors.Is(err, ErrHostPermissionRequired) {
			t.Fatalf("expected host required on add, got %v", err)
		}
		if err := uc.RemoveMember(ctx, RemoveMemberInput{UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID}); !errors.Is(err, ErrHostPermissionRequired) {
			t.Fatalf("expected host required on remove, got %v", err)
		}
	})

	t.Run("autosave persist branches", func(t *testing.T) {
		docID, docs, cache := autosaveFixture()

		recheck := &recheckDirtyCache{stubDocumentCache: *cache, recheckDirtyErr: true}
		uc := newAutoSaveUseCaseForCache(docs, &stubVersionRepo{}, recheck, &stubAutoSaveLock{locked: true})
		if err := uc.SaveDocument(ctx, docID); err == nil || !strings.Contains(err.Error(), "recheck dirty flag") {
			t.Fatalf("recheck dirty: %v", err)
		}

		notDirty := &clearOnRecheckCache{stubDocumentCache: *cache}
		uc = newAutoSaveUseCaseForCache(docs, &stubVersionRepo{}, notDirty, &stubAutoSaveLock{locked: true})
		if err := uc.SaveDocument(ctx, docID); err != nil {
			t.Fatalf("not dirty after recheck: %v", err)
		}

		revCache := &revisionFailCache{stubDocumentCache: *cache}
		uc = newAutoSaveUseCaseForCache(docs, &stubVersionRepo{}, revCache, &stubAutoSaveLock{locked: true})
		if err := uc.SaveDocument(ctx, docID); err == nil || !strings.Contains(err.Error(), "read start revision") {
			t.Fatalf("revision error: %v", err)
		}

		stateErr := &contentStateErrCache{stubDocumentCache: *cache}
		uc = newAutoSaveUseCaseForCache(docs, &stubVersionRepo{}, stateErr, &stubAutoSaveLock{locked: true})
		if err := uc.SaveDocument(ctx, docID); err == nil || !strings.Contains(err.Error(), "load cached content") {
			t.Fatalf("content state error: %v", err)
		}

		orig := newDocumentVersionFn
		newDocumentVersionFn = func(entity.Document, string, entity.DocumentVersionType, uuid.UUID, *uuid.UUID, time.Time) (entity.DocumentVersion, error) {
			return entity.DocumentVersion{}, fmt.Errorf("version entity failed")
		}
		t.Cleanup(func() { newDocumentVersionFn = orig })
		uc = newAutoSaveUseCaseForCache(docs, &stubVersionRepo{}, cache, &stubAutoSaveLock{locked: true})
		if err := uc.SaveDocument(ctx, docID); err == nil || !strings.Contains(err.Error(), "create document version entity") {
			t.Fatalf("autosave version entity: %v", err)
		}
	})

	t.Run("websocket register prepare error and load doc error", func(t *testing.T) {
		userID, _, doc, _, _ := wsFixture()
		docs := &stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}}
		uc := newWebSocketUseCase(docs, &stubUserRepoFull{}, &stubDocumentCache{}, &stubWebSocketSessions{}, &stubDocumentEditors{}, &stubAccessCheck{err: ErrWorkspaceAccessDenied})
		_, err := uc.RegisterConnection(ctx, RegisterWebSocketConnectionInput{UserID: userID, DocumentID: doc.ID, ConnectionID: uuid.New()})
		if !errors.Is(err, ErrWorkspaceAccessDenied) {
			t.Fatalf("register prepare error: %v", err)
		}

		userID, _, doc, _, access := wsFixture()
		docs = &stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}, findErr: fmt.Errorf("find failed")}
		uc = newWebSocketUseCase(docs, &stubUserRepoFull{}, &stubDocumentCache{}, &stubWebSocketSessions{}, &stubDocumentEditors{}, access)
		_, err = uc.PrepareConnection(ctx, PrepareWebSocketConnectionInput{UserID: userID, DocumentID: doc.ID})
		if err == nil {
			t.Fatal("expected load doc error")
		}
	})
}

type findVersionErrRepo struct {
	stubVersionRepo
	findErr error
}

func (r *findVersionErrRepo) FindByID(context.Context, uuid.UUID) (*entity.DocumentVersion, bool, error) {
	return nil, false, r.findErr
}

type createFailOnSecondVersionRepo struct {
	stubVersionRepo
	createCalls int
	failOnCall  int
}

func (r *createFailOnSecondVersionRepo) Create(ctx context.Context, version *entity.DocumentVersion) error {
	r.createCalls++
	if r.createCalls == r.failOnCall {
		return fmt.Errorf("save restore version failed")
	}
	return r.stubVersionRepo.Create(ctx, version)
}

func (r *createFailOnSecondVersionRepo) FindByID(ctx context.Context, id uuid.UUID) (*entity.DocumentVersion, bool, error) {
	return r.stubVersionRepo.FindByID(ctx, id)
}

type clearOnRecheckCache struct {
	stubDocumentCache
	checks int
}

func (c *clearOnRecheckCache) IsDirty(ctx context.Context, id uuid.UUID) (bool, error) {
	c.checks++
	if c.checks > 1 {
		return false, nil
	}
	return c.stubDocumentCache.IsDirty(ctx, id)
}

type revisionFailCache struct {
	stubDocumentCache
}

func (c *revisionFailCache) GetRevision(context.Context, uuid.UUID) (uint64, error) {
	return 0, fmt.Errorf("revision failed")
}

type contentStateErrCache struct {
	stubDocumentCache
}

func (c *contentStateErrCache) GetContentState(context.Context, uuid.UUID) (DocumentContentState, bool, error) {
	return DocumentContentState{}, false, fmt.Errorf("state failed")
}

func TestCoverage_Hooks_FinalGaps(t *testing.T) {
	ctx := context.Background()

	t.Run("account delete early paths", func(t *testing.T) {
		hostID, workspaceID, targetID, member, _ := accountFixture()

		uc := NewAccountUseCase(&mockAccountUserRepo{users: map[uuid.UUID]*entity.User{}}, &trackingRefreshTokenRepo{}, member, &mockWorkspaceRepo{}, &mockAuditLogRepo{}, &mockTransactionManager{}, &mockAccessCheck{err: ErrHostPermissionRequired}, nil)
		uc.now = accountTestNow
		_, err := uc.DeleteAccount(ctx, DeleteAccountInput{UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID})
		if !errors.Is(err, ErrHostPermissionRequired) {
			t.Fatalf("expected host required, got %v", err)
		}

		uc = NewAccountUseCase(&mockAccountUserRepo{users: map[uuid.UUID]*entity.User{}}, &trackingRefreshTokenRepo{}, member, &mockWorkspaceRepo{}, &mockAuditLogRepo{}, &mockTransactionManager{}, &mockAccessCheck{}, nil)
		uc.now = accountTestNow
		_, err = uc.DeleteAccount(ctx, DeleteAccountInput{UserID: hostID, WorkspaceID: workspaceID, TargetUserID: targetID})
		if !errors.Is(err, ErrTargetUserNotFound) {
			t.Fatalf("expected target not found in tx, got %v", err)
		}
	})

	t.Run("document update invalid title and delete logical delete", func(t *testing.T) {
		userID, workspaceID, doc := versionFixture()
		docs := &stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}}
		uc := newDocumentUseCase(docs, docAccess(workspaceID), nil, nil, nil)
		_, err := uc.UpdateDocument(ctx, UpdateDocumentInput{UserID: userID, DocumentID: doc.ID, Title: ""})
		if !errors.Is(err, ErrValidation) {
			t.Fatalf("expected title validation, got %v", err)
		}

		uc = NewDocumentUseCase(docs, &mockAuditLogRepo{}, &mockTransactionManager{}, docAccess(workspaceID), nil, nil, nil)
		uc.now = usecaseTestNow
		_, err = uc.DeleteDocument(ctx, DeleteDocumentInput{UserID: uuid.Nil, DocumentID: doc.ID})
		if err == nil {
			t.Fatal("expected logical delete error")
		}
	})

	t.Run("autosave replace content and conflict", func(t *testing.T) {
		docID, docs, cache := autosaveFixture()
		orig := replaceDocumentContentFn
		replaceDocumentContentFn = func(*entity.Document, string, uuid.UUID, uint64, time.Time) error {
			return entity.ErrDocumentConflict
		}
		t.Cleanup(func() { replaceDocumentContentFn = orig })
		uc := newAutoSaveUseCaseForCache(docs, &stubVersionRepo{}, cache, &stubAutoSaveLock{locked: true})
		if err := uc.SaveDocument(ctx, docID); err == nil || !strings.Contains(err.Error(), "document revision conflict") {
			t.Fatalf("expected conflict error, got %v", err)
		}

		docID, docs, cache = autosaveFixture()
		replaceDocumentContentFn = func(*entity.Document, string, uuid.UUID, uint64, time.Time) error {
			return fmt.Errorf("replace failed")
		}
		t.Cleanup(func() { replaceDocumentContentFn = orig })
		uc = newAutoSaveUseCaseForCache(docs, &stubVersionRepo{}, cache, &stubAutoSaveLock{locked: true})
		if err := uc.SaveDocument(ctx, docID); err == nil {
			t.Fatal("expected replace content error")
		}

		docID, docs, cache = autosaveFixture()
		replaceDocumentContentFn = orig
		origVer := newDocumentVersionFn
		newDocumentVersionFn = func(entity.Document, string, entity.DocumentVersionType, uuid.UUID, *uuid.UUID, time.Time) (entity.DocumentVersion, error) {
			return entity.DocumentVersion{}, fmt.Errorf("autosave version entity failed")
		}
		t.Cleanup(func() { newDocumentVersionFn = origVer })
		uc = newAutoSaveUseCaseForCache(docs, &stubVersionRepo{}, cache, &stubAutoSaveLock{locked: true})
		if err := uc.SaveDocument(ctx, docID); err == nil || !strings.Contains(err.Error(), "create document version entity") {
			t.Fatalf("expected autosave tx version entity error, got %v", err)
		}

		docID, docs, cache = autosaveFixture()
		uc = newAutoSaveUseCaseForCache(docs, &stubVersionRepo{}, &contentStateErrCache{stubDocumentCache: *cache}, &stubAutoSaveLock{locked: true})
		if err := uc.SaveDocument(ctx, docID); err == nil || !strings.Contains(err.Error(), "load cached content") {
			t.Fatalf("expected load cached content error in persist, got %v", err)
		}
	})

	t.Run("websocket load doc not found", func(t *testing.T) {
		userID, _, _, _, access := wsFixture()
		uc := newWebSocketUseCase(&stubDocumentRepo{}, &stubUserRepoFull{}, &stubDocumentCache{}, &stubWebSocketSessions{}, &stubDocumentEditors{}, access)
		_, err := uc.PrepareConnection(ctx, PrepareWebSocketConnectionInput{UserID: userID, DocumentID: uuid.New()})
		if !errors.Is(err, ErrDocumentNotFound) {
			t.Fatalf("expected not found, got %v", err)
		}
	})

	t.Run("version manual save tx branches", func(t *testing.T) {
		userID, workspaceID, doc := versionFixture()
		base := &stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}}
		cache := &stubDocumentCache{content: map[uuid.UUID]DocumentContentState{doc.ID: {Content: "edited", UpdatedBy: userID, UpdatedAt: usecaseTestNow()}}, hasState: true}

		cases := []struct {
			name string
			docs *stubDocumentRepo
			vers *stubVersionRepo
			want string
		}{
			{"find for update", &stubDocumentRepo{byID: base.byID, findForUpdateErr: fmt.Errorf("lock failed")}, &stubVersionRepo{}, "find document for manual save"},
			{"not found", &stubDocumentRepo{byID: base.byID, findForUpdateMiss: true}, &stubVersionRepo{}, "document not found"},
			{"deleted", func() *stubDocumentRepo {
				deleted := doc
				now := usecaseTestNow()
				deleted.DeletedAt = &now
				return &stubDocumentRepo{byID: base.byID, findForUpdateDoc: &deleted}
			}(), &stubVersionRepo{}, "document deleted"},
			{"version save", base, &stubVersionRepo{err: fmt.Errorf("save failed")}, "save manual save version"},
			{"replace content", func() *stubDocumentRepo {
				locked := doc
				return &stubDocumentRepo{byID: base.byID, findForUpdateDoc: &locked}
			}(), &stubVersionRepo{}, "replace failed"},
			{"cache set", base, &stubVersionRepo{}, "update document cache after manual save"},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				c := cache
				if tc.name == "cache set" {
					c = &stubDocumentCache{content: map[uuid.UUID]DocumentContentState{doc.ID: {Content: "edited", UpdatedBy: userID, UpdatedAt: usecaseTestNow()}}, hasState: true, setErr: fmt.Errorf("cache failed")}
				}
				if tc.name == "replace content" {
					orig := replaceDocumentContentFn
					replaceDocumentContentFn = func(*entity.Document, string, uuid.UUID, uint64, time.Time) error {
						return fmt.Errorf("replace failed")
					}
					t.Cleanup(func() { replaceDocumentContentFn = orig })
				}
				uc := NewVersionUseCase(tc.docs, tc.vers, &mockTransactionManager{}, docAccess(workspaceID), c, nil, &mockAuditLogRepo{})
				uc.now = usecaseTestNow
				_, err := uc.SaveManualVersion(ctx, SaveManualVersionInput{UserID: userID, DocumentID: doc.ID})
				if err == nil || !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("expected %q, got %v", tc.want, err)
				}
			})
		}
	})

	t.Run("version restore remaining branches", func(t *testing.T) {
		userID, workspaceID, doc := versionFixture()
		now := usecaseTestNow()
		target, _ := entity.NewDocumentVersion(doc, "restored", entity.DocumentVersionManualSave, userID, nil, now)
		docs := &stubDocumentRepo{byID: map[uuid.UUID]*entity.Document{doc.ID: &doc}}
		versions := &stubVersionRepo{byID: map[uuid.UUID]*entity.DocumentVersion{target.ID: &target}}

		_, err := newVersionUseCase(&stubDocumentRepo{findErr: fmt.Errorf("find failed")}, versions, docAccess(workspaceID), nil, nil).RestoreVersion(ctx, RestoreVersionInput{UserID: userID, DocumentID: doc.ID, VersionID: target.ID})
		if err == nil {
			t.Fatal("expected load doc error")
		}

		_, err = newVersionUseCase(docs, versions, &stubAccessCheck{err: ErrWorkspaceAccessDenied}, nil, nil).RestoreVersion(ctx, RestoreVersionInput{UserID: userID, DocumentID: doc.ID, VersionID: target.ID})
		if !errors.Is(err, ErrWorkspaceAccessDenied) {
			t.Fatalf("expected access denied, got %v", err)
		}

		badCache := &stubDocumentCache{getErr: fmt.Errorf("content failed")}
		uc := NewVersionUseCase(docs, versions, &mockTransactionManager{}, docAccess(workspaceID), badCache, nil, &mockAuditLogRepo{})
		uc.now = usecaseTestNow
		_, err = uc.RestoreVersion(ctx, RestoreVersionInput{UserID: userID, DocumentID: doc.ID, VersionID: target.ID})
		if err == nil || !strings.Contains(err.Error(), "load current document content") {
			t.Fatalf("expected current content error, got %v", err)
		}

		uc = NewVersionUseCase(&stubDocumentRepo{byID: docs.byID, findForUpdateMiss: true}, versions, &mockTransactionManager{}, docAccess(workspaceID), nil, nil, &mockAuditLogRepo{})
		uc.now = usecaseTestNow
		_, err = uc.RestoreVersion(ctx, RestoreVersionInput{UserID: userID, DocumentID: doc.ID, VersionID: target.ID})
		if !errors.Is(err, ErrDocumentNotFound) {
			t.Fatalf("expected not found in tx, got %v", err)
		}

		deleted := doc
		deletedAt := usecaseTestNow()
		deleted.DeletedAt = &deletedAt
		uc = NewVersionUseCase(&stubDocumentRepo{byID: docs.byID, findForUpdateDoc: &deleted}, versions, &mockTransactionManager{}, docAccess(workspaceID), nil, nil, &mockAuditLogRepo{})
		uc.now = usecaseTestNow
		_, err = uc.RestoreVersion(ctx, RestoreVersionInput{UserID: userID, DocumentID: doc.ID, VersionID: target.ID})
		if !errors.Is(err, ErrDocumentDeleted) {
			t.Fatalf("expected deleted in tx, got %v", err)
		}

		t.Run("before restore entity error", func(t *testing.T) {
			orig := newDocumentVersionFn
			newDocumentVersionFn = func(entity.Document, string, entity.DocumentVersionType, uuid.UUID, *uuid.UUID, time.Time) (entity.DocumentVersion, error) {
				return entity.DocumentVersion{}, fmt.Errorf("restore entity failed")
			}
			t.Cleanup(func() { newDocumentVersionFn = orig })
			uc := NewVersionUseCase(docs, versions, &mockTransactionManager{}, docAccess(workspaceID), nil, nil, &mockAuditLogRepo{})
			uc.now = usecaseTestNow
			_, err := uc.RestoreVersion(ctx, RestoreVersionInput{UserID: userID, DocumentID: doc.ID, VersionID: target.ID})
			if err == nil || !strings.Contains(err.Error(), "create before_restore version") {
				t.Fatalf("expected before restore version entity error, got %v", err)
			}
		})

		t.Run("restore replace error", func(t *testing.T) {
			orig := replaceDocumentContentFn
			replaceDocumentContentFn = func(*entity.Document, string, uuid.UUID, uint64, time.Time) error {
				return fmt.Errorf("restore replace failed")
			}
			t.Cleanup(func() { replaceDocumentContentFn = orig })
			uc := NewVersionUseCase(docs, versions, &mockTransactionManager{}, docAccess(workspaceID), nil, nil, &mockAuditLogRepo{})
			uc.now = usecaseTestNow
			_, err := uc.RestoreVersion(ctx, RestoreVersionInput{UserID: userID, DocumentID: doc.ID, VersionID: target.ID})
			if err == nil || !strings.Contains(err.Error(), "restore replace failed") {
				t.Fatalf("expected restore replace error, got %v", err)
			}
		})

		t.Run("restore version entity error", func(t *testing.T) {
			calls := 0
			orig := newDocumentVersionFn
			newDocumentVersionFn = func(d entity.Document, content string, vt entity.DocumentVersionType, by uuid.UUID, src *uuid.UUID, now time.Time) (entity.DocumentVersion, error) {
				calls++
				if calls == 2 {
					return entity.DocumentVersion{}, fmt.Errorf("restore version entity failed")
				}
				return entity.NewDocumentVersion(d, content, vt, by, src, now)
			}
			t.Cleanup(func() { newDocumentVersionFn = orig })
			uc := NewVersionUseCase(docs, versions, &mockTransactionManager{}, docAccess(workspaceID), nil, nil, &mockAuditLogRepo{})
			uc.now = usecaseTestNow
			_, err := uc.RestoreVersion(ctx, RestoreVersionInput{UserID: userID, DocumentID: doc.ID, VersionID: target.ID})
			if err == nil || !strings.Contains(err.Error(), "create restore version") {
				t.Fatalf("expected restore version entity error, got %v", err)
			}
		})

		t.Run("save restore version error", func(t *testing.T) {
			createCalls := 0
			failVersions := &createFailOnSecondVersionRepo{
				stubVersionRepo: stubVersionRepo{byID: map[uuid.UUID]*entity.DocumentVersion{target.ID: &target}},
				failOnCall:      2,
			}
			_ = createCalls
			uc := NewVersionUseCase(docs, failVersions, &mockTransactionManager{}, docAccess(workspaceID), nil, nil, &mockAuditLogRepo{})
			uc.now = usecaseTestNow
			_, err := uc.RestoreVersion(ctx, RestoreVersionInput{UserID: userID, DocumentID: doc.ID, VersionID: target.ID})
			if err == nil || !strings.Contains(err.Error(), "save restore version") {
				t.Fatalf("expected save restore version error, got %v", err)
			}
		})
	})

	t.Run("list versions load doc error", func(t *testing.T) {
		userID, workspaceID, doc := versionFixture()
		uc := newVersionUseCase(&stubDocumentRepo{findErr: fmt.Errorf("find failed")}, &stubVersionRepo{}, docAccess(workspaceID), nil, nil)
		_, err := uc.ListVersions(ctx, ListVersionsInput{UserID: userID, DocumentID: doc.ID})
		if err == nil {
			t.Fatal("expected load doc error")
		}
	})

	t.Run("workspace remaining tx paths", func(t *testing.T) {
		hostID, workspaceID, users, workspaces, members := setupWorkspaceFixture()
		newUserID := uuid.New()
		users.users[newUserID] = &entity.User{ID: newUserID, Status: entity.UserStatusActive, CreatedAt: usecaseTestNow(), UpdatedAt: usecaseTestNow()}

		uc := NewWorkspaceUseCase(users, workspaces, members, errAuditLogRepo{err: fmt.Errorf("audit failed")}, &mockTransactionManager{}, nil, nil)
		uc.now = usecaseTestNow
		_, err := uc.CreateWorkspace(ctx, CreateWorkspaceInput{UserID: newUserID, Name: "Another"})
		if err == nil || !strings.Contains(err.Error(), "save audit log") {
			t.Fatalf("expected create audit save error, got %v", err)
		}

		workspaces.findForUpdateMiss = true
		uc = NewWorkspaceUseCase(users, workspaces, members, &mockAuditLogRepo{}, &mockTransactionManager{}, nil, nil)
		uc.now = usecaseTestNow
		_, err = uc.UpdateWorkspace(ctx, UpdateWorkspaceInput{UserID: hostID, WorkspaceID: workspaceID, Name: "New"})
		if !errors.Is(err, ErrWorkspaceNotFound) {
			t.Fatalf("expected not found in update tx, got %v", err)
		}

		hostID, workspaceID, users, workspaces, members = setupWorkspaceFixture()
		now := usecaseTestNow()
		workspaces.byID[workspaceID].DeletedAt = &now
		uc = NewWorkspaceUseCase(users, workspaces, members, &mockAuditLogRepo{}, &mockTransactionManager{}, nil, nil)
		uc.now = usecaseTestNow
		_, err = uc.UpdateWorkspace(ctx, UpdateWorkspaceInput{UserID: hostID, WorkspaceID: workspaceID, Name: "New"})
		if !errors.Is(err, ErrWorkspaceAlreadyDeleted) {
			t.Fatalf("expected already deleted in update tx, got %v", err)
		}

		hostID, workspaceID, users, workspaces, members = setupWorkspaceFixture()
		deletedWS := *workspaces.byID[workspaceID]
		deletedAt := usecaseTestNow()
		deletedWS.DeletedAt = &deletedAt
		workspaces.findForUpdateDoc = &deletedWS
		uc = newWorkspaceUseCase(users, workspaces, members, nil, nil)
		_, err = uc.UpdateWorkspace(ctx, UpdateWorkspaceInput{UserID: hostID, WorkspaceID: workspaceID, Name: "New"})
		if !errors.Is(err, ErrWorkspaceAlreadyDeleted) {
			t.Fatalf("expected already deleted in update tx via for-update doc, got %v", err)
		}

		origRename := renameWorkspaceFn
		renameWorkspaceFn = func(*entity.Workspace, string, time.Time) error { return fmt.Errorf("rename failed") }
		t.Cleanup(func() { renameWorkspaceFn = origRename })
		hostID, workspaceID, users, workspaces, members = setupWorkspaceFixture()
		uc = newWorkspaceUseCase(users, workspaces, members, nil, nil)
		_, err = uc.UpdateWorkspace(ctx, UpdateWorkspaceInput{UserID: hostID, WorkspaceID: workspaceID, Name: "New"})
		if err == nil || !strings.Contains(err.Error(), "rename failed") {
			t.Fatalf("expected rename error, got %v", err)
		}

		hostID, workspaceID, users, workspaces, members = setupWorkspaceFixture()
		workspaces.findForUpdateDoc = &deletedWS
		uc = newWorkspaceUseCase(users, workspaces, members, &stubDocumentFlush{}, nil)
		_, err = uc.DeleteWorkspace(ctx, DeleteWorkspaceInput{UserID: hostID, WorkspaceID: workspaceID, Reason: "done"})
		if !errors.Is(err, ErrWorkspaceAlreadyDeleted) {
			t.Fatalf("expected already deleted in delete tx via for-update doc, got %v", err)
		}

		origDelete := logicalDeleteWorkspaceFn
		logicalDeleteWorkspaceFn = func(*entity.Workspace, uuid.UUID, string, time.Time) error { return fmt.Errorf("logical delete failed") }
		t.Cleanup(func() { logicalDeleteWorkspaceFn = origDelete })
		hostID, workspaceID, users, workspaces, members = setupWorkspaceFixture()
		uc = newWorkspaceUseCase(users, workspaces, members, &stubDocumentFlush{}, nil)
		_, err = uc.DeleteWorkspace(ctx, DeleteWorkspaceInput{UserID: hostID, WorkspaceID: workspaceID, Reason: "done"})
		if err == nil || !strings.Contains(err.Error(), "logical delete failed") {
			t.Fatalf("expected logical delete workspace error, got %v", err)
		}

		hostID, workspaceID, users, workspaces, members = setupWorkspaceFixture()
		memberID := uuid.New()
		users.users[memberID] = &entity.User{ID: memberID, Status: entity.UserStatusActive, CreatedAt: usecaseTestNow(), UpdatedAt: usecaseTestNow()}
		wsMember, _ := entity.NewWorkspaceMember(workspaceID, memberID, entity.WorkspaceRoleMember, usecaseTestNow())
		members.members[memberKey(workspaceID, memberID)] = &wsMember
		uc = newWorkspaceUseCase(users, workspaces, members, &stubDocumentFlush{}, nil)
		_, err = uc.DeleteWorkspace(ctx, DeleteWorkspaceInput{UserID: memberID, WorkspaceID: workspaceID, Reason: "done"})
		if !errors.Is(err, ErrHostPermissionRequired) {
			t.Fatalf("expected delete access error, got %v", err)
		}

		hostID, workspaceID, users, workspaces, members = setupWorkspaceFixture()
		workspaces.findForUpdateMiss = true
		uc = newWorkspaceUseCase(users, workspaces, members, &stubDocumentFlush{}, nil)
		_, err = uc.DeleteWorkspace(ctx, DeleteWorkspaceInput{UserID: hostID, WorkspaceID: workspaceID, Reason: "done"})
		if !errors.Is(err, ErrWorkspaceNotFound) {
			t.Fatalf("expected delete not found in tx, got %v", err)
		}

		workspaces.byID[workspaceID].DeletedAt = &now
		_, err = uc.DeleteWorkspace(ctx, DeleteWorkspaceInput{UserID: hostID, WorkspaceID: workspaceID, Reason: "done"})
		if !errors.Is(err, ErrWorkspaceAlreadyDeleted) {
			t.Fatalf("expected already deleted in delete tx, got %v", err)
		}
	})

	t.Run("batch stat error via hook", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "notehub-backup-old.sql"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		oldTime := usecaseTestNow().Add(-48 * time.Hour)
		if err := os.Chtimes(filepath.Join(dir, "notehub-backup-old.sql"), oldTime, oldTime); err != nil {
			t.Fatal(err)
		}
		orig := backupFileInfoFn
		backupFileInfoFn = func(os.DirEntry) (os.FileInfo, error) { return nil, fmt.Errorf("stat failed") }
		t.Cleanup(func() { backupFileInfoFn = orig })

		uc := NewDatabaseBackupUseCase("postgres://x", dir, time.Hour, &mockAuditLogRepo{})
		if _, err := uc.pruneOldBackups(usecaseTestNow()); err == nil || !strings.Contains(err.Error(), "stat backup file") {
			t.Fatalf("expected stat error, got %v", err)
		}
	})

	t.Run("batch prune skips non-matching entries", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "ignore-me.txt"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "notehub-backup-old.sql"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		oldTime := usecaseTestNow().Add(-48 * time.Hour)
		if err := os.Chtimes(filepath.Join(dir, "notehub-backup-old.sql"), oldTime, oldTime); err != nil {
			t.Fatal(err)
		}
		uc := NewDatabaseBackupUseCase("postgres://x", dir, time.Hour, &mockAuditLogRepo{})
		removed, err := uc.pruneOldBackups(usecaseTestNow())
		if err != nil || removed != 1 {
			t.Fatalf("expected one removed backup, got removed=%d err=%v", removed, err)
		}
	})
}
