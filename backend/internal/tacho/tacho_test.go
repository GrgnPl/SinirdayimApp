package tacho

import (
	"testing"
	"time"
)

var depart = time.Date(2026, 9, 28, 6, 0, 0, 0, time.UTC)

func drive(h float64, kmh float64) Activity {
	d := time.Duration(h * float64(time.Hour))
	return Activity{Kind: KindDrive, Duration: d, DistanceKm: h * kmh}
}

func kinds(p Plan) []Kind {
	out := make([]Kind, len(p.Steps))
	for i, s := range p.Steps {
		out[i] = s.Kind
	}
	return out
}

func eqKinds(t *testing.T, got []Kind, want ...Kind) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("steps = %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("steps = %v, want %v", got, want)
		}
	}
}

func TestShortTripNoStops(t *testing.T) {
	p, err := Schedule([]Activity{drive(3, 70)}, depart, DriverState{})
	if err != nil {
		t.Fatal(err)
	}
	eqKinds(t, kinds(p), KindDrive)
	if !p.Arrival.Equal(depart.Add(3 * time.Hour)) {
		t.Errorf("arrival = %v", p.Arrival)
	}
}

func TestBreakAfterFourAndHalfHours(t *testing.T) {
	p, _ := Schedule([]Activity{drive(7.5, 70)}, depart, DriverState{})
	eqKinds(t, kinds(p), KindDrive, KindBreak, KindDrive)
	if p.Steps[0].Duration != MaxContinuousDriving {
		t.Errorf("first drive = %v", p.Steps[0].Duration)
	}
	if got := p.Steps[1].FromKm; got != 4.5*70 {
		t.Errorf("break at km %v, want %v", got, 4.5*70)
	}
	want := depart.Add(7*time.Hour + 30*time.Minute + BreakDuration)
	if !p.Arrival.Equal(want) {
		t.Errorf("arrival = %v, want %v", p.Arrival, want)
	}
}

func TestExistingStateIsRespected(t *testing.T) {
	// Already drove 4h since the last break: only 30 min left.
	p, _ := Schedule([]Activity{drive(2, 60)}, depart, DriverState{ContinuousDriving: 4 * time.Hour, DailyDriving: 4 * time.Hour})
	eqKinds(t, kinds(p), KindDrive, KindBreak, KindDrive)
	if p.Steps[0].Duration != 30*time.Minute {
		t.Errorf("first drive = %v", p.Steps[0].Duration)
	}
}

func TestDailyRestAfterNineHours(t *testing.T) {
	p, _ := Schedule([]Activity{drive(12, 70)}, depart, DriverState{})
	// 4h30 drive, break, 4h30 drive, daily rest (9h reached), 3h drive
	eqKinds(t, kinds(p), KindDrive, KindBreak, KindDrive, KindDailyRest, KindDrive)
	if p.Steps[3].Reason != ReasonDaily {
		t.Errorf("reason = %s", p.Steps[3].Reason)
	}
	want := depart.Add(12*time.Hour + BreakDuration + DailyRestDuration)
	if !p.Arrival.Equal(want) {
		t.Errorf("arrival = %v, want %v", p.Arrival, want)
	}
}

func TestExtendedDayUsesTenHours(t *testing.T) {
	p, _ := Schedule([]Activity{drive(12, 70)}, depart, DriverState{ExtendedDaysLeft: 1})
	// 4h30, break, 4h30, break, 1h (10h total), daily rest, 2h
	eqKinds(t, kinds(p), KindDrive, KindBreak, KindDrive, KindBreak, KindDrive, KindDailyRest, KindDrive)
	var firstDay time.Duration
	for _, s := range p.Steps[:5] {
		if s.Kind == KindDrive {
			firstDay += s.Duration
		}
	}
	if firstDay != ExtendedDailyDriving {
		t.Errorf("first day driving = %v, want 10h", firstDay)
	}
}

func TestDutyWindowForcesRest(t *testing.T) {
	// 2h drive, 10h border wait (not a daily rest), then 3h drive:
	// duty started at departure, so after 12h only 1h of duty is left.
	acts := []Activity{drive(2, 60), {Kind: KindBorderWait, Duration: 10 * time.Hour, Ref: "tr-ge-sarp"}, drive(3, 60)}
	p, _ := Schedule(acts, depart, DriverState{})
	eqKinds(t, kinds(p), KindDrive, KindBorderWait, KindDrive, KindDailyRest, KindDrive)
	if p.Steps[2].Duration != time.Hour || p.Steps[3].Reason != ReasonDuty {
		t.Errorf("unexpected steps %+v", p.Steps)
	}
}

func TestLongBorderWaitCountsAsDailyRest(t *testing.T) {
	// 8h driving, 30h wait at Sarp (>= 11h rest), then 6h: a fresh day.
	acts := []Activity{drive(4, 60), drive(4, 60), {Kind: KindBorderWait, Duration: 30 * time.Hour, Ref: "tr-ge-sarp"}, drive(6, 60)}
	p, _ := Schedule(acts, depart, DriverState{})
	eqKinds(t, kinds(p), KindDrive, KindBreak, KindDrive, KindBorderWait, KindDrive, KindBreak, KindDrive)
	if p.DailyRests != 0 {
		t.Errorf("no separate daily rest expected, got %v", p.DailyRests)
	}
}

func TestInvalidState(t *testing.T) {
	if _, err := Schedule(nil, depart, DriverState{ContinuousDriving: 5 * time.Hour}); err == nil {
		t.Fatal("expected error")
	}
}

// fixedFinder offers places at fixed kilometre marks.
type fixedFinder struct {
	places map[float64]string // km -> ref
	rests  map[float64]bool   // places suitable for a daily rest
}

func (f fixedFinder) Find(kind Kind, from, to float64) (float64, string, bool) {
	best, ok := -1.0, false
	for km := range f.places {
		if km < from || km > to || (kind == KindDailyRest && !f.rests[km]) {
			continue
		}
		if km > best {
			best, ok = km, true
		}
	}
	return best, f.places[best], ok
}

func TestBreakSnapsToLastPlaceBeforeLimit(t *testing.T) {
	// Split into many short segments like a real route: 7.5 h at 60 km/h.
	var acts []Activity
	for range 30 {
		acts = append(acts, drive(0.25, 60))
	}
	// Limit falls at km 270; places at 200 (too early), 240 and 290 (too late).
	f := fixedFinder{places: map[float64]string{200: "a", 240: "b", 290: "c"}}
	p, _ := ScheduleWith(acts, depart, DriverState{}, f)
	eqKinds(t, kinds(p), KindDrive, KindBreak, KindDrive)
	br := p.Steps[1]
	if br.Ref != "b" || br.FromKm < 239.99 || br.FromKm > 240.01 {
		t.Fatalf("break = %+v, want at km 240 (ref b)", br)
	}
	if p.Steps[0].Duration != 4*time.Hour {
		t.Errorf("first drive = %v, want 4h", p.Steps[0].Duration)
	}
}

func TestNoPlaceInWindowKeepsRoadStop(t *testing.T) {
	f := fixedFinder{places: map[float64]string{100: "far"}}
	p, _ := ScheduleWith([]Activity{drive(7.5, 60)}, depart, DriverState{}, f)
	if br := p.Steps[1]; br.Ref != "" || br.FromKm != 270 {
		t.Fatalf("break = %+v, want road stop at km 270", br)
	}
}

func TestDailyRestUsesWiderWindowAndSuitablePlaces(t *testing.T) {
	// Day limit (9 h) reached at km 540 (60 km/h, with a break at 270).
	f := fixedFinder{
		places: map[float64]string{450: "parking", 530: "lay-by"},
		rests:  map[float64]bool{450: true},
	}
	p, _ := ScheduleWith([]Activity{drive(12, 60)}, depart, DriverState{ExtendedDaysLeft: 0}, f)
	var rest *Step
	for i := range p.Steps {
		if p.Steps[i].Kind == KindDailyRest {
			rest = &p.Steps[i]
		}
	}
	if rest == nil || rest.Ref != "parking" || rest.FromKm < 449.99 || rest.FromKm > 450.01 {
		t.Fatalf("rest = %+v, want at km 450 (parking)", rest)
	}
}

func TestDayEndsEarlyAtGoodPlaceInsteadOfLateBreak(t *testing.T) {
	// 10 h day available. Continuous limit at 4.5 h (km 270), again at 9 h
	// (km 540 after the first break); the day ends at 10 h (km 600). At the
	// second break the day has only 1 h left, so the rest is taken at the
	// truck parking (km 500) instead of a break at km 540 and a roadside
	// rest at km 600. The remaining 220 km (3 h 40) need no further break.
	f := fixedFinder{
		places: map[float64]string{500: "parking"},
		rests:  map[float64]bool{500: true},
	}
	p, _ := ScheduleWith([]Activity{drive(12, 60)}, depart, DriverState{ExtendedDaysLeft: 1}, f)
	eqKinds(t, kinds(p), KindDrive, KindBreak, KindDrive, KindDailyRest, KindDrive)
	rest := p.Steps[3]
	if rest.Ref != "parking" || rest.FromKm < 499.99 || rest.FromKm > 500.01 || rest.Reason != ReasonDaily {
		t.Fatalf("rest = %+v, want daily rest at km 500", rest)
	}
}

func stepsOf(p Plan, k Kind) []Step {
	var out []Step
	for _, s := range p.Steps {
		if s.Kind == k {
			out = append(out, s)
		}
	}
	return out
}

func TestSplitBreakSecondPartIsThirtyMinutes(t *testing.T) {
	p, _ := Schedule([]Activity{drive(6, 60)}, depart, DriverState{SplitBreakTaken: true})
	br := stepsOf(p, KindBreak)
	if len(br) != 1 || br[0].Duration != SplitBreakSecond || !br[0].Reduced {
		t.Fatalf("breaks = %+v, want one 30 min second part", br)
	}
}

func TestShortBorderWaitIsFirstPartOfSplitBreak(t *testing.T) {
	acts := []Activity{drive(2, 60), {Kind: KindBorderWait, Duration: 20 * time.Minute, Ref: "x"}, drive(4, 60)}
	p, _ := Schedule(acts, depart, DriverState{})
	br := stepsOf(p, KindBreak)
	if len(br) != 1 || br[0].Duration != SplitBreakSecond {
		t.Fatalf("breaks = %+v, want a 30 min second part after the 20 min wait", br)
	}
	// 20 min does not reset continuous driving: break still after 4h30 of driving.
	if br[0].FromKm != 270 {
		t.Errorf("break at km %v, want 270", br[0].FromKm)
	}
}

func TestReducedDailyRestUsedWhileAvailable(t *testing.T) {
	p, _ := Schedule([]Activity{drive(24, 60)}, depart, DriverState{ReducedRestsLeft: 1})
	rests := stepsOf(p, KindDailyRest)
	if len(rests) != 2 {
		t.Fatalf("rests = %+v, want 2", rests)
	}
	if rests[0].Duration != ReducedDailyRest || !rests[0].Reduced {
		t.Errorf("first rest = %+v, want 9h reduced", rests[0])
	}
	if rests[1].Duration != DailyRestDuration || rests[1].Reduced {
		t.Errorf("second rest = %+v, want regular 11h", rests[1])
	}
}

func TestBorderWaitCountsAsReducedRest(t *testing.T) {
	acts := []Activity{drive(4, 60), {Kind: KindBorderWait, Duration: 10 * time.Hour, Ref: "x"}, drive(8, 60)}
	p, _ := Schedule(acts, depart, DriverState{ReducedRestsLeft: 1})
	if w := stepsOf(p, KindBorderWait)[0]; w.CountsAs != KindDailyRest {
		t.Fatalf("wait counts as %q, want daily rest", w.CountsAs)
	}
	if n := len(stepsOf(p, KindDailyRest)); n != 0 {
		t.Errorf("got %d extra daily rests, want none", n)
	}
	// Without a reduction left, 10h is not a daily rest: only a break.
	p, _ = Schedule(acts, depart, DriverState{})
	if w := stepsOf(p, KindBorderWait)[0]; w.CountsAs != KindBreak {
		t.Errorf("wait counts as %q, want break", w.CountsAs)
	}
}

func TestWeeklyDrivingLimitForcesWeeklyRest(t *testing.T) {
	p, _ := Schedule([]Activity{drive(3, 60)}, depart, DriverState{WeeklyDriving: 55 * time.Hour})
	eqKinds(t, kinds(p), KindDrive, KindWeeklyRest, KindDrive)
	// Monday 07:00: the limit lifts next Monday, so the rest lasts until then.
	w := p.Steps[1]
	nextWeek := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)
	if w.Reason != ReasonWeekly || w.FromKm != 60 || !w.Start.Add(w.Duration).Equal(nextWeek) {
		t.Fatalf("weekly rest = %+v, want until %v", w, nextWeek)
	}
}

func TestBiweeklyLimit(t *testing.T) {
	st := DriverState{WeeklyDriving: 40 * time.Hour, PrevWeekDriving: 49 * time.Hour}
	p, _ := Schedule([]Activity{drive(3, 60)}, depart, st)
	if w := stepsOf(p, KindWeeklyRest); len(w) != 1 || w[0].FromKm != 60 {
		t.Fatalf("weekly rest = %+v, want after 1h (90h over two weeks)", w)
	}
}

func TestNewCalendarWeekResetsWeeklyDriving(t *testing.T) {
	sunday := time.Date(2026, 9, 27, 23, 30, 0, 0, time.UTC)
	p, _ := Schedule([]Activity{drive(3, 60)}, sunday, DriverState{WeeklyDriving: 55 * time.Hour})
	if w := stepsOf(p, KindWeeklyRest); len(w) != 0 {
		t.Fatalf("weekly rest = %+v, want none: the week ends after 30 min", w)
	}
}

func TestWeeklyRestDueAfterSixDays(t *testing.T) {
	st := DriverState{LastWeeklyRestEnd: depart.Add(-MaxBetweenWeeklyRest + time.Hour)}
	p, _ := Schedule([]Activity{drive(3, 60)}, depart, st)
	w := stepsOf(p, KindWeeklyRest)
	if len(w) != 1 || w[0].Reason != ReasonWeeklyDue || w[0].FromKm != 60 {
		t.Fatalf("weekly rest = %+v, want due after 1h", w)
	}
}

func TestInvalidStates(t *testing.T) {
	for _, st := range []DriverState{
		{ReducedRestsLeft: 4},
		{ExtendedDaysLeft: 3},
		{WeeklyDriving: 57 * time.Hour},
	} {
		if _, err := Schedule(nil, depart, st); err == nil {
			t.Errorf("state %+v: expected error", st)
		}
	}
}

func TestAfterIdle(t *testing.T) {
	tired := DriverState{ContinuousDriving: 4 * time.Hour, DailyDriving: 8 * time.Hour, DutyStartedAt: depart.Add(-10 * time.Hour)}
	end := depart.Add(12 * time.Hour)

	if st := tired.AfterIdle(12*time.Hour, end); st.DailyDriving != 0 || st.ContinuousDriving != 0 || !st.DutyStartedAt.Equal(end) {
		t.Errorf("12h idle: %+v, want a fresh day", st)
	}
	withReduced := tired
	withReduced.ReducedRestsLeft = 1
	if st := withReduced.AfterIdle(9*time.Hour+30*time.Minute, end); st.DailyDriving != 0 || st.ReducedRestsLeft != 0 {
		t.Errorf("9.5h idle with a reduction: %+v, want reduced daily rest", st)
	}
	if st := tired.AfterIdle(time.Hour, end); st.ContinuousDriving != 0 || st.DailyDriving != 8*time.Hour {
		t.Errorf("1h idle: %+v, want a break only", st)
	}
	if st := tired.AfterIdle(20*time.Minute, end); !st.SplitBreakTaken || st.ContinuousDriving != 4*time.Hour {
		t.Errorf("20 min idle: %+v, want first part of a split break", st)
	}
	fresh := DriverState{DutyStartedAt: depart}
	if st := fresh.AfterIdle(7*time.Hour, end); !st.DutyStartedAt.Equal(end) {
		t.Errorf("fresh driver: duty starts at %v, want %v", st.DutyStartedAt, end)
	}
}
