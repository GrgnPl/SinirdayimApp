package osm

import (
	"strings"
	"testing"

	"github.com/burakaydin/sinir-bekleme/backend/internal/domain"
)

const fixture = `{"elements":[
 {"type":"way","id":11,"center":{"lat":41.0,"lon":39.7},"tags":{"highway":"services","name":"Opet Dinlenme Tesisi","toilets":"yes","restaurant":"yes"}},
 {"type":"node","id":22,"lat":41.1,"lon":39.8,"tags":{"highway":"rest_area"}},
 {"type":"way","id":33,"center":{"lat":41.2,"lon":39.9},"tags":{"amenity":"parking","hgv":"designated","fee":"no","supervised":"yes","capacity:hgv":"40"}},
 {"type":"node","id":44,"lat":41.3,"lon":40.0,"tags":{"amenity":"fuel","hgv":"yes","brand":"Shell"}},
 {"type":"node","id":55,"lat":41.4,"lon":40.1,"tags":{"amenity":"cafe"}},
 {"type":"node","id":66,"lat":41.5,"lon":40.2,"tags":{"highway":"services","name":"Ulaştırma Bakanlığı Karayolu Denetleme İstasyonu"}},
 {"type":"node","id":77,"lat":41.6,"lon":40.3,"tags":{"amenity":"parking","hgv":"yes","access":"private"}}
]}`

func TestParseRestAreas(t *testing.T) {
	areas, err := ParseRestAreas(strings.NewReader(fixture), "TR")
	if err != nil {
		t.Fatal(err)
	}
	if len(areas) != 4 {
		t.Fatalf("got %d areas, want 4 (cafe, weigh station and private parking skipped)", len(areas))
	}

	s := areas[0]
	if s.ID != "osm:way/11" || s.Kind != domain.RestAreaServices || s.Name != "Opet Dinlenme Tesisi" ||
		s.Location.Lat != 41.0 || !*s.Toilets || !*s.Restaurant || s.Shower != nil || s.Country != "TR" {
		t.Errorf("services: %+v", s)
	}
	if areas[1].Kind != domain.RestAreaRest || areas[1].Name != "" {
		t.Errorf("rest area: %+v", areas[1])
	}
	p := areas[2]
	if p.Kind != domain.RestAreaTruckParking || *p.Fee || !*p.Supervised || *p.HGVCapacity != 40 || !p.SuitsDailyRest() {
		t.Errorf("truck parking: %+v", p)
	}
	f := areas[3]
	if f.Kind != domain.RestAreaTruckFuel || f.Name != "Shell" || f.SuitsDailyRest() {
		t.Errorf("fuel: %+v", f)
	}
}

func TestParseBorderControls(t *testing.T) {
	const data = `{"elements":[
 {"type":"node","id":1,"lat":41.5178,"lon":41.5475,"tags":{"barrier":"border_control","name":"Sarp Gümrük Kapısı"}},
 {"type":"way","id":2,"center":{"lat":41.72,"lon":26.36},"tags":{"barrier":"border_control"}},
 {"type":"node","id":3,"lat":41.0,"lon":40.0,"tags":{"barrier":"gate"}}
]}`
	pts, err := ParseBorderControls(strings.NewReader(data), "TR")
	if err != nil {
		t.Fatal(err)
	}
	if len(pts) != 2 || pts[0].ID != "osm:node/1" || pts[0].Name != "Sarp Gümrük Kapısı" || pts[1].Location.Lng != 26.36 {
		t.Fatalf("points = %+v", pts)
	}
}
