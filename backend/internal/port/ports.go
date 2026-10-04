// Package port defines the interfaces between the use cases and the outside
// world. New integrations (data sources, storage, notifiers) implement these.
package port

import (
	"context"
	"time"

	"github.com/burakaydin/sinir-bekleme/backend/internal/domain"
)

// SnapshotSource is a data provider integration (official site, third-party
// API, crowdsourcing, camera counting...). Adding a new source means
// implementing this interface and registering it in cmd/api.
type SnapshotSource interface {
	// ID is a short, unique provider id such as "und".
	ID() string
	// Interval is how often the source should be polled.
	Interval() time.Duration
	// Fetch returns the latest snapshots the source publishes.
	Fetch(ctx context.Context) ([]domain.Snapshot, error)
}

// CrossingCatalog is the reference list of known crossings.
type CrossingCatalog interface {
	List(ctx context.Context) ([]domain.Crossing, error)
	Get(ctx context.Context, id domain.CrossingID) (domain.Crossing, error)
}

// SnapshotRepository persists observations.
type SnapshotRepository interface {
	// Save stores snapshots, ignoring exact duplicates
	// (same crossing, direction, source and observation time).
	Save(ctx context.Context, snaps []domain.Snapshot) error
	// Latest returns the newest snapshot per (direction, source) for a crossing.
	Latest(ctx context.Context, id domain.CrossingID) ([]domain.Snapshot, error)
	// History returns snapshots for a crossing observed at or after since, oldest first.
	History(ctx context.Context, id domain.CrossingID, since time.Time) ([]domain.Snapshot, error)
}

// Estimator turns observations into wait estimates.
type Estimator interface {
	// Estimate uses the newest official snapshots.
	Estimate(c domain.Crossing, dir domain.Direction, latest []domain.Snapshot) domain.WaitEstimate
	// WithReports refines an estimate with recent driver reports.
	WithReports(est domain.WaitEstimate, c domain.Crossing, reports []domain.DriverReport, dailyThroughput *int) domain.WaitEstimate
	// Trend is how the queue changes, from snapshot history of one source and direction.
	Trend(c domain.Crossing, history []domain.Snapshot) *domain.Trend
	// Outlook is the expected wait for each of the next hours.
	Outlook(est domain.WaitEstimate, dailyThroughput *int, trend *domain.Trend, hours int) []domain.HourOutlook
}

// Router computes truck routes (Valhalla, HERE, TomTom...). The route must
// pass through every via point in order without stopping.
type Router interface {
	Route(ctx context.Context, from, to domain.GeoPoint, via ...domain.GeoPoint) (domain.Route, error)
}

// Geocoder turns free text into places.
type Geocoder interface {
	Search(ctx context.Context, query string, limit int) ([]domain.Place, error)
}

// Dataset is a periodically refreshed collection of reference data
// (rest areas, border posts...) from one provider.
type Dataset[T any] interface {
	ID() string
	Interval() time.Duration
	Fetch(ctx context.Context) ([]T, error)
}

// GeoStore keeps located reference data, replaced per provider.
type GeoStore[T any] interface {
	// Replace swaps everything previously stored for sourceID.
	Replace(ctx context.Context, sourceID string, items []T) error
	// InBounds returns items inside the south-west / north-east box.
	InBounds(ctx context.Context, sw, ne domain.GeoPoint) ([]T, error)
}

type (
	// RestAreaSource provides stopping places (OSM, operator feeds, user reports).
	RestAreaSource = Dataset[domain.RestArea]
	RestAreaStore  = GeoStore[domain.RestArea]
	// BorderPointSource provides border control posts from map data.
	BorderPointSource = Dataset[domain.BorderPoint]
	BorderPointStore  = GeoStore[domain.BorderPoint]
)

// ReportStore keeps driver reports.
type ReportStore interface {
	Add(ctx context.Context, r domain.DriverReport) error
	// Since returns reports for a crossing at or after t, newest first.
	Since(ctx context.Context, id domain.CrossingID, t time.Time) ([]domain.DriverReport, error)
}
