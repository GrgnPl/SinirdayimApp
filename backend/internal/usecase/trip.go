package usecase

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/GrgnPl/SinirdayimApp/backend/internal/domain"
	"github.com/GrgnPl/SinirdayimApp/backend/internal/geo"
	"github.com/GrgnPl/SinirdayimApp/backend/internal/port"
	"github.com/GrgnPl/SinirdayimApp/backend/internal/tacho"
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
	// StateAt is when Driver was measured (usually "now"). A later
	// departure counts the time in between as rest. Zero = DepartAt.
	StateAt time.Time
	// Appointment, when set, pins the route to its crossing and plans the
	// wait there around the booked slot.
	Appointment *Appointment
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
	// Procedures that apply in this direction (appointment, truck park...).
	Procedures []domain.Procedure `json:"procedures,omitempty"`
	// SuggestedAppointment is the slot to book when the crossing requires
	// one: the planned arrival rounded up to the next half hour.
	SuggestedAppointment *time.Time `json:"suggestedAppointment,omitempty"`
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
	// Appointment tells how the trip fits a booked slot, when one was given.
	Appointment *AppointmentPlan `json:"appointment,omitempty"`
}

type TripService struct {
	Router  port.Router
	Catalog port.CrossingCatalog
	Status  *StatusService
	// RestAreas is optional; without it stops are placed on the road.
	RestAreas port.RestAreaStore
	// BorderPoints is optional; with it, crossings missing from the catalog
	// are still reported (without wait data) instead of silently ignored.
	BorderPoints port.BorderPointStore
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
	if !req.StateAt.IsZero() && req.DepartAt.After(req.StateAt) {
		req.Driver = req.Driver.AfterIdle(req.DepartAt.Sub(req.StateAt), req.DepartAt)
	}
	// With an appointment, the route must go through its crossing.
	withDirect := req.Appointment == nil
	var candidates []domain.Crossing
	if req.Appointment != nil {
		c, err := s.Catalog.Get(ctx, req.Appointment.CrossingID)
		if err != nil {
			return TripPlan{}, err
		}
		candidates = []domain.Crossing{c}
	} else {
		var err error
		if candidates, err = s.candidateCrossings(ctx, req.Origin, req.Destination); err != nil {
			return TripPlan{}, err
		}
	}

	type result struct {
		p   planned
		err error
	}
	// Index 0 is the direct route, then one route through each candidate.
	results := make([]result, len(candidates)+1)
	var wg sync.WaitGroup
	for i := range results {
		if i == 0 && !withDirect {
			results[i].err = errSkipped
			continue
		}
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
		case errors.Is(r.err, errSkipped):
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

	sched, appt, err := scheduleFor(r, req, nil)
	if err != nil {
		return planned{}, err
	}
	return planned{plan: assemble(r, sched, nil, appt), r: r}, nil
}

var errSkipped = errors.New("skipped")

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
	sched, appt, err := scheduleFor(p.r, req, stops)
	if err != nil {
		return TripPlan{}, err
	}
	return assemble(p.r, sched, stops, appt), nil
}

// assemble turns a schedule into the API plan.
func assemble(r *routed, sched tacho.Plan, stops *routeStops, appt *AppointmentPlan) TripPlan {
	arrivals := map[string]time.Time{}
	for _, st := range sched.Steps {
		if st.Kind == tacho.KindBorderWait {
			arrivals[st.Ref] = st.Start
		}
	}
	// Copy: routes are shared between candidate plans.
	crossings := append([]CrossingOnRoute(nil), r.crossings...)
	allKnown := true
	for i := range crossings {
		c := &crossings[i]
		if appt != nil && c.ID == appt.CrossingID {
			c.WaitKnown = true // the booked slot defines the wait
			appt.CrossingName = c.Name
		}
		for _, p := range c.Procedures {
			if at, ok := arrivals[string(c.ID)]; ok && p.Kind == domain.ProcedureAppointment {
				slot := suggestedSlot(at)
				c.SuggestedAppointment = &slot
			}
		}
		allKnown = allKnown && c.WaitKnown
	}
	out := TripPlan{
		DistanceKm:    r.cum[len(r.cum)-1],
		Departure:     sched.Departure,
		Arrival:       sched.Arrival,
		Crossings:     crossings,
		Polyline:      r.route.Polyline,
		AllWaitsKnown: allKnown,
		Appointment:   appt,
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
		dir, crossed := direction(h.c, route.Shape, cum, h.idx)
		if !crossed {
			continue
		}
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
			WaitKnown:  est.WaitMinutes != nil,
			Procedures: h.c.ProceduresFor(dir),
		})
		idxs = append(idxs, h.idx)
	}

	unknown, err := s.unknownCrossings(ctx, route, cum, out)
	if err != nil {
		return nil, nil, err
	}
	for _, u := range unknown {
		out = append(out, u)
		idxs = append(idxs, min(sort.SearchFloat64s(cum, u.AtKm), len(cum)-1))
	}
	order := make([]int, len(out))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return out[order[a]].AtKm < out[order[b]].AtKm })
	sortedOut := make([]CrossingOnRoute, len(out))
	sortedIdx := make([]int, len(out))
	for i, j := range order {
		sortedOut[i], sortedIdx[i] = out[j], idxs[j]
	}
	return sortedOut, sortedIdx, nil
}

const (
	// borderPointOnRouteKm is how close a border post must be to the route.
	borderPointOnRouteKm = 0.3
	// sameCrossingKm merges posts of one crossing (both sides, several lanes)
	// and matches them to catalog crossings.
	sameCrossingKm = 5.0
)

// unknownCrossings finds border posts on the route that are not catalog
// crossings. They have no wait data, which the plan must not hide.
func (s *TripService) unknownCrossings(ctx context.Context, route domain.Route, cum []float64, known []CrossingOnRoute) ([]CrossingOnRoute, error) {
	if s.BorderPoints == nil {
		return nil, nil
	}
	sw, ne := bounds(route.Shape, 0.05)
	points, err := s.BorderPoints.InBounds(ctx, sw, ne)
	if err != nil {
		return nil, err
	}
	type onRoute struct {
		p  domain.BorderPoint
		km float64
	}
	var hits []onRoute
	for _, p := range points {
		if nearKnown(p.Location, known) {
			continue
		}
		km, offset, ok := projectOnRoute(route.Shape, cum, p.Location)
		if ok && offset <= borderPointOnRouteKm {
			hits = append(hits, onRoute{p, km})
		}
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].km < hits[j].km })

	// Group posts that belong to one crossing (both sides, several lanes).
	type group struct {
		first     onRoute
		name      string
		countries []string
	}
	var groups []*group
	for _, h := range hits {
		if n := len(groups); n > 0 && h.km-groups[n-1].first.km <= sameCrossingKm {
			g := groups[n-1]
			g.name = firstNonEmpty(g.name, h.p.Name)
			g.countries = appendUnique(g.countries, h.p.Country)
			continue
		}
		groups = append(groups, &group{first: h, name: h.p.Name, countries: appendUnique(nil, h.p.Country)})
	}

	var out []CrossingOnRoute
	for _, g := range groups {
		// Lone posts inside one country are often internal checkpoints;
		// count them only when the name says it is a border crossing.
		if len(g.countries) < 2 && !looksLikeCrossing(g.name) {
			continue
		}
		c := CrossingOnRoute{
			ID:       domain.CrossingID(g.first.p.ID),
			Name:     firstNonEmpty(g.name, "Sınır kapısı (veri yok)"),
			AtKm:     g.first.km,
			Location: g.first.p.Location,
			Estimate: domain.WaitEstimate{
				CrossingID: domain.CrossingID(g.first.p.ID), Level: domain.LevelUnknown,
				Confidence: domain.ConfidenceLow, Method: "none", BasedOn: []string{},
			},
		}
		if len(g.countries) > 0 {
			c.From = g.countries[0]
		}
		if len(g.countries) > 1 {
			c.To = g.countries[1]
		}
		out = append(out, c)
	}
	return out, nil
}

// nearKnown reports whether p belongs to a catalog crossing on the route.
// Straight-line distance: truck terminals can make the route wander around
// a gate, so distances along the route would be misleading.
func nearKnown(p domain.GeoPoint, crossings []CrossingOnRoute) bool {
	for _, c := range crossings {
		if geo.DistanceKm(p, c.Location) <= sameCrossingKm {
			return true
		}
	}
	return false
}

var crossingWords = []string{"kapı", "gümrük", "hudut", "border", "crossing", "customs", "checkpoint", "пункт", "кпп", "გამშვები", "مرز"}

func looksLikeCrossing(name string) bool {
	n := strings.ToLower(name)
	for _, w := range crossingWords {
		if strings.Contains(n, w) {
			return true
		}
	}
	return false
}

func appendUnique(list []string, v string) []string {
	if v == "" {
		return list
	}
	for _, x := range list {
		if x == v {
			return list
		}
	}
	return append(list, v)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// direction decides which way the route passes a crossing. It takes the
// stretch of route within onRouteThresholdKm of the gate (truck terminals
// make routes loop around gates) and compares the sides of points a few km
// before entering and after leaving it: A → B is export from the crossing's
// first country. ok is false when both are on the same side, i.e. the route
// touches the gate without crossing there.
func direction(c domain.Crossing, shape []domain.GeoPoint, cum []float64, idx int) (dir domain.Direction, ok bool) {
	const probeKm = 5
	lo, hi := idx, idx
	for lo > 0 && geo.DistanceKm(shape[lo-1], c.Location) <= onRouteThresholdKm {
		lo--
	}
	for hi < len(shape)-1 && geo.DistanceKm(shape[hi+1], c.Location) <= onRouteThresholdKm {
		hi++
	}
	before := side(c, geo.PointAtKm(shape, cum, max(cum[lo]-probeKm, 0)))
	after := side(c, geo.PointAtKm(shape, cum, min(cum[hi]+probeKm, cum[len(cum)-1])))
	if before == after {
		return "", false
	}
	if before == 0 {
		return domain.DirectionExport, true
	}
	return domain.DirectionImport, true
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
			// Unknown waits become zero-length waits so the crossing still
			// shows up in the timeline.
			wait := tacho.Activity{Kind: tacho.KindBorderWait, Ref: string(crossings[next].ID)}
			if c := crossings[next]; c.WaitKnown {
				wait.Duration = time.Duration(*c.Estimate.WaitMinutes) * time.Minute
			}
			acts = append(acts, wait)
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
