package domain

import "time"

// Route is a truck route between two points.
type Route struct {
	DistanceKm float64
	Duration   time.Duration
	Shape      []GeoPoint
	Polyline   string // encoded polyline (precision 6) for clients to draw
	Steps      []RouteStep
}

// RouteStep is a maneuver-sized piece of the route with its own travel time.
type RouteStep struct {
	Duration   time.Duration
	DistanceKm float64
	BeginIdx   int // index into Route.Shape
	EndIdx     int
}

// Place is a geocoding result.
type Place struct {
	Name     string   `json:"name"`
	Label    string   `json:"label"` // "Samsun, Türkiye"
	Country  string   `json:"country"`
	Location GeoPoint `json:"location"`
}
