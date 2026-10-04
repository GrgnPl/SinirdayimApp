package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/burakaydin/sinir-bekleme/backend/internal/adapter/repository/memory"
	"github.com/burakaydin/sinir-bekleme/backend/internal/catalog"
	"github.com/burakaydin/sinir-bekleme/backend/internal/domain"
	"github.com/burakaydin/sinir-bekleme/backend/internal/estimator"
	"github.com/burakaydin/sinir-bekleme/backend/internal/geo"
	"github.com/burakaydin/sinir-bekleme/backend/internal/tacho"
)

// lineRouter returns a straight route through the given points, each step
// driven at a constant speed.
type lineRouter struct {
	points []domain.GeoPoint
	kmh    float64
}

func (l lineRouter) Route(_ context.Context, from, to domain.GeoPoint, via ...domain.GeoPoint) (domain.Route, error) {
	points := l.points
	if len(via) > 0 {
		points = via
	}
	shape := append([]domain.GeoPoint{from}, append(append([]domain.GeoPoint{}, points...), to)...)
	cum := geo.Cumulative(shape)
	r := domain.Route{Shape: shape, DistanceKm: cum[len(cum)-1], Polyline: "x"}
	for i := 1; i < len(shape); i++ {
		km := cum[i] - cum[i-1]
		r.Steps = append(r.Steps, domain.RouteStep{
			DistanceKm: km,
			Duration:   time.Duration(km / l.kmh * float64(time.Hour)),
			BeginIdx:   i - 1,
			EndIdx:     i,
		})
	}
	return r, nil
}

func newTripService(t *testing.T, router lineRouter, snaps ...domain.Snapshot) *TripService {
	t.Helper()
	repo := memory.New()
	if err := repo.Save(context.Background(), snaps); err != nil {
		t.Fatal(err)
	}
	cat := catalog.NewStatic()
	status := &StatusService{Catalog: cat, Repo: repo, Estimator: &estimator.Queue{Now: func() time.Time { return depart }}}
	return &TripService{Router: router, Catalog: cat, Status: status}
}

var (
	depart   = time.Date(2026, 9, 28, 6, 0, 0, 0, time.UTC)
	hopa     = domain.GeoPoint{Lat: 41.40, Lng: 41.43}
	sarp     = domain.GeoPoint{Lat: 41.5178, Lng: 41.5475}
	batumi   = domain.GeoPoint{Lat: 41.645, Lng: 41.640}
	sarpWait = domain.Snapshot{
		CrossingID: "tr-ge-sarp", Direction: domain.DirectionExport, Source: "und",
		ObservedAt: depart.Add(-time.Hour), QueueKm: domain.Ptr(1.0), DailyThroughput: domain.Ptr(24 * 55),
	} // 55 trucks at 55/h -> 60 min
)

func TestPlanInsertsBorderWaitAtCrossing(t *testing.T) {
	svc := newTripService(t, lineRouter{points: []domain.GeoPoint{sarp}, kmh: 60}, sarpWait)
	plan, err := svc.Plan(context.Background(), TripRequest{Origin: hopa, Destination: batumi, DepartAt: depart})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Crossings) != 1 {
		t.Fatalf("crossings = %+v", plan.Crossings)
	}
	c := plan.Crossings[0]
	if c.ID != "tr-ge-sarp" || c.Direction != domain.DirectionExport || c.From != "TR" || c.To != "GE" || !c.WaitKnown {
		t.Errorf("crossing = %+v", c)
	}

	var kinds []tacho.Kind
	for _, s := range plan.Steps {
		kinds = append(kinds, s.Kind)
	}
	want := []tacho.Kind{tacho.KindDrive, tacho.KindBorderWait, tacho.KindDrive}
	if len(kinds) != len(want) || kinds[0] != want[0] || kinds[1] != want[1] || kinds[2] != want[2] {
		t.Fatalf("steps = %v", kinds)
	}
	wait := plan.Steps[1]
	if wait.DurationMin != 60 || wait.CrossingID != "tr-ge-sarp" {
		t.Errorf("wait step = %+v", wait)
	}
	if d := geo.DistanceKm(wait.Location, sarp); d > 0.1 {
		t.Errorf("wait located %.2f km from the gate", d)
	}
	if plan.Totals.BorderWaitMin != 60 {
		t.Errorf("totals = %+v", plan.Totals)
	}
}

func TestPlanDetectsImportDirection(t *testing.T) {
	svc := newTripService(t, lineRouter{points: []domain.GeoPoint{sarp}, kmh: 60})
	plan, err := svc.Plan(context.Background(), TripRequest{Origin: batumi, Destination: hopa, DepartAt: depart})
	if err != nil {
		t.Fatal(err)
	}
	c := plan.Crossings[0]
	if c.Direction != domain.DirectionImport || c.From != "GE" || c.To != "TR" {
		t.Errorf("crossing = %+v", c)
	}
	// No data for this direction: a zero-length wait marks the crossing.
	if c.WaitKnown || plan.Totals.BorderWaitMin != 0 {
		t.Errorf("unexpected wait: %+v", plan.Totals)
	}
	var marked bool
	for _, st := range plan.Steps {
		marked = marked || (st.Kind == tacho.KindBorderWait && st.DurationMin == 0 && st.CrossingID == "tr-ge-sarp")
	}
	if !marked {
		t.Errorf("steps = %+v, want a zero-length border step for the crossing", plan.Steps)
	}
}

func TestPlanIgnoresFarCrossings(t *testing.T) {
	far := domain.GeoPoint{Lat: 40.0, Lng: 40.0}
	svc := newTripService(t, lineRouter{kmh: 60})
	plan, err := svc.Plan(context.Background(), TripRequest{Origin: far, Destination: hopa, DepartAt: depart})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Crossings) != 0 {
		t.Errorf("crossings = %+v", plan.Crossings)
	}
}

func TestPlanPutsBreakAtRestArea(t *testing.T) {
	// Straight road north from Hopa, 6 h at 60 km/h: one break needed at 4.5 h (270 km).
	far := domain.GeoPoint{Lat: 41.40 + 360.0/111.2, Lng: 41.43}
	at := func(km float64) domain.GeoPoint { return domain.GeoPoint{Lat: 41.40 + km/111.2, Lng: 41.43} }
	svc := newTripService(t, lineRouter{points: []domain.GeoPoint{at(100), at(200), at(250), at(300)}, kmh: 60})
	store := memory.NewRestAreas()
	store.Replace(context.Background(), "test", []domain.RestArea{
		{ID: "early", Kind: domain.RestAreaServices, Location: at(150)},
		{ID: "good", Name: "Kaçkar Tesisleri", Kind: domain.RestAreaServices, Location: domain.GeoPoint{Lat: at(240).Lat, Lng: 41.435}},
		{ID: "offroad", Kind: domain.RestAreaServices, Location: domain.GeoPoint{Lat: at(260).Lat, Lng: 41.60}},
	})
	svc.RestAreas = store

	plan, err := svc.Plan(context.Background(), TripRequest{Origin: hopa, Destination: far, DepartAt: depart})
	if err != nil {
		t.Fatal(err)
	}
	var br *TripStep
	for i := range plan.Steps {
		if plan.Steps[i].Kind == tacho.KindBreak {
			br = &plan.Steps[i]
		}
	}
	if br == nil || br.RestArea == nil || br.RestArea.ID != "good" {
		t.Fatalf("break = %+v, want at rest area 'good'", br)
	}
	if br.Location != br.RestArea.Location {
		t.Errorf("break location %v should be the rest area's %v", br.Location, br.RestArea.Location)
	}
}

func TestSideOfCrossing(t *testing.T) {
	cat := catalog.NewStatic()
	get := func(id domain.CrossingID) domain.Crossing {
		c, err := cat.Get(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	istanbul := domain.GeoPoint{Lat: 41.01, Lng: 28.97}
	hopaFar := domain.GeoPoint{Lat: 41.39, Lng: 41.42}
	sofia := domain.GeoPoint{Lat: 42.70, Lng: 23.32}
	samsun := domain.GeoPoint{Lat: 41.29, Lng: 36.33}
	tbilisi := domain.GeoPoint{Lat: 41.72, Lng: 44.83}

	cases := []struct {
		crossing domain.CrossingID
		a, b     domain.GeoPoint
		same     bool
	}{
		{"tr-bg-hamzabeyli", istanbul, hopaFar, true}, // both in Türkiye, far from the gate
		{"tr-bg-kapikule", istanbul, sofia, false},
		{"tr-ge-sarp", samsun, tbilisi, false},
		{"tr-ge-sarp", samsun, hopaFar, true},
	}
	for _, tc := range cases {
		c := get(tc.crossing)
		if got := side(c, tc.a) == side(c, tc.b); got != tc.same {
			t.Errorf("%s: same side = %v, want %v", tc.crossing, got, tc.same)
		}
	}
}

func TestPlanReportsCrossingMissingFromCatalog(t *testing.T) {
	// A straight road far from any catalog crossing, with border posts on
	// both sides of an unknown border at km ~100.
	at := func(km float64) domain.GeoPoint { return domain.GeoPoint{Lat: 44.0 + km/111.2, Lng: 30.0} }
	svc := newTripService(t, lineRouter{points: []domain.GeoPoint{at(50), at(150)}, kmh: 60})
	store := memory.NewBorderPoints()
	store.Replace(context.Background(), "test", []domain.BorderPoint{
		{ID: "osm:node/1", Name: "Kapı A", Location: at(99.5), Country: "AA"},
		{ID: "osm:node/2", Location: domain.GeoPoint{Lat: at(100.3).Lat, Lng: 30.001}, Country: "BB"},
		{ID: "osm:node/3", Location: domain.GeoPoint{Lat: at(120).Lat, Lng: 30.2}, Country: "BB"}, // 16 km off the road
		{ID: "osm:node/4", Location: at(170), Country: "BB"},                                      // lone internal checkpoint
	})
	svc.BorderPoints = store

	plan, err := svc.Plan(context.Background(), TripRequest{Origin: at(0), Destination: at(200), DepartAt: depart})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Crossings) != 1 {
		t.Fatalf("crossings = %+v, want one merged unknown crossing", plan.Crossings)
	}
	c := plan.Crossings[0]
	if c.Name != "Kapı A" || c.From != "AA" || c.To != "BB" || c.WaitKnown || c.Estimate.Level != domain.LevelUnknown {
		t.Errorf("crossing = %+v", c)
	}
	if plan.AllWaitsKnown {
		t.Error("plan with an unknown crossing must not claim all waits are known")
	}
}

func TestPlanIgnoresGateTouchedWithoutCrossing(t *testing.T) {
	// Hopa → Sarp gate → back towards Hopa: the route never enters Georgia.
	back := domain.GeoPoint{Lat: 41.35, Lng: 41.40}
	svc := newTripService(t, lineRouter{points: []domain.GeoPoint{sarp}, kmh: 60}, sarpWait)
	plan, err := svc.Plan(context.Background(), TripRequest{Origin: hopa, Destination: back, DepartAt: depart})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Crossings) != 0 || plan.Totals.BorderWaitMin != 0 {
		t.Fatalf("crossings = %+v, wait = %d; want none", plan.Crossings, plan.Totals.BorderWaitMin)
	}
}
