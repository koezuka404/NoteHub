package usecase

import (
	"context"
	"log"
	"time"
)

type ICleanupRefreshTokenRepository interface {
	MarkExpiredBefore(ctx context.Context, now time.Time) (int64, error)
	DeleteStaleBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

type ICleanupRedisStore interface {
	CleanupEphemeralKeys(ctx context.Context) (int, error)
}

type CleanupUseCase struct {
	refreshTokens ICleanupRefreshTokenRepository
	redis         ICleanupRedisStore
	retention     time.Duration
	now           func() time.Time
}

func NewCleanupUseCase(
	refreshTokens ICleanupRefreshTokenRepository,
	redis ICleanupRedisStore,
	retention time.Duration,
) *CleanupUseCase {
	if retention <= 0 {
		retention = 30 * 24 * time.Hour
	}
	return &CleanupUseCase{
		refreshTokens: refreshTokens,
		redis:         redis,
		retention:     retention,
		now:           time.Now,
	}
}

func (u *CleanupUseCase) RunOnce(ctx context.Context) error {
	now := u.now()

	marked, err := u.refreshTokens.MarkExpiredBefore(ctx, now)
	if err != nil {
		return err
	}

	cutoff := now.Add(-u.retention)
	deleted, err := u.refreshTokens.DeleteStaleBefore(ctx, cutoff)
	if err != nil {
		return err
	}

	removed, err := u.redis.CleanupEphemeralKeys(ctx)
	if err != nil {
		return err
	}

	if marked > 0 || deleted > 0 || removed > 0 {
		log.Printf(
			"cleanup batch: marked %d expired refresh tokens, deleted %d stale refresh tokens, removed %d redis keys",
			marked,
			deleted,
			removed,
		)
	}
	return nil
}
