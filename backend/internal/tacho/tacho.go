// Package tacho schedules breaks and rests along a trip according to the
// EU 561/2006 / AETR driving time rules. It is pure logic: no I/O, no clock.
//
// Rules applied:
//   - After 4h30 of driving, a 45 min break. It may be split into 15 + 30 min:
//     once a 15 min part is taken, the next break only needs 30 min.
//   - Daily driving max 9h, or 10h on up to two days per calendar week.
//   - Daily rest of 11h, reduced to 9h up to three times between weekly
//     rests. It must start within 24h minus the rest length (13h / 15h) of
//     the end of the previous daily rest.
//   - Weekly driving max 56h per calendar week (Monday 00:00) and 90h over
//     two consecutive weeks; weekly rest of 45h when a weekly limit is hit or
//     six 24h periods have passed since the last weekly rest.
//   - Waiting at a border with the engine off counts as rest: 15 min as the
//     first part of a split break, a full break, and 9h / 11h as a daily rest.
//
// Stops are placed as late as the rules allow. With a StopFinder, a stop is
// pulled back to the last suitable place (rest area, truck parking) within a
// search window before the limit.
//
// Not modeled: split daily rest (3h + 9h), reduced weekly rest (24h) and its
// compensation, ferry/train interruptions.
package tacho

import (
	"errors"
	"sort"
	"time"
)

const (
	MaxContinuousDriving = 4*time.Hour + 30*time.Minute
	BreakDuration        = 45 * time.Minute
	SplitBreakFirst      = 15 * time.Minute
	SplitBreakSecond     = 30 * time.Minute
	MaxDailyDriving      = 9 * time.Hour
	ExtendedDailyDriving = 10 * time.Hour
	DailyRestDuration    = 11 * time.Hour
	ReducedDailyRest     = 9 * time.Hour
	MaxDutyPeriod        = 24*time.Hour - DailyRestDuration // 13h
	MaxReducedRests      = 3
	MaxExtendedDays      = 2
	MaxWeeklyDriving     = 56 * time.Hour
	MaxBiweeklyDriving   = 90 * time.Hour
	WeeklyRestDuration   = 45 * time.Hour
	MaxBetweenWeeklyRest = 6 * 24 * time.Hour

	// How much driving before a limit a stop may be pulled back to reach a
	// proper place. Rests deserve a longer search: a safe spot for the night
	// matters more than a few extra kilometres.
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
	KindWeeklyRest Kind = "weekly_rest"
)

// Activity is one piece of the trip as it would happen without any rules:
// a driving segment or a waiting period at a border.
type Activity struct {
	Kind       Kind
	Duration   time.Duration
	DistanceKm float64 // for drives
	Ref        string  // e.g. crossing id for border waits
}

// DriverState is the driver's tachograph state at departure. The zero value
// is a rested driver at the start of a week with no reduced rests or
// extended days available; callers set what the driver actually has.
type DriverState struct {
	ContinuousDriving time.Duration // driving since the last qualifying break
	SplitBreakTaken   bool          // a 15 min first part was taken since then
	DailyDriving      time.Duration // driving since the last daily rest
	DutyStartedAt     time.Time     // end of the last daily rest; zero = departure time
	ExtendedDaysLeft  int           // 10h days still available this week (0-2)
	ReducedRestsLeft  int           // 9h daily rests still available (0-3)
	WeeklyDriving     time.Duration // driving in the current calendar week
	PrevWeekDriving   time.Duration // driving in the previous calendar week
	LastWeeklyRestEnd time.Time     // zero = not tracked
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
	// Reduced marks a 9h daily rest, or a 30 min second part of a split break.
	Reduced bool `json:"reduced,omitempty"`
	// CountsAs tells what rule a border wait satisfied (break, daily rest...).
	CountsAs Kind `json:"countsAs,omitempty"`
}

// Plan is the scheduled trip.
type Plan struct {
	Steps       []Step
	Departure   time.Time
	Arrival     time.Time
	Driving     time.Duration
	Breaks      time.Duration
	DailyRests  time.Duration
	WeeklyRests time.Duration
	BorderWaits time.Duration
}

// Reasons attached to inserted stops.
const (
	ReasonContinuous = "continuous_driving_limit" // 4h30 reached
	ReasonDaily      = "daily_driving_limit"      // 9h / 10h reached
	ReasonDuty       = "duty_period_limit"        // 13h / 15h since last daily rest
	ReasonWeekly     = "weekly_driving_limit"     // 56h / 90h reached
	ReasonWeeklyDue  = "weekly_rest_due"          // six 24h periods since last weekly rest
)

var ErrInvalidState = errors.New("tacho: invalid driver state")

// Schedule walks the activities and inserts breaks and rests as late as the
// rules allow, at arbitrary points on the road.
func Schedule(activities []Activity, depart time.Time, st DriverState) (Plan, error) {
	return ScheduleWith(activities, depart, st, nil)
}

// ScheduleWith is Schedule with stops snapped to places chosen by finder.
func ScheduleWith(activities []Activity, depart time.Time, st DriverState, finder StopFinder) (Plan, error) {
	if !valid(st) {
		return Plan{}, ErrInvalidState
	}
	s := &scheduler{
		now:          depart,
		cont:         st.ContinuousDriving,
		splitTaken:   st.SplitBreakTaken,
		daily:        st.DailyDriving,
		dutyStart:    st.DutyStartedAt,
		extendedLeft: st.ExtendedDaysLeft,
		reducedLeft:  st.ReducedRestsLeft,
		weekly:       st.WeeklyDriving,
		prevWeek:     st.PrevWeekDriving,
		nextWeek:     nextMonday(depart),
		finder:       finder,
		track:        newTrack(activities),
		plan:         Plan{Departure: depart},
	}
	if !st.LastWeeklyRestEnd.IsZero() {
		s.weeklyRestDue = st.LastWeeklyRestEnd.Add(MaxBetweenWeeklyRest)
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

func valid(st DriverState) bool {
	return st.ContinuousDriving >= 0 && st.ContinuousDriving <= MaxContinuousDriving &&
		st.DailyDriving >= 0 && st.DailyDriving <= ExtendedDailyDriving &&
		st.ExtendedDaysLeft >= 0 && st.ExtendedDaysLeft <= MaxExtendedDays &&
		st.ReducedRestsLeft >= 0 && st.ReducedRestsLeft <= MaxReducedRests &&
		st.WeeklyDriving >= 0 && st.WeeklyDriving <= MaxWeeklyDriving &&
		st.PrevWeekDriving >= 0 && st.PrevWeekDriving <= MaxWeeklyDriving
}

type scheduler struct {
	now           time.Time
	pos           time.Duration // driving time done along the route
	cont          time.Duration
	splitTaken    bool
	daily         time.Duration
	dailyLimit    time.Duration
	dutyStart     time.Time
	extendedLeft  int
	reducedLeft   int
	weekly        time.Duration
	prevWeek      time.Duration
	nextWeek      time.Time // start of the next calendar week
	weeklyRestDue time.Time // zero = not tracked
	finder        StopFinder
	track         track
	plan          Plan
}

// pickDailyLimit uses an extended (10h) day when one is left.
func (s *scheduler) pickDailyLimit() time.Duration {
	if s.extendedLeft > 0 {
		return ExtendedDailyDriving
	}
	return MaxDailyDriving
}

// dailyRest is the rest the driver will take at the end of the current day:
// reduced while reductions are left, which also widens the duty window.
func (s *scheduler) dailyRest() (time.Duration, bool) {
	if s.reducedLeft > 0 {
		return ReducedDailyRest, true
	}
	return DailyRestDuration, false
}

func (s *scheduler) breakNeeded() time.Duration {
	if s.splitTaken {
		return SplitBreakSecond
	}
	return BreakDuration
}

// driveUntil drives to route driving time end, stopping whenever a limit
// is reached.
func (s *scheduler) driveUntil(end time.Duration) {
	for s.pos < end {
		rest, _ := s.dailyRest()
		contLeft := MaxContinuousDriving - s.cont
		dailyLeft := s.dailyLimit - s.daily
		dutyLeft := s.dutyStart.Add(24*time.Hour - rest).Sub(s.now)
		weekLeft := s.weekLeft()
		dueLeft := time.Duration(1<<62 - 1)
		if !s.weeklyRestDue.IsZero() {
			dueLeft = s.weeklyRestDue.Sub(s.now)
		}
		budget := min(contLeft, dailyLeft, dutyLeft, weekLeft, dueLeft)

		kind, reason := KindBreak, ReasonContinuous
		switch budget {
		case weekLeft:
			kind, reason = KindWeeklyRest, ReasonWeekly
		case dueLeft:
			kind, reason = KindWeeklyRest, ReasonWeeklyDue
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
				why := ReasonDaily
				if dutyLeft < dailyLeft {
					why = ReasonDuty
				}
				s.advance(min(max(s.track.timeAt(km), s.pos), target))
				s.stop(KindDailyRest, why, r)
				continue
			}
		}

		stopAt, ref := target, ""
		if s.finder != nil {
			window := BreakSearchWindow
			if kind != KindBreak {
				window = RestSearchWindow
			}
			// Rest areas suitable for a daily rest also suit a weekly one.
			findKind := kind
			if kind == KindWeeklyRest {
				findKind = KindDailyRest
			}
			from := s.track.kmAt(max(s.pos, target-window))
			to := s.track.kmAt(target)
			if km, r, ok := s.finder.Find(findKind, from, to); ok && km >= from && km <= to {
				stopAt, ref = min(max(s.track.timeAt(km), s.pos), target), r
			}
		}
		s.advance(stopAt)
		s.stop(kind, reason, ref)
	}
}

// weekLeft is how long the driver may still drive under the weekly limits,
// counting the fresh allowance of the next calendar week if it starts first.
func (s *scheduler) weekLeft() time.Duration {
	left := min(MaxWeeklyDriving-s.weekly, MaxBiweeklyDriving-s.prevWeek-s.weekly)
	untilRoll := s.nextWeek.Sub(s.now)
	if left <= untilRoll {
		return left
	}
	thisWeek := s.weekly + untilRoll
	return untilRoll + min(MaxWeeklyDriving, MaxBiweeklyDriving-thisWeek)
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
	// Weekly driving counts per calendar week; split at the boundary.
	if end := s.now.Add(d); !end.Before(s.nextWeek) {
		before := s.nextWeek.Sub(s.now)
		s.weekly += before
		s.rollWeek()
		s.weekly += d - before
	} else {
		s.weekly += d
	}
	s.now = s.now.Add(d)
	s.pos = t
	s.cont += d
	s.daily += d
	s.plan.Driving += d
}

// rollWeek moves to the next calendar week. Extended days are per week.
func (s *scheduler) rollWeek() {
	s.prevWeek = s.weekly
	s.weekly = 0
	s.extendedLeft = MaxExtendedDays
	s.nextWeek = s.nextWeek.AddDate(0, 0, 7)
}

// passTime lets time pass without driving, rolling calendar weeks.
func (s *scheduler) passTime(d time.Duration) {
	s.now = s.now.Add(d)
	for !s.now.Before(s.nextWeek) {
		s.rollWeek()
	}
}

func (s *scheduler) stop(kind Kind, reason, ref string) {
	km := s.track.kmAt(s.pos)
	step := Step{Kind: kind, Start: s.now, FromKm: km, ToKm: km, Reason: reason, Ref: ref}
	switch kind {
	case KindBreak:
		step.Duration = s.breakNeeded()
		step.Reduced = s.splitTaken
		s.plan.Breaks += step.Duration
		s.cont, s.splitTaken = 0, false
	case KindDailyRest:
		step.Duration, step.Reduced = s.dailyRest()
		s.plan.DailyRests += step.Duration
	case KindWeeklyRest:
		step.Duration = WeeklyRestDuration
		// A weekly driving limit only lifts when the next calendar week
		// starts, so the rest lasts at least until then.
		if reason == ReasonWeekly {
			step.Duration = max(step.Duration, s.nextWeek.Sub(s.now))
		}
		s.plan.WeeklyRests += step.Duration
	}
	s.plan.Steps = append(s.plan.Steps, step)
	s.passTime(step.Duration)
	switch kind {
	case KindDailyRest:
		if step.Reduced {
			s.reducedLeft--
		}
		s.resetDay()
	case KindWeeklyRest:
		s.resetWeekRest()
	}
}

func (s *scheduler) resetDay() {
	if s.dailyLimit == ExtendedDailyDriving && s.daily > MaxDailyDriving {
		s.extendedLeft = max(s.extendedLeft-1, 0)
	}
	s.cont, s.splitTaken = 0, false
	s.daily = 0
	s.dutyStart = s.now
	s.dailyLimit = s.pickDailyLimit()
}

// resetWeekRest applies a weekly rest: a new day, reductions restored and
// the next weekly rest due in six 24h periods.
func (s *scheduler) resetWeekRest() {
	s.resetDay()
	s.reducedLeft = MaxReducedRests
	s.weeklyRestDue = s.now.Add(MaxBetweenWeeklyRest)
}

// wait records a border wait. A zero-length wait still appears in the
// timeline: it marks a crossing whose wait is unknown.
func (s *scheduler) wait(a Activity) {
	a.Duration = max(a.Duration, 0)
	km := s.track.kmAt(s.pos)
	step := Step{Kind: KindBorderWait, Start: s.now, Duration: a.Duration, FromKm: km, ToKm: km, Ref: a.Ref}
	s.plan.BorderWaits += a.Duration

	// Engine off in the queue: the wait counts as the longest rest it covers.
	restDur, reduced := s.dailyRest()
	switch {
	case a.Duration >= DailyRestDuration:
		step.CountsAs = KindDailyRest
		reduced = false
	case reduced && a.Duration >= restDur:
		step.CountsAs = KindDailyRest
	case a.Duration >= s.breakNeeded():
		step.CountsAs = KindBreak
	case a.Duration >= SplitBreakFirst && !s.splitTaken:
		s.splitTaken = true
	}
	s.plan.Steps = append(s.plan.Steps, step)
	s.passTime(a.Duration)
	switch step.CountsAs {
	case KindDailyRest:
		if reduced {
			s.reducedLeft--
		}
		s.resetDay()
	case KindBreak:
		s.cont, s.splitTaken = 0, false
	}
}

// nextMonday is the start of the calendar week after t, in t's location.
func nextMonday(t time.Time) time.Time {
	y, m, d := t.Date()
	day := time.Date(y, m, d, 0, 0, 0, 0, t.Location())
	offset := (int(time.Monday) - int(day.Weekday()) + 7) % 7
	if offset == 0 {
		offset = 7
	}
	return day.AddDate(0, 0, offset)
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

// AfterIdle returns the driver state after not driving for d, ending at end:
// a long enough stop counts as a daily rest (regular, or reduced while
// reductions are left), a break, or the first part of a split break. A
// driver who has not driven since the last daily rest starts the duty
// period at end.
func (st DriverState) AfterIdle(d time.Duration, end time.Time) DriverState {
	switch {
	case d >= DailyRestDuration || (st.ReducedRestsLeft > 0 && d >= ReducedDailyRest):
		if d < DailyRestDuration {
			st.ReducedRestsLeft--
		}
		st.ContinuousDriving, st.SplitBreakTaken, st.DailyDriving = 0, false, 0
		st.DutyStartedAt = end
	case d >= BreakDuration || (st.SplitBreakTaken && d >= SplitBreakSecond):
		st.ContinuousDriving, st.SplitBreakTaken = 0, false
	case d >= SplitBreakFirst:
		st.SplitBreakTaken = true
	}
	// No driving yet in this duty period: the day starts when driving does.
	if st.DailyDriving == 0 {
		st.DutyStartedAt = end
	}
	return st
}
