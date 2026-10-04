package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/burakaydin/sinir-bekleme/backend/internal/adapter/repository/memory"
	"github.com/burakaydin/sinir-bekleme/backend/internal/catalog"
	"github.com/burakaydin/sinir-bekleme/backend/internal/domain"
	"github.com/burakaydin/sinir-bekleme/backend/internal/estimator"
)

func newQueueService(t *testing.T, snaps ...domain.Snapshot) *QueueService {
	t.Helper()
	repo := memory.New()
	if err := repo.Save(context.Background(), snaps); err != nil {
		t.Fatal(err)
	}
	now := func() time.Time { return depart }
	return &QueueService{
		Catalog: catalog.NewStatic(), Snapshots: repo, Reports: memory.NewReports(),
		Estimator: &estimator.Queue{Now: now}, Now: now,
	}
}

func sarpSnap(ago time.Duration, km float64) domain.Snapshot {
	return domain.Snapshot{
		CrossingID: "tr-ge-sarp", Direction: domain.DirectionExport, Source: "und",
		ObservedAt: depart.Add(-ago), QueueKm: domain.Ptr(km), DailyThroughput: domain.Ptr(24 * 55),
	}
}

func exportOf(t *testing.T, d QueueDetail) DirectionQueue {
	t.Helper()
	for _, dq := range d.Directions {
		if dq.Direction == domain.DirectionExport {
			return dq
		}
	}
	t.Fatal("no export direction")
	return DirectionQueue{}
}

func TestQueueTrendAndOutlook(t *testing.T) {
	// 4 km (220 trucks) 10 h ago, 6 km (330 trucks) 2 h ago: +13.75 trucks/h.
	svc := newQueueService(t, sarpSnap(10*time.Hour, 4), sarpSnap(2*time.Hour, 6))
	dq := exportOf(t, mustDetail(t, svc))
	if dq.Trend == nil || dq.Trend.Direction != "growing" || dq.Trend.VehiclesPerHour < 13 || dq.Trend.VehiclesPerHour > 14 {
		t.Fatalf("trend = %+v", dq.Trend)
	}
	if len(dq.Outlook) != OutlookHours {
		t.Fatalf("outlook has %d hours", len(dq.Outlook))
	}
	// Now: 330 + 2 h × 13.75 ≈ 357 trucks at 55/h ≈ 390 min.
	if w := dq.Outlook[0].WaitMinutes; w < 380 || w > 400 {
		t.Errorf("wait now = %d", w)
	}
	// The queue keeps growing for 6 h, then stays.
	if dq.Outlook[6].Vehicles <= dq.Outlook[0].Vehicles || dq.Outlook[11].Vehicles != dq.Outlook[6].Vehicles {
		t.Errorf("outlook vehicles = %d, %d, %d", dq.Outlook[0].Vehicles, dq.Outlook[6].Vehicles, dq.Outlook[11].Vehicles)
	}
}

func TestDriverReportsOverrideOfficialWait(t *testing.T) {
	svc := newQueueService(t, sarpSnap(8*time.Hour, 6))
	ctx := context.Background()
	for _, w := range []int{120, 150, 600} {
		if _, err := svc.Report(ctx, domain.DriverReport{
			CrossingID: "tr-ge-sarp", Direction: domain.DirectionExport, Kind: domain.ReportPassed,
			At: depart.Add(-time.Hour), WaitMinutes: domain.Ptr(w),
		}); err != nil {
			t.Fatal(err)
		}
	}
	dq := exportOf(t, mustDetail(t, svc))
	if *dq.Estimate.WaitMinutes != 150 || dq.Estimate.Method != "driver_reports" || dq.Estimate.Confidence != domain.ConfidenceHigh {
		t.Errorf("estimate = %+v, want median 150 from drivers", dq.Estimate)
	}
	if *dq.Official.WaitMinutes == 150 {
		t.Error("official estimate must stay separate")
	}
	if w := dq.Outlook[0].WaitMinutes; w < 140 || w > 160 {
		t.Errorf("outlook starts at %d min, want ~150 from the measured wait", w)
	}
	if len(dq.Reports) != 3 || dq.Sources[0].Source != "drivers" {
		t.Errorf("reports = %d, sources = %+v", len(dq.Reports), dq.Sources)
	}
}

func TestInQueueReportUsesThroughput(t *testing.T) {
	svc := newQueueService(t, sarpSnap(8*time.Hour, 6))
	_, err := svc.Report(context.Background(), domain.DriverReport{
		CrossingID: "tr-ge-sarp", Direction: domain.DirectionExport, Kind: domain.ReportInQueue, VehiclesAhead: domain.Ptr(110),
	})
	if err != nil {
		t.Fatal(err)
	}
	est := exportOf(t, mustDetail(t, svc)).Estimate
	if *est.Vehicles != 110 || *est.WaitMinutes != 120 || est.Method != "driver_queue/throughput" {
		t.Errorf("estimate = %+v, want 110 trucks / 2 h", est)
	}
}

func TestInvalidReports(t *testing.T) {
	svc := newQueueService(t)
	ctx := context.Background()
	for name, r := range map[string]domain.DriverReport{
		"no data":  {CrossingID: "tr-ge-sarp", Direction: domain.DirectionExport, Kind: domain.ReportInQueue},
		"bad kind": {CrossingID: "tr-ge-sarp", Direction: domain.DirectionExport, Kind: "x", WaitMinutes: domain.Ptr(1)},
		"bad dir":  {CrossingID: "tr-ge-sarp", Direction: "up", Kind: domain.ReportPassed, WaitMinutes: domain.Ptr(1)},
		"too old":  {CrossingID: "tr-ge-sarp", Direction: domain.DirectionExport, Kind: domain.ReportPassed, WaitMinutes: domain.Ptr(1), At: depart.Add(-13 * time.Hour)},
		"negative": {CrossingID: "tr-ge-sarp", Direction: domain.DirectionExport, Kind: domain.ReportInQueue, VehiclesAhead: domain.Ptr(-1)},
	} {
		if _, err := svc.Report(ctx, r); !errors.Is(err, ErrInvalidReport) {
			t.Errorf("%s: err = %v, want ErrInvalidReport", name, err)
		}
	}
	if _, err := svc.Report(ctx, domain.DriverReport{CrossingID: "nope", Direction: domain.DirectionExport, Kind: domain.ReportPassed, WaitMinutes: domain.Ptr(1)}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("unknown crossing: err = %v", err)
	}
}

func mustDetail(t *testing.T, svc *QueueService) QueueDetail {
	t.Helper()
	d, err := svc.Detail(context.Background(), "tr-ge-sarp")
	if err != nil {
		t.Fatal(err)
	}
	return d
}
