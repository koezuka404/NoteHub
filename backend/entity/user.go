package entity

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey"`
	Name         string     `gorm:"size:100;not null"`
	Email        string     `gorm:"size:255;not null;uniqueIndex:uq_users_email_active,where:deleted_at IS NULL"`
	PasswordHash string     `gorm:"size:255;not null;column:password_hash"`
	Status       UserStatus `gorm:"size:20;not null;default:active;index"`
	AuthVersion  uint       `gorm:"not null;default:1"`
	SuspendedAt  *time.Time `gorm:"index"`
	DeletedAt    *time.Time `gorm:"index"`
	CreatedAt    time.Time  `gorm:"not null"`
	UpdatedAt    time.Time  `gorm:"not null"`
}

func (User) TableName() string { return "users" }

func NewUser(name, email, passwordHash string, now time.Time) (User, error) {
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))
	passwordHash = strings.TrimSpace(passwordHash)
	if name == "" || email == "" || passwordHash == "" {
		return User{}, fmt.Errorf("name, email and password hash are required")
	}
	return User{ID: uuid.New(), Name: name, Email: email, PasswordHash: passwordHash, Status: UserStatusActive, AuthVersion: 1, CreatedAt: now, UpdatedAt: now}, nil
}

func (u User) IsDeleted() bool {
	return u.Status == UserStatusDeleted || u.DeletedAt != nil
}

func (u User) IsSuspended() bool {
	return u.Status == UserStatusSuspended
}

func (u User) CanAuthenticate() bool {
	return u.Status == UserStatusActive && !u.IsDeleted()
}

func (u *User) Suspend(now time.Time) error {
	if u.IsDeleted() {
		return ErrUserDeleted
	}
	if u.Status != UserStatusActive {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidStateTransition, u.Status, UserStatusSuspended)
	}
	u.Status = UserStatusSuspended
	u.SuspendedAt = timePointer(now)
	u.AuthVersion++
	u.UpdatedAt = now
	return nil
}

func (u *User) Reactivate(now time.Time) error {
	if u.IsDeleted() {
		return ErrUserDeleted
	}
	if u.Status != UserStatusSuspended {
		return ErrUserNotSuspended
	}
	u.Status = UserStatusActive
	u.SuspendedAt = nil
	u.AuthVersion++
	u.UpdatedAt = now
	return nil
}

func (u *User) LogicalDelete(anonymizedEmail, unusablePasswordHash string, now time.Time) error {
	if u.IsDeleted() {
		return ErrUserDeleted
	}
	if u.Status != UserStatusSuspended {
		return ErrUserNotSuspended
	}
	anonymizedEmail = strings.ToLower(strings.TrimSpace(anonymizedEmail))
	unusablePasswordHash = strings.TrimSpace(unusablePasswordHash)
	if anonymizedEmail == "" || unusablePasswordHash == "" {
		return fmt.Errorf("anonymized email and unusable password hash are required")
	}
	u.Name = "Deleted User"
	u.Email = anonymizedEmail
	u.PasswordHash = unusablePasswordHash
	u.Status = UserStatusDeleted
	u.DeletedAt = timePointer(now)
	u.AuthVersion++
	u.UpdatedAt = now
	return nil
}

func timePointer(value time.Time) *time.Time { return &value }
