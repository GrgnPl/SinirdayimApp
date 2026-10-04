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
	// SideRefs are reference points well inside each country (index matches
	// Countries). They tell which way a route passes the crossing.
	SideRefs [2]GeoPoint `json:"-"`
	// Procedures are formalities trucks must go through, per direction.
	Procedures []Procedure `json:"procedures,omitempty"`
}

// ProcedureKind is the type of formality at a crossing.
type ProcedureKind string

const (
	// ProcedureAppointment: a time slot must be booked in an official
	// queue system before reaching the gate.
	ProcedureAppointment ProcedureKind = "appointment"
	// ProcedureTruckPark: trucks must enter a licensed park where the
	// electronic queue is run; no remote booking.
	ProcedureTruckPark ProcedureKind = "truck_park"
)

// Procedure is a formality at a crossing in one direction.
type Procedure struct {
	Kind      ProcedureKind `json:"kind"`
	Direction Direction     `json:"direction"`
	System    string        `json:"system"` // "RSS – Randevulu Sanal Sıra Sistemi"
	URL       string        `json:"url,omitempty"`
	Mandatory bool          `json:"mandatory"`
	Fee       string        `json:"fee,omitempty"` // free text, e.g. "80 GEL / römork"
	Note      string        `json:"note,omitempty"`
}

// ProceduresFor returns the procedures that apply in direction dir.
func (c Crossing) ProceduresFor(dir Direction) []Procedure {
	var out []Procedure
	for _, p := range c.Procedures {
		if p.Direction == dir {
			out = append(out, p)
		}
	}
	return out
}

// AppointmentSystem returns the booking procedure for direction dir, if any.
func (c Crossing) AppointmentSystem(dir Direction) (Procedure, bool) {
	for _, p := range c.ProceduresFor(dir) {
		if p.Kind == ProcedureAppointment {
			return p, true
		}
	}
	return Procedure{}, false
}
