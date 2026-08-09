package batch

import (
	"context"
	"log"
	"time"

	"github.com/koezuka404/notehub/usecase"
)

type CleanupBatch struct {
	cleanup  *usecase.CleanupUseCase
	interval time.Duration
}

func NewCleanupBatch(cleanup *usecase.CleanupUseCase, interval time.Duration) *CleanupBatch {
	if interval <= 0 {
		interval = time.Hour
	}
	return &CleanupBatch{cleanup: cleanup, interval: interval}
}

func (b *CleanupBatch) Run(ctx context.Context) {
	ticker := time.NewTicker(b.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
			if err := runCleanupOnce(b, runCtx); err != nil {
				log.Printf("cleanup batch: %v", err)
			}
			cancel()
		}
	}
}
