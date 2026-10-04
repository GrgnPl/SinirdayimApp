package und

import (
	"strings"
	"testing"
	"time"

	"github.com/GrgnPl/SinirdayimApp/backend/internal/domain"
)

const fixture = `<html><body>
<table class="table table-bordered"><thead><tr><th>25-09-2026 10:36:42</th><th>İHRACAT</th><th>TIR PARKI</th><th>ÇIKIŞ SAYISI</th><th>İTHALAT</th><th>RO-LA BEKLEMELERİ</th></tr></thead>
<tbody>
<tr><td class="first-item">Kapıkule (TR) - Kapitan Andreevo (BG)</td><td>1 KM</td><td>1.797 ARAÇ</td><td>904 ARAÇ</td><td>- KM</td><td>- ARAÇ</td></tr>
<tr><td class="first-item">Sarp (TR) - Sarpi (GE)</td><td>16 KM</td><td>- ARAÇ</td><td>642 ARAÇ</td><td>19 KM</td><td>- ARAÇ</td></tr>
<tr><td class="first-item">Maribor (SLO) - Wels (AT)</td><td>- KM</td><td>- ARAÇ</td><td>- ARAÇ</td><td>- KM</td><td>- ARAÇ</td></tr>
<tr><td class="first-item">Bilinmeyen (XX) - Yok (YY)</td><td>3 KM</td><td>- ARAÇ</td><td>- ARAÇ</td><td>- KM</td><td>- ARAÇ</td></tr>
</tbody></table>
<table><tr><th>Başka tablo</th></tr><tr><td>a</td><td>b</td><td>c</td><td>d</td><td>e</td></tr></table>
</body></html>`

func TestParse(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Istanbul")
	snaps, err := Parse(strings.NewReader(fixture), loc)
	if err != nil {
		t.Fatal(err)
	}
	// Kapıkule export, Sarp export, Sarp import. Maribor has no data, unknown row skipped.
	if len(snaps) != 3 {
		t.Fatalf("got %d snapshots, want 3: %+v", len(snaps), snaps)
	}

	wantAt := time.Date(2026, 9, 25, 10, 36, 42, 0, loc)
	k := snaps[0]
	if k.CrossingID != "tr-bg-kapikule" || k.Direction != domain.DirectionExport || !k.ObservedAt.Equal(wantAt) {
		t.Errorf("kapikule: %+v", k)
	}
	if *k.QueueKm != 1 || *k.ParkedVehicles != 1797 || *k.DailyThroughput != 904 {
		t.Errorf("kapikule values: km=%v park=%v thr=%v", *k.QueueKm, *k.ParkedVehicles, *k.DailyThroughput)
	}

	se, si := snaps[1], snaps[2]
	if se.CrossingID != "tr-ge-sarp" || *se.QueueKm != 16 || se.ParkedVehicles != nil || *se.DailyThroughput != 642 {
		t.Errorf("sarp export: %+v", se)
	}
	if si.Direction != domain.DirectionImport || *si.QueueKm != 19 || si.DailyThroughput != nil {
		t.Errorf("sarp import: %+v", si)
	}
}
