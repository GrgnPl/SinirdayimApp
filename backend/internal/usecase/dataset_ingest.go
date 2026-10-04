package usecase

import (
	"context"
	"log/slog"
	"time"

	"github.com/GrgnPl/SinirdayimApp/backend/internal/port"
)

// DatasetIngestor loads reference data from a source into a store. A failed
// refresh keeps the previously loaded data.
type DatasetIngestor[T any] struct {
	Name  string // for logs, e.g. "rest areas"
	Store port.GeoStore[T]
	Log   *slog.Logger
}

func (in *DatasetIngestor[T]) RunOnce(ctx context.Context, src port.Dataset[T]) {
	start := time.Now()
	items, err := src.Fetch(ctx)
	if err != nil {
		in.Log.Error("dataset fetch failed", "dataset", in.Name, "source", src.ID(), "err", err)
		return
	}
	if err := in.Store.Replace(ctx, src.ID(), items); err != nil {
		in.Log.Error("dataset save failed", "dataset", in.Name, "source", src.ID(), "err", err)
		return
	}
	in.Log.Info("dataset ingested", "dataset", in.Name, "source", src.ID(), "count", len(items), "took", time.Since(start).Round(time.Second))
}
