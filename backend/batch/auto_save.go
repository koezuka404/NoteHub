package batch

import (
	"context"
	"log"
	"time"

	"github.com/koezuka404/notehub/usecase"
)

type AutoSaveBatch struct {
	autosave *usecase.DocumentAutoSaveUseCase
	interval time.Duration
}

func NewAutoSaveBatch(autosave *usecase.DocumentAutoSaveUseCase, interval time.Duration) *AutoSaveBatch {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	return &AutoSaveBatch{autosave: autosave, interval: interval}
}

func (b *AutoSaveBatch) Run(ctx context.Context) {
	ticker := time.NewTicker(b.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			if err := runAutoSaveOnce(b, runCtx); err != nil {
				log.Printf("autosave batch: %v", err)
			}
			cancel()
		}
	}
}
