// Package photon implements port.Geocoder with Komoot's Photon (OSM data).
package photon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/GrgnPl/SinirdayimApp/backend/internal/domain"
)

// DefaultURL is the public instance: fair use only, fine for development.
const DefaultURL = "https://photon.komoot.io"

type Geocoder struct {
	URL    string
	Client *http.Client
}

func New(baseURL string) *Geocoder {
	if baseURL == "" {
		baseURL = DefaultURL
	}
	return &Geocoder{URL: baseURL, Client: &http.Client{Timeout: 10 * time.Second}}
}

type featureCollection struct {
	Features []struct {
		Geometry struct {
			Coordinates []float64 `json:"coordinates"` // [lon, lat]
		} `json:"geometry"`
		Properties struct {
			Name        string `json:"name"`
			City        string `json:"city"`
			State       string `json:"state"`
			Country     string `json:"country"`
			CountryCode string `json:"countrycode"`
		} `json:"properties"`
	} `json:"features"`
}

func (g *Geocoder) Search(ctx context.Context, query string, limit int) ([]domain.Place, error) {
	q := url.Values{}
	q.Set("q", query)
	q.Set("limit", strconv.Itoa(limit))
	// Cities, towns and villages first; that is what drivers type.
	q.Add("osm_tag", "place")
	// Photon has no Turkish; English keeps names in Latin script
	// ("Tbilisi" rather than "თბილისი").
	q.Set("lang", "en")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.URL+"/api/?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "sinirdayim/0.1")
	resp, err := g.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("photon: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("photon: status %d", resp.StatusCode)
	}
	var fc featureCollection
	if err := json.NewDecoder(resp.Body).Decode(&fc); err != nil {
		return nil, fmt.Errorf("photon: decode: %w", err)
	}

	places := make([]domain.Place, 0, len(fc.Features))
	seen := map[string]bool{}
	for _, f := range fc.Features {
		if len(f.Geometry.Coordinates) != 2 {
			continue
		}
		p := f.Properties
		parts := []string{p.Name}
		for _, s := range []string{p.State, p.Country} {
			if s != "" && s != p.Name {
				parts = append(parts, s)
			}
		}
		label := strings.Join(parts, ", ")
		if seen[label] {
			continue // a city and its province often share a name
		}
		seen[label] = true
		places = append(places, domain.Place{
			Name:     p.Name,
			Label:    label,
			Country:  strings.ToUpper(p.CountryCode),
			Location: domain.GeoPoint{Lat: f.Geometry.Coordinates[1], Lng: f.Geometry.Coordinates[0]},
		})
	}
	return places, nil
}
