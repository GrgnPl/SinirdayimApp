package usecase

import (
	"context"
	"log/slog"
	"time"

	"github.com/burakaydin/sinir-bekleme/backend/internal/port"
)

// RestAreaIngestor refreshes stopping places from every source on its own
// interval. A failed refresh keeps the previously loaded places.
type RestAreaIngestor struct {
	Sources []port.RestAreaSource
	Store   port.RestAreaStore
	Log     *slog.Logger
}

func (in *RestAreaIngestor) Run(ctx context.Context) {
	for _, src := range in.Sources {
		go func() {
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
		}()
	}
	<-ctx.Done()
}

func (in *RestAreaIngestor) RunOnce(ctx context.Context, src port.RestAreaSource) {
	start := time.Now()
	areas, err := src.Fetch(ctx)
	if err != nil {
		in.Log.Error("rest area fetch failed", "source", src.ID(), "err", err)
		return
	}
	if err := in.Store.Replace(ctx, src.ID(), areas); err != nil {
		in.Log.Error("rest area save failed", "source", src.ID(), "err", err)
		return
	}
	in.Log.Info("rest areas ingested", "source", src.ID(), "count", len(areas), "took", time.Since(start).Round(time.Second))
}
