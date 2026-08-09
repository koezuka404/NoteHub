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

const testTokenHash = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

type authServiceExt struct {
	mockAuthService
	isLoginLockedErr error
	resetErr           error
	recordFailErr      error
	revokeAccessErr    error
}

func (a *authServiceExt) IsLoginLocked(context.Context, string) (bool, time.Duration, error) {
	if a.isLoginLockedErr != nil {
		return false, 0, a.isLoginLockedErr
	}
	return a.mockAuthService.IsLoginLocked(context.Background(), "")
}

func (a *authServiceExt) ResetLoginFailures(context.Context, string) error {
	if a.resetErr != nil {
		return a.resetErr
	}
	return a.mockAuthService.ResetLoginFailures(context.Background(), "")
}

func (a *authServiceExt) RecordLoginFailure(context.Context, string) (bool, time.Duration, error) {
	if a.recordFailErr != nil {
		return false, 0, a.recordFailErr
	}
	return a.mockAuthService.RecordLoginFailure(context.Background(), "")
}

func (a *authServiceExt) RevokeAccessToken(context.Context, uuid.UUID, time.Duration) error {
	if a.revokeAccessErr != nil {
		return a.revokeAccessErr
	}
	return a.mockAuthService.RevokeAccessToken(context.Background(), uuid.Nil, 0)
}

type trackingRefreshRepo struct {
	stubRefreshRepoFull
	created []*entity.RefreshToken
}

func (r *trackingRefreshRepo) Create(_ context.Context, token *entity.RefreshToken) error {
	if r.err != nil {
		return r.err
	}
	if r.byHash == nil {
		r.byHash = map[string]*entity.RefreshToken{}
	}
	copy := *token
	r.byHash[token.TokenHash] = &copy
	r.created = append(r.created, &copy)
	return nil
}

func newRefreshAuthUseCase(users *stubUserRepoFull, refresh *trackingRefreshRepo, auth IAuthService, audit *mockAuditLogRepo) *AuthUseCase {
	uc := NewAuthUseCase(users, refresh, audit, auth, &mockTransactionManager{}, 24*time.Hour)
	uc.now = usecaseTestNow
	return uc
}

func activeRefreshToken(userID uuid.UUID, now time.Time) entity.RefreshToken {
	token, _ := entity.NewRefreshToken(userID, testTokenHash, uuid.New(), now.Add(24*time.Hour), now)
	return token
}

func TestRefresh_Required(t *testing.T) {
	uc := newRefreshAuthUseCase(&stubUserRepoFull{}, &trackingRefreshRepo{}, &mockAuthService{}, &mockAuditLogRepo{})

	_, err := uc.Refresh(context.Background(), RefreshInput{})
	if !errors.Is(err, ErrRefreshTokenRequired) {
		t.Fatalf("expected ErrRefreshTokenRequired, got %v", err)
	}
}

func TestRefresh_Invalid(t *testing.T) {
	uc := newRefreshAuthUseCase(&stubUserRepoFull{}, &trackingRefreshRepo{}, &mockAuthService{}, &mockAuditLogRepo{})

	_, err := uc.Refresh(context.Background(), RefreshInput{RefreshToken: "refresh-token"})
	if !errors.Is(err, ErrRefreshTokenInvalid) {
		t.Fatalf("expected ErrRefreshTokenInvalid, got %v", err)
	}
}

func TestRefresh_Expired(t *testing.T) {
	now := usecaseTestNow()
	userID := uuid.New()
	token, _ := entity.NewRefreshToken(userID, testTokenHash, uuid.New(), now.Add(-time.Hour), now.Add(-48*time.Hour))
	refresh := &trackingRefreshRepo{stubRefreshRepoFull: stubRefreshRepoFull{byHash: map[string]*entity.RefreshToken{
		testTokenHash: &token,
	}}}
	uc := newRefreshAuthUseCase(&stubUserRepoFull{}, refresh, &mockAuthService{}, &mockAuditLogRepo{})

	_, err := uc.Refresh(context.Background(), RefreshInput{RefreshToken: "refresh-token"})
	if !errors.Is(err, ErrRefreshTokenExpired) {
		t.Fatalf("expected ErrRefreshTokenExpired, got %v", err)
	}
}

func TestRefresh_Reused(t *testing.T) {
	now := usecaseTestNow()
	userID := uuid.New()
	token := activeRefreshToken(userID, now)
	rotated := now
	token.RotatedAt = &rotated
	token.Status = entity.RefreshTokenStatusRotated
	refresh := &trackingRefreshRepo{stubRefreshRepoFull: stubRefreshRepoFull{byHash: map[string]*entity.RefreshToken{
		testTokenHash: &token,
	}}}
	users := newStubUserRepoFull(map[uuid.UUID]*entity.User{
		userID: {ID: userID, Status: entity.UserStatusActive, AuthVersion: 1},
	})
	audit := &mockAuditLogRepo{}
	uc := newRefreshAuthUseCase(users, refresh, &mockAuthService{}, audit)

	_, err := uc.Refresh(context.Background(), RefreshInput{
		RefreshToken: "refresh-token",
		IPAddress:    "127.0.0.1",
		UserAgent:    "test",
	})
	if !errors.Is(err, ErrRefreshTokenReused) {
		t.Fatalf("expected ErrRefreshTokenReused, got %v", err)
	}
	if len(audit.logs) != 1 || audit.logs[0].Action != "REFRESH_TOKEN_REUSED" {
		t.Fatalf("expected reused audit log, got %+v", audit.logs)
	}
}

func TestRefresh_Success(t *testing.T) {
	now := usecaseTestNow()
	userID := uuid.New()
	token := activeRefreshToken(userID, now)
	refresh := &trackingRefreshRepo{stubRefreshRepoFull: stubRefreshRepoFull{byHash: map[string]*entity.RefreshToken{
		testTokenHash: &token,
	}}}
	users := newStubUserRepoFull(map[uuid.UUID]*entity.User{
		userID: {ID: userID, Status: entity.UserStatusActive, AuthVersion: 2},
	})
	uc := newRefreshAuthUseCase(users, refresh, &mockAuthService{}, &mockAuditLogRepo{})

	out, err := uc.Refresh(context.Background(), RefreshInput{
		RefreshToken: "refresh-token",
		IPAddress:    "127.0.0.1",
		UserAgent:    "test-agent",
	})
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if out.AccessToken != "access-token" || out.RefreshToken != "refresh-token" {
		t.Fatalf("unexpected tokens: %+v", out)
	}
	if len(refresh.created) != 1 {
		t.Fatalf("expected next refresh token saved, got %d", len(refresh.created))
	}
	updated := refresh.byHash[testTokenHash]
	if updated == nil || updated.Status != entity.RefreshTokenStatusRotated {
		t.Fatalf("expected current token rotated, got %+v", updated)
	}
}

func TestRefresh_UserDeleted(t *testing.T) {
	now := usecaseTestNow()
	userID := uuid.New()
	token := activeRefreshToken(userID, now)
	deletedAt := now
	users := newStubUserRepoFull(map[uuid.UUID]*entity.User{
		userID: {ID: userID, Status: entity.UserStatusDeleted, DeletedAt: &deletedAt},
	})
	refresh := &trackingRefreshRepo{stubRefreshRepoFull: stubRefreshRepoFull{byHash: map[string]*entity.RefreshToken{
		testTokenHash: &token,
	}}}
	uc := newRefreshAuthUseCase(users, refresh, &mockAuthService{}, &mockAuditLogRepo{})

	_, err := uc.Refresh(context.Background(), RefreshInput{RefreshToken: "refresh-token"})
	if !errors.Is(err, ErrAccountDeleted) {
		t.Fatalf("expected ErrAccountDeleted, got %v", err)
	}
}

func TestRefresh_UserSuspended(t *testing.T) {
	now := usecaseTestNow()
	userID := uuid.New()
	token := activeRefreshToken(userID, now)
	users := newStubUserRepoFull(map[uuid.UUID]*entity.User{
		userID: {ID: userID, Status: entity.UserStatusSuspended},
	})
	refresh := &trackingRefreshRepo{stubRefreshRepoFull: stubRefreshRepoFull{byHash: map[string]*entity.RefreshToken{
		testTokenHash: &token,
	}}}
	uc := newRefreshAuthUseCase(users, refresh, &mockAuthService{}, &mockAuditLogRepo{})

	_, err := uc.Refresh(context.Background(), RefreshInput{RefreshToken: "refresh-token"})
	if !errors.Is(err, ErrAccountSuspended) {
		t.Fatalf("expected ErrAccountSuspended, got %v", err)
	}
}

func TestLogout_InvalidInput(t *testing.T) {
	uc := newRefreshAuthUseCase(&stubUserRepoFull{}, &trackingRefreshRepo{}, &mockAuthService{}, &mockAuditLogRepo{})

	_, err := uc.Logout(context.Background(), LogoutInput{
		CSRFValidated: true,
	})
	if !errors.Is(err, ErrAccessTokenInvalid) {
		t.Fatalf("expected ErrAccessTokenInvalid, got %v", err)
	}
}

func TestLogout_CSRFInvalid(t *testing.T) {
	uc := newRefreshAuthUseCase(&stubUserRepoFull{}, &trackingRefreshRepo{}, &mockAuthService{}, &mockAuditLogRepo{})
	jti := uuid.New()
	exp := usecaseTestNow().Add(time.Hour)

	_, err := uc.Logout(context.Background(), LogoutInput{
		UserID: uuid.New(), AccessTokenJTI: jti, AccessTokenExp: exp,
	})
	if !errors.Is(err, ErrCSRFTokenInvalid) {
		t.Fatalf("expected ErrCSRFTokenInvalid, got %v", err)
	}
}

func TestLogout_OwnerMismatch(t *testing.T) {
	now := usecaseTestNow()
	userID := uuid.New()
	otherUser := uuid.New()
	token := activeRefreshToken(otherUser, now)
	refresh := &trackingRefreshRepo{stubRefreshRepoFull: stubRefreshRepoFull{byHash: map[string]*entity.RefreshToken{
		testTokenHash: &token,
	}}}
	uc := newRefreshAuthUseCase(&stubUserRepoFull{}, refresh, &mockAuthService{}, &mockAuditLogRepo{})

	_, err := uc.Logout(context.Background(), LogoutInput{
		UserID:         userID,
		AccessTokenJTI: uuid.New(),
		AccessTokenExp: now.Add(time.Hour),
		RefreshToken:   "refresh-token",
		CSRFValidated:  true,
	})
	if !errors.Is(err, ErrTokenOwnerMismatch) {
		t.Fatalf("expected ErrTokenOwnerMismatch, got %v", err)
	}
}

func TestLogout_SuccessWithoutRefreshToken(t *testing.T) {
	now := usecaseTestNow()
	userID := uuid.New()
	audit := &mockAuditLogRepo{}
	auth := &authServiceExt{}
	uc := newRefreshAuthUseCase(&stubUserRepoFull{}, &trackingRefreshRepo{}, auth, audit)

	_, err := uc.Logout(context.Background(), LogoutInput{
		UserID:         userID,
		AccessTokenJTI: uuid.New(),
		AccessTokenExp: now.Add(time.Hour),
		CSRFValidated:  true,
		IPAddress:      "127.0.0.1",
		UserAgent:      "test",
	})
	if err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if len(audit.logs) != 1 || audit.logs[0].Action != "LOGOUT" {
		t.Fatalf("expected LOGOUT audit, got %+v", audit.logs)
	}
}

func TestLogout_SuccessWithRefreshToken(t *testing.T) {
	now := usecaseTestNow()
	userID := uuid.New()
	token := activeRefreshToken(userID, now)
	refresh := &trackingRefreshRepo{stubRefreshRepoFull: stubRefreshRepoFull{byHash: map[string]*entity.RefreshToken{
		testTokenHash: &token,
	}}}
	uc := newRefreshAuthUseCase(&stubUserRepoFull{}, refresh, &mockAuthService{}, &mockAuditLogRepo{})

	_, err := uc.Logout(context.Background(), LogoutInput{
		UserID:         userID,
		AccessTokenJTI: uuid.New(),
		AccessTokenExp: now.Add(time.Hour),
		RefreshToken:   "refresh-token",
		CSRFValidated:  true,
	})
	if err != nil {
		t.Fatalf("Logout: %v", err)
	}
	updated := refresh.byHash[testTokenHash]
	if updated == nil || updated.Status != entity.RefreshTokenStatusRevoked {
		t.Fatalf("expected refresh token revoked, got %+v", updated)
	}
}

func TestLogout_RevokeAccessTokenError(t *testing.T) {
	now := usecaseTestNow()
	auth := &authServiceExt{revokeAccessErr: fmt.Errorf("redis down")}
	uc := newRefreshAuthUseCase(&stubUserRepoFull{}, &trackingRefreshRepo{}, auth, &mockAuditLogRepo{})

	_, err := uc.Logout(context.Background(), LogoutInput{
		UserID:         uuid.New(),
		AccessTokenJTI: uuid.New(),
		AccessTokenExp: now.Add(time.Hour),
		CSRFValidated:  true,
	})
	if !errors.Is(err, ErrAuthServiceUnavailable) {
		t.Fatalf("expected ErrAuthServiceUnavailable, got %v", err)
	}
}

func TestLogin_IsLoginLockedError(t *testing.T) {
	auth := &authServiceExt{isLoginLockedErr: fmt.Errorf("redis unavailable")}
	uc := newLoginAuthUseCaseWithAudit(&mockUserRepo{}, &mockRefreshTokenRepo{}, auth, &mockAuditLogRepo{})

	_, err := uc.Login(context.Background(), LoginInput{Email: "user@example.com", Password: "Pass1234"})
	if !errors.Is(err, ErrAuthServiceUnavailable) {
		t.Fatalf("expected ErrAuthServiceUnavailable, got %v", err)
	}
}

func TestLogin_FindByEmailError(t *testing.T) {
	users := &mockUserRepo{err: fmt.Errorf("db error")}
	uc := newLoginAuthUseCase(users, &mockRefreshTokenRepo{}, &mockAuthService{})

	_, err := uc.Login(context.Background(), LoginInput{Email: "user@example.com", Password: "Pass1234"})
	if err == nil || errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected find user error, got %v", err)
	}
}

func TestLogin_ResetLoginFailuresError(t *testing.T) {
	user := &entity.User{
		ID: uuid.New(), Email: "user@example.com", PasswordHash: "hash",
		Status: entity.UserStatusActive, AuthVersion: 1,
	}
	auth := &authServiceExt{resetErr: fmt.Errorf("redis down")}
	uc := newLoginAuthUseCase(&mockUserRepo{user: user, found: true}, &mockRefreshTokenRepo{}, auth)

	_, err := uc.Login(context.Background(), LoginInput{Email: "user@example.com", Password: "Pass1234"})
	if !errors.Is(err, ErrAuthServiceUnavailable) {
		t.Fatalf("expected ErrAuthServiceUnavailable, got %v", err)
	}
}

func TestLogin_RecordLoginFailureError(t *testing.T) {
	auth := &authServiceExt{recordFailErr: fmt.Errorf("redis down")}
	uc := newLoginAuthUseCase(&mockUserRepo{found: false}, &mockRefreshTokenRepo{}, auth)

	_, err := uc.Login(context.Background(), LoginInput{Email: "user@example.com", Password: "secret"})
	if !errors.Is(err, ErrAuthServiceUnavailable) {
		t.Fatalf("expected ErrAuthServiceUnavailable, got %v", err)
	}
}

func TestLogin_RecordLoginFailureLocksAccount(t *testing.T) {
	auth := &authServiceExt{mockAuthService: mockAuthService{recordLocked: true}}
	audit := &mockAuditLogRepo{}
	uc := newLoginAuthUseCaseWithAudit(&mockUserRepo{found: false}, &mockRefreshTokenRepo{}, auth, audit)

	_, err := uc.Login(context.Background(), LoginInput{
		Email: "user@example.com", Password: "secret", IPAddress: "127.0.0.1",
	})
	if !errors.Is(err, ErrLoginTemporarilyLocked) {
		t.Fatalf("expected ErrLoginTemporarilyLocked, got %v", err)
	}
	if len(audit.logs) != 1 || audit.logs[0].Action != "LOGIN_RATE_LIMITED" {
		t.Fatalf("expected rate limit audit, got %+v", audit.logs)
	}
}

func TestLogin_Success_NilAuditLogs(t *testing.T) {
	userID := uuid.New()
	user := &entity.User{
		ID: userID, Name: "Alice", Email: "user@example.com",
		PasswordHash: "stored-hash", Status: entity.UserStatusActive, AuthVersion: 2,
	}
	uc := NewAuthUseCase(&mockUserRepo{user: user, found: true}, &mockRefreshTokenRepo{}, nil, &mockAuthService{}, &mockTransactionManager{}, 24*time.Hour)
	uc.now = usecaseTestNow

	out, err := uc.Login(context.Background(), LoginInput{
		Email: "user@example.com", Password: "Pass1234",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if out.AccessToken == "" {
		t.Fatal("expected access token")
	}
}
