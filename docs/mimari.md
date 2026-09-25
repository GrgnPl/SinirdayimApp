# Mimari

```
sinir-bekleme/
├── backend/          Go – veri toplama, tahmin, REST API
│   ├── cmd/api/                  composition root (bağımlılıklar burada bağlanır)
│   └── internal/
│       ├── domain/               saf iş tipleri (Crossing, Snapshot, WaitEstimate)
│       ├── port/                 arayüzler: SnapshotSource, SnapshotRepository, CrossingCatalog, Estimator
│       ├── usecase/              Ingestor (kaynakları periyodik çeker), StatusService (sorgular)
│       ├── estimator/            tahmin modelleri (şimdilik Queue: kuyruk / saatlik geçiş)
│       ├── catalog/              kapı referans listesi
│       └── adapter/
│           ├── source/und/       UND scraper
│           ├── repository/memory bellek içi depo (Postgres adapter'ı gelecek)
│           └── httpapi/          JSON REST
└── mobile/           Kotlin Multiplatform + Compose (Android / iOS) – sıradaki adım
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

## API

| Uç nokta | Açıklama |
|---|---|
| `GET /v1/crossings` | Tüm kapılar + ihracat/ithalat tahmini |
| `GET /v1/crossings/{id}?hours=48` | Tek kapı + ham geçmiş (grafik için) |
| `GET /healthz` | Sağlık kontrolü |

## Çalıştırma

```bash
cd backend && go run ./cmd/api
```
