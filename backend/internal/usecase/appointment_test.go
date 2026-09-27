package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/burakaydin/sinir-bekleme/backend/internal/domain"
	"github.com/burakaydin/sinir-bekleme/backend/internal/geo"
	"github.com/burakaydin/sinir-bekleme/backend/internal/tacho"
)

func sarpAppointment(at time.Time) *Appointment {
	return &Appointment{CrossingID: "tr-ge-sarp", At: at}
}

func TestAppointmentOnTime(t *testing.T) {
	svc := newTripService(t, lineRouter{points: []domain.GeoPoint{sarp}, kmh: 60}, sarpWait)
	slot := depart.Add(4 * time.Hour)
	plan, err := svc.Plan(context.Background(), TripRequest{Origin: hopa, Destination: batumi, DepartAt: depart, Appointment: sarpAppointment(slot)})
	if err != nil {
		t.Fatal(err)
	}
	a := plan.Appointment
	if a == nil || !a.OnTime || a.SlackMin <= 0 || a.CrossingName != "Sarp – Sarpi" {
		t.Fatalf("appointment = %+v", a)
	}
	// Hopa → Sarp at 60 km/h.
	drive := time.Duration(geo.DistanceKm(hopa, sarp) / 60 * float64(time.Hour))
	if got := a.ArriveAt.Sub(depart); got < drive-time.Minute || got > drive+time.Minute {
		t.Errorf("arrive after %v, want ~%v", got, drive)
	}
	// The wait lasts until the slot plus passing time; the known queue estimate is ignored.
	var wait *TripStep
	for i := range plan.Steps {
		if plan.Steps[i].Kind == tacho.KindBorderWait {
			wait = &plan.Steps[i]
		}
	}
	if wait == nil || !wait.End.Equal(slot.Add(AppointmentPassTime)) {
		t.Fatalf("wait = %+v, want to end at slot + %v", wait, AppointmentPassTime)
	}
	// Latest departure reaches the gate 30 min before the slot.
	if a.LatestDeparture == nil {
		t.Fatal("latest departure missing")
	}
	want := slot.Add(-AppointmentMargin - drive)
	if d := want.Sub(*a.LatestDeparture); d < 0 || d > 6*time.Minute {
		t.Errorf("latest departure = %v, want just before %v", *a.LatestDeparture, want)
	}
	if !plan.AllWaitsKnown {
		t.Error("a booked slot makes the wait known")
	}
}

func TestAppointmentLate(t *testing.T) {
	svc := newTripService(t, lineRouter{points: []domain.GeoPoint{sarp}, kmh: 60})
	slot := depart.Add(5 * time.Minute)
	plan, err := svc.Plan(context.Background(), TripRequest{Origin: hopa, Destination: batumi, DepartAt: depart, Appointment: sarpAppointment(slot)})
	if err != nil {
		t.Fatal(err)
	}
	a := plan.Appointment
	if a.OnTime || a.SlackMin >= 0 {
		t.Fatalf("appointment = %+v, want late", a)
	}
	if a.LatestDeparture == nil || !a.LatestDeparture.Before(depart) {
		t.Errorf("latest departure = %v, want before the planned departure", a.LatestDeparture)
	}
}

func TestSuggestedAppointmentAtKapikule(t *testing.T) {
	edirne := domain.GeoPoint{Lat: 41.677, Lng: 26.556}
	kapikule := domain.GeoPoint{Lat: 41.7206, Lng: 26.3594}
	svilengrad := domain.GeoPoint{Lat: 41.767, Lng: 26.199}
	svc := newTripService(t, lineRouter{points: []domain.GeoPoint{kapikule}, kmh: 60})

	plan, err := svc.Plan(context.Background(), TripRequest{Origin: edirne, Destination: svilengrad, DepartAt: depart})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Crossings) != 1 {
		t.Fatalf("crossings = %+v", plan.Crossings)
	}
	c := plan.Crossings[0]
	if len(c.Procedures) != 1 || c.Procedures[0].Kind != domain.ProcedureAppointment || !c.Procedures[0].Mandatory {
		t.Fatalf("procedures = %+v, want mandatory RSS appointment", c.Procedures)
	}
	// ~17 km from Edirne: arrival 06:17, suggested slot 06:30.
	if c.SuggestedAppointment == nil || !c.SuggestedAppointment.Equal(depart.Add(30*time.Minute)) {
		t.Errorf("suggested = %v, want %v", c.SuggestedAppointment, depart.Add(30*time.Minute))
	}

	// Coming from Bulgaria (import) RSS does not apply.
	plan, err = svc.Plan(context.Background(), TripRequest{Origin: svilengrad, Destination: edirne, DepartAt: depart})
	if err != nil {
		t.Fatal(err)
	}
	if c := plan.Crossings[0]; len(c.Procedures) != 0 || c.SuggestedAppointment != nil {
		t.Errorf("import crossing = %+v, want no procedures", c)
	}
}

func TestLaterDepartureRestsAtHome(t *testing.T) {
	// A tired driver (8h driven today, duty started 10h ago) with a slot
	// 14h away: leaving now means resting on the way, leaving at the latest
	// departure means the wait counts as the daily rest.
	svc := newTripService(t, lineRouter{points: []domain.GeoPoint{sarp}, kmh: 60})
	st := tacho.DriverState{DailyDriving: 8 * time.Hour, DutyStartedAt: depart.Add(-10 * time.Hour)}
	slot := depart.Add(14 * time.Hour)
	plan, err := svc.Plan(context.Background(), TripRequest{
		Origin: hopa, Destination: batumi, DepartAt: depart, Driver: st, Appointment: sarpAppointment(slot),
	})
	if err != nil {
		t.Fatal(err)
	}
	later := plan.Appointment.LaterDeparture
	if later == nil {
		t.Fatal("later departure option missing")
	}
	if !later.OnTime || !later.Departure.Equal(*plan.Appointment.LatestDeparture) {
		t.Errorf("later = %+v", later)
	}
	if later.RestMin != 0 {
		t.Errorf("later option rests %d min on the way, want 0 (rested before leaving)", later.RestMin)
	}
}

func TestLaterDepartureCountsAsRest(t *testing.T) {
	// State entered now (tired), departure 12 h later: the plan starts rested.
	svc := newTripService(t, lineRouter{points: []domain.GeoPoint{sarp}, kmh: 60})
	st := tacho.DriverState{ContinuousDriving: 4 * time.Hour, DailyDriving: 9 * time.Hour, DutyStartedAt: depart.Add(-12 * time.Hour)}
	plan, err := svc.Plan(context.Background(), TripRequest{
		Origin: hopa, Destination: batumi, DepartAt: depart.Add(12 * time.Hour), StateAt: depart, Driver: st,
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Totals.DailyRestMin != 0 || plan.Totals.BreakMin != 0 {
		t.Errorf("totals = %+v, want no stops: the driver rested before leaving", plan.Totals)
	}
}

func TestLatestDepartureCountsWaitingAsRest(t *testing.T) {
	// Planned departure now, slot in 16 h. The driver has been on duty for
	// 11 h with 0 h driving left... except that waiting 11 h at home resets
	// the day, so the latest departure is just before the slot.
	svc := newTripService(t, lineRouter{points: []domain.GeoPoint{sarp}, kmh: 60})
	st := tacho.DriverState{DailyDriving: 9 * time.Hour, DutyStartedAt: depart.Add(-11 * time.Hour)}
	slot := depart.Add(16 * time.Hour)
	plan, err := svc.Plan(context.Background(), TripRequest{
		Origin: hopa, Destination: batumi, DepartAt: depart, Driver: st, Appointment: sarpAppointment(slot),
	})
	if err != nil {
		t.Fatal(err)
	}
	drive := time.Duration(geo.DistanceKm(hopa, sarp) / 60 * float64(time.Hour))
	want := slot.Add(-AppointmentMargin - drive)
	got := plan.Appointment.LatestDeparture
	if got == nil || want.Sub(*got) < 0 || want.Sub(*got) > 6*time.Minute {
		t.Fatalf("latest departure = %v, want just before %v", got, want)
	}
}
