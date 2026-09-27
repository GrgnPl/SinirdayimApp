# Sıra kaydı (randevu) – araştırma

Tarih: 2026-09-26

## Mevcut resmi sistemler

### Türkiye – Randevulu Sanal Sıra Sistemi (RSS), Ticaret Bakanlığı
- Adres: rss.ticaret.gov.tr (+ Android / iOS uygulaması, kapıda kiosk). Türkiye dışından / bu ağdan HTTPS erişimi yok.
- Kapıkule: 13.07.2020 pilot, 15.09.2020 24:00'ten itibaren **tüm tır çıkışları RSS ile zorunlu**.
  Hamzabeyli ve Karkamış'a genişletileceği duyurulmuştu; güncel kapı listesi doğrulanamadı.
- İşleyiş (TOBB broşürü):
  - Gümrük işlemlerini bitiren sürücü, gümrük müdürlüğünün tanımladığı zaman dilimlerinden tarih + saat seçer.
  - Randevu, BİLGE'de beyan edilen çıkış gümrüğüne bağlı ("Belgenin çıkış gümrüğü randevu almaya uygun değildir").
  - Karekod ile kapıya bitişik bekleme alanına (TIR parkı) giriş/çıkış.
  - Randevusuz çıkış yok; kioskta yeni randevu alınabilir.
  - Kaçırılan randevu: gümrüğün tanıdığı süre aşılırsa yeni randevu gerekir, **ücret iade edilmez** (sistem ücretli).
  - Geç kalacaksa sürüş sırasında **erteleme** yapılabilir.
  - Yük ve araç özelliğine göre öncelik algoritması; sıra gelince LED tabela + push + SMS.
  - Uygulamada kapıların **doluluk durumu** görülebiliyor.
- Hesap: sürücü için TC kimlik / pasaport + Türk GSM numarası. Filo yöneticisi için kurumsal hesap (YKTS kaydı şart),
  ama her sürücünün yine bireysel hesabı olmalı.
- Dışa açık API: **yok** (yayınlanmış doküman bulunamadı).
- Destek: 444 8 482.

### Gürcistan – TIR parkı elektronik sırası (Gelir İdaresi, rs.ge)
- Sarpi, Vale (Türkgözü karşısı), Kazbegi, Red Bridge, Sadakhlo, Lagodekhi, Guguti, Ninotsminda, Kartsakhi'ye giden
  tüm yük araçları kapıya bitişik lisanslı TIR parklarına girmek zorunda (35 park). Ücret 80 GEL / römork.
- Sıra, parka giriş anında elektronik sisteme alınarak yönetiliyor; **uzaktan randevu yok**, canlı veri / API yok.

### Diğer (koridor dışı, referans)
- GoSwift (Estonya, Letonya, Litvanya, Finlandiya): uzaktan sıra rezervasyonu, öncelikli slot, özel şirket imtiyazı.
- eCherha (Ukrayna): mobil uygulama ile elektronik sıra.
- Her ikisinde de herkese açık bir API dokümanı bulunamadı.

## Sonuç

Kapıda sıraya öncelik ancak resmi sistem verir. Bizim uygulamada alınacak bir "sıra"nın kapıda hükmü olmaz.
Sürücü adına RSS'ten randevu almak da mümkün değil: sürücünün kendi hesabı ve beyanı gerekiyor, API yok.

## Önerilen ürün yaklaşımı

1. **Randevuya göre plan (şimdi yapılabilir)**
   - Rota RSS'li bir kapıdan geçiyorsa plan, takograf molaları dahil kapıya varış saatini hesaplar ve
     "RSS'ten şu saat aralığında randevu al" önerir; RSS uygulamasına / sitesine yönlendirir.
   - Sürücü aldığı randevu saatini uygulamaya girer; plan tersine çalışır: **en geç kalkış saati**, yolda nerede
     mola / dinlenme verileceği ve kapıya randevu penceresinde varış. Kapıdaki bekleme = randevuya kalan süre.
   - Yolda gecikme öngörülürse (ileride GPS ile) "randevunu ertele" uyarısı.
2. **Gürcistan tarafı**: Sarp / Türkgözü rotalarında Gürcistan TIR parkı zorunluluğunu ve ücretini plana ekle.
3. **Resmi entegrasyon (iş geliştirme)**: Ticaret Bakanlığı ile RSS slot doluluğu (okuma) ve sürücü onaylı
   randevu (yazma) için API görüşmesi. Onay gelirse sürücü tek ekrandan randevu alabilir.
4. **Kendi rezervasyonumuz**: resmi sistemi olmayan yerlerde kapı sırası değil, **TIR parkı / güvenli otopark
   rezervasyonu** (işletmelerle anlaşmalı, ticari model).
