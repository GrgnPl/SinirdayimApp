package osm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/burakaydin/sinir-bekleme/backend/internal/domain"
)

// RestAreas implements port.RestAreaSource: services, rest areas, truck
// parkings and fuel stations that accept trucks.
type RestAreas struct{ *Client }

func (RestAreas) ID() string                { return "osm" }
func (r RestAreas) Interval() time.Duration { return r.MaxAge }

func (r RestAreas) Fetch(ctx context.Context) ([]domain.RestArea, error) {
	return fetchAll(ctx, r.Client, restAreaDataset)
}

var restAreaDataset = dataset[domain.RestArea]{
	name: "restareas",
	query: func(cc string) string {
		return countryArea(cc) + `(` +
			`nwr["highway"="services"](area.a);` +
			`nwr["highway"="rest_area"](area.a);` +
			`nwr["amenity"="parking"]["hgv"~"^(yes|designated)$"](area.a);` +
			`nwr["amenity"="fuel"]["hgv"~"^(yes|designated)$"](area.a);` +
			`);out center tags;`
	},
	parse: ParseRestAreas,
}

// ParseRestAreas converts an Overpass JSON response into rest areas.
func ParseRestAreas(r io.Reader, country string) ([]domain.RestArea, error) {
	var resp response
	if err := json.NewDecoder(r).Decode(&resp); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	out := make([]domain.RestArea, 0, len(resp.Elements))
	for _, e := range resp.Elements {
		kind, ok := kindOf(e.Tags)
		if !ok || notForStopping(e.Tags) {
			continue
		}
		lat, lng, ok := e.position()
		if !ok {
			continue
		}
		t := e.Tags
		out = append(out, domain.RestArea{
			ID:          e.id(),
			Name:        firstNonEmpty(t["name:tr"], t["name"], t["brand"], t["operator"]),
			Kind:        kind,
			Location:    domain.GeoPoint{Lat: lat, Lng: lng},
			Country:     country,
			Toilets:     yesNo(t["toilets"]),
			Shower:      yesNo(t["shower"]),
			Restaurant:  yesNo(t["restaurant"]),
			Fee:         yesNo(t["fee"]),
			Supervised:  yesNo(t["supervised"]),
			HGVCapacity: count(t["capacity:hgv"]),
			Hours:       t["opening_hours"],
		})
	}
	return out, nil
}

func kindOf(t map[string]string) (domain.RestAreaKind, bool) {
	switch {
	case t["highway"] == "services":
		return domain.RestAreaServices, true
	case t["highway"] == "rest_area":
		return domain.RestAreaRest, true
	case t["amenity"] == "parking":
		return domain.RestAreaTruckParking, true
	case t["amenity"] == "fuel":
		return domain.RestAreaTruckFuel, true
	}
	return "", false
}

// Weigh and inspection stations are sometimes tagged as services; trucks can
// not rest there.
var notStoppingNames = []string{"denetleme istasyonu", "kantar", "weigh", "gümrük", "customs"}

func notForStopping(t map[string]string) bool {
	if t["hgv"] == "no" || t["access"] == "no" || t["access"] == "private" {
		return true
	}
	name := strings.ToLower(t["name"] + " " + t["name:tr"] + " " + t["name:en"])
	for _, n := range notStoppingNames {
		if strings.Contains(name, n) {
			return true
		}
	}
	return false
}

func yesNo(v string) *bool {
	switch strings.ToLower(v) {
	case "yes", "designated", "customers", "interval":
		return domain.Ptr(true)
	case "no":
		return domain.Ptr(false)
	}
	return nil
}

func count(v string) *int {
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || n <= 0 {
		return nil
	}
	return &n
}
