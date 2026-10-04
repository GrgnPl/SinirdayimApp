// Package catalog provides the reference list of border crossings.
package catalog

import (
	"context"
	"fmt"

	"github.com/GrgnPl/SinirdayimApp/backend/internal/domain"
)

// Static is an in-code catalog. Gate coordinates follow the OSM
// barrier=border_control posts;
// SideRefs are nearby towns on each side of the border.
// It can later be replaced by a DB-backed catalog behind port.CrossingCatalog.
type Static struct {
	items []domain.Crossing
	byID  map[domain.CrossingID]domain.Crossing
}

// Procedures, see docs/sira-kaydi.md.
var (
	rss = domain.Procedure{
		Kind:      domain.ProcedureAppointment,
		Direction: domain.DirectionExport,
		System:    "RSS – Randevulu Sanal Sıra Sistemi",
		URL:       "https://rss.ticaret.gov.tr",
		Mandatory: true,
		Note:      "Randevu, BİLGE'de beyan edilen çıkış gümrüğüne bağlıdır. Kaçırılan randevunun ücreti iade edilmez; yoldayken ertelenebilir.",
	}
	georgianTruckPark = domain.Procedure{
		Kind:      domain.ProcedureTruckPark,
		Direction: domain.DirectionImport, // leaving Georgia towards Türkiye
		System:    "Gürcistan lisanslı TIR parkı elektronik sırası",
		URL:       "https://www.rs.ge/TirPark-en",
		Mandatory: true,
		Fee:       "80 GEL / römork",
		Note:      "Gümrüğe gitmeden önce kapıya bitişik lisanslı TIR parkına girilmesi zorunludur; sıra parka girişte verilir.",
	}
)

func NewStatic() *Static {
	items := []domain.Crossing{
		{ID: "tr-bg-kapikule", Name: "Kapıkule – Kapitan Andreevo", Countries: [2]string{"TR", "BG"}, Location: domain.GeoPoint{Lat: 41.7206, Lng: 26.3594}, Lanes: 1, SideRefs: [2]domain.GeoPoint{{Lat: 41.677, Lng: 26.556}, {Lat: 41.767, Lng: 26.199}}, Procedures: []domain.Procedure{rss}},
		{ID: "tr-bg-hamzabeyli", Name: "Hamzabeyli – Lesovo", Countries: [2]string{"TR", "BG"}, Location: domain.GeoPoint{Lat: 41.9600, Lng: 26.6080}, Lanes: 1, SideRefs: [2]domain.GeoPoint{{Lat: 41.677, Lng: 26.556}, {Lat: 42.160, Lng: 26.560}}},
		{ID: "tr-gr-ipsala", Name: "İpsala – Kipi", Countries: [2]string{"TR", "GR"}, Location: domain.GeoPoint{Lat: 40.9369, Lng: 26.3372}, Lanes: 1, SideRefs: [2]domain.GeoPoint{{Lat: 40.921, Lng: 26.383}, {Lat: 40.894, Lng: 26.173}}},
		{ID: "tr-ir-gurbulak", Name: "Gürbulak – Bazargan", Countries: [2]string{"TR", "IR"}, Location: domain.GeoPoint{Lat: 39.4133, Lng: 44.3765}, Lanes: 1, SideRefs: [2]domain.GeoPoint{{Lat: 39.547, Lng: 44.084}, {Lat: 39.392, Lng: 44.600}}},
		{ID: "tr-iq-habur", Name: "Habur – İbrahim Halil", Countries: [2]string{"TR", "IQ"}, Location: domain.GeoPoint{Lat: 37.1428, Lng: 42.5767}, Lanes: 1, SideRefs: [2]domain.GeoPoint{{Lat: 37.249, Lng: 42.470}, {Lat: 37.144, Lng: 42.687}}},
		{ID: "tr-ge-turkgozu", Name: "Türkgözü – Vale", Countries: [2]string{"TR", "GE"}, Location: domain.GeoPoint{Lat: 41.5877, Lng: 42.8185}, Lanes: 1, SideRefs: [2]domain.GeoPoint{{Lat: 41.180, Lng: 42.700}, {Lat: 41.640, Lng: 42.980}}, Procedures: []domain.Procedure{georgianTruckPark}}, // Posof-Ardahan / Akhaltsikhe
		{ID: "tr-ge-sarp", Name: "Sarp – Sarpi", Countries: [2]string{"TR", "GE"}, Location: domain.GeoPoint{Lat: 41.5178, Lng: 41.5475}, Lanes: 1, SideRefs: [2]domain.GeoPoint{{Lat: 41.392, Lng: 41.419}, {Lat: 41.645, Lng: 41.640}}, Procedures: []domain.Procedure{georgianTruckPark}},
		{ID: "si-at-maribor", Name: "Maribor – Wels", Countries: [2]string{"SI", "AT"}, Location: domain.GeoPoint{Lat: 46.6703, Lng: 15.6553}, Lanes: 1, SideRefs: [2]domain.GeoPoint{{Lat: 46.557, Lng: 15.646}, {Lat: 47.070, Lng: 15.440}}},
		{ID: "bg-rs-kalotina", Name: "Kalotina – Gradina", Countries: [2]string{"BG", "RS"}, Location: domain.GeoPoint{Lat: 42.9978, Lng: 22.8761}, Lanes: 1, SideRefs: [2]domain.GeoPoint{{Lat: 42.920, Lng: 22.930}, {Lat: 43.016, Lng: 22.775}}},
		{ID: "rs-hr-batrovci", Name: "Batrovci – Bajakovo", Countries: [2]string{"RS", "HR"}, Location: domain.GeoPoint{Lat: 45.0450, Lng: 19.1036}, Lanes: 1, SideRefs: [2]domain.GeoPoint{{Lat: 45.127, Lng: 19.227}, {Lat: 45.050, Lng: 18.700}}},
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
