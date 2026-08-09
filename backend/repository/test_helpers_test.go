package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func testNow() time.Time {
	return time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
}

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&entity.User{},
		&entity.RefreshToken{},
		&entity.AuditLog{},
		&entity.Workspace{},
		&entity.WorkspaceMember{},
		&entity.Document{},
		&entity.DocumentVersion{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func closeDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db(): %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}
}

func seedUser(t *testing.T, db *gorm.DB) *entity.User {
	t.Helper()
	user, err := entity.NewUser("Alice", "alice@example.com", "hash", testNow())
	if err != nil {
		t.Fatalf("new user: %v", err)
	}
	if err := NewUserRepository(db).Create(context.Background(), &user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	return &user
}

func validTokenHash() string {
	return strings.Repeat("a", 64)
}

func seedRefreshToken(t *testing.T, db *gorm.DB, userID uuid.UUID) *entity.RefreshToken {
	t.Helper()
	now := testNow()
	token, err := entity.NewRefreshToken(userID, validTokenHash(), uuid.New(), now.Add(time.Hour), now)
	if err != nil {
		t.Fatalf("new refresh token: %v", err)
	}
	if err := NewRefreshTokenRepository(db).Create(context.Background(), &token); err != nil {
		t.Fatalf("create refresh token: %v", err)
	}
	return &token
}

func seedWorkspace(t *testing.T, db *gorm.DB, host *entity.User) *entity.Workspace {
	t.Helper()
	now := testNow()
	ws, err := entity.NewWorkspace(host.ID, "Team", now)
	if err != nil {
		t.Fatalf("new workspace: %v", err)
	}
	if err := NewWorkspaceRepository(db).Create(context.Background(), &ws); err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	member, err := entity.NewWorkspaceMember(ws.ID, host.ID, entity.WorkspaceRoleHost, now)
	if err != nil {
		t.Fatalf("new member: %v", err)
	}
	if err := NewWorkspaceMemberRepository(db).Create(context.Background(), &member); err != nil {
		t.Fatalf("create member: %v", err)
	}
	return &ws
}

func seedDocument(t *testing.T, db *gorm.DB, ws *entity.Workspace, actor *entity.User) *entity.Document {
	t.Helper()
	doc, err := entity.NewDocument(ws.ID, actor.ID, "Doc", "content", testNow())
	if err != nil {
		t.Fatalf("new document: %v", err)
	}
	if err := NewDocumentRepository(db).Create(context.Background(), &doc); err != nil {
		t.Fatalf("create document: %v", err)
	}
	return &doc
}
