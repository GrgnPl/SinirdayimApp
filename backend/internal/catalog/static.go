// Package catalog provides the reference list of border crossings.
package catalog

import (
	"context"
	"fmt"

	"github.com/burakaydin/sinir-bekleme/backend/internal/domain"
)

// Static is an in-code catalog. Coordinates are approximate gate locations.
// It can later be replaced by a DB-backed catalog behind port.CrossingCatalog.
type Static struct {
	items []domain.Crossing
	byID  map[domain.CrossingID]domain.Crossing
}

func NewStatic() *Static {
	items := []domain.Crossing{
		{ID: "tr-bg-kapikule", Name: "Kapıkule – Kapitan Andreevo", Countries: [2]string{"TR", "BG"}, Location: domain.GeoPoint{Lat: 41.7206, Lng: 26.3594}, Lanes: 1},
		{ID: "tr-bg-hamzabeyli", Name: "Hamzabeyli – Lesovo", Countries: [2]string{"TR", "BG"}, Location: domain.GeoPoint{Lat: 41.9772, Lng: 26.5147}, Lanes: 1},
		{ID: "tr-gr-ipsala", Name: "İpsala – Kipi", Countries: [2]string{"TR", "GR"}, Location: domain.GeoPoint{Lat: 40.9369, Lng: 26.3372}, Lanes: 1},
		{ID: "tr-ir-gurbulak", Name: "Gürbulak – Bazargan", Countries: [2]string{"TR", "IR"}, Location: domain.GeoPoint{Lat: 39.4331, Lng: 44.4264}, Lanes: 1},
		{ID: "tr-iq-habur", Name: "Habur – İbrahim Halil", Countries: [2]string{"TR", "IQ"}, Location: domain.GeoPoint{Lat: 37.1428, Lng: 42.5767}, Lanes: 1},
		{ID: "tr-ge-sarp", Name: "Sarp – Sarpi", Countries: [2]string{"TR", "GE"}, Location: domain.GeoPoint{Lat: 41.5178, Lng: 41.5475}, Lanes: 1},
		{ID: "si-at-maribor", Name: "Maribor – Wels", Countries: [2]string{"SI", "AT"}, Location: domain.GeoPoint{Lat: 46.6703, Lng: 15.6553}, Lanes: 1},
		{ID: "bg-rs-kalotina", Name: "Kalotina – Gradina", Countries: [2]string{"BG", "RS"}, Location: domain.GeoPoint{Lat: 42.9978, Lng: 22.8761}, Lanes: 1},
		{ID: "rs-hr-batrovci", Name: "Batrovci – Bajakovo", Countries: [2]string{"RS", "HR"}, Location: domain.GeoPoint{Lat: 45.0450, Lng: 19.1036}, Lanes: 1},
	}
	byID := make(map[domain.CrossingID]domain.Crossing, len(items))
	for _, c := range items {
		byID[c.ID] = c
	}
	return &Static{items: items, byID: byID}
}

func (s *Static) List(context.Context) ([]domain.Crossing, error) {
	return append([]domain.Crossing(nil), s.items...), nil
}

func (s *Static) Get(_ context.Context, id domain.CrossingID) (domain.Crossing, error) {
	c, ok := s.byID[id]
	if !ok {
		return domain.Crossing{}, fmt.Errorf("crossing %q: %w", id, domain.ErrNotFound)
	}
	return c, nil
}
