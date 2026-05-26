package timer

import (
	"context"
	"log/slog"
	"time"
)

type Indexer interface {
	BuildIndex(ctx context.Context) error
}

type Initiator struct {
	indexer  Indexer
	interval time.Duration
}

func New(indexer Indexer, interval time.Duration) *Initiator {
	return &Initiator{indexer: indexer, interval: interval}
}

func (i *Initiator) Start(ctx context.Context) {
	if err := i.indexer.BuildIndex(ctx); err != nil {
		slog.Error("build index", "err", err)
	}

	go func() {
		ticker := time.NewTicker(i.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				slog.Info("Index ticker stopped")
				return

			case <-ticker.C:
				if err := i.indexer.BuildIndex(ctx); err != nil {
					slog.Error("rebuild index", "err", err)
				}
			}
		}
	}()
}
