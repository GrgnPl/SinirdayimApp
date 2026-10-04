// Package valhalla implements port.Router with a Valhalla server using the
// truck costing model.
package valhalla

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/burakaydin/sinir-bekleme/backend/internal/domain"
	"github.com/burakaydin/sinir-bekleme/backend/internal/geo"
)

// DefaultURL is the FOSSGIS public instance: fair use only, fine for
// development. Production should run its own Valhalla.
const DefaultURL = "https://valhalla1.openstreetmap.de"

// Truck describes the vehicle for route restrictions (heights, weights...).
type Truck struct {
	HeightM  float64 `json:"height"`
	WidthM   float64 `json:"width"`
	LengthM  float64 `json:"length"`
	WeightT  float64 `json:"weight"`
	AxleLoad float64 `json:"axle_load"`
}

// StandardTIR is a typical 40 t semi-trailer.
var StandardTIR = Truck{HeightM: 4.0, WidthM: 2.55, LengthM: 16.5, WeightT: 40, AxleLoad: 11.5}

type Router struct {
	URL    string
	Client *http.Client
	Truck  Truck
}

func New(url string) *Router {
	if url == "" {
		url = DefaultURL
	}
	return &Router{URL: url, Client: &http.Client{Timeout: 60 * time.Second}, Truck: StandardTIR}
}

type location struct {
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
	Type string  `json:"type,omitempty"` // "through" for via points
}

type request struct {
	Locations      []location       `json:"locations"`
	Costing        string           `json:"costing"`
	CostingOptions map[string]Truck `json:"costing_options"`
	Units          string           `json:"units"`
	DirectionsType string           `json:"directions_type"`
}

type response struct {
	Trip struct {
		Legs []struct {
			Shape     string `json:"shape"`
			Maneuvers []struct {
				Time            float64 `json:"time"`
				Length          float64 `json:"length"`
				BeginShapeIndex int     `json:"begin_shape_index"`
				EndShapeIndex   int     `json:"end_shape_index"`
			} `json:"maneuvers"`
		} `json:"legs"`
		Summary struct {
			Time   float64 `json:"time"`
			Length float64 `json:"length"`
		} `json:"summary"`
	} `json:"trip"`
	Error string `json:"error"`
}

func (r *Router) Route(ctx context.Context, from, to domain.GeoPoint, via ...domain.GeoPoint) (domain.Route, error) {
	locs := []location{{Lat: from.Lat, Lon: from.Lng}}
	for _, v := range via {
		// "through" keeps a single leg and lets the route use edges that
		// OSM marks as destination-only for trucks, which is common on
		// border gate roads.
		locs = append(locs, location{Lat: v.Lat, Lon: v.Lng, Type: "through"})
	}
	locs = append(locs, location{Lat: to.Lat, Lon: to.Lng})
	body, _ := json.Marshal(request{
		Locations:      locs,
		Costing:        "truck",
		CostingOptions: map[string]Truck{"truck": r.Truck},
		Units:          "kilometers",
		DirectionsType: "maneuvers",
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.URL+"/route", bytes.NewReader(body))
	if err != nil {
		return domain.Route{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "sinirdayim/0.1")
	resp, err := r.Client.Do(req)
	if err != nil {
		return domain.Route{}, fmt.Errorf("valhalla: %w", err)
	}
	defer resp.Body.Close()

	var out response
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return domain.Route{}, fmt.Errorf("valhalla: decode: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusBadRequest {
			return domain.Route{}, fmt.Errorf("%w: %s", domain.ErrNoRoute, out.Error)
		}
		return domain.Route{}, fmt.Errorf("valhalla: status %d: %s", resp.StatusCode, out.Error)
	}
	if len(out.Trip.Legs) != 1 {
		return domain.Route{}, fmt.Errorf("valhalla: expected 1 leg, got %d", len(out.Trip.Legs))
	}

	leg := out.Trip.Legs[0]
	shape, err := geo.DecodePolyline(leg.Shape, 6)
	if err != nil {
		return domain.Route{}, fmt.Errorf("valhalla: %w", err)
	}
	route := domain.Route{
		DistanceKm: out.Trip.Summary.Length,
		Duration:   seconds(out.Trip.Summary.Time),
		Shape:      shape,
		Polyline:   leg.Shape,
	}
	for _, m := range leg.Maneuvers {
		route.Steps = append(route.Steps, domain.RouteStep{
			Duration:   seconds(m.Time),
			DistanceKm: m.Length,
			BeginIdx:   m.BeginShapeIndex,
			EndIdx:     m.EndShapeIndex,
		})
	}
	return route, nil
}

func seconds(s float64) time.Duration { return time.Duration(s * float64(time.Second)) }
