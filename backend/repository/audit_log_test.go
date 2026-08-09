package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
)

func TestAuditLogRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAuditLogRepository(db)
	ctx := context.Background()
	userID := uuid.New()
	log, err := entity.NewAuditLog(&userID, "LOGIN", "user", &userID, nil, testNow())
	if err != nil {
		t.Fatalf("new audit log: %v", err)
	}
	if err := repo.Create(ctx, &log); err != nil {
		t.Fatalf("create: %v", err)
	}
}

func TestAuditLogRepository_DBError(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAuditLogRepository(db)
	ctx := context.Background()
	userID := uuid.New()
	log, err := entity.NewAuditLog(&userID, "LOGIN", "user", &userID, nil, testNow())
	if err != nil {
		t.Fatalf("new audit log: %v", err)
	}
	closeDB(t, db)
	if err := repo.Create(ctx, &log); err == nil {
		t.Fatal("expected create error")
	}
}

func TestAuditLogRepository_WithinTransaction(t *testing.T) {
	db := setupTestDB(t)
	txMgr := NewTransactionManager(db)
	repo := NewAuditLogRepository(db)
	userID := uuid.New()

	if err := txMgr.WithinTransaction(context.Background(), func(ctx context.Context) error {
		log, err := entity.NewAuditLog(&userID, "LOGOUT", "user", &userID, nil, testNow())
		if err != nil {
			return err
		}
		return repo.Create(ctx, &log)
	}); err != nil {
		t.Fatalf("tx create audit log: %v", err)
	}
}
