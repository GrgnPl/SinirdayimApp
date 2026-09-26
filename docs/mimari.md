# Mimari

```
sinir-bekleme/
├── backend/          Go – veri toplama, tahmin, REST API
│   ├── cmd/api/                  composition root (bağımlılıklar burada bağlanır)
│   └── internal/
│       ├── domain/               saf iş tipleri (Crossing, Snapshot, WaitEstimate)
│       ├── port/                 arayüzler: SnapshotSource, SnapshotRepository, CrossingCatalog, Estimator
│       ├── usecase/              Ingestor, StatusService, TripService (rota planı)
│       ├── estimator/            tahmin modelleri (şimdilik Queue: kuyruk / saatlik geçiş)
│       ├── tacho/                takograf kuralları (AB 561/2006, AETR) – saf mantık
│       ├── geo/                  mesafe, polyline çözümleme
│       ├── catalog/              kapı referans listesi
│       └── adapter/
│           ├── source/und/       UND scraper
│           ├── repository/memory bellek içi depo (Postgres adapter'ı gelecek)
│           ├── routing/valhalla  tır rotası (varsayılan: FOSSGIS public sunucusu)
│           ├── geocoding/photon  yer arama
│           └── httpapi/          JSON REST
└── mobile/           Kotlin Multiplatform + Compose (Android / iOS)
    ├── composeApp/src/commonMain   domain / data / presentation / di – tüm UI ortak
    └── iosApp/                     SwiftUI kabuğu (Config.xcconfig, Local.xcconfig ile API adresi)
```

Bağımlılık yönü: `adapter → usecase → port → domain`. `domain` hiçbir şeye bağımlı değildir.

## Yeni veri kaynağı eklemek

1. `internal/adapter/source/<isim>/` altında `port.SnapshotSource` implement et (`ID`, `Interval`, `Fetch`).
2. `cmd/api/main.go` içinde `sources` listesine ekle.

Kaynaklar birbirinden bağımsızdır; biri hata verirse diğerleri çalışmaya devam eder.
Kullanıcı bildirimi, GPS ölçümü, kamera sayımı ve Nakordoni API de aynı arayüzle eklenecek.

## Tahmin

`estimator.Queue`:
- araç = kuyruk km × 55 × şerit + TIR parkındaki araç
- bekleme = araç ÷ (günlük çıkış ÷ 24)
- Kaynak bekleme süresini doğrudan veriyorsa o kullanılır (`reported`, güven: yüksek).
- 12 saatten eski veri → güven: düşük.

İleride geçmiş veriyle eğitilmiş bir model, aynı `port.Estimator` arayüzünün arkasına konacak.

## Rota planlama (takograf)

`TripService.Plan`:
1. **Aday kapılar**: iki tarafı başlangıç ve varışı ayıran, sapması ≤ %50 olan en fazla 4 kapı.
2. Doğrudan rota + her aday kapıdan "through" noktasıyla rota paralel hesaplanır.
   (Kapı yolları OSM'de tırlar için çoğu zaman "yalnızca varış" etiketli; zorlamadan Valhalla kapıdan geçmiyor.)
3. Her rotada kapılar geometriden bulunur (≤ 3 km), yön `SideRefs` ile belirlenir, bekleme tahmini eklenir.
4. `tacho.Schedule` molaları yerleştirir:
   - 4 sa 30 dk sürüş → 45 dk mola
   - günlük 9 sa (haftada 2 gün 10 sa) → 11 sa dinlenme
   - son dinlenmeden 13 sa sonra görev süresi dolar → 11 sa dinlenme
   - sınırda ≥ 45 dk bekleme mola, ≥ 11 sa bekleme günlük dinlenme sayılır
5. En erken varan plan seçilir; bekleme verisi eksik planlar geride sıralanır ve işaretlenir.

Henüz yok: bölünmüş mola (15+30), 9 saatlik kısaltılmış dinlenme, haftalık limitler (56/90 sa),
haftalık dinlenme, dinlenme tesislerine yerleştirme.

## API

| Uç nokta | Açıklama |
|---|---|
| `GET /v1/crossings` | Tüm kapılar + ihracat/ithalat tahmini |
| `GET /v1/crossings/{id}?hours=48` | Tek kapı + ham geçmiş (grafik için) |
| `POST /v1/trips/plan` | Rota + takograf molaları + sınır beklemeleri + alternatifler |
| `GET /v1/places?q=` | Yer arama |
| `GET /healthz` | Sağlık kontrolü |

## Çalıştırma

```bash
cd backend && go run ./cmd/api
```

Ortam değişkenleri: `PORT` (8080), `VALHALLA_URL`, `PHOTON_URL`.
