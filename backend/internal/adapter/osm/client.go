// Package osm loads truck-relevant places from OpenStreetMap through the
// Overpass API, one country at a time: rest areas and border controls.
package osm

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const DefaultURL = "https://overpass-api.de/api/interpreter"

// DefaultCountries covers Türkiye and the neighbours our crossings lead to.
var DefaultCountries = []string{"TR", "GE", "BG", "GR", "RS", "AM", "AZ", "IR"}

// Client holds what every dataset shares: endpoint, politeness, retries and
// the on-disk cache.
type Client struct {
	URL       string
	Countries []string
	HTTP      *http.Client
	// Pause between queries; the public server allows two concurrent
	// queries and asks clients to be gentle.
	Pause   time.Duration
	Retries int
	// CacheDir, when set, keeps each country's last response on disk and
	// reuses it while younger than MaxAge.
	CacheDir string
	MaxAge   time.Duration
	Log      *slog.Logger
}

func NewClient(url string, countries []string) *Client {
	if url == "" {
		url = DefaultURL
	}
	if len(countries) == 0 {
		countries = DefaultCountries
	}
	return &Client{
		URL:       url,
		Countries: countries,
		HTTP:      &http.Client{Timeout: 3 * time.Minute},
		Pause:     10 * time.Second,
		Retries:   3,
		MaxAge:    24 * time.Hour,
		Log:       slog.New(slog.DiscardHandler),
	}
}

// dataset is one kind of OSM object: how to query it and how to parse it.
type dataset[T any] struct {
	name  string // used in logs and cache file names
	query func(country string) string
	parse func(r io.Reader, country string) ([]T, error)
}

// fetchAll loads the dataset for every country. It fails only when no
// country succeeded, so one flaky country does not wipe the others.
func fetchAll[T any](ctx context.Context, c *Client, ds dataset[T]) ([]T, error) {
	var all []T
	var lastErr error
	ok := 0
	queried := false
	for _, cc := range c.Countries {
		if items, fresh := fromCache(c, ds, cc); fresh {
			c.Log.Info("osm from cache", "dataset", ds.name, "country", cc, "count", len(items))
			ok++
			all = append(all, items...)
			continue
		}
		if queried {
			if err := sleep(ctx, c.Pause); err != nil {
				return nil, err
			}
		}
		queried = true
		items, err := fetchCountry(ctx, c, ds, cc)
		if err != nil {
			lastErr = fmt.Errorf("overpass %s %s: %w", ds.name, cc, err)
			c.Log.Warn("osm fetch failed", "dataset", ds.name, "country", cc, "err", err)
			continue
		}
		c.Log.Info("osm fetched", "dataset", ds.name, "country", cc, "count", len(items))
		ok++
		all = append(all, items...)
	}
	if ok == 0 && lastErr != nil {
		return nil, lastErr
	}
	return all, nil
}

func fetchCountry[T any](ctx context.Context, c *Client, ds dataset[T], cc string) ([]T, error) {
	q := ds.query(cc)
	var lastErr error
	for attempt := 0; attempt <= c.Retries; attempt++ {
		if attempt > 0 {
			if err := sleep(ctx, time.Duration(attempt)*20*time.Second); err != nil {
				return nil, err
			}
		}
		data, err := c.post(ctx, q)
		if err != nil {
			lastErr = err
			continue
		}
		items, err := ds.parse(bytes.NewReader(data), cc)
		if err != nil {
			lastErr = err
			continue
		}
		toCache(c, ds.name, cc, data)
		return items, nil
	}
	return nil, lastErr
}

func (c *Client) post(ctx context.Context, q string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, strings.NewReader(url.Values{"data": {q}}.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "sinirdayim/0.1")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func cachePath(c *Client, name, cc string) string {
	return filepath.Join(c.CacheDir, "osm-"+name+"-"+cc+".json")
}

func fromCache[T any](c *Client, ds dataset[T], cc string) ([]T, bool) {
	if c.CacheDir == "" {
		return nil, false
	}
	path := cachePath(c, ds.name, cc)
	info, err := os.Stat(path)
	if err != nil || time.Since(info.ModTime()) > c.MaxAge {
		return nil, false
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer f.Close()
	items, err := ds.parse(f, cc)
	return items, err == nil
}

func toCache(c *Client, name, cc string, data []byte) {
	if c.CacheDir == "" {
		return
	}
	if err := os.MkdirAll(c.CacheDir, 0o755); err != nil {
		c.Log.Warn("osm cache", "err", err)
		return
	}
	if err := os.WriteFile(cachePath(c, name, cc), data, 0o644); err != nil {
		c.Log.Warn("osm cache", "err", err)
	}
}

// countryArea selects the country's boundary as area .a for the query body.
func countryArea(cc string) string {
	return `[out:json][timeout:150];area["ISO3166-1"="` + cc + `"][admin_level=2]->.a;`
}

// element is an Overpass JSON element with "out center tags".
type element struct {
	Type   string  `json:"type"`
	ID     int64   `json:"id"`
	Lat    float64 `json:"lat"`
	Lon    float64 `json:"lon"`
	Center *struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	} `json:"center"`
	Tags map[string]string `json:"tags"`
}

type response struct {
	Elements []element `json:"elements"`
}

func (e element) id() string { return fmt.Sprintf("osm:%s/%d", e.Type, e.ID) }

func (e element) position() (lat, lng float64, ok bool) {
	if e.Center != nil {
		return e.Center.Lat, e.Center.Lon, true
	}
	return e.Lat, e.Lon, e.Lat != 0 || e.Lon != 0
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
