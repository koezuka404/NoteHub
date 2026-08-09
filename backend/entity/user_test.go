package entity

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func testNow() time.Time {
	return time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
}

func activeUser() User {
	return User{
		ID:           uuid.New(),
		Name:         "Alice",
		Email:        "alice@example.com",
		PasswordHash: "hash",
		Status:       UserStatusActive,
		AuthVersion:  1,
		CreatedAt:    testNow(),
		UpdatedAt:    testNow(),
	}
}

func TestUser_Suspend(t *testing.T) {
	user := activeUser()
	now := testNow()

	if err := user.Suspend(now); err != nil {
		t.Fatalf("suspend: %v", err)
	}
	if user.Status != UserStatusSuspended {
		t.Fatalf("status = %q", user.Status)
	}
	if user.SuspendedAt == nil || !user.SuspendedAt.Equal(now) {
		t.Fatal("expected suspended_at")
	}
	if user.AuthVersion != 2 {
		t.Fatalf("auth_version = %d", user.AuthVersion)
	}
	if user.CanAuthenticate() {
		t.Fatal("suspended user should not authenticate")
	}
}

func TestUser_Suspend_AlreadySuspended(t *testing.T) {
	user := activeUser()
	_ = user.Suspend(testNow())
	err := user.Suspend(testNow())
	if !errors.Is(err, ErrInvalidStateTransition) {
		t.Fatalf("expected ErrInvalidStateTransition, got %v", err)
	}
}

func TestUser_Reactivate(t *testing.T) {
	user := activeUser()
	now := testNow()
	_ = user.Suspend(now)

	if err := user.Reactivate(now.Add(time.Minute)); err != nil {
		t.Fatalf("reactivate: %v", err)
	}
	if user.Status != UserStatusActive {
		t.Fatalf("status = %q", user.Status)
	}
	if user.SuspendedAt != nil {
		t.Fatal("expected suspended_at cleared")
	}
	if user.AuthVersion != 3 {
		t.Fatalf("auth_version = %d", user.AuthVersion)
	}
	if !user.CanAuthenticate() {
		t.Fatal("reactivated user should authenticate")
	}
}

func TestUser_Reactivate_NotSuspended(t *testing.T) {
	user := activeUser()
	err := user.Reactivate(testNow())
	if !errors.Is(err, ErrUserNotSuspended) {
		t.Fatalf("expected ErrUserNotSuspended, got %v", err)
	}
}

func TestUser_LogicalDelete(t *testing.T) {
	user := activeUser()
	now := testNow()
	_ = user.Suspend(now)

	if err := user.LogicalDelete("deleted+"+user.ID.String()+"@notehub.invalid", "unused-hash", now.Add(time.Minute)); err != nil {
		t.Fatalf("logical delete: %v", err)
	}
	if user.Status != UserStatusDeleted {
		t.Fatalf("status = %q", user.Status)
	}
	if user.Email != "deleted+"+user.ID.String()+"@notehub.invalid" {
		t.Fatalf("email = %q", user.Email)
	}
	if user.DeletedAt == nil {
		t.Fatal("expected deleted_at")
	}
	if user.CanAuthenticate() {
		t.Fatal("deleted user should not authenticate")
	}
}

func TestUser_LogicalDelete_RequiresSuspended(t *testing.T) {
	user := activeUser()
	err := user.LogicalDelete("deleted@test.invalid", "hash", testNow())
	if !errors.Is(err, ErrUserNotSuspended) {
		t.Fatalf("expected ErrUserNotSuspended, got %v", err)
	}
}

func TestUser_TableName(t *testing.T) {
	if (User{}).TableName() != "users" {
		t.Fatalf("table name = %q", (User{}).TableName())
	}
}

func TestNewUser(t *testing.T) {
	now := testNow()
	user, err := NewUser("  Alice  ", "  ALICE@Example.COM  ", " hash ", now)
	if err != nil {
		t.Fatalf("NewUser: %v", err)
	}
	if user.Name != "Alice" || user.Email != "alice@example.com" || user.PasswordHash != "hash" {
		t.Fatalf("unexpected user: %+v", user)
	}
	if user.Status != UserStatusActive || user.AuthVersion != 1 {
		t.Fatalf("unexpected status/auth version: %+v", user)
	}
}

func TestNewUser_ValidationErrors(t *testing.T) {
	now := testNow()
	if _, err := NewUser("", "a@b.com", "hash", now); err == nil {
		t.Fatal("expected error for empty name")
	}
	if _, err := NewUser("name", "", "hash", now); err == nil {
		t.Fatal("expected error for empty email")
	}
	if _, err := NewUser("name", "a@b.com", " ", now); err == nil {
		t.Fatal("expected error for empty password hash")
	}
}

func TestUser_IsSuspendedAndIsDeleted(t *testing.T) {
	user := activeUser()
	if user.IsSuspended() || user.IsDeleted() {
		t.Fatal("active user should not be suspended or deleted")
	}
	_ = user.Suspend(testNow())
	if !user.IsSuspended() {
		t.Fatal("expected suspended")
	}

	deleted := activeUser()
	deleted.DeletedAt = timePointer(testNow())
	if !deleted.IsDeleted() {
		t.Fatal("user with deleted_at should be deleted")
	}
}

func TestUser_Suspend_DeletedUser(t *testing.T) {
	user := activeUser()
	now := testNow()
	_ = user.Suspend(now)
	_ = user.LogicalDelete("deleted+"+user.ID.String()+"@notehub.invalid", "hash", now.Add(time.Minute))
	err := user.Suspend(now.Add(2 * time.Minute))
	if !errors.Is(err, ErrUserDeleted) {
		t.Fatalf("expected ErrUserDeleted, got %v", err)
	}
}

func TestUser_Reactivate_DeletedUser(t *testing.T) {
	user := activeUser()
	now := testNow()
	_ = user.Suspend(now)
	_ = user.LogicalDelete("deleted+"+user.ID.String()+"@notehub.invalid", "hash", now.Add(time.Minute))
	err := user.Reactivate(now.Add(2 * time.Minute))
	if !errors.Is(err, ErrUserDeleted) {
		t.Fatalf("expected ErrUserDeleted, got %v", err)
	}
}

func TestUser_LogicalDelete_AlreadyDeleted(t *testing.T) {
	user := activeUser()
	now := testNow()
	_ = user.Suspend(now)
	email := "deleted+" + user.ID.String() + "@notehub.invalid"
	_ = user.LogicalDelete(email, "hash", now.Add(time.Minute))
	err := user.LogicalDelete(email, "hash", now.Add(2*time.Minute))
	if !errors.Is(err, ErrUserDeleted) {
		t.Fatalf("expected ErrUserDeleted, got %v", err)
	}
}

func TestUser_LogicalDelete_EmptyAnonymizedFields(t *testing.T) {
	user := activeUser()
	_ = user.Suspend(testNow())
	err := user.LogicalDelete("  ", "hash", testNow())
	if err == nil {
		t.Fatal("expected error for empty anonymized email")
	}
}
