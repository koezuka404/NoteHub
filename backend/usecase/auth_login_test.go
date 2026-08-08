package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
)

type mockUserRepo struct {
	user  *entity.User
	found bool
	err   error
}

func (m *mockUserRepo) Create(context.Context, *entity.User) error { return nil }
func (m *mockUserRepo) FindByID(context.Context, uuid.UUID) (*entity.User, bool, error) {
	return nil, false, nil
}
func (m *mockUserRepo) FindByIDForUpdate(context.Context, uuid.UUID) (*entity.User, bool, error) {
	return nil, false, nil
}
func (m *mockUserRepo) Update(context.Context, *entity.User) error { return nil }
func (m *mockUserRepo) FindByEmail(_ context.Context, email string) (*entity.User, bool, error) {
	if m.err != nil {
		return nil, false, m.err
	}
	if !m.found {
		return nil, false, nil
	}
	if m.user != nil && m.user.Email != email {
		return nil, false, nil
	}
	return m.user, true, nil
}
func (m *mockUserRepo) ExistsByEmail(context.Context, string) (bool, error) { return false, nil }
func (m *mockUserRepo) IncrementAuthVersion(context.Context, uuid.UUID, time.Time) error {
	return nil
}

type mockRefreshTokenRepo struct {
	created *entity.RefreshToken
	err     error
}

func (m *mockRefreshTokenRepo) Create(_ context.Context, token *entity.RefreshToken) error {
	if m.err != nil {
		return m.err
	}
	m.created = token
	return nil
}
func (m *mockRefreshTokenRepo) FindByHashForUpdate(context.Context, string) (*entity.RefreshToken, bool, error) {
	return nil, false, nil
}
func (m *mockRefreshTokenRepo) Update(context.Context, *entity.RefreshToken) error { return nil }
func (m *mockRefreshTokenRepo) RevokeFamily(context.Context, uuid.UUID, time.Time) error {
	return nil
}
func (m *mockRefreshTokenRepo) RevokeAllByUserID(context.Context, uuid.UUID, time.Time) error {
	return nil
}
func (m *mockRefreshTokenRepo) MarkExpiredBefore(context.Context, time.Time) (int64, error) {
	return 0, nil
}
func (m *mockRefreshTokenRepo) DeleteStaleBefore(context.Context, time.Time) (int64, error) {
	return 0, nil
}

type mockAuditLogRepo struct {
	logs []*entity.AuditLog
}

func (m *mockAuditLogRepo) Create(_ context.Context, log *entity.AuditLog) error {
	copy := *log
	m.logs = append(m.logs, &copy)
	return nil
}

type mockTransactionManager struct{}

func (m *mockTransactionManager) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type mockAuthService struct {
	loginLocked      bool
	recordLocked     bool
	compareErr       error
	compareCalls     int
	resetCalled      bool
	recordFailCalled bool
}

func (m *mockAuthService) HashPassword(string) (string, error) { return "", nil }
func (m *mockAuthService) ComparePassword(_, _ string) error {
	m.compareCalls++
	return m.compareErr
}
func (m *mockAuthService) GenerateAccessToken(userID uuid.UUID, authVersion uint, now time.Time) (string, time.Time, error) {
	return "access-token", now.Add(15 * time.Minute), nil
}
func (m *mockAuthService) GenerateRefreshToken() (string, error) { return "refresh-token", nil }
func (m *mockAuthService) GenerateCSRFToken() (string, error)    { return "csrf-token", nil }
func (m *mockAuthService) HashToken(string) string {
	return "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
}
func (m *mockAuthService) RevokeAccessToken(context.Context, uuid.UUID, time.Duration) error {
	return nil
}
func (m *mockAuthService) IsAccessTokenRevoked(context.Context, uuid.UUID) (bool, error) {
	return false, nil
}
func (m *mockAuthService) IsLoginLocked(context.Context, string) (bool, time.Duration, error) {
	return m.loginLocked, 0, nil
}
func (m *mockAuthService) RecordLoginFailure(context.Context, string) (bool, time.Duration, error) {
	m.recordFailCalled = true
	return m.recordLocked, 0, nil
}
func (m *mockAuthService) ResetLoginFailures(context.Context, string) error {
	m.resetCalled = true
	return nil
}

func newLoginAuthUseCase(users *mockUserRepo, refresh *mockRefreshTokenRepo, auth *mockAuthService) *AuthUseCase {
	return newLoginAuthUseCaseWithAudit(users, refresh, auth, &mockAuditLogRepo{})
}

func newLoginAuthUseCaseWithAudit(
	users *mockUserRepo,
	refresh *mockRefreshTokenRepo,
	auth *mockAuthService,
	audit *mockAuditLogRepo,
) *AuthUseCase {
	uc := NewAuthUseCase(users, refresh, audit, auth, &mockTransactionManager{}, 24*time.Hour)
	uc.now = func() time.Time {
		return time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	}
	return uc
}

func TestLogin_ValidationError(t *testing.T) {
	uc := newLoginAuthUseCase(&mockUserRepo{}, &mockRefreshTokenRepo{}, &mockAuthService{})

	_, err := uc.Login(context.Background(), LoginInput{Email: "", Password: ""})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestLogin_AccountLocked(t *testing.T) {
	auth := &mockAuthService{loginLocked: true}
	auditRepo := &mockAuditLogRepo{}
	uc := newLoginAuthUseCaseWithAudit(&mockUserRepo{}, &mockRefreshTokenRepo{}, auth, auditRepo)

	_, err := uc.Login(context.Background(), LoginInput{Email: "user@example.com", Password: "secret"})
	if !errors.Is(err, ErrLoginTemporarilyLocked) {
		t.Fatalf("expected ErrLoginTemporarilyLocked, got %v", err)
	}
	if len(auditRepo.logs) != 1 || auditRepo.logs[0].Action != "LOGIN_RATE_LIMITED" {
		t.Fatalf("expected LOGIN_RATE_LIMITED audit log, got %+v", auditRepo.logs)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	auth := &mockAuthService{}
	uc := newLoginAuthUseCase(&mockUserRepo{found: false}, &mockRefreshTokenRepo{}, auth)

	_, err := uc.Login(context.Background(), LoginInput{Email: "user@example.com", Password: "secret"})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
	if auth.compareCalls != 1 {
		t.Fatalf("expected dummy password compare, calls = %d", auth.compareCalls)
	}
	if !auth.recordFailCalled {
		t.Fatal("expected login failure to be recorded")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	user := &entity.User{
		ID:           uuid.New(),
		Name:         "Alice",
		Email:        "user@example.com",
		PasswordHash: "stored-hash",
		Status:       entity.UserStatusActive,
		AuthVersion:  1,
	}
	auth := &mockAuthService{compareErr: fmt.Errorf("password mismatch")}
	uc := newLoginAuthUseCase(&mockUserRepo{user: user, found: true}, &mockRefreshTokenRepo{}, auth)

	_, err := uc.Login(context.Background(), LoginInput{Email: "user@example.com", Password: "wrong"})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_SuspendedUser(t *testing.T) {
	user := &entity.User{
		ID:           uuid.New(),
		Email:        "user@example.com",
		PasswordHash: "stored-hash",
		Status:       entity.UserStatusSuspended,
	}
	auth := &mockAuthService{}
	uc := newLoginAuthUseCase(&mockUserRepo{user: user, found: true}, &mockRefreshTokenRepo{}, auth)

	_, err := uc.Login(context.Background(), LoginInput{Email: "user@example.com", Password: "secret"})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_Success(t *testing.T) {
	userID := uuid.New()
	user := &entity.User{
		ID:           userID,
		Name:         "Alice",
		Email:        "user@example.com",
		PasswordHash: "stored-hash",
		Status:       entity.UserStatusActive,
		AuthVersion:  2,
	}
	auth := &mockAuthService{}
	refreshRepo := &mockRefreshTokenRepo{}
	uc := newLoginAuthUseCase(&mockUserRepo{user: user, found: true}, refreshRepo, auth)

	out, err := uc.Login(context.Background(), LoginInput{
		Email:     "  User@Example.com ",
		Password:  "Pass1234",
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if out.AccessToken != "access-token" {
		t.Fatalf("access token = %q", out.AccessToken)
	}
	if out.RefreshToken != "refresh-token" {
		t.Fatalf("refresh token = %q", out.RefreshToken)
	}
	if out.CSRFToken != "csrf-token" {
		t.Fatalf("csrf token = %q", out.CSRFToken)
	}
	if out.TokenType != "Bearer" {
		t.Fatalf("token type = %q", out.TokenType)
	}
	if out.User.ID != userID || out.User.Email != "user@example.com" {
		t.Fatalf("unexpected user output: %+v", out.User)
	}
	if !auth.resetCalled {
		t.Fatal("expected login failures to reset")
	}
	if refreshRepo.created == nil {
		t.Fatal("expected refresh token to be saved")
	}
	if refreshRepo.created.UserID != userID {
		t.Fatalf("refresh token user id = %v", refreshRepo.created.UserID)
	}
	if refreshRepo.created.IPAddress != "127.0.0.1" || refreshRepo.created.UserAgent != "test-agent" {
		t.Fatalf("unexpected refresh metadata: %+v", refreshRepo.created)
	}
}

func TestLogin_Success_RecordsAuditLog(t *testing.T) {
	userID := uuid.New()
	user := &entity.User{
		ID:           userID,
		Name:         "Alice",
		Email:        "user@example.com",
		PasswordHash: "stored-hash",
		Status:       entity.UserStatusActive,
		AuthVersion:  2,
	}
	auditRepo := &mockAuditLogRepo{}
	uc := newLoginAuthUseCaseWithAudit(
		&mockUserRepo{user: user, found: true},
		&mockRefreshTokenRepo{},
		&mockAuthService{},
		auditRepo,
	)

	_, err := uc.Login(context.Background(), LoginInput{
		Email:     "user@example.com",
		Password:  "Pass1234",
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if len(auditRepo.logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(auditRepo.logs))
	}
	log := auditRepo.logs[0]
	if log.Action != "LOGIN" || log.ResourceType != "user" {
		t.Fatalf("unexpected audit log: %+v", log)
	}
	if log.ActorUserID == nil || *log.ActorUserID != userID {
		t.Fatalf("unexpected actor: %+v", log.ActorUserID)
	}
	if log.ResourceID == nil || *log.ResourceID != userID {
		t.Fatalf("unexpected resource id: %+v", log.ResourceID)
	}
	if log.IPAddress != "127.0.0.1" || log.UserAgent != "test-agent" {
		t.Fatalf("unexpected audit metadata: %+v", log)
	}
}

func TestLogin_WrongPassword_RecordsFailedAuditLog(t *testing.T) {
	userID := uuid.New()
	user := &entity.User{
		ID:           userID,
		Email:        "user@example.com",
		PasswordHash: "stored-hash",
		Status:       entity.UserStatusActive,
	}
	auditRepo := &mockAuditLogRepo{}
	uc := newLoginAuthUseCaseWithAudit(
		&mockUserRepo{user: user, found: true},
		&mockRefreshTokenRepo{},
		&mockAuthService{compareErr: fmt.Errorf("password mismatch")},
		auditRepo,
	)

	_, err := uc.Login(context.Background(), LoginInput{
		Email:     "user@example.com",
		Password:  "wrong",
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
	if len(auditRepo.logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(auditRepo.logs))
	}
	log := auditRepo.logs[0]
	if log.Action != "LOGIN_FAILED" {
		t.Fatalf("unexpected action: %q", log.Action)
	}
	if log.ActorUserID == nil || *log.ActorUserID != userID {
		t.Fatalf("unexpected actor: %+v", log.ActorUserID)
	}
}

func TestLogin_UserNotFound_RecordsFailedAuditLog(t *testing.T) {
	auditRepo := &mockAuditLogRepo{}
	uc := newLoginAuthUseCaseWithAudit(
		&mockUserRepo{found: false},
		&mockRefreshTokenRepo{},
		&mockAuthService{},
		auditRepo,
	)

	_, err := uc.Login(context.Background(), LoginInput{
		Email:     "missing@example.com",
		Password:  "secret",
		IPAddress: "127.0.0.1",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
	if len(auditRepo.logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(auditRepo.logs))
	}
	log := auditRepo.logs[0]
	if log.Action != "LOGIN_FAILED" {
		t.Fatalf("unexpected action: %q", log.Action)
	}
	if log.ActorUserID != nil || log.ResourceID != nil {
		t.Fatalf("expected nil actor/resource for unknown user, got actor=%+v resource=%+v", log.ActorUserID, log.ResourceID)
	}
}
