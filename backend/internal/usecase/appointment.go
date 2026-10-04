package usecase

import (
	"errors"
	"time"

	"github.com/GrgnPl/SinirdayimApp/backend/internal/domain"
	"github.com/GrgnPl/SinirdayimApp/backend/internal/tacho"
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
	// LaterDeparture is the trip when leaving at LatestDeparture instead,
	// with the wait until then counted as rest. Nil when it is not
	// meaningfully later than the planned departure.
	LaterDeparture *DepartureOption `json:"laterDeparture,omitempty"`
}

// DepartureOption summarizes a trip for another departure time.
type DepartureOption struct {
	Departure     time.Time `json:"departure"`
	Arrival       time.Time `json:"arrival"`
	ArriveAt      time.Time `json:"arriveAt"` // at the appointment gate
	OnTime        bool      `json:"onTime"`
	TotalMin      int       `json:"totalMin"`
	RestMin       int       `json:"restMin"` // daily and weekly rests on the way
	BreakMin      int       `json:"breakMin"`
	BorderWaitMin int       `json:"borderWaitMin"`
}

// minLaterDeparture is how much later a departure must be to be offered.
const minLaterDeparture = 30 * time.Minute

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
		if dep.Sub(req.DepartAt) >= minLaterDeparture {
			if opt, err := departAt(acts, wi, req, finder, id, dep); err == nil {
				ap.LaterDeparture = &opt
			}
		}
	}
	return plan, ap, nil
}

// departAt plans the trip leaving at dep instead; the driver rests until then.
func departAt(acts []tacho.Activity, wi int, req TripRequest, finder tacho.StopFinder, id string, dep time.Time) (DepartureOption, error) {
	acts = append([]tacho.Activity(nil), acts...)
	st := req.Driver.AfterIdle(dep.Sub(req.DepartAt), dep)
	acts[wi].Duration = 0
	arrive, err := gateArrival(acts, dep, st, finder, id)
	if err != nil {
		return DepartureOption{}, err
	}
	at := req.Appointment.At
	acts[wi].Duration = max(at.Sub(arrive), 0) + AppointmentPassTime
	p, err := tacho.ScheduleWith(acts, dep, st, finder)
	if err != nil {
		return DepartureOption{}, err
	}
	return DepartureOption{
		Departure:     dep,
		Arrival:       p.Arrival.Truncate(time.Second),
		ArriveAt:      arrive.Truncate(time.Second),
		OnTime:        !arrive.After(at),
		TotalMin:      minutes(p.Arrival.Sub(dep)),
		RestMin:       minutes(p.DailyRests + p.WeeklyRests),
		BreakMin:      minutes(p.Breaks),
		BorderWaitMin: minutes(p.BorderWaits),
	}, nil
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

// latestDeparture finds the latest departure that reaches the gate by
// target, counting any wait before a later departure as rest. Arrival is not
// monotone in departure (a long enough wait resets the day), so the last
// 36 h are scanned in 5 min steps; earlier departures are binary-searched.
func latestDeparture(acts []tacho.Activity, req TripRequest, finder tacho.StopFinder, id string, target time.Time) (time.Time, bool) {
	const step = 5 * time.Minute
	const scan = 36 * time.Hour
	arrivesBy := func(dep time.Time) bool {
		st := req.Driver
		if dep.After(req.DepartAt) {
			st = st.AfterIdle(dep.Sub(req.DepartAt), dep)
		}
		a, err := gateArrival(acts, dep, st, finder, id)
		return err == nil && !a.After(target)
	}
	hi := target.Truncate(step)
	for dep := hi; !dep.Before(hi.Add(-scan)); dep = dep.Add(-step) {
		if arrivesBy(dep) {
			return dep, true
		}
	}
	lo, hi := target.Add(-latestDepartureSearch), hi.Add(-scan)
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
	return lo.Truncate(step), true
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
