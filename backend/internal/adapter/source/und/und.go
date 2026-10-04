// Package und scrapes the "Sınır Kapıları Yoğunluk Durumu" page published by
// UND (Uluslararası Nakliyeciler Derneği).
package und

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/GrgnPl/SinirdayimApp/backend/internal/domain"
)

const (
	sourceID   = "und"
	defaultURL = "https://www.und.org.tr/sinir-kapilari-yogunluk-durumu"
)

// crossingIDs maps the first word of UND's row label to our crossing id.
var crossingIDs = map[string]domain.CrossingID{
	"Kapıkule":   "tr-bg-kapikule",
	"Hamzabeyli": "tr-bg-hamzabeyli",
	"Ipsala":     "tr-gr-ipsala",
	"İpsala":     "tr-gr-ipsala",
	"Gürbulak":   "tr-ir-gurbulak",
	"Habur":      "tr-iq-habur",
	"Sarp":       "tr-ge-sarp",
	"Maribor":    "si-at-maribor",
	"Kalotina":   "bg-rs-kalotina",
	"Batrovci":   "rs-hr-batrovci",
}

// Source implements port.SnapshotSource.
type Source struct {
	URL      string
	Client   *http.Client
	Location *time.Location
}

func New() (*Source, error) {
	loc, err := time.LoadLocation("Europe/Istanbul")
	if err != nil {
		return nil, fmt.Errorf("und: load timezone: %w", err)
	}
	return &Source{
		URL:      defaultURL,
		Client:   &http.Client{Timeout: 20 * time.Second},
		Location: loc,
	}, nil
}

func (s *Source) ID() string              { return sourceID }
func (s *Source) Interval() time.Duration { return 30 * time.Minute }

func (s *Source) Fetch(ctx context.Context) ([]domain.Snapshot, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "sinirdayim/0.1")
	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("und: fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("und: unexpected status %d", resp.StatusCode)
	}
	return Parse(resp.Body, s.Location)
}

// Parse extracts snapshots from the UND page. Each table on the page is one
// report; its first header cell holds the report time ("25-09-2026 10:36:42").
// Columns: name | İHRACAT (km) | TIR PARKI (araç) | ÇIKIŞ SAYISI (araç) | İTHALAT (km) | RO-LA (araç)
func Parse(r io.Reader, loc *time.Location) ([]domain.Snapshot, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("und: parse html: %w", err)
	}

	var out []domain.Snapshot
	for _, table := range findAll(doc, "table") {
		headers := findAll(table, "th")
		if len(headers) == 0 {
			continue
		}
		observedAt, err := time.ParseInLocation("02-01-2006 15:04:05", text(headers[0]), loc)
		if err != nil {
			continue // not a report table
		}
		for _, row := range findAll(table, "tr") {
			cells := findAll(row, "td")
			if len(cells) < 5 {
				continue
			}
			id, ok := crossingID(text(cells[0]))
			if !ok {
				continue
			}
			export := domain.Snapshot{
				CrossingID:      id,
				Direction:       domain.DirectionExport,
				Source:          sourceID,
				ObservedAt:      observedAt,
				QueueKm:         parseFloat(text(cells[1])),
				ParkedVehicles:  parseInt(text(cells[2])),
				DailyThroughput: parseInt(text(cells[3])),
			}
			if hasData(export) {
				out = append(out, export)
			}
			imp := domain.Snapshot{
				CrossingID: id,
				Direction:  domain.DirectionImport,
				Source:     sourceID,
				ObservedAt: observedAt,
				QueueKm:    parseFloat(text(cells[4])),
			}
			if hasData(imp) {
				out = append(out, imp)
			}
		}
	}
	return out, nil
}

func crossingID(label string) (domain.CrossingID, bool) {
	first, _, _ := strings.Cut(strings.TrimSpace(label), " ")
	id, ok := crossingIDs[first]
	return id, ok
}

func hasData(s domain.Snapshot) bool {
	return s.QueueKm != nil || s.ParkedVehicles != nil || s.DailyThroughput != nil
}

// parseInt reads "1.797 ARAÇ" -> 1797; "- ARAÇ" -> nil.
func parseInt(s string) *int {
	num, _, _ := strings.Cut(strings.TrimSpace(s), " ")
	num = strings.ReplaceAll(num, ".", "")
	n, err := strconv.Atoi(num)
	if err != nil {
		return nil
	}
	return &n
}

// parseFloat reads "16 KM" or "2,5 KM"; "- KM" -> nil.
func parseFloat(s string) *float64 {
	num, _, _ := strings.Cut(strings.TrimSpace(s), " ")
	num = strings.ReplaceAll(num, ",", ".")
	f, err := strconv.ParseFloat(num, 64)
	if err != nil {
		return nil
	}
	return &f
}

func findAll(n *html.Node, tag string) []*html.Node {
	var out []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && c.Data == tag {
				out = append(out, c)
				if tag == "table" {
					continue // don't descend into nested tables
				}
			}
			walk(c)
		}
	}
	walk(n)
	return out
}

func text(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.Join(strings.Fields(b.String()), " ")
}
