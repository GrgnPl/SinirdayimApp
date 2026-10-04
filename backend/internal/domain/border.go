package domain

// BorderPoint is a border control post from map data. Unlike Crossing, it
// carries no wait data; it tells that a route crosses a border there.
type BorderPoint struct {
	ID       string   `json:"id"`
	Name     string   `json:"name,omitempty"`
	Location GeoPoint `json:"location"`
	Country  string   `json:"country,omitempty"`
}
