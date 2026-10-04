package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/burakaydin/sinir-bekleme/backend/internal/domain"
	"github.com/burakaydin/sinir-bekleme/backend/internal/port"
)

const (
	// OutlookHours is how far the hourly outlook looks ahead.
	OutlookHours = 12
	// reportListWindow is how far back reports are listed on the detail.
	reportListWindow = 24 * time.Hour
	// trendHistory is how much snapshot history the trend looks at.
	trendHistory = 72 * time.Hour
	// maxReportAge / maxReportAhead bound report timestamps sent by clients.
	maxReportAge   = 12 * time.Hour
	maxReportAhead = 5 * time.Minute
)

var ErrInvalidReport = errors.New("invalid report")

// SourceStatus tells what one source last said about a direction.
type SourceStatus struct {
	Source     string           `json:"source"`
	ObservedAt time.Time        `json:"observedAt"`
	Snapshot   *domain.Snapshot `json:"snapshot,omitempty"`
	Reports    int              `json:"reports,omitempty"` // for driver reports: how many in the window
}

// DirectionQueue is everything known about the queue in one direction.
type DirectionQueue struct {
	Direction domain.Direction `json:"direction"`
	// Estimate combines official data and recent driver reports.
	Estimate domain.WaitEstimate `json:"estimate"`
	// Official is the estimate from official data only.
	Official        domain.WaitEstimate   `json:"official"`
	DailyThroughput *int                  `json:"dailyThroughput,omitempty"`
	Trend           *domain.Trend         `json:"trend,omitempty"`
	Outlook         []domain.HourOutlook  `json:"outlook"`
	Sources         []SourceStatus        `json:"sources"`
	Reports         []domain.DriverReport `json:"reports"`
	Procedures      []domain.Procedure    `json:"procedures,omitempty"`
}

// QueueDetail is the detailed queue view of one crossing.
type QueueDetail struct {
	Crossing   domain.Crossing  `json:"crossing"`
	Directions []DirectionQueue `json:"directions"`
	At         time.Time        `json:"at"`
}

type QueueService struct {
	Catalog   port.CrossingCatalog
	Snapshots port.SnapshotRepository
	Reports   port.ReportStore
	Estimator port.Estimator
	Now       func() time.Time
}

func (s *QueueService) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// Detail builds the queue view for both directions of a crossing.
func (s *QueueService) Detail(ctx context.Context, id domain.CrossingID) (QueueDetail, error) {
	c, err := s.Catalog.Get(ctx, id)
	if err != nil {
		return QueueDetail{}, err
	}
	now := s.now()
	latest, err := s.Snapshots.Latest(ctx, id)
	if err != nil {
		return QueueDetail{}, err
	}
	history, err := s.Snapshots.History(ctx, id, now.Add(-trendHistory))
	if err != nil {
		return QueueDetail{}, err
	}
	reports, err := s.Reports.Since(ctx, id, now.Add(-reportListWindow))
	if err != nil {
		return QueueDetail{}, err
	}

	out := QueueDetail{Crossing: c, At: now}
	for _, dir := range []domain.Direction{domain.DirectionExport, domain.DirectionImport} {
		out.Directions = append(out.Directions, s.direction(c, dir, latest, history, reports))
	}
	return out, nil
}

func (s *QueueService) direction(c domain.Crossing, dir domain.Direction, latest, history []domain.Snapshot, reports []domain.DriverReport) DirectionQueue {
	official := s.Estimator.Estimate(c, dir, latest)
	throughput := latestThroughput(latest, dir)
	dq := DirectionQueue{
		Direction:       dir,
		Official:        official,
		Estimate:        s.Estimator.WithReports(official, c, reports, throughput),
		DailyThroughput: throughput,
		Procedures:      c.ProceduresFor(dir),
		Reports:         []domain.DriverReport{},
		Sources:         []SourceStatus{},
	}
	dq.Trend = s.Estimator.Trend(c, sameSource(history, dir, official.BasedOn))
	dq.Outlook = s.Estimator.Outlook(dq.Estimate, throughput, dq.Trend, OutlookHours)
	if dq.Outlook == nil {
		dq.Outlook = []domain.HourOutlook{}
	}

	for _, snap := range latest {
		if snap.Direction == dir {
			snap := snap
			dq.Sources = append(dq.Sources, SourceStatus{Source: snap.Source, ObservedAt: snap.ObservedAt, Snapshot: &snap})
		}
	}
	var newest time.Time
	for _, r := range reports {
		if r.Direction == dir {
			dq.Reports = append(dq.Reports, r)
			if r.At.After(newest) {
				newest = r.At
			}
		}
	}
	if len(dq.Reports) > 0 {
		dq.Sources = append(dq.Sources, SourceStatus{Source: "drivers", ObservedAt: newest, Reports: len(dq.Reports)})
	}
	sort.Slice(dq.Sources, func(i, j int) bool { return dq.Sources[i].ObservedAt.After(dq.Sources[j].ObservedAt) })
	return dq
}

// Report validates and stores a driver report.
func (s *QueueService) Report(ctx context.Context, r domain.DriverReport) (domain.DriverReport, error) {
	if _, err := s.Catalog.Get(ctx, r.CrossingID); err != nil {
		return domain.DriverReport{}, err
	}
	now := s.now()
	if r.At.IsZero() {
		r.At = now
	}
	if err := validateReport(r, now); err != nil {
		return domain.DriverReport{}, err
	}
	r.Note = strings.TrimSpace(r.Note)
	// Keep only the fields that belong to the kind.
	if r.Kind == domain.ReportInQueue {
		r.WaitMinutes = nil
	} else {
		r.VehiclesAhead, r.QueueKm = nil, nil
	}
	r.ID = newID()
	if err := s.Reports.Add(ctx, r); err != nil {
		return domain.DriverReport{}, err
	}
	return r, nil
}

func validateReport(r domain.DriverReport, now time.Time) error {
	bad := func(msg string) error { return fmt.Errorf("%w: %s", ErrInvalidReport, msg) }
	if r.Direction != domain.DirectionExport && r.Direction != domain.DirectionImport {
		return bad("direction must be export or import")
	}
	if r.At.Before(now.Add(-maxReportAge)) || r.At.After(now.Add(maxReportAhead)) {
		return bad("report time is out of range")
	}
	if len([]rune(r.Note)) > domain.MaxReportNoteLen {
		return bad("note is too long")
	}
	switch r.Kind {
	case domain.ReportInQueue:
		if r.VehiclesAhead == nil && r.QueueKm == nil {
			return bad("in_queue needs vehiclesAhead or queueKm")
		}
		if r.VehiclesAhead != nil && (*r.VehiclesAhead < 0 || *r.VehiclesAhead > domain.MaxReportVehicles) {
			return bad("vehiclesAhead out of range")
		}
		if r.QueueKm != nil && (*r.QueueKm < 0 || *r.QueueKm > domain.MaxReportQueueKm) {
			return bad("queueKm out of range")
		}
	case domain.ReportPassed:
		if r.WaitMinutes == nil || *r.WaitMinutes < 0 || *r.WaitMinutes > domain.MaxReportWaitMin {
			return bad("passed needs waitMinutes in range")
		}
	default:
		return bad("kind must be in_queue or passed")
	}
	return nil
}

// latestThroughput is the newest daily throughput reported for a direction.
func latestThroughput(latest []domain.Snapshot, dir domain.Direction) *int {
	var best *domain.Snapshot
	for i := range latest {
		s := &latest[i]
		if s.Direction == dir && s.DailyThroughput != nil && (best == nil || s.ObservedAt.After(best.ObservedAt)) {
			best = s
		}
	}
	if best == nil {
		return nil
	}
	return best.DailyThroughput
}

// sameSource keeps the history of the direction from the official estimate's
// source, so a trend never mixes two sources' scales.
func sameSource(history []domain.Snapshot, dir domain.Direction, basedOn []string) []domain.Snapshot {
	src := ""
	for _, b := range basedOn {
		if b != "drivers" {
			src = b
			break
		}
	}
	var out []domain.Snapshot
	for _, h := range history {
		if h.Direction == dir && h.Source == src {
			out = append(out, h)
		}
	}
	return out
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
