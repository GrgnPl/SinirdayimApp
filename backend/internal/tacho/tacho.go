// Package tacho schedules breaks and rests along a trip according to the
// EU 561/2006 / AETR driving time rules. It is pure logic: no I/O, no clock.
//
// Rules applied:
//   - After 4h30 of driving, a 45 min break (no split 15+30 yet).
//   - Daily driving max 9h, or 10h on up to two days per week.
//   - Daily rest of 11h, which must start within 13h of the end of the
//     previous daily rest (24h period).
//   - Waiting at a border with the engine off counts as a break when it lasts
//     45 min or more, and as a daily rest when it lasts 11h or more.
//
// Stops are placed as late as the rules allow. With a StopFinder, a stop is
// pulled back to the last suitable place (rest area, truck parking) within a
// search window before the limit.
//
// Not yet modeled: reduced daily rest (9h), split breaks, weekly limits
// (56h / 90h) and weekly rest.
package tacho

import (
	"errors"
	"sort"
	"time"
)

const (
	MaxContinuousDriving = 4*time.Hour + 30*time.Minute
	BreakDuration        = 45 * time.Minute
	MaxDailyDriving      = 9 * time.Hour
	ExtendedDailyDriving = 10 * time.Hour
	DailyRestDuration    = 11 * time.Hour
	MaxDutyPeriod        = 24*time.Hour - DailyRestDuration // 13h

	// How much driving before a limit a stop may be pulled back to reach a
	// proper place. Daily rests deserve a longer search: a safe spot for the
	// night matters more than a few extra kilometres.
	BreakSearchWindow = 45 * time.Minute
	RestSearchWindow  = 2 * time.Hour
)

// StopFinder picks where to stop. Find returns the position (km along the
// route) of the best place for a stop of the given kind within [fromKm, toKm],
// with an opaque reference to it.
type StopFinder interface {
	Find(kind Kind, fromKm, toKm float64) (km float64, ref string, ok bool)
}

// Kind of an activity in the input or output timeline.
type Kind string

const (
	KindDrive      Kind = "drive"
	KindBorderWait Kind = "border_wait"
	KindBreak      Kind = "break"
	KindDailyRest  Kind = "daily_rest"
)

// Activity is one piece of the trip as it would happen without any rules:
// a driving segment or a waiting period at a border.
type Activity struct {
	Kind       Kind
	Duration   time.Duration
	DistanceKm float64 // for drives
	Ref        string  // e.g. crossing id for border waits
}

// DriverState is the driver's tachograph state at departure.
type DriverState struct {
	ContinuousDriving time.Duration // driving since the last qualifying break
	DailyDriving      time.Duration // driving since the last daily rest
	DutyStartedAt     time.Time     // end of the last daily rest; zero = departure time
	ExtendedDaysLeft  int           // 10h days still available this week (0-2)
}

// Step is one entry of the planned timeline.
type Step struct {
	Kind     Kind          `json:"kind"`
	Start    time.Time     `json:"start"`
	Duration time.Duration `json:"duration"`
	FromKm   float64       `json:"fromKm"` // position along the route
	ToKm     float64       `json:"toKm"`
	Ref      string        `json:"ref,omitempty"`
	Reason   string        `json:"reason,omitempty"` // why a break/rest was inserted
}

// Plan is the scheduled trip.
type Plan struct {
	Steps       []Step
	Departure   time.Time
	Arrival     time.Time
	Driving     time.Duration
	Breaks      time.Duration
	DailyRests  time.Duration
	BorderWaits time.Duration
}

// Reasons attached to inserted stops.
const (
	ReasonContinuous = "continuous_driving_limit" // 4h30 reached
	ReasonDaily      = "daily_driving_limit"      // 9h / 10h reached
	ReasonDuty       = "duty_period_limit"        // 13h since last daily rest
)

var ErrInvalidState = errors.New("tacho: invalid driver state")

// Schedule walks the activities and inserts breaks and daily rests as late as
// the rules allow, at arbitrary points on the road.
func Schedule(activities []Activity, depart time.Time, st DriverState) (Plan, error) {
	return ScheduleWith(activities, depart, st, nil)
}

// ScheduleWith is Schedule with stops snapped to places chosen by finder.
func ScheduleWith(activities []Activity, depart time.Time, st DriverState, finder StopFinder) (Plan, error) {
	if st.ContinuousDriving < 0 || st.DailyDriving < 0 || st.ExtendedDaysLeft < 0 ||
		st.ContinuousDriving > MaxContinuousDriving || st.DailyDriving > ExtendedDailyDriving {
		return Plan{}, ErrInvalidState
	}
	s := &scheduler{
		now:          depart,
		cont:         st.ContinuousDriving,
		daily:        st.DailyDriving,
		dutyStart:    st.DutyStartedAt,
		extendedLeft: st.ExtendedDaysLeft,
		finder:       finder,
		track:        newTrack(activities),
		plan:         Plan{Departure: depart},
	}
	if s.dutyStart.IsZero() || s.dutyStart.After(depart) {
		s.dutyStart = depart
	}
	s.dailyLimit = s.pickDailyLimit()

	// Driving between border waits is one continuous stretch, so a stop can
	// be placed anywhere in it regardless of how the route was segmented.
	var driven time.Duration
	for _, a := range activities {
		switch a.Kind {
		case KindDrive:
			driven += max(a.Duration, 0)
		case KindBorderWait:
			s.driveUntil(driven)
			s.wait(a)
		}
	}
	s.driveUntil(driven)
	s.plan.Arrival = s.now
	return s.plan, nil
}

type scheduler struct {
	now          time.Time
	pos          time.Duration // driving time done along the route
	cont         time.Duration
	daily        time.Duration
	dailyLimit   time.Duration
	dutyStart    time.Time
	extendedLeft int
	finder       StopFinder
	track        track
	plan         Plan
}

// pickDailyLimit uses an extended (10h) day when one is left.
func (s *scheduler) pickDailyLimit() time.Duration {
	if s.extendedLeft > 0 {
		return ExtendedDailyDriving
	}
	return MaxDailyDriving
}

// driveUntil drives to route driving time end, stopping whenever a limit
// is reached.
func (s *scheduler) driveUntil(end time.Duration) {
	for s.pos < end {
		contLeft := MaxContinuousDriving - s.cont
		dailyLeft := s.dailyLimit - s.daily
		dutyLeft := s.dutyStart.Add(MaxDutyPeriod).Sub(s.now)
		budget := min(contLeft, dailyLeft, dutyLeft)

		kind, reason := KindBreak, ReasonContinuous
		switch budget {
		case dailyLeft:
			kind, reason = KindDailyRest, ReasonDaily
		case dutyLeft:
			kind, reason = KindDailyRest, ReasonDuty
		}

		if budget <= 0 {
			s.stop(kind, reason, "")
			continue
		}
		target := s.pos + budget
		if target >= end {
			s.advance(end)
			return
		}

		// A break shortly before the day ends would leave only a short
		// stretch to find a place for the night. If a good place for the
		// daily rest is within reach now, end the day there instead.
		dayLeft := min(dailyLeft, dutyLeft)
		if kind == KindBreak && s.finder != nil && dayLeft-contLeft < RestSearchWindow {
			from := s.track.kmAt(max(s.pos, target-RestSearchWindow))
			if km, r, ok := s.finder.Find(KindDailyRest, from, s.track.kmAt(target)); ok {
				rest := ReasonDaily
				if dutyLeft < dailyLeft {
					rest = ReasonDuty
				}
				s.advance(min(max(s.track.timeAt(km), s.pos), target))
				s.stop(KindDailyRest, rest, r)
				continue
			}
		}

		stopAt, ref := target, ""
		if s.finder != nil {
			window := BreakSearchWindow
			if kind == KindDailyRest {
				window = RestSearchWindow
			}
			from := s.track.kmAt(max(s.pos, target-window))
			to := s.track.kmAt(target)
			if km, r, ok := s.finder.Find(kind, from, to); ok && km >= from && km <= to {
				stopAt, ref = min(max(s.track.timeAt(km), s.pos), target), r
			}
		}
		s.advance(stopAt)
		s.stop(kind, reason, ref)
	}
}

// advance drives from the current position to route driving time t.
func (s *scheduler) advance(t time.Duration) {
	d := t - s.pos
	if d <= 0 {
		return
	}
	from, to := s.track.kmAt(s.pos), s.track.kmAt(t)
	// Merge with the previous drive step so the timeline stays readable.
	if n := len(s.plan.Steps); n > 0 && s.plan.Steps[n-1].Kind == KindDrive {
		last := &s.plan.Steps[n-1]
		last.Duration += d
		last.ToKm = to
	} else {
		s.plan.Steps = append(s.plan.Steps, Step{Kind: KindDrive, Start: s.now, Duration: d, FromKm: from, ToKm: to})
	}
	s.now = s.now.Add(d)
	s.pos = t
	s.cont += d
	s.daily += d
	s.plan.Driving += d
}

func (s *scheduler) stop(kind Kind, reason, ref string) {
	km := s.track.kmAt(s.pos)
	d := BreakDuration
	if kind == KindDailyRest {
		d = DailyRestDuration
	}
	s.plan.Steps = append(s.plan.Steps, Step{Kind: kind, Start: s.now, Duration: d, FromKm: km, ToKm: km, Reason: reason, Ref: ref})
	s.now = s.now.Add(d)
	if kind == KindDailyRest {
		s.plan.DailyRests += d
		s.resetDay()
	} else {
		s.plan.Breaks += d
		s.cont = 0
	}
}

func (s *scheduler) resetDay() {
	if s.dailyLimit == ExtendedDailyDriving && s.daily > MaxDailyDriving {
		s.extendedLeft--
	}
	s.cont = 0
	s.daily = 0
	s.dutyStart = s.now
	s.dailyLimit = s.pickDailyLimit()
}

func (s *scheduler) wait(a Activity) {
	if a.Duration <= 0 {
		return
	}
	km := s.track.kmAt(s.pos)
	s.plan.Steps = append(s.plan.Steps, Step{Kind: KindBorderWait, Start: s.now, Duration: a.Duration, FromKm: km, ToKm: km, Ref: a.Ref})
	s.now = s.now.Add(a.Duration)
	s.plan.BorderWaits += a.Duration
	switch {
	case a.Duration >= DailyRestDuration:
		s.resetDay()
	case a.Duration >= BreakDuration:
		s.cont = 0
	}
}

// track maps driving time along the route to distance and back, as a
// piecewise-linear function built from the drive activities.
type track struct {
	t  []time.Duration
	km []float64
}

func newTrack(activities []Activity) track {
	tr := track{t: []time.Duration{0}, km: []float64{0}}
	var t time.Duration
	var km float64
	for _, a := range activities {
		if a.Kind != KindDrive {
			continue
		}
		t += max(a.Duration, 0)
		km += a.DistanceKm
		tr.t = append(tr.t, t)
		tr.km = append(tr.km, km)
	}
	return tr
}

func (tr track) kmAt(t time.Duration) float64 {
	i := sort.Search(len(tr.t), func(i int) bool { return tr.t[i] >= t })
	switch {
	case i == 0:
		return tr.km[0]
	case i >= len(tr.t):
		return tr.km[len(tr.km)-1]
	}
	span := tr.t[i] - tr.t[i-1]
	if span <= 0 {
		return tr.km[i]
	}
	f := float64(t-tr.t[i-1]) / float64(span)
	return tr.km[i-1] + (tr.km[i]-tr.km[i-1])*f
}

func (tr track) timeAt(km float64) time.Duration {
	i := sort.SearchFloat64s(tr.km, km)
	switch {
	case i == 0:
		return tr.t[0]
	case i >= len(tr.km):
		return tr.t[len(tr.t)-1]
	}
	span := tr.km[i] - tr.km[i-1]
	if span <= 0 {
		return tr.t[i]
	}
	f := (km - tr.km[i-1]) / span
	return tr.t[i-1] + time.Duration(float64(tr.t[i]-tr.t[i-1])*f)
}
