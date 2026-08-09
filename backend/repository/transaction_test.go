package repository

import (
	"context"
	"errors"
	"testing"
)

func TestTransactionContext(t *testing.T) {
	db := setupTestDB(t)
	ctx := WithTransactionContext(context.Background(), db)
	got := dbFromContext(ctx, db)
	if got == nil {
		t.Fatal("expected transaction db from context")
	}

	fallback := dbFromContext(context.Background(), db)
	if fallback == nil {
		t.Fatal("expected fallback db")
	}
}

func TestTransactionManager_CommitAndRollback(t *testing.T) {
	db := setupTestDB(t)
	user := seedUser(t, db)
	txMgr := NewTransactionManager(db)
	users := NewUserRepository(db)

	if err := txMgr.WithinTransaction(context.Background(), func(ctx context.Context) error {
		_, found, err := users.FindByIDForUpdate(ctx, user.ID)
		if err != nil {
			return err
		}
		if !found {
			return errors.New("user not found in tx")
		}
		return nil
	}); err != nil {
		t.Fatalf("commit tx: %v", err)
	}

	if err := txMgr.WithinTransaction(context.Background(), func(context.Context) error {
		return errors.New("rollback")
	}); err == nil {
		t.Fatal("expected rollback error")
	}
}
