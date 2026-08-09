package redis

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

type mockRevocationCommands struct {
	setErr error
	getErr error
	getVal string
}

func (m *mockRevocationCommands) Set(context.Context, string, interface{}, time.Duration) *goredis.StatusCmd {
	cmd := goredis.NewStatusCmd(context.Background())
	if m.setErr != nil {
		cmd.SetErr(m.setErr)
	}
	return cmd
}

func (m *mockRevocationCommands) Get(_ context.Context, key string) *goredis.StringCmd {
	cmd := goredis.NewStringCmd(context.Background(), key)
	if m.getErr != nil {
		if errors.Is(m.getErr, goredis.Nil) {
			cmd.SetErr(goredis.Nil)
		} else {
			cmd.SetErr(m.getErr)
		}
		return cmd
	}
	cmd.SetVal(m.getVal)
	return cmd
}

func newRevocationStore(commands IRevocationCommands) *AccessTokenRevocationStore {
	return &AccessTokenRevocationStore{
		commands:    commands,
		withTimeout: func(ctx context.Context) (context.Context, context.CancelFunc) {
			return context.WithCancel(ctx)
		},
	}
}

func TestAccessTokenRevocationStore_RevokeAndIsRevoked(t *testing.T) {
	client, _ := newTestClient(t)
	store := NewAccessTokenRevocationStore(client)
	ctx := context.Background()
	jti := uuid.New()

	if err := store.Revoke(ctx, uuid.Nil, time.Minute); err != nil {
		t.Fatalf("Revoke nil jti: %v", err)
	}
	if err := store.Revoke(ctx, jti, 0); err != nil {
		t.Fatalf("Revoke zero ttl: %v", err)
	}
	if err := store.Revoke(ctx, jti, time.Minute); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	revoked, err := store.IsRevoked(ctx, jti)
	if err != nil || !revoked {
		t.Fatalf("IsRevoked() = %v, %v", revoked, err)
	}

	revoked, err = store.IsRevoked(ctx, uuid.Nil)
	if err != nil || revoked {
		t.Fatalf("nil jti IsRevoked() = %v, %v", revoked, err)
	}
}

func TestAccessTokenRevocationStore_Errors(t *testing.T) {
	ctx := context.Background()
	jti := uuid.New()

	t.Run("revoke set error", func(t *testing.T) {
		store := newRevocationStore(&mockRevocationCommands{setErr: errors.New("set failed")})
		if err := store.Revoke(ctx, jti, time.Minute); err == nil {
			t.Fatal("expected revoke error")
		}
	})

	t.Run("is revoked read error", func(t *testing.T) {
		store := newRevocationStore(&mockRevocationCommands{getErr: errors.New("get failed")})
		if _, err := store.IsRevoked(ctx, jti); err == nil {
			t.Fatal("expected is revoked error")
		}
	})

	t.Run("is revoked missing", func(t *testing.T) {
		store := newRevocationStore(&mockRevocationCommands{getErr: goredis.Nil})
		revoked, err := store.IsRevoked(ctx, jti)
		if err != nil || revoked {
			t.Fatalf("IsRevoked() = %v, %v", revoked, err)
		}
	})
}
