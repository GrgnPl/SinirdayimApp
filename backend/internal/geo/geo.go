// Package geo has small geometry helpers shared by adapters and use cases.
package geo

import (
	"errors"
	"math"
	"sort"

	"github.com/burakaydin/sinir-bekleme/backend/internal/domain"
)

const earthRadiusKm = 6371.0

// DistanceKm is the great-circle distance between two points.
func DistanceKm(a, b domain.GeoPoint) float64 {
	la1, lo1 := rad(a.Lat), rad(a.Lng)
	la2, lo2 := rad(b.Lat), rad(b.Lng)
	h := math.Pow(math.Sin((la2-la1)/2), 2) + math.Cos(la1)*math.Cos(la2)*math.Pow(math.Sin((lo2-lo1)/2), 2)
	return 2 * earthRadiusKm * math.Asin(math.Sqrt(h))
}

func rad(d float64) float64 { return d * math.Pi / 180 }

// Cumulative returns the distance from the first point to each point.
func Cumulative(shape []domain.GeoPoint) []float64 {
	out := make([]float64, len(shape))
	for i := 1; i < len(shape); i++ {
		out[i] = out[i-1] + DistanceKm(shape[i-1], shape[i])
	}
	return out
}

// Nearest returns the index of the shape point closest to p and its distance.
func Nearest(shape []domain.GeoPoint, p domain.GeoPoint) (int, float64) {
	best, bestD := -1, math.Inf(1)
	for i, s := range shape {
		if d := DistanceKm(s, p); d < bestD {
			best, bestD = i, d
		}
	}
	return best, bestD
}

// PointAtKm interpolates the position at distance km along the shape.
func PointAtKm(shape []domain.GeoPoint, cum []float64, km float64) domain.GeoPoint {
	if len(shape) == 0 {
		return domain.GeoPoint{}
	}
	i := sort.SearchFloat64s(cum, km)
	if i <= 0 {
		return shape[0]
	}
	if i >= len(shape) {
		return shape[len(shape)-1]
	}
	seg := cum[i] - cum[i-1]
	if seg == 0 {
		return shape[i]
	}
	t := (km - cum[i-1]) / seg
	return domain.GeoPoint{
		Lat: shape[i-1].Lat + (shape[i].Lat-shape[i-1].Lat)*t,
		Lng: shape[i-1].Lng + (shape[i].Lng-shape[i-1].Lng)*t,
	}
}

var ErrBadPolyline = errors.New("geo: malformed polyline")

// DecodePolyline decodes a Google-style encoded polyline with the given
// precision (5 for Google, 6 for Valhalla/OSRM polyline6).
func DecodePolyline(s string, precision int) ([]domain.GeoPoint, error) {
	factor := math.Pow10(precision)
	var pts []domain.GeoPoint
	var lat, lng int
	for i := 0; i < len(s); {
		var vals [2]int
		for k := range vals {
			shift, result := 0, 0
			for {
				if i >= len(s) {
					return nil, ErrBadPolyline
				}
				b := int(s[i]) - 63
				i++
				result |= (b & 0x1f) << shift
				shift += 5
				if b < 0x20 {
					break
				}
			}
			if result&1 != 0 {
				vals[k] = ^(result >> 1)
			} else {
				vals[k] = result >> 1
			}
		}
		lat += vals[0]
		lng += vals[1]
		pts = append(pts, domain.GeoPoint{Lat: float64(lat) / factor, Lng: float64(lng) / factor})
	}
	return pts, nil
}
