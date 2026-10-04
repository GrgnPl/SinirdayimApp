// Command api runs the ingestion loop and the REST API in one process.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/burakaydin/sinir-bekleme/backend/internal/adapter/geocoding/photon"
	"github.com/burakaydin/sinir-bekleme/backend/internal/adapter/httpapi"
	"github.com/burakaydin/sinir-bekleme/backend/internal/adapter/osm"
	"github.com/burakaydin/sinir-bekleme/backend/internal/adapter/repository/memory"
	"github.com/burakaydin/sinir-bekleme/backend/internal/adapter/routing/valhalla"
	"github.com/burakaydin/sinir-bekleme/backend/internal/adapter/source/und"
	"github.com/burakaydin/sinir-bekleme/backend/internal/catalog"
	"github.com/burakaydin/sinir-bekleme/backend/internal/domain"
	"github.com/burakaydin/sinir-bekleme/backend/internal/estimator"
	"github.com/burakaydin/sinir-bekleme/backend/internal/port"
	"github.com/burakaydin/sinir-bekleme/backend/internal/usecase"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	undSrc, err := und.New()
	if err != nil {
		return err
	}
	// Register new data integrations here.
	sources := []port.SnapshotSource{undSrc}

	repo := memory.New()
	reports := memory.NewReports()
	crossings := catalog.NewStatic()
	est := estimator.NewQueue()
	status := &usecase.StatusService{
		Catalog:   crossings,
		Repo:      repo,
		Estimator: est,
		Reports:   reports,
	}
	queue := &usecase.QueueService{Catalog: crossings, Snapshots: repo, Reports: reports, Estimator: est}
	restAreas := memory.NewRestAreas()
	borderPoints := memory.NewBorderPoints()
	trips := &usecase.TripService{
		Router:       valhalla.New(os.Getenv("VALHALLA_URL")),
		Catalog:      crossings,
		Status:       status,
		RestAreas:    restAreas,
		BorderPoints: borderPoints,
	}
	ingestor := &usecase.Ingestor{Sources: sources, Repo: repo, Log: log}
	go ingestor.Run(ctx)

	var countries []string
	if v := os.Getenv("OSM_COUNTRIES"); v != "" {
		countries = strings.Split(v, ",")
	}
	osmClient := osm.NewClient(os.Getenv("OVERPASS_URL"), countries)
	osmClient.Log = log
	osmClient.CacheDir = envOr("CACHE_DIR", defaultCacheDir())
	borderIngest := &usecase.DatasetIngestor[domain.BorderPoint]{Name: "border points", Store: borderPoints, Log: log}
	restIngest := &usecase.DatasetIngestor[domain.RestArea]{Name: "rest areas", Store: restAreas, Log: log}
	// Both datasets share one client and refresh one after another to stay
	// within the public Overpass server's limits.
	go func() {
		for {
			borderIngest.RunOnce(ctx, osm.BorderControls{Client: osmClient})
			restIngest.RunOnce(ctx, osm.RestAreas{Client: osmClient})
			select {
			case <-ctx.Done():
				return
			case <-time.After(osmClient.MaxAge):
			}
		}
	}()

	addr := ":" + envOr("PORT", "8080")
	srv := &http.Server{
		Addr: addr,
		Handler: (&httpapi.Handler{
			Status:   status,
			Trips:    trips,
			Queue:    queue,
			Geocoder: photon.New(os.Getenv("PHOTON_URL")),
			Log:      log,
		}).Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	log.Info("listening", "addr", addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// defaultCacheDir is the per-user cache folder, e.g. ~/Library/Caches/sinirdayim.
func defaultCacheDir() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "sinirdayim")
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
