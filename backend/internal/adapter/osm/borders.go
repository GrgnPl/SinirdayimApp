package osm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/GrgnPl/SinirdayimApp/backend/internal/domain"
)

// BorderControls implements port.BorderPointSource with OSM
// barrier=border_control objects. They reveal crossings a route uses even
// when we have no wait data for them.
type BorderControls struct{ *Client }

func (BorderControls) ID() string                { return "osm" }
func (b BorderControls) Interval() time.Duration { return b.MaxAge }

func (b BorderControls) Fetch(ctx context.Context) ([]domain.BorderPoint, error) {
	return fetchAll(ctx, b.Client, borderDataset)
}

var borderDataset = dataset[domain.BorderPoint]{
	name: "borders",
	query: func(cc string) string {
		return countryArea(cc) + `nwr["barrier"="border_control"](area.a);out center tags;`
	},
	parse: ParseBorderControls,
}

// ParseBorderControls converts an Overpass JSON response into border points.
func ParseBorderControls(r io.Reader, country string) ([]domain.BorderPoint, error) {
	var resp response
	if err := json.NewDecoder(r).Decode(&resp); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	out := make([]domain.BorderPoint, 0, len(resp.Elements))
	for _, e := range resp.Elements {
		lat, lng, ok := e.position()
		if !ok || e.Tags["barrier"] != "border_control" {
			continue
		}
		t := e.Tags
		out = append(out, domain.BorderPoint{
			ID:       e.id(),
			Name:     firstNonEmpty(t["name:tr"], t["name"], t["name:en"]),
			Location: domain.GeoPoint{Lat: lat, Lng: lng},
			Country:  country,
		})
	}
	return out, nil
}
