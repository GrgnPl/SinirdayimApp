# Veri Kaynakları

Tüm veriler canlı kaynaklardan periyodik olarak çekilir; statik veri yok.

## Türkiye

| Kaynak | Kapsam | Veri | Sıklık | Erişim |
|---|---|---|---|---|
| UND – Sınır Kapıları Yoğunluk Durumu<br>https://www.und.org.tr/sinir-kapilari-yogunluk-durumu | 9 kapı: Sarp, Kapıkule, Hamzabeyli, İpsala, Gürbulak, Habur, … | İhracat kuyruğu (km + araç), TIR park, çıkış/ithalat araç sayısı, RO-LA | Günde ~1-2 snapshot | HTML tablo → scraping |
| Ticaret Bakanlığı – Anlık Saha Yoğunluk<br>https://uygulamalar.gumruk.gov.tr/websahaozet/ | Hamzabeyli, Kapıkule, İpsala, Pazarkule | Bekleyen araç sayısı (giriş/çıkış) | Anlık | HTML → scraping (TR dışından erişim sorunlu olabilir, DNS çözülmedi) |
| tirtakip.com | Öncüpınar | Saha giriş sıra takibi | Anlık | HTML → scraping |

## Avrupa

| Kaynak | Kapsam | Veri | Erişim |
|---|---|---|---|
| Nakordoni API<br>https://nakordoni.eu/en/developers | 556+ kapı, 39 ülke (UA–AB ağırlıklı) | Kuyruk, tahmini bekleme, veri yaşı, kaynak | REST API, ücretsiz plan 1000 çağrı/gün (atıf zorunlu) |
| GoSwift e-kuyruk<br>https://www.estonianborder.eu/ | EE/LV/LT–RU/BY | Rezervasyonlu kuyruk | Web |
| BorderAlarm<br>https://borderalarm.com/ | AB | Kullanıcı bildirimi | Web |

## Kendi kaynaklarımız

- Kullanıcı bildirimi (kuyruktayım / araç sayısı)
- Uygulama içi GPS: kapı bölgesinde bekleme süresi ölçümü
- İleride: açık kamera görüntülerinden araç sayımı

## Açık sorular

- Gürcistan tarafı (Sarpi) için resmi veri var mı?
- UND ve Bakanlık verisinin kullanım/izin şartları
