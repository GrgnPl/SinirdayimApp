package estimator

import (
	"testing"
	"time"

	"github.com/GrgnPl/SinirdayimApp/backend/internal/domain"
)

var (
	now  = time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	sarp = domain.Crossing{ID: "tr-ge-sarp", Lanes: 1}
)

func newQueue() *Queue { return &Queue{Now: func() time.Time { return now }} }

func TestQueueThroughput(t *testing.T) {
	snaps := []domain.Snapshot{{
		CrossingID: sarp.ID, Direction: domain.DirectionExport, Source: "und",
		ObservedAt:      now.Add(-time.Hour),
		QueueKm:         domain.Ptr(16.0),
		DailyThroughput: domain.Ptr(642),
	}}
	est := newQueue().Estimate(sarp, domain.DirectionExport, snaps)

	// 16 km * 55 = 880 trucks; 642/24 = 26.75 per hour -> ~1973 min (~33 h)
	if *est.Vehicles != 880 {
		t.Errorf("vehicles = %d, want 880", *est.Vehicles)
	}
	if *est.WaitMinutes != 1973 {
		t.Errorf("wait = %d, want 1973", *est.WaitMinutes)
	}
	if est.Level != domain.LevelHigh || est.Confidence != domain.ConfidenceMedium || est.Method != "queue/throughput" {
		t.Errorf("unexpected estimate %+v", est)
	}
}

func TestQueueOnlyAndStale(t *testing.T) {
	snaps := []domain.Snapshot{{
		CrossingID: sarp.ID, Direction: domain.DirectionImport, Source: "und",
		ObservedAt: now.Add(-24 * time.Hour),
		QueueKm:    domain.Ptr(2.0),
	}}
	est := newQueue().Estimate(sarp, domain.DirectionImport, snaps)
	if est.WaitMinutes != nil || *est.Vehicles != 110 || est.Level != domain.LevelMedium {
		t.Errorf("unexpected estimate %+v", est)
	}
	if est.Confidence != domain.ConfidenceLow {
		t.Errorf("stale data should be low confidence, got %s", est.Confidence)
	}
}

func TestReportedWaitWins(t *testing.T) {
	snaps := []domain.Snapshot{
		{CrossingID: sarp.ID, Direction: domain.DirectionExport, Source: "und", ObservedAt: now, QueueKm: domain.Ptr(10.0), DailyThroughput: domain.Ptr(500)},
		{CrossingID: sarp.ID, Direction: domain.DirectionExport, Source: "crowd", ObservedAt: now.Add(-time.Hour), WaitMinutes: domain.Ptr(90)},
	}
	est := newQueue().Estimate(sarp, domain.DirectionExport, snaps)
	if *est.WaitMinutes != 90 || est.Method != "reported" || est.Level != domain.LevelLow {
		t.Errorf("unexpected estimate %+v", est)
	}
}

func TestNoData(t *testing.T) {
	est := newQueue().Estimate(sarp, domain.DirectionExport, nil)
	if est.Level != domain.LevelUnknown || est.Vehicles != nil {
		t.Errorf("unexpected estimate %+v", est)
	}
}
