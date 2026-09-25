// Package domain holds the core business types. It has no dependencies on
// frameworks, storage or data sources.
package domain

// CrossingID is a stable, URL-safe identifier such as "tr-ge-sarp".
type CrossingID string

// Direction is the flow of traffic relative to the reporting country.
type Direction string

const (
	DirectionExport Direction = "export" // leaving the country
	DirectionImport Direction = "import" // entering the country
)

// GeoPoint is a WGS84 coordinate.
type GeoPoint struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// Crossing is a border crossing point for freight traffic.
type Crossing struct {
	ID        CrossingID `json:"id"`
	Name      string     `json:"name"`      // "Sarp – Sarpi"
	Countries [2]string  `json:"countries"` // ISO-3166 alpha-2, e.g. ["TR","GE"]
	Location  GeoPoint   `json:"location"`
	// Lanes is the typical number of truck queue lanes; used to convert queue
	// length (km) into a vehicle count.
	Lanes int `json:"lanes"`
}
