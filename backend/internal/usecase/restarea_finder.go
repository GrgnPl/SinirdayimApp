package usecase

import (
	"math"
	"sort"

	"github.com/GrgnPl/SinirdayimApp/backend/internal/domain"
	"github.com/GrgnPl/SinirdayimApp/backend/internal/geo"
	"github.com/GrgnPl/SinirdayimApp/backend/internal/tacho"
)

// maxRestAreaOffsetKm is how far from the route line a place may be to count
// as "on the way". Kept tight: a detour eats driving time.
const maxRestAreaOffsetKm = 1.0

type placedArea struct {
	area domain.RestArea
	km   float64
}

// routeStops implements tacho.StopFinder over the rest areas along a route.
type routeStops struct {
	items []placedArea // sorted by km
	byID  map[string]domain.RestArea
}

func newRouteStops(areas []domain.RestArea, shape []domain.GeoPoint, cum []float64) *routeStops {
	rs := &routeStops{byID: map[string]domain.RestArea{}}
	for _, a := range areas {
		km, offset, ok := projectOnRoute(shape, cum, a.Location)
		if !ok || offset > maxRestAreaOffsetKm {
			continue
		}
		rs.items = append(rs.items, placedArea{a, km})
		rs.byID[a.ID] = a
	}
	sort.Slice(rs.items, func(i, j int) bool { return rs.items[i].km < rs.items[j].km })
	return rs
}

// Find returns the last suitable place in [fromKm, toKm]: stopping as late as
// possible keeps the most driving time for the day.
func (rs *routeStops) Find(kind tacho.Kind, fromKm, toKm float64) (float64, string, bool) {
	for i := len(rs.items) - 1; i >= 0; i-- {
		p := rs.items[i]
		if p.km > toKm {
			continue
		}
		if p.km < fromKm {
			break
		}
		if kind == tacho.KindDailyRest && !p.area.SuitsDailyRest() {
			continue
		}
		return p.km, p.area.ID, true
	}
	return 0, "", false
}

// projectOnRoute returns the distance along the route of p's projection onto
// the route line and how far p is from it. It scans every 10th vertex, then
// projects onto the segments around the closest one.
func projectOnRoute(shape []domain.GeoPoint, cum []float64, p domain.GeoPoint) (km, offset float64, ok bool) {
	if len(shape) < 2 {
		return 0, 0, false
	}
	const stride = 10
	best, bestD := 0, math.Inf(1)
	for i := 0; i < len(shape); i += stride {
		if d := geo.DistanceKm(shape[i], p); d < bestD {
			best, bestD = i, d
		}
	}
	offset = math.Inf(1)
	for i := max(best-stride, 0); i < min(best+stride, len(shape)-1); i++ {
		t, d := projectOnSegment(shape[i], shape[i+1], p)
		if d < offset {
			offset = d
			km = cum[i] + t*(cum[i+1]-cum[i])
		}
	}
	return km, offset, true
}

// projectOnSegment projects p onto segment a-b using a local flat
// approximation (fine for segments of a few km). It returns the position
// along the segment (0..1) and the distance from it in km.
func projectOnSegment(a, b, p domain.GeoPoint) (float64, float64) {
	kx := math.Cos(a.Lat*math.Pi/180) * 111.32 // km per degree of longitude
	const ky = 110.57                          // km per degree of latitude
	bx, by := (b.Lng-a.Lng)*kx, (b.Lat-a.Lat)*ky
	px, py := (p.Lng-a.Lng)*kx, (p.Lat-a.Lat)*ky
	t := 0.0
	if l2 := bx*bx + by*by; l2 > 0 {
		t = max(0, min(1, (px*bx+py*by)/l2))
	}
	dx, dy := px-t*bx, py-t*by
	return t, math.Hypot(dx, dy)
}
