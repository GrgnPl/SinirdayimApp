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
│           ├── osm/              Overpass: dinlenme tesisleri ve sınır kontrol noktaları (günlük, disk önbellekli)
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
4. Katalogda olmayan kapılar OSM `barrier=border_control` noktalarından bulunur ve "bekleme verisi yok"
   olarak plana eklenir (iki ülkenin noktası birlikte olmalı ya da adı kapı/gümrük/hudut içermeli).
   Rota kapıya gidip geri dönüyorsa (kapının iki yanı aynı ülkede) o kapı sayılmaz.
5. `tacho.ScheduleWith` molaları yerleştirir (AB 561/2006, AETR):
   - 4 sa 30 dk sürüş → 45 dk mola, ya da 15 + 30 bölünmüş
   - günlük 9 sa (haftada 2 gün 10 sa) sürüş
   - 11 sa günlük dinlenme; iki haftalık dinlenme arasında 3 kez 9 sa (görev penceresi 13 / 15 sa)
   - takvim haftasında 56 sa, iki haftada 90 sa sürüş; dolunca yeni haftaya kadar haftalık dinlenme
   - son haftalık dinlenmeden 6 × 24 sa sonra 45 sa haftalık dinlenme
   - sınırdaki bekleme kapsadığı en uzun dinlenme sayılır (15 dk bölünmüş molanın ilk kısmı, mola, 9 / 11 sa dinlenme)
6. Duraklar gerçek tesislere oturtulur: mola sınırdan önceki son 45 dk, dinlenme son 2 sa sürüş içindeki
   son uygun tesise çekilir (dinlenme için servis alanı / tır parkı). Günün bitmesine 2 saatten az kala
   mola gerekiyorsa ve uygun tesis varsa gün orada bitirilir.
7. Aday rotalar tesissiz karşılaştırılır; en erken varan (bekleme verisi eksiksiz olanlar önce) seçilir ve
   tesislerle yeniden planlanır.

Henüz yok: bölünmüş günlük dinlenme (3 + 9), kısaltılmış haftalık dinlenme (24 sa) ve telafisi,
feribot / tren, kapıya varış saatine göre bekleme tahmini.

## Yol haritası

- Kapı randevu sistemi (resmi e-kuyruk entegrasyonu / tır parkı rezervasyonu) – kapsam netleşecek
- Tesis bildirimleri: doluluk, güvenlik, fiyat
- PostgreSQL, kendi Valhalla / Overpass / harita sunucuları

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

Ortam değişkenleri: `PORT` (8080), `VALHALLA_URL`, `PHOTON_URL`, `OVERPASS_URL`, `OSM_COUNTRIES` (TR,GE,BG,…),
`CACHE_DIR` (varsayılan kullanıcı önbellek klasörü / sinirdayim).
