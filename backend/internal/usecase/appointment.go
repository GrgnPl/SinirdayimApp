package usecase

import (
	"errors"
	"time"

	"github.com/burakaydin/sinir-bekleme/backend/internal/domain"
	"github.com/burakaydin/sinir-bekleme/backend/internal/tacho"
)

const (
	// AppointmentPassTime is how long passing the gate takes once the slot
	// starts. The official systems do not publish it; one hour is an
	// assumption to be replaced by measured data.
	AppointmentPassTime = time.Hour
	// AppointmentMargin is how early the truck should reach the gate.
	AppointmentMargin = 30 * time.Minute
	// latestDepartureSearch bounds how far back a departure is searched.
	latestDepartureSearch = 10 * 24 * time.Hour
)

var ErrAppointmentNotOnRoute = errors.New("appointment crossing is not on the route")

// Appointment is a booked slot at a crossing (e.g. RSS at Kapıkule).
type Appointment struct {
	CrossingID domain.CrossingID
	At         time.Time
}

// AppointmentPlan tells how the trip fits the booked slot.
type AppointmentPlan struct {
	CrossingID   domain.CrossingID `json:"crossingId"`
	CrossingName string            `json:"crossingName"`
	At           time.Time         `json:"at"`
	ArriveAt     time.Time         `json:"arriveAt"` // at the gate, with the planned stops
	// SlackMin is the time between arrival and the slot; negative when late.
	SlackMin int  `json:"slackMin"`
	OnTime   bool `json:"onTime"`
	// LatestDeparture is the latest departure that still reaches the gate
	// AppointmentMargin before the slot, with the same driver state. Nil
	// when even departing now is too late.
	LatestDeparture *time.Time `json:"latestDeparture,omitempty"`
}

// scheduleFor schedules a routed trip. With an appointment, the wait at its
// crossing becomes the time until the slot plus the passing time.
func scheduleFor(r *routed, req TripRequest, finder tacho.StopFinder) (tacho.Plan, *AppointmentPlan, error) {
	if req.Appointment == nil {
		p, err := tacho.ScheduleWith(r.activities, req.DepartAt, req.Driver, finder)
		return p, nil, err
	}
	id := string(req.Appointment.CrossingID)
	acts := append([]tacho.Activity(nil), r.activities...)
	wi := -1
	for i, a := range acts {
		if a.Kind == tacho.KindBorderWait && a.Ref == id {
			wi = i
			break
		}
	}
	if wi < 0 {
		return tacho.Plan{}, nil, ErrAppointmentNotOnRoute
	}

	// Arrival at the gate does not depend on the wait there, so measure it
	// with no wait first, then plan the real wait.
	acts[wi].Duration = 0
	arrive, err := gateArrival(acts, req.DepartAt, req.Driver, finder, id)
	if err != nil {
		return tacho.Plan{}, nil, err
	}
	at := req.Appointment.At
	acts[wi].Duration = max(at.Sub(arrive), 0) + AppointmentPassTime
	plan, err := tacho.ScheduleWith(acts, req.DepartAt, req.Driver, finder)
	if err != nil {
		return tacho.Plan{}, nil, err
	}

	ap := &AppointmentPlan{
		CrossingID: req.Appointment.CrossingID,
		At:         at,
		ArriveAt:   arrive.Truncate(time.Second),
		SlackMin:   minutes(at.Sub(arrive)),
		OnTime:     !arrive.After(at),
	}
	acts[wi].Duration = 0
	if dep, ok := latestDeparture(acts, req, finder, id, at.Add(-AppointmentMargin)); ok {
		ap.LatestDeparture = &dep
	}
	return plan, ap, nil
}

// gateArrival is when the truck reaches the crossing's wait step.
func gateArrival(acts []tacho.Activity, depart time.Time, st tacho.DriverState, finder tacho.StopFinder, id string) (time.Time, error) {
	p, err := tacho.ScheduleWith(acts, depart, st, finder)
	if err != nil {
		return time.Time{}, err
	}
	for _, s := range p.Steps {
		if s.Kind == tacho.KindBorderWait && s.Ref == id {
			return s.Start, nil
		}
	}
	return time.Time{}, ErrAppointmentNotOnRoute
}

// latestDeparture binary-searches the latest departure that reaches the
// gate by target. Later departures arrive later (the rules only add stops),
// so the search is well defined. The driver state is taken as given at any
// departure time.
func latestDeparture(acts []tacho.Activity, req TripRequest, finder tacho.StopFinder, id string, target time.Time) (time.Time, bool) {
	arrivesBy := func(dep time.Time) bool {
		a, err := gateArrival(acts, dep, req.Driver, finder, id)
		return err == nil && !a.After(target)
	}
	lo, hi := target.Add(-latestDepartureSearch), target
	if !arrivesBy(lo) {
		return time.Time{}, false
	}
	for hi.Sub(lo) > time.Minute {
		mid := lo.Add(hi.Sub(lo) / 2)
		if arrivesBy(mid) {
			lo = mid
		} else {
			hi = mid
		}
	}
	return lo.Truncate(5 * time.Minute), true
}

// suggestedSlot rounds an arrival up to the next half hour: slots are
// offered in fixed time windows.
func suggestedSlot(arrive time.Time) time.Time {
	s := arrive.Truncate(30 * time.Minute)
	if s.Before(arrive) {
		s = s.Add(30 * time.Minute)
	}
	return s
}
