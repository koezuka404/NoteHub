package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
	"gorm.io/gorm"
)

func TestUserRepository_CRUD(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user, err := entity.NewUser("Alice", "alice@example.com", "hash", testNow())
	if err != nil {
		t.Fatalf("new user: %v", err)
	}
	if err := repo.Create(ctx, &user); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, found, err := repo.FindByEmail(ctx, "ALICE@example.com")
	if err != nil || !found || got.Email != "alice@example.com" {
		t.Fatalf("find by email: found=%v err=%v got=%+v", found, err, got)
	}

	exists, err := repo.ExistsByEmail(ctx, "alice@example.com")
	if err != nil || !exists {
		t.Fatalf("exists by email: exists=%v err=%v", exists, err)
	}

	byID, found, err := repo.FindByID(ctx, user.ID)
	if err != nil || !found || byID.ID != user.ID {
		t.Fatalf("find by id: %+v", err)
	}

	locked, found, err := repo.FindByIDForUpdate(ctx, user.ID)
	if err != nil || !found {
		t.Fatalf("find for update: %v", err)
	}
	locked.Name = "Alice Updated"
	if err := repo.Update(ctx, locked); err != nil {
		t.Fatalf("update: %v", err)
	}

	if err := repo.IncrementAuthVersion(ctx, user.ID, testNow().Add(time.Minute)); err != nil {
		t.Fatalf("increment auth version: %v", err)
	}
	updated, _, _ := repo.FindByID(ctx, user.ID)
	if updated.AuthVersion != 2 {
		t.Fatalf("auth_version = %d", updated.AuthVersion)
	}
}

func TestUserRepository_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	if _, found, err := repo.FindByEmail(ctx, "missing@example.com"); err != nil || found {
		t.Fatalf("find by email missing: found=%v err=%v", found, err)
	}
	if exists, err := repo.ExistsByEmail(ctx, "missing@example.com"); err != nil || exists {
		t.Fatalf("exists missing: exists=%v err=%v", exists, err)
	}
	if _, found, err := repo.FindByID(ctx, uuid.New()); err != nil || found {
		t.Fatalf("find by id missing: found=%v err=%v", found, err)
	}
	if _, found, err := repo.FindByIDForUpdate(ctx, uuid.New()); err != nil || found {
		t.Fatalf("find by id for update missing: found=%v err=%v", found, err)
	}
	if err := repo.IncrementAuthVersion(ctx, uuid.New(), testNow()); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("increment missing user: %v", err)
	}
}

func TestUserRepository_DBErrors(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()
	user := seedUser(t, db)
	closeDB(t, db)

	if err := repo.Create(ctx, user); err == nil {
		t.Fatal("expected create error")
	}
	if _, _, err := repo.FindByEmail(ctx, "alice@example.com"); err == nil {
		t.Fatal("expected find by email error")
	}
	if _, err := repo.ExistsByEmail(ctx, "alice@example.com"); err == nil {
		t.Fatal("expected exists error")
	}
	if _, _, err := repo.FindByID(ctx, user.ID); err == nil {
		t.Fatal("expected find by id error")
	}
	if _, _, err := repo.FindByIDForUpdate(ctx, user.ID); err == nil {
		t.Fatal("expected find for update error")
	}
	if err := repo.Update(ctx, user); err == nil {
		t.Fatal("expected update error")
	}
	if err := repo.IncrementAuthVersion(ctx, user.ID, testNow()); err == nil {
		t.Fatal("expected increment error")
	}
}

func TestUserRepository_WithinTransaction(t *testing.T) {
	db := setupTestDB(t)
	user := seedUser(t, db)
	txMgr := NewTransactionManager(db)
	repo := NewUserRepository(db)

	if err := txMgr.WithinTransaction(context.Background(), func(ctx context.Context) error {
		_, found, err := repo.FindByIDForUpdate(ctx, user.ID)
		if err != nil || !found {
			return err
		}
		return repo.IncrementAuthVersion(ctx, user.ID, testNow())
	}); err != nil {
		t.Fatalf("tx increment: %v", err)
	}
}
