// Package overpass loads truck stopping places from OpenStreetMap through the
// Overpass API, one country at a time.
package overpass

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/burakaydin/sinir-bekleme/backend/internal/domain"
)

const (
	sourceID   = "osm"
	DefaultURL = "https://overpass-api.de/api/interpreter"
)

// DefaultCountries covers Türkiye and the neighbours our crossings lead to.
var DefaultCountries = []string{"TR", "GE", "BG", "GR", "RS", "AM", "AZ", "IR"}

type Source struct {
	URL       string
	Countries []string
	Client    *http.Client
	// Pause between countries; the public server allows two concurrent
	// queries and asks clients to be gentle.
	Pause   time.Duration
	Retries int
	// CacheDir, when set, keeps each country's last response on disk and
	// reuses it while younger than the refresh interval.
	CacheDir string
	Log      *slog.Logger
}

func New(url string, countries []string) *Source {
	if url == "" {
		url = DefaultURL
	}
	if len(countries) == 0 {
		countries = DefaultCountries
	}
	return &Source{
		URL:       url,
		Countries: countries,
		Client:    &http.Client{Timeout: 3 * time.Minute},
		Pause:     10 * time.Second,
		Retries:   3,
		Log:       slog.New(slog.DiscardHandler),
	}
}

func (s *Source) ID() string              { return sourceID }
func (s *Source) Interval() time.Duration { return 24 * time.Hour }

// Fetch returns places for every country that could be loaded. It fails only
// when no country succeeded, so one flaky country does not wipe the others.
func (s *Source) Fetch(ctx context.Context) ([]domain.RestArea, error) {
	var all []domain.RestArea
	var lastErr error
	ok := 0
	queried := false
	for _, cc := range s.Countries {
		if areas, fresh := s.fromCache(cc); fresh {
			s.Log.Info("rest areas from cache", "country", cc, "count", len(areas))
			ok++
			all = append(all, areas...)
			continue
		}
		if queried {
			if err := sleep(ctx, s.Pause); err != nil {
				return nil, err
			}
		}
		queried = true
		areas, err := s.fetchCountry(ctx, cc)
		if err != nil {
			lastErr = fmt.Errorf("overpass %s: %w", cc, err)
			s.Log.Warn("rest areas failed", "country", cc, "err", err)
			continue
		}
		s.Log.Info("rest areas fetched", "country", cc, "count", len(areas))
		ok++
		all = append(all, areas...)
	}
	if ok == 0 && lastErr != nil {
		return nil, lastErr
	}
	return all, nil
}

func (s *Source) fetchCountry(ctx context.Context, cc string) ([]domain.RestArea, error) {
	q := query(cc)
	var lastErr error
	for attempt := 0; attempt <= s.Retries; attempt++ {
		if attempt > 0 {
			if err := sleep(ctx, time.Duration(attempt)*20*time.Second); err != nil {
				return nil, err
			}
		}
		body, err := s.post(ctx, q)
		if err != nil {
			lastErr = err
			continue
		}
		data, err := io.ReadAll(body)
		body.Close()
		if err != nil {
			lastErr = err
			continue
		}
		areas, err := Parse(bytes.NewReader(data), cc)
		if err != nil {
			lastErr = err
			continue
		}
		s.toCache(cc, data)
		return areas, nil
	}
	return nil, lastErr
}

func (s *Source) cachePath(cc string) string {
	return filepath.Join(s.CacheDir, "overpass-"+cc+".json")
}

func (s *Source) fromCache(cc string) ([]domain.RestArea, bool) {
	if s.CacheDir == "" {
		return nil, false
	}
	info, err := os.Stat(s.cachePath(cc))
	if err != nil || time.Since(info.ModTime()) > s.Interval() {
		return nil, false
	}
	f, err := os.Open(s.cachePath(cc))
	if err != nil {
		return nil, false
	}
	defer f.Close()
	areas, err := Parse(f, cc)
	return areas, err == nil
}

func (s *Source) toCache(cc string, data []byte) {
	if s.CacheDir == "" {
		return
	}
	if err := os.MkdirAll(s.CacheDir, 0o755); err != nil {
		s.Log.Warn("rest area cache", "err", err)
		return
	}
	if err := os.WriteFile(s.cachePath(cc), data, 0o644); err != nil {
		s.Log.Warn("rest area cache", "err", err)
	}
}

func (s *Source) post(ctx context.Context, q string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.URL, strings.NewReader(url.Values{"data": {q}}.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "sinirdayim/0.1")
	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	return resp.Body, nil
}

func query(cc string) string {
	return `[out:json][timeout:150];area["ISO3166-1"="` + cc + `"][admin_level=2]->.a;(` +
		`nwr["highway"="services"](area.a);` +
		`nwr["highway"="rest_area"](area.a);` +
		`nwr["amenity"="parking"]["hgv"~"^(yes|designated)$"](area.a);` +
		`nwr["amenity"="fuel"]["hgv"~"^(yes|designated)$"](area.a);` +
		`);out center tags;`
}

type response struct {
	Elements []struct {
		Type   string                      `json:"type"`
		ID     int64                       `json:"id"`
		Lat    float64                     `json:"lat"`
		Lon    float64                     `json:"lon"`
		Center *struct{ Lat, Lon float64 } `json:"center"`
		Tags   map[string]string           `json:"tags"`
	} `json:"elements"`
}

// Parse converts an Overpass JSON response into rest areas.
func Parse(r io.Reader, country string) ([]domain.RestArea, error) {
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
		loc := domain.GeoPoint{Lat: e.Lat, Lng: e.Lon}
		if e.Center != nil {
			loc = domain.GeoPoint{Lat: e.Center.Lat, Lng: e.Center.Lon}
		}
		if loc.Lat == 0 && loc.Lng == 0 {
			continue
		}
		t := e.Tags
		out = append(out, domain.RestArea{
			ID:          fmt.Sprintf("osm:%s/%d", e.Type, e.ID),
			Name:        firstNonEmpty(t["name:tr"], t["name"], t["brand"], t["operator"]),
			Kind:        kind,
			Location:    loc,
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

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
