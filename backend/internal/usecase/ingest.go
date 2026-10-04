package usecase

import (
	"context"
	"log/slog"
	"time"

	"github.com/GrgnPl/SinirdayimApp/backend/internal/port"
)

// Ingestor polls every registered source on its own interval and stores the
// results. Sources are independent: one failing does not stop the others.
type Ingestor struct {
	Sources []port.SnapshotSource
	Repo    port.SnapshotRepository
	Log     *slog.Logger
}

// Run blocks until ctx is cancelled.
func (in *Ingestor) Run(ctx context.Context) {
	done := make(chan struct{})
	for _, src := range in.Sources {
		go func() {
			in.loop(ctx, src)
			done <- struct{}{}
		}()
	}
	for range in.Sources {
		<-done
	}
}

func (in *Ingestor) loop(ctx context.Context, src port.SnapshotSource) {
	in.RunOnce(ctx, src)
	t := time.NewTicker(src.Interval())
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			in.RunOnce(ctx, src)
		}
	}
}

// RunOnce fetches a single source and saves its snapshots.
func (in *Ingestor) RunOnce(ctx context.Context, src port.SnapshotSource) {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	snaps, err := src.Fetch(ctx)
	if err != nil {
		in.Log.Error("source fetch failed", "source", src.ID(), "err", err)
		return
	}
	if err := in.Repo.Save(ctx, snaps); err != nil {
		in.Log.Error("save snapshots failed", "source", src.ID(), "err", err)
		return
	}
	in.Log.Info("source ingested", "source", src.ID(), "snapshots", len(snaps))
}
