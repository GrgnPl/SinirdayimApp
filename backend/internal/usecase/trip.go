package usecase

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/burakaydin/sinir-bekleme/backend/internal/domain"
	"github.com/burakaydin/sinir-bekleme/backend/internal/geo"
	"github.com/burakaydin/sinir-bekleme/backend/internal/port"
	"github.com/burakaydin/sinir-bekleme/backend/internal/tacho"
)

const (
	// onRouteThresholdKm is how close the route must pass a gate to count it.
	onRouteThresholdKm = 3.0
	// maxDetourRatio drops candidate crossings that are too far off the
	// straight line between origin and destination.
	maxDetourRatio = 1.5
	// maxCandidates bounds the number of routing calls per plan.
	maxCandidates = 4
)

type TripRequest struct {
	Origin      domain.GeoPoint
	Destination domain.GeoPoint
	DepartAt    time.Time
	Driver      tacho.DriverState
}

// CrossingOnRoute is a border crossing the route passes through.
type CrossingOnRoute struct {
	ID        domain.CrossingID   `json:"id"`
	Name      string              `json:"name"`
	From      string              `json:"from"` // country code the truck leaves
	To        string              `json:"to"`
	Direction domain.Direction    `json:"direction"`
	AtKm      float64             `json:"atKm"`
	Location  domain.GeoPoint     `json:"location"`
	Estimate  domain.WaitEstimate `json:"estimate"`
	// WaitKnown is false when no wait time is available; the plan then
	// assumes no wait at this crossing.
	WaitKnown bool `json:"waitKnown"`
}

type TripStep struct {
	Kind        tacho.Kind      `json:"kind"`
	Start       time.Time       `json:"start"`
	End         time.Time       `json:"end"`
	DurationMin int             `json:"durationMin"`
	FromKm      float64         `json:"fromKm"`
	ToKm        float64         `json:"toKm"`
	Location    domain.GeoPoint `json:"location"` // where the step starts
	Reason      string          `json:"reason,omitempty"`
	// Reduced marks a 9h daily rest or the 30 min second part of a split break.
	Reduced bool `json:"reduced,omitempty"`
	// CountsAs tells which rest a border wait satisfied, if any.
	CountsAs   tacho.Kind `json:"countsAs,omitempty"`
	CrossingID string     `json:"crossingId,omitempty"`
	// RestArea is where a break or daily rest is planned, when a suitable
	// place was found near the limit; otherwise the stop is on the road.
	RestArea *domain.RestArea `json:"restArea,omitempty"`
}

type TripTotals struct {
	DrivingMin    int `json:"drivingMin"`
	BreakMin      int `json:"breakMin"`
	DailyRestMin  int `json:"dailyRestMin"`
	WeeklyRestMin int `json:"weeklyRestMin"`
	BorderWaitMin int `json:"borderWaitMin"`
	TotalMin      int `json:"totalMin"`
}

// TripAlternative summarizes a plan through a different set of crossings.
type TripAlternative struct {
	Via           []string  `json:"via"` // crossing names in order
	DistanceKm    float64   `json:"distanceKm"`
	Arrival       time.Time `json:"arrival"`
	TotalMin      int       `json:"totalMin"`
	BorderWaitMin int       `json:"borderWaitMin"`
	// AllWaitsKnown is false when some crossing on this route has no wait
	// data; its total then excludes that wait.
	AllWaitsKnown bool `json:"allWaitsKnown"`
	Selected      bool `json:"selected"`
}

type TripPlan struct {
	DistanceKm float64           `json:"distanceKm"`
	Departure  time.Time         `json:"departure"`
	Arrival    time.Time         `json:"arrival"`
	Totals     TripTotals        `json:"totals"`
	Crossings  []CrossingOnRoute `json:"crossings"`
	Steps      []TripStep        `json:"steps"`
	Polyline   string            `json:"polyline"` // encoded, precision 6
	// AllWaitsKnown is false when a crossing on the route has no wait data.
	AllWaitsKnown bool              `json:"allWaitsKnown"`
	Alternatives  []TripAlternative `json:"alternatives"`
}

type TripService struct {
	Router  port.Router
	Catalog port.CrossingCatalog
	Status  *StatusService
	// RestAreas is optional; without it stops are placed on the road.
	RestAreas port.RestAreaStore
}

// routed is a computed route with everything needed to schedule it again.
type routed struct {
	route      domain.Route
	cum        []float64
	crossings  []CrossingOnRoute
	activities []tacho.Activity
}

type planned struct {
	plan TripPlan
	r    *routed
}

// Plan evaluates the direct route and routes forced through each plausible
// border crossing, then returns the one that arrives first. Plans whose
// border waits are all known win over plans with unknown waits.
func (s *TripService) Plan(ctx context.Context, req TripRequest) (TripPlan, error) {
	candidates, err := s.candidateCrossings(ctx, req.Origin, req.Destination)
	if err != nil {
		return TripPlan{}, err
	}

	type result struct {
		p   planned
		err error
	}
	results := make([]result, len(candidates)+1)
	var wg sync.WaitGroup
	for i := range results {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var via []domain.GeoPoint
			if i > 0 {
				via = []domain.GeoPoint{candidates[i-1].Location}
			}
			p, err := s.planRoute(ctx, req, via)
			results[i] = result{p, err}
		}()
	}
	wg.Wait()

	var plans []planned
	var firstErr error
	for _, r := range results {
		switch {
		case r.err == nil:
			plans = append(plans, r.p)
		case firstErr == nil:
			firstErr = r.err
		}
	}
	if len(plans) == 0 {
		return TripPlan{}, firstErr
	}
	plans = dedupeByCrossings(plans)
	sort.SliceStable(plans, func(i, j int) bool { return better(plans[i].plan, plans[j].plan) })

	// Candidates are compared with stops on the road; only the chosen route
	// is rescheduled around real rest areas.
	best, err := s.withRestAreas(ctx, req, plans[0])
	if err != nil {
		return TripPlan{}, err
	}
	plans[0].plan = best
	for i, pl := range plans {
		p := pl.plan
		best.Alternatives = append(best.Alternatives, TripAlternative{
			Via:           crossingNames(p),
			DistanceKm:    p.DistanceKm,
			Arrival:       p.Arrival,
			TotalMin:      p.Totals.TotalMin,
			BorderWaitMin: p.Totals.BorderWaitMin,
			AllWaitsKnown: p.AllWaitsKnown,
			Selected:      i == 0,
		})
	}
	return best, nil
}

// candidateCrossings returns crossings whose two sides separate origin and
// destination and that do not require a large detour, closest first.
func (s *TripService) candidateCrossings(ctx context.Context, origin, dest domain.GeoPoint) ([]domain.Crossing, error) {
	all, err := s.Catalog.List(ctx)
	if err != nil {
		return nil, err
	}
	direct := max(geo.DistanceKm(origin, dest), 1)
	type cand struct {
		c     domain.Crossing
		ratio float64
	}
	var cands []cand
	for _, c := range all {
		if side(c, origin) == side(c, dest) {
			continue
		}
		ratio := (geo.DistanceKm(origin, c.Location) + geo.DistanceKm(c.Location, dest)) / direct
		if ratio <= maxDetourRatio {
			cands = append(cands, cand{c, ratio})
		}
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].ratio < cands[j].ratio })
	out := make([]domain.Crossing, 0, min(len(cands), maxCandidates))
	for i := 0; i < len(cands) && i < maxCandidates; i++ {
		out = append(out, cands[i].c)
	}
	return out, nil
}

// side is 0 or 1 depending on which side of the crossing p lies. The border
// is approximated by the line through the gate perpendicular to the axis
// from SideRefs[0] to SideRefs[1]; unlike comparing distances to the two
// reference towns, this stays right for points far from the crossing.
func side(c domain.Crossing, p domain.GeoPoint) int {
	kx := math.Cos(c.Location.Lat * math.Pi / 180) // shrink longitude like a local map
	ax := (c.SideRefs[1].Lng - c.SideRefs[0].Lng) * kx
	ay := c.SideRefs[1].Lat - c.SideRefs[0].Lat
	px := (p.Lng - c.Location.Lng) * kx
	py := p.Lat - c.Location.Lat
	if px*ax+py*ay <= 0 {
		return 0
	}
	return 1
}

// better orders plans: fully known waits first, then earliest arrival, then
// shortest distance.
func better(a, b TripPlan) bool {
	if a.AllWaitsKnown != b.AllWaitsKnown {
		return a.AllWaitsKnown
	}
	if !a.Arrival.Equal(b.Arrival) {
		return a.Arrival.Before(b.Arrival)
	}
	return a.DistanceKm < b.DistanceKm
}

// dedupeByCrossings keeps the fastest plan for each sequence of crossings.
func dedupeByCrossings(plans []planned) []planned {
	best := map[string]int{}
	var out []planned
	for _, p := range plans {
		key := strings.Join(crossingNames(p.plan), "|")
		if i, ok := best[key]; ok {
			if p.plan.Arrival.Before(out[i].plan.Arrival) {
				out[i] = p
			}
			continue
		}
		best[key] = len(out)
		out = append(out, p)
	}
	return out
}

func crossingNames(p TripPlan) []string {
	names := make([]string, len(p.Crossings))
	for i, c := range p.Crossings {
		names[i] = c.Name
	}
	return names
}

// planRoute routes (optionally through via points) and schedules the trip
// with stops on the road.
func (s *TripService) planRoute(ctx context.Context, req TripRequest, via []domain.GeoPoint) (planned, error) {
	route, err := s.Router.Route(ctx, req.Origin, req.Destination, via...)
	if err != nil {
		return planned{}, err
	}
	if len(route.Shape) < 2 {
		return planned{}, fmt.Errorf("%w: empty route", domain.ErrNoRoute)
	}
	cum := geo.Cumulative(route.Shape)

	crossings, idxs, err := s.crossingsOnRoute(ctx, route, cum)
	if err != nil {
		return planned{}, err
	}
	r := &routed{route: route, cum: cum, crossings: crossings, activities: buildActivities(route, cum, crossings, idxs)}

	sched, err := tacho.Schedule(r.activities, req.DepartAt, req.Driver)
	if err != nil {
		return planned{}, err
	}
	return planned{plan: assemble(r, sched, nil), r: r}, nil
}

// withRestAreas reschedules a route so breaks and rests fall on real places.
// Without a store, or when none is near the route, the plan is unchanged.
func (s *TripService) withRestAreas(ctx context.Context, req TripRequest, p planned) (TripPlan, error) {
	if s.RestAreas == nil {
		return p.plan, nil
	}
	sw, ne := bounds(p.r.route.Shape, 0.05)
	areas, err := s.RestAreas.InBounds(ctx, sw, ne)
	if err != nil {
		return TripPlan{}, err
	}
	stops := newRouteStops(areas, p.r.route.Shape, p.r.cum)
	if len(stops.items) == 0 {
		return p.plan, nil
	}
	sched, err := tacho.ScheduleWith(p.r.activities, req.DepartAt, req.Driver, stops)
	if err != nil {
		return TripPlan{}, err
	}
	return assemble(p.r, sched, stops), nil
}

// assemble turns a schedule into the API plan.
func assemble(r *routed, sched tacho.Plan, stops *routeStops) TripPlan {
	allKnown := true
	for _, c := range r.crossings {
		allKnown = allKnown && c.WaitKnown
	}
	out := TripPlan{
		DistanceKm:    r.cum[len(r.cum)-1],
		Departure:     sched.Departure,
		Arrival:       sched.Arrival,
		Crossings:     r.crossings,
		Polyline:      r.route.Polyline,
		AllWaitsKnown: allKnown,
		Totals: TripTotals{
			DrivingMin:    minutes(sched.Driving),
			BreakMin:      minutes(sched.Breaks),
			DailyRestMin:  minutes(sched.DailyRests),
			WeeklyRestMin: minutes(sched.WeeklyRests),
			BorderWaitMin: minutes(sched.BorderWaits),
			TotalMin:      minutes(sched.Arrival.Sub(sched.Departure)),
		},
	}
	for _, st := range sched.Steps {
		step := TripStep{
			Kind:        st.Kind,
			Start:       st.Start,
			End:         st.Start.Add(st.Duration),
			DurationMin: minutes(st.Duration),
			FromKm:      st.FromKm,
			ToKm:        st.ToKm,
			Location:    geo.PointAtKm(r.route.Shape, r.cum, st.FromKm),
			Reason:      st.Reason,
			Reduced:     st.Reduced,
			CountsAs:    st.CountsAs,
		}
		switch {
		case st.Kind == tacho.KindBorderWait:
			step.CrossingID = st.Ref
		case st.Ref != "" && stops != nil:
			if a, ok := stops.byID[st.Ref]; ok {
				step.RestArea = &a
				step.Location = a.Location
			}
		}
		out.Steps = append(out.Steps, step)
	}
	return out
}

// bounds returns the south-west and north-east corners around the points.
func bounds(points []domain.GeoPoint, pad float64) (domain.GeoPoint, domain.GeoPoint) {
	sw, ne := points[0], points[0]
	for _, p := range points {
		sw.Lat, sw.Lng = min(sw.Lat, p.Lat), min(sw.Lng, p.Lng)
		ne.Lat, ne.Lng = max(ne.Lat, p.Lat), max(ne.Lng, p.Lng)
	}
	return domain.GeoPoint{Lat: sw.Lat - pad, Lng: sw.Lng - pad}, domain.GeoPoint{Lat: ne.Lat + pad, Lng: ne.Lng + pad}
}

// crossingsOnRoute returns known crossings the route passes, ordered along
// the route, with the shape index where each is passed.
func (s *TripService) crossingsOnRoute(ctx context.Context, route domain.Route, cum []float64) ([]CrossingOnRoute, []int, error) {
	all, err := s.Catalog.List(ctx)
	if err != nil {
		return nil, nil, err
	}
	type hit struct {
		c   domain.Crossing
		idx int
	}
	var hits []hit
	for _, c := range all {
		idx, d := geo.Nearest(route.Shape, c.Location)
		if d <= onRouteThresholdKm {
			hits = append(hits, hit{c, idx})
		}
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].idx < hits[j].idx })

	out := make([]CrossingOnRoute, 0, len(hits))
	idxs := make([]int, 0, len(hits))
	for _, h := range hits {
		dir := direction(h.c, route.Shape, cum, h.idx)
		est, err := s.Status.Estimate(ctx, h.c, dir)
		if err != nil {
			return nil, nil, err
		}
		from, to := h.c.Countries[0], h.c.Countries[1]
		if dir == domain.DirectionImport {
			from, to = to, from
		}
		out = append(out, CrossingOnRoute{
			ID: h.c.ID, Name: h.c.Name, From: from, To: to, Direction: dir,
			AtKm: cum[h.idx], Location: h.c.Location, Estimate: est,
			WaitKnown: est.WaitMinutes != nil,
		})
		idxs = append(idxs, h.idx)
	}
	return out, idxs, nil
}

// direction decides which way the route passes a crossing by looking at a
// point a few km before the gate: on side A the truck goes A → B (export
// from the crossing's first country).
func direction(c domain.Crossing, shape []domain.GeoPoint, cum []float64, idx int) domain.Direction {
	before := geo.PointAtKm(shape, cum, max(cum[idx]-5, 0))
	if side(c, before) == 0 {
		return domain.DirectionExport
	}
	return domain.DirectionImport
}

// buildActivities turns route steps into drive activities, splitting the step
// that contains a crossing and inserting the border wait there.
func buildActivities(route domain.Route, cum []float64, crossings []CrossingOnRoute, idxs []int) []tacho.Activity {
	var acts []tacho.Activity
	next := 0
	for _, st := range route.Steps {
		begin, end := st.BeginIdx, st.EndIdx
		for next < len(idxs) && idxs[next] >= begin && idxs[next] <= end {
			split := idxs[next]
			acts = append(acts, part(st, cum, begin, split))
			if c := crossings[next]; c.WaitKnown {
				acts = append(acts, tacho.Activity{
					Kind:     tacho.KindBorderWait,
					Duration: time.Duration(*c.Estimate.WaitMinutes) * time.Minute,
					Ref:      string(c.ID),
				})
			}
			begin = split
			next++
		}
		acts = append(acts, part(st, cum, begin, end))
	}
	return acts
}

// part is the share of a route step between two shape indices, with time
// proportional to distance.
func part(st domain.RouteStep, cum []float64, from, to int) tacho.Activity {
	stepKm := cum[st.EndIdx] - cum[st.BeginIdx]
	km := cum[to] - cum[from]
	d := st.Duration
	if stepKm > 0 {
		d = time.Duration(float64(st.Duration) * km / stepKm)
	} else if from != st.BeginIdx || to != st.EndIdx {
		d = 0
	}
	return tacho.Activity{Kind: tacho.KindDrive, Duration: d, DistanceKm: km}
}

func minutes(d time.Duration) int { return int(d.Round(time.Minute) / time.Minute) }
