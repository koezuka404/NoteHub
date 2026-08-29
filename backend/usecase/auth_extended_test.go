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

func TestValidatePassword(t *testing.T) {
	if validatePassword("Pass1234") != true {
		t.Fatal("expected valid password")
	}
	if validatePassword("short1") || validatePassword("onlyletters") || validatePassword("12345678") {
		t.Fatal("expected invalid password")
	}
	if validatePassword("        ") {
		t.Fatal("whitespace-only password should fail")
	}
	if !validatePassword("あいうえおか12") {
		t.Fatal("multibyte password of 8 runes should be valid")
	}
	if validatePassword("あいうえおかきくけこさしすせそ12") {
		t.Fatal("multibyte password over 15 runes should be invalid")
	}
}

func TestValidateRegisterInput(t *testing.T) {
	if err := validateRegisterInput(RegisterInput{Name: "Alice", Email: "alice@example.com", Password: "Pass1234"}); err != nil {
		t.Fatalf("valid input: %v", err)
	}
	if err := validateRegisterInput(RegisterInput{Name: "", Email: "alice@example.com", Password: "Pass1234"}); !errors.Is(err, ErrValidation) {
		t.Fatalf("empty name: %v", err)
	}
	if err := validateRegisterInput(RegisterInput{Name: "Alice", Email: "bad", Password: "Pass1234"}); !errors.Is(err, ErrValidation) {
		t.Fatalf("bad email: %v", err)
	}
	if err := validateRegisterInput(RegisterInput{Name: "Alice", Email: "alice@example.com", Password: "short"}); !errors.Is(err, ErrPasswordInvalid) {
		t.Fatalf("bad password: %v", err)
	}
}

func TestRegister_Success(t *testing.T) {
	users := &stubUserRepoFull{}
	audit := &mockAuditLogRepo{}
	uc := NewAuthUseCase(users, &mockRefreshTokenRepo{}, audit, &mockAuthService{}, &mockTransactionManager{}, 24*time.Hour)
	uc.now = usecaseTestNow

	out, err := uc.Register(context.Background(), RegisterInput{
		Name: "  Alice  ", Email: " Alice@Example.com ", Password: "Pass1234", IPAddress: "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if out.Email != "alice@example.com" || out.Name != "Alice" {
		t.Fatalf("unexpected output: %+v", out)
	}
	if len(audit.logs) != 1 || audit.logs[0].Action != "USER_REGISTERED" {
		t.Fatalf("audit logs: %+v", audit.logs)
	}
}

func TestRegister_EmailExists(t *testing.T) {
	users := newStubUserRepoFull(map[uuid.UUID]*entity.User{
		uuid.New(): {Email: "alice@example.com"},
	})
	uc := NewAuthUseCase(users, &mockRefreshTokenRepo{}, &mockAuditLogRepo{}, &mockAuthService{}, &mockTransactionManager{}, time.Hour)
	uc.now = usecaseTestNow

	_, err := uc.Register(context.Background(), RegisterInput{Name: "Alice", Email: "alice@example.com", Password: "Pass1234"})
	if !errors.Is(err, ErrEmailAlreadyExists) {
		t.Fatalf("expected ErrEmailAlreadyExists, got %v", err)
	}
}

func TestRegister_HashPasswordError(t *testing.T) {
	auth := &failingHashAuthService{}
	uc := NewAuthUseCase(&stubUserRepoFull{}, &mockRefreshTokenRepo{}, &mockAuditLogRepo{}, auth, &mockTransactionManager{}, time.Hour)
	uc.now = usecaseTestNow

	_, err := uc.Register(context.Background(), RegisterInput{Name: "Alice", Email: "new@example.com", Password: "Pass1234"})
	if err == nil {
		t.Fatal("expected hash error")
	}
}

type failingHashAuthService struct{ mockAuthService }

func (*failingHashAuthService) HashPassword(string) (string, error) {
	return "", fmt.Errorf("hash failed")
}

func TestGetCurrentUser_Success(t *testing.T) {
	userID := uuid.New()
	users := newStubUserRepoFull(map[uuid.UUID]*entity.User{
		userID: {ID: userID, Name: "Alice", Email: "alice@example.com", Status: entity.UserStatusActive, AuthVersion: 1},
	})
	uc := NewAuthUseCase(users, &mockRefreshTokenRepo{}, &mockAuditLogRepo{}, &mockAuthService{}, &mockTransactionManager{}, time.Hour)

	out, err := uc.GetCurrentUser(context.Background(), GetCurrentUserInput{UserID: userID})
	if err != nil {
		t.Fatalf("GetCurrentUser: %v", err)
	}
	if out.Email != "alice@example.com" {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestGetCurrentUser_NotAvailable(t *testing.T) {
	userID := uuid.New()
	users := newStubUserRepoFull(map[uuid.UUID]*entity.User{
		userID: {ID: userID, Status: entity.UserStatusSuspended},
	})
	uc := NewAuthUseCase(users, &mockRefreshTokenRepo{}, &mockAuditLogRepo{}, &mockAuthService{}, &mockTransactionManager{}, time.Hour)

	_, err := uc.GetCurrentUser(context.Background(), GetCurrentUserInput{UserID: userID})
	if !errors.Is(err, ErrAccountSuspended) {
		t.Fatalf("expected ErrAccountSuspended, got %v", err)
	}
}
