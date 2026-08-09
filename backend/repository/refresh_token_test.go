package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
)

func TestRefreshTokenRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRefreshTokenRepository(db)
	ctx := context.Background()
	user := seedUser(t, db)
	now := testNow()

	active := seedRefreshToken(t, db, user.ID)

	found, ok, err := repo.FindByHashForUpdate(ctx, active.TokenHash)
	if err != nil || !ok || found.ID != active.ID {
		t.Fatalf("find by hash: ok=%v err=%v", ok, err)
	}

	found.Status = entity.RefreshTokenStatusRotated
	if err := repo.Update(ctx, found); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := repo.RevokeFamily(ctx, active.FamilyID, now); err != nil {
		t.Fatalf("revoke family: %v", err)
	}
	if err := repo.RevokeAllByUserID(ctx, user.ID, now); err != nil {
		t.Fatalf("revoke all by user: %v", err)
	}

	expiredHash := validTokenHash()
	expiredHash = expiredHash[:63] + "c"
	expired, err := entity.NewRefreshToken(user.ID, expiredHash, uuid.New(), now.Add(-time.Hour), now.Add(-2*time.Hour))
	if err != nil {
		t.Fatalf("new expired token: %v", err)
	}
	if err := repo.Create(ctx, &expired); err != nil {
		t.Fatalf("create expired: %v", err)
	}

	marked, err := repo.MarkExpiredBefore(ctx, now)
	if err != nil || marked < 1 {
		t.Fatalf("mark expired: marked=%d err=%v", marked, err)
	}

	deleted, err := repo.DeleteStaleBefore(ctx, now.Add(24*time.Hour))
	if err != nil || deleted < 1 {
		t.Fatalf("delete stale: deleted=%d err=%v", deleted, err)
	}
}

func TestRefreshTokenRepository_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRefreshTokenRepository(db)
	if _, ok, err := repo.FindByHashForUpdate(context.Background(), validTokenHash()); err != nil || ok {
		t.Fatalf("expected not found, ok=%v err=%v", ok, err)
	}
}

func TestRefreshTokenRepository_DBErrors(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRefreshTokenRepository(db)
	ctx := context.Background()
	user := seedUser(t, db)
	token := seedRefreshToken(t, db, user.ID)
	closeDB(t, db)

	if err := repo.Create(ctx, token); err == nil {
		t.Fatal("expected create error")
	}
	if _, _, err := repo.FindByHashForUpdate(ctx, token.TokenHash); err == nil {
		t.Fatal("expected find error")
	}
	if err := repo.Update(ctx, token); err == nil {
		t.Fatal("expected update error")
	}
	if err := repo.RevokeFamily(ctx, token.FamilyID, testNow()); err == nil {
		t.Fatal("expected revoke family error")
	}
	if err := repo.RevokeAllByUserID(ctx, user.ID, testNow()); err == nil {
		t.Fatal("expected revoke all error")
	}
	if _, err := repo.MarkExpiredBefore(ctx, testNow()); err == nil {
		t.Fatal("expected mark expired error")
	}
	if _, err := repo.DeleteStaleBefore(ctx, testNow()); err == nil {
		t.Fatal("expected delete stale error")
	}
}

func TestRefreshTokenRepository_WithinTransaction(t *testing.T) {
	db := setupTestDB(t)
	user := seedUser(t, db)
	token := seedRefreshToken(t, db, user.ID)
	txMgr := NewTransactionManager(db)
	repo := NewRefreshTokenRepository(db)

	if err := txMgr.WithinTransaction(context.Background(), func(ctx context.Context) error {
		found, ok, err := repo.FindByHashForUpdate(ctx, token.TokenHash)
		if err != nil || !ok {
			return err
		}
		found.UserAgent = "tx"
		return repo.Update(ctx, found)
	}); err != nil {
		t.Fatalf("tx update: %v", err)
	}
}
