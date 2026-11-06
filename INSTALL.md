# Lucy kurulumu

Lucy tek bir Docker container'ı olarak çalışır. Veriler `lucy-data` volume'ünde tutulur.

## Kurulum

Bu klasörü sunucuya kopyala, sonra:

```bash
cp .env.example .env
```

`.env` içine bir ntfy konusu yaz. ntfy.sh üzerindeki konular herkese açıktır, bu yüzden tahmin edilemez bir ad kullan:

```bash
openssl rand -hex 12
```

`.env` yalnızca ilk açılışta okunur, sonrasında bildirim ayarları Ayarlar ekranından değişir. Telefondaki ntfy uygulamasında aynı konuya abone ol, sonra Lucy'yi başlat:

```bash
docker compose up -d --build
```

`http://sunucu-adresi:8080` adresini hemen aç ve şifreni belirle. Şifre belirlenene kadar sayfayı ilk açan kişi şifreyi belirleyebilir.

## Güncelleme

```bash
docker compose up -d --build
```

Veriler korunur, veritabanı kendiliğinden güncellenir.

## Yedekleme

Monitörlerin ve bildirim ayarlarının yedeği için Ayarlar ekranındaki **Dışa aktar** yeterlidir. Dosyada ntfy token'ı bulunmaz, monitör başlıkları ise açık yazılır.

Geçmişi de içeren tam yedek için veritabanını kopyala.

Veritabanı çalışırken kopyalanırsa bozuk yedek çıkabilir, bu yüzden önce durdur:

```bash
docker compose stop
```

```bash
docker run --rm -v lucy_lucy-data:/data:ro -v "$PWD":/backup alpine tar czf /backup/lucy-yedek.tgz -C /data .
```

```bash
docker compose start
```

Volume adının başındaki `lucy_`, klasör adından gelir. Emin değilsen `docker volume ls` ile bak.

## Şifreyi unutursan

```bash
docker exec -it lucy lucy reset-password
```

## Bilmen gerekenler

- Lucy HTTPS sunmaz. Ev dışından erişeceksen Tailscale veya HTTPS veren bir ters proxy kullan.
- Lucy izlediği uygulamalarla aynı sunucuda çalışmamalı. Sunucu çökerse Lucy de çöker ve haber veremez.
- Lucy kapalıyken bildirim gelmez. O aralıklar grafikte "veri yok" olarak görünür.
