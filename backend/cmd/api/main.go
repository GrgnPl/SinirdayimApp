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
	"github.com/burakaydin/sinir-bekleme/backend/internal/adapter/repository/memory"
	"github.com/burakaydin/sinir-bekleme/backend/internal/adapter/restarea/overpass"
	"github.com/burakaydin/sinir-bekleme/backend/internal/adapter/routing/valhalla"
	"github.com/burakaydin/sinir-bekleme/backend/internal/adapter/source/und"
	"github.com/burakaydin/sinir-bekleme/backend/internal/catalog"
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
	crossings := catalog.NewStatic()
	status := &usecase.StatusService{
		Catalog:   crossings,
		Repo:      repo,
		Estimator: estimator.NewQueue(),
	}
	restAreas := memory.NewRestAreas()
	trips := &usecase.TripService{
		Router:    valhalla.New(os.Getenv("VALHALLA_URL")),
		Catalog:   crossings,
		Status:    status,
		RestAreas: restAreas,
	}
	ingestor := &usecase.Ingestor{Sources: sources, Repo: repo, Log: log}
	go ingestor.Run(ctx)

	var countries []string
	if v := os.Getenv("REST_AREA_COUNTRIES"); v != "" {
		countries = strings.Split(v, ",")
	}
	osm := overpass.New(os.Getenv("OVERPASS_URL"), countries)
	osm.Log = log
	osm.CacheDir = envOr("CACHE_DIR", defaultCacheDir())
	restIngestor := &usecase.RestAreaIngestor{
		Sources: []port.RestAreaSource{osm},
		Store:   restAreas,
		Log:     log,
	}
	go restIngestor.Run(ctx)

	addr := ":" + envOr("PORT", "8080")
	srv := &http.Server{
		Addr: addr,
		Handler: (&httpapi.Handler{
			Status:   status,
			Trips:    trips,
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
