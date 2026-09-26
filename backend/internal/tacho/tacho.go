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
// Not yet modeled: reduced daily rest (9h), split breaks, weekly limits
// (56h / 90h) and weekly rest.
package tacho

import (
	"errors"
	"time"
)

const (
	MaxContinuousDriving = 4*time.Hour + 30*time.Minute
	BreakDuration        = 45 * time.Minute
	MaxDailyDriving      = 9 * time.Hour
	ExtendedDailyDriving = 10 * time.Hour
	DailyRestDuration    = 11 * time.Hour
	MaxDutyPeriod        = 24*time.Hour - DailyRestDuration // 13h
)

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
// the rules allow.
func Schedule(activities []Activity, depart time.Time, st DriverState) (Plan, error) {
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
		plan:         Plan{Departure: depart},
	}
	if s.dutyStart.IsZero() || s.dutyStart.After(depart) {
		s.dutyStart = depart
	}
	s.dailyLimit = s.pickDailyLimit()

	for _, a := range activities {
		switch a.Kind {
		case KindDrive:
			s.drive(a)
		case KindBorderWait:
			s.wait(a)
		}
	}
	s.plan.Arrival = s.now
	return s.plan, nil
}

type scheduler struct {
	now          time.Time
	km           float64
	cont         time.Duration
	daily        time.Duration
	dailyLimit   time.Duration
	dutyStart    time.Time
	extendedLeft int
	plan         Plan
}

// pickDailyLimit uses an extended (10h) day when one is left.
func (s *scheduler) pickDailyLimit() time.Duration {
	if s.extendedLeft > 0 {
		return ExtendedDailyDriving
	}
	return MaxDailyDriving
}

func (s *scheduler) drive(a Activity) {
	remaining := a.Duration
	if remaining <= 0 {
		s.km += a.DistanceKm
		return
	}
	speed := a.DistanceKm / a.Duration.Hours() // km per hour for this segment

	for remaining > 0 {
		dutyLeft := s.dutyStart.Add(MaxDutyPeriod).Sub(s.now)
		allowed := min(remaining, MaxContinuousDriving-s.cont, s.dailyLimit-s.daily, dutyLeft)

		if allowed <= 0 {
			switch {
			case s.daily >= s.dailyLimit:
				s.dailyRest(ReasonDaily)
			case dutyLeft <= 0:
				s.dailyRest(ReasonDuty)
			default:
				s.takeBreak(ReasonContinuous)
			}
			continue
		}

		dist := speed * allowed.Hours()
		s.addDrive(allowed, dist)
		remaining -= allowed
	}
}

func (s *scheduler) addDrive(d time.Duration, km float64) {
	// Merge with the previous drive step so the timeline stays readable.
	if n := len(s.plan.Steps); n > 0 && s.plan.Steps[n-1].Kind == KindDrive {
		last := &s.plan.Steps[n-1]
		last.Duration += d
		last.ToKm += km
	} else {
		s.plan.Steps = append(s.plan.Steps, Step{Kind: KindDrive, Start: s.now, Duration: d, FromKm: s.km, ToKm: s.km + km})
	}
	s.now = s.now.Add(d)
	s.km += km
	s.cont += d
	s.daily += d
	s.plan.Driving += d
}

func (s *scheduler) takeBreak(reason string) {
	s.plan.Steps = append(s.plan.Steps, Step{Kind: KindBreak, Start: s.now, Duration: BreakDuration, FromKm: s.km, ToKm: s.km, Reason: reason})
	s.now = s.now.Add(BreakDuration)
	s.cont = 0
	s.plan.Breaks += BreakDuration
}

func (s *scheduler) dailyRest(reason string) {
	s.plan.Steps = append(s.plan.Steps, Step{Kind: KindDailyRest, Start: s.now, Duration: DailyRestDuration, FromKm: s.km, ToKm: s.km, Reason: reason})
	s.now = s.now.Add(DailyRestDuration)
	s.plan.DailyRests += DailyRestDuration
	s.resetDay()
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
	s.plan.Steps = append(s.plan.Steps, Step{Kind: KindBorderWait, Start: s.now, Duration: a.Duration, FromKm: s.km, ToKm: s.km, Ref: a.Ref})
	s.now = s.now.Add(a.Duration)
	s.plan.BorderWaits += a.Duration
	switch {
	case a.Duration >= DailyRestDuration:
		s.resetDay()
	case a.Duration >= BreakDuration:
		s.cont = 0
	}
}
