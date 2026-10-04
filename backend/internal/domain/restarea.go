package domain

// RestAreaKind is what a stopping place offers trucks.
type RestAreaKind string

const (
	RestAreaServices     RestAreaKind = "services"      // motorway services: fuel, food, parking
	RestAreaRest         RestAreaKind = "rest_area"     // lay-by / rest area, usually no fuel
	RestAreaTruckParking RestAreaKind = "truck_parking" // parking designated for trucks
	RestAreaTruckFuel    RestAreaKind = "truck_fuel"    // fuel station that accepts trucks
)

// RestArea is a place where a truck can take a break or a daily rest.
// Facility fields are optional: nil means unknown.
type RestArea struct {
	ID       string       `json:"id"` // source-scoped, e.g. "osm:way/123"
	Name     string       `json:"name,omitempty"`
	Kind     RestAreaKind `json:"kind"`
	Location GeoPoint     `json:"location"`
	Country  string       `json:"country,omitempty"`

	Toilets     *bool  `json:"toilets,omitempty"`
	Shower      *bool  `json:"shower,omitempty"`
	Restaurant  *bool  `json:"restaurant,omitempty"`
	Fee         *bool  `json:"fee,omitempty"`
	Supervised  *bool  `json:"supervised,omitempty"` // guarded / CCTV
	HGVCapacity *int   `json:"hgvCapacity,omitempty"`
	Hours       string `json:"openingHours,omitempty"`
}

// SuitsDailyRest reports whether the place is a sensible spot for an 11h rest.
func (r RestArea) SuitsDailyRest() bool {
	return r.Kind == RestAreaServices || r.Kind == RestAreaTruckParking
}
