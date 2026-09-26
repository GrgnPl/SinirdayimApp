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
