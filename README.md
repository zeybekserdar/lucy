# Handoff: Lucy — kişisel uptime izleme

## Genel bakış
Lucy, ev sunucusunda çalışan, tek kullanıcılı bir uptime izleme web uygulaması. HTTP(S) kontrolleri yapar, SSL sertifikası ve alan adı süresini izler, sorun olduğunda **ntfy** ile telefona bildirim gönderir. Kullanıcı uygulamayı çoğunlukla birkaç saniyeliğine açıp “her şey yolunda mı?” sorusuna cevap alır; arayüz bu yüzden sorun yokken sakin ve neredeyse boş, sorun varken dikkat doğrudan sorunlu monitöre gider.

Kapsam dışı: kullanıcı/rol yönetimi, açık durum sayfası, bakım penceresi, ntfy dışı bildirim kanalları, TCP/Ping/DNS/heartbeat monitörleri.

## Tasarım dosyaları hakkında
`designs/` klasöründeki dosyalar **HTML ile hazırlanmış tasarım referanslarıdır**: hedeflenen görünüm ve davranışı gösteren prototiplerdir, doğrudan kopyalanacak üretim kodu değildir. Görev, bu tasarımları **React** ile (proje henüz yoksa Vite + React + TypeScript önerilir) uygulamanın kendi kalıplarıyla yeniden kurmaktır. `.dc.html` dosyaları tarayıcıda doğrudan açılır (`support.js` aynı klasörde olmalı); `Lucy.dc.html` tüm ekran ve durumların bulunduğu ana tuvaldir, `Lucy Sistem.dc.html` token ve bileşen sayfasıdır.

## Doğruluk düzeyi
**Yüksek doğruluk (hi-fi).** Renkler, tipografi, boşluklar, köşe yarıçapları ve etkileşimler nihaidir. `tokens.css` doğrudan kullanılabilir. Arayüz metinleri Türkçe örnek metinlerdir; İngilizce çeviri i18n dosyasında yapılmalıdır.

## Seçilen yönler
- **Ana ekran:** `Lucy.dc.html` → **2a “Odak kartı”** (`LucyHome2.dc.html`, `variant="focus"`). 1a–1h ve 2b/2c alternatif olarak kaldı; boş durum (1h) ve “internet kesik” (1g) durumlarının metin/yerleşimi 1. turdan alınmalı.
- **Logo:** **3c “Oturan Lucy · profil”** (`assets/lucy-logo.svg`).
- **Detay:** masaüstünde sağdan panel (≥ 900 px), telefonda ayrı sayfa.

---

## Ekranlar

Ortak kabuk: zemin `--bg`, yazı `--font-sans` 15/1.4, `font-variant-numeric: tabular-nums`. İçerik `width: calc(100% - 32px)` ve ortalı; en büyük genişlikler: ana ekran 880, detay 720, ayarlar 680, form 640 px. Başlık boyutları kapsayıcı genişliğine göre `clamp()` + `cqi` ile küçülür (kök: `container-type: inline-size`).

### Üst çubuk (tüm ana sayfalarda)
- Yükseklik ≈ 70 (padding 18px 0). Solda logo 22 px + “Lucy” 17/650, −0.01em, gap 8.
- Sağda: **Monitör ekle** (birincil, pill, 34 px, + ikonu 14 px; telefonda yalnızca ikon ve `aria-label`), **Ayarlar** (hayalet düğme).

### 1. Giriş (`LucyLogin.dc.html`)
- Dikey ve yatay ortalı sütun, max 340 px. Logo 64 px → mb 24. Başlık 28/650. Açıklama 15, `--ink2`, mb 28.
- Alan: etiket 13/550, input 44 px, radius 12, kenar `--sep-strong`, 17 px, letter-spacing .1em. Odakta kenar `--focus` + 3 px halka (`color-mix(--focus 30%)`).
- Birincil düğme 44 px, radius 12, 16/600, tam genişlik.
- **Hâller:**
  - `login`: “Lucy” / “Devam etmek için şifreni gir.” / “Giriş yap”.
  - `setup` (ilk açılış, hiç şifre yokken): “Lucy’ye hoş geldin”, “Yeni şifre” + “Şifre (tekrar)”, altında canlı doğrulama satırı (onay ikonu + “En az 12 karakter · iki alan eşleşiyor”), düğme “Şifreyi belirle ve başla”.
  - `error`: input kenarı `--down` + halka, altında `role="alert"` ikonlu “Şifre yanlış. Tekrar dene.”
- Alt not 13, `--ink2`: `lucy reset-password` komutu ile sıfırlama.

### 2. Ana ekran — 2a Odak kartı (`LucyHome2.dc.html`)
1. **Durum cümlesi** (hero): 30 px durum ikonu + h1 `clamp(24px,4.2cqi,34px)`/650/−0.02em, “ · meta” kısmı `--ink2`/500. Altında 15 px `--ink2` alt satır. Padding üst `clamp(28px,7cqi,56px)`. `aria-live="polite"`.
   - Her şey yolunda: ✓ (dolu `--ok`) “Tümü yolunda · 7 monitör” / “Son kontrol 12 saniye önce”.
   - Kesinti: ✕ (dolu `--down`) “api.app kesintide · 14 dk” / “Diğer 6 monitör çalışıyor · son kontrol 8 sn önce”.
   - İnternet kesik: ? (dolu `--unk`) “Ev interneti kesik · 6 dk” / “Dış monitörler bilinmiyor sayılıyor ve kesintiye yazılmıyor. Yerel monitörler çalışıyor.”
2. **Odak kartı** (yalnızca sorun varsa): yüzey, radius 18, mb 32.
   - Her kesinti için bir blok: zemin `--down-tint`, padding `clamp(16px,3cqi,24px)`, gap 14. Satır: ikon 24 + ad 20/650 + “Kesinti” rozeti (dolu `--down`, `--on-status`, 12/600) + sağda süre 20/600 `--down-ink`. Açıklama (`--ink2`): “HTTP 503 Service Unavailable · son kontrol 8 sn önce · ntfy bildirimi 14:02’de gönderildi”. **Son 24 saat** şeridi: 48 yarım saatlik çubuk, 28 px yükseklik, gap 2, altta “24 sa önce / Şimdi” 12 px. Düğmeler: “Ayrıntılar” (birincil), “Şimdi kontrol et” (ikincil).
   - Uyarı/bilinmiyor monitörleri: tek satır (ikon 20 + **ad** — metin `--warn-ink`/`--unk-ink` + ›), üstte `--sep` çizgisi.
3. **Liste başlığı** 13/600 `--ink2`: “Çalışıyor · 5” (sorun yoksa “Monitörler · 7”).
4. **Kompakt liste**: yüzey kartı, radius 16. Satır grid `20px | 1fr | 178px | 64px | 56px`, gap 14, padding 11px 20px; ✓ ikonu 18 (dolgusuz, `--ok-ink`); ad 15/550 + (varsa) not 13 (“Dün 4 sa 12 dk kesinti”, “Alan adı süresi bilinmiyor” → `--unk-ink`); 30 çubuk 4×16 px gap 2; uptime 500; yanıt `--ink2`.
   - **Telefon (<600 px)**: satır iki sıra — (ikon + ad … uptime) / (30 çubuk esnek genişlik … yanıt). Hiçbir şey yatay taşmaz; ad tek satır + üç nokta, diğer metinler sarılır.
5. **Açıklama satırı** (legend) 12 px: Çalışıyor, Kısmi kesinti, Kesinti, Bilinmiyor, “Veri yok · Lucy kapalı”.
- Sıralama: kesinti → uyarı → bilinmiyor → çalışıyor; kendi içinde ada göre.
- **Boş durum (ilk kullanım, 1h):** üst çubukta “Monitör ekle” gizli; ortada logo 56 px, “Henüz izlenen bir şey yok”, açıklama, “İlk monitörü ekle” (birincil 40 px) + “YAML’dan içe aktar” (ikincil).

### 3. Monitör detayı (`LucyDetail.dc.html`)
- Masaüstü: ana ekranın üstünde, sağa yaslı **520 px panel** (`role="dialog"`), arkada scrim (açık `oklch(0 0 0/.22)`, koyu `/.5`), sol kenar `--sep` + gölge. Esc ve scrim tıklaması kapatır, odak panelde kilitli, kapanınca tetikleyen satıra döner. Telefon: tam sayfa, solda “‹ Monitörler”.
- Sıra (gap 16):
  1. Üst sıra: (sayfa: geri) … “Düzenle” (hayalet) + (panel: × 34 px yuvarlak).
  2. Ad `clamp(24px,5cqi,30px)`/650 + URL mono 13 `--ink2`. Durum satırı: ikon 18 + durum metni 600 (renk = durum ink) + meta 14 `--ink2` + sağda “Şimdi kontrol et” (ikincil 32 px).
  3. **Uptime üçlüsü**: tek kart, 3 eşit sütun, `--sep` dikey ayırıcı; etiket 12 `--ink2` (“Son 24 saat / Son 7 gün / Son 30 gün”), değer `clamp(18px,4cqi,22px)`/600 (kesintideyse 24 saat değeri `--down-ink`).
  4. **Yanıt süresi grafiği**: başlık 15/600 + “ortalama N ms” + segment (24 sa / 7 gün / 30 gün; varsayılan 7 gün). Çizim alanı 170 px; y ekseni 0 / orta / max (11 px), kesikli yatay ızgara `--sep`, alt çizgi `--sep-strong`. Çizgi `--ink` 1.5 px, alan dolgusu `--ink` %6.
     - **Kesinti** bandı: `color-mix(in oklch, var(--down) 16%, transparent)` + üstte 2 px `--down` + “Kesinti” etiketi; çizgi bu aralıkta kesilir.
     - **Veri yok** bandı (Lucy kapalıyken): renksiz, `repeating-linear-gradient(135deg, var(--sep-strong) 0 1.5px, transparent 1.5px 6px)` + “Veri yok” etiketi; çizgi kesilir. Kesintiden **renk, desen ve metinle** ayrılır ve uptime hesabına girmez.
     - Eksen etiketleri: 24 sa → saatler, 7 gün → gün kısaltmaları (Cmt, Paz, …, Bugün), 30 gün → tarihler. Lejant grafiğin altında.
  5. **Son kontrol**: durum kodu mono 14/600 (durum rengi) + meta; her koşul bir satır: ✓/✕ glif + koşul adı + “Beklenen: …” + sonuç (renkli, 550).
  6. **Sertifika ve alan adı**: iki satır; sağda ikon + kalan gün. Durumlar: ≥ 15 gün `ok`, ≤ 14 gün `warn` (“7 gün”), bilinmiyor `unk` (“Bilinmiyor” + açıklama: “.home yerel bir alan adı; WHOIS kaydı olmadığı için süre öğrenilemiyor.”), süresi dolmuş `down`.
  7. **Olaylar · son 30 gün**: dikey zaman çizelgesi. Kesinti: dolu `--down` nokta, “Kesinti” `--down-ink` + süre, altında zaman aralığı · neden. Veri yok: kesikli boş halka, `--ink2`, “Lucy kapalıydı, kesinti sayılmadı”. Süren kesinti “14 dk · sürüyor”.

### 4. Monitör ekle / düzenle (`LucyEditor.dc.html`)
- Sayfa (masaüstünde 640 px sütun; ileride modal da olabilir). “‹ Vazgeç”, başlık “Yeni monitör” / “api.app’i düzenle”.
- **Temel kart** (her zaman görünür): Adres (mono, zorunlu), Ad (isteğe bağlı; placeholder = host), Kontrol aralığı (segment: 30 sn / 1 dk / 5 dk / 15 dk, varsayılan 1 dk), Alarm eşiği (− N + stepper, 1–10, varsayılan 3; yanında “art arda başarısız kontrolden sonra bildir · yaklaşık N dakika” canlı hesaplanır).
- **Gelişmiş ayarlar**: varsayılan **kapalı** bir açılır düğme (tam genişlik kart, `aria-expanded`, ok 180 ms döner). Kapalıyken özet: “Yöntem, başlıklar, gövde, zaman aşımı ve koşullar. Varsayılan: GET, 10 sn, durum kodu 200–299.”; düzenlemede gerçek değerlerin özeti.
  - **İstek kartı**: Yöntem segmenti (GET/POST/PUT/HEAD, mono), Zaman aşımı (sn), Başlıklar (ad/değer satırları + kaldır, “+ Başlık ekle”), Gövde (mono textarea; GET/HEAD’de placeholder “GET isteklerinde gövde gönderilmez.”).
  - **Koşullar kartı**: “Hepsi sağlanırsa monitör çalışıyor sayılır.” Koşul türleri: Durum kodu (arasında/eşittir), Yanıt süresi (altında), Gövde içeriği (içerir/içermez), JSON yolu (yol + eşittir/içerir + değer). “+ Koşul ekle”.
- **Doğrulama** (gönderimde ve alan bırakıldığında):
  - Adres `http://` veya `https://` ile başlamalı → hata + “https://… kullan” öneri düğmesi.
  - Zaman aşımı < kontrol aralığı → “Kontrol aralığından (60 sn) kısa olmalı.”
  - JSON yolu `$` ile başlamalı → “JSON yolu $ ile başlamalı, örneğin $.status”.
  - Hata stili: kenar `--down` + 3 px halka (`color-mix(--down 22%)`), altta 14 px ikon + 13 px `--down-ink` metin, `aria-invalid`, `aria-describedby`. Formun üstünde özet uyarısı (`role="alert"`, `--down-tint`): “2 alanı düzeltmen gerekiyor.” Gelişmiş bölümde hata varsa bölüm otomatik açılır.
- **Şimdi test et**: kaydetmeden tek kontrol çalıştırır. Sonuç kartı formun altında (`aria-live`): başarısızsa başlık zemini `--down-tint`, “Test başarısız · 3 koşuldan 2’si sağlanmadı · saat”, istek özeti mono (“GET https://api.app/health → 503 Service Unavailable · 41 ms · text/html”), her koşul ✓/✕ + açıklama. Başarılıysa aynı yapı, ✓ ve `--ok` tonları. Test sürerken düğme “Test ediliyor…” ve devre dışı.
- Alt sıra: “Şimdi test et” (ikincil 40) … “İptal” (hayalet) + “Monitörü ekle” / “Değişiklikleri kaydet” (birincil).

### 5. Ayarlar (`LucySettings.dc.html`)
Gruplu listeler; grup başlığı 13/600 `--ink2`. Satır: etiket (flex 1 1 150) + kontrol (flex 2 1 260), dar ekranda alt alta sarılır.
- **Bildirimler · ntfy**: Sunucu adresi, Konu, Erişim token’ı (maskeli; “Göster/Gizle”). “Test bildirimi gönder” → sonuç yanında: ✓ “Gönderildi · ntfy 200 döndü, 14:20” veya ✕ hata metni. Alt açıklama: hangi olaylarda bildirim gittiği.
- **Görünüm**: Dil (Türkçe / English), Tema (Sistem / Açık / Koyu) segmentleri; değişiklik anında uygulanır.
- **Yedekleme**: YAML dışa aktar (token dahil değil), YAML’dan içe aktar (dosya seç → önizleme → onay; aynı adresliler güncellenir).
- **Güvenlik**: Şifre (“Şifreyi değiştir” → mevcut / yeni / tekrar alanları açılır), Oturumu kapat (`--down-ink` metin düğmesi).
- Altta 12 px sürüm satırı.

---

## Etkileşim ve davranış
- **Hover**: satır ve hayalet düğmede zemin `--hover`, 150 ms.
- **Odak**: tüm etkileşimli öğelerde `:focus-visible` → 2 px `--focus` outline, 2 px ofset (liste satırlarında −2 px içe).
- **Panel**: 220 ms sağdan kayma + scrim opaklığı, `cubic-bezier(.2,.8,.2,1)`. `prefers-reduced-motion` ile yalnız opaklık.
- **Uptime çubukları**: her gün bir çubuk (en sağ = bugün); `title` / tooltip “N gün önce · Durum”; kapsayıcı `role="img"` + özet `aria-label`.
- **İnternet kesik algısı**: Lucy dış bağlantıyı kaybettiğinde dış alan adlarına giden kontroller `unknown` olur, kesinti sayılmaz, bildirim kuyruğa alınır; `.home`/yerel adresler kontrol edilmeye devam eder.
- **Veri yok**: Lucy çalışmadığı aralıklar `no_data`; uptime yüzdesinin paydasına girmez.
- **Canlı güncelleme**: ana ekran ve detay 5–10 sn’de bir yenilenir (SSE veya polling); hero cümlesi `aria-live`.
- **Duyarlılık**: 375 px’de hiçbir satır yatay taşmaz. Panel → sayfa eşiği 900 px; liste satırı iki sıraya geçiş eşiği 600 px.
- **i18n**: TR/EN. Türkçe metinler daha uzundur; tüm metin kapları sarılır (`text-wrap: pretty`), sabit yükseklik yok. Sayı biçimi yerel ayara göre (`99,20%` TR / `99.20%` EN), süreler `Intl.RelativeTimeFormat` ile.

## Durum yönetimi (öneri)
- `auth`: `{ status: 'needsSetup' | 'loggedOut' | 'loggedIn', error? }`
- `monitors`: `Monitor[]` — `{ id, name, url, method, headers, body, timeoutSec, intervalSec, alertThreshold, conditions[], status: 'up'|'warn'|'down'|'unknown', since, lastCheck, responseMs, uptime: {d1,d7,d30}, days: DayState[30], ssl: {daysLeft|null, expiresAt, issuer}, domain: {daysLeft|null, expiresAt, reason?} }`
- `DayState = 'ok' | 'partial' | 'down' | 'unknown' | 'no_data'`
- `network`: `{ online: boolean, offlineSince? }` → hero ve satır durumları.
- `ui`: `selectedMonitorId` (panel/sayfa route’u `/m/:id`), `chartRange: '24h'|'7d'|'30d'`, `editorAdvancedOpen`, `testResult`.
- `settings`: `{ ntfy: {server, topic, token}, language, theme }`.
- Uç noktalar (öneri): `GET /api/monitors`, `GET /api/monitors/:id?range=7d` (seri + olaylar), `POST /api/monitors`, `PUT /api/monitors/:id`, `POST /api/monitors/test`, `POST /api/monitors/:id/check`, `POST /api/notify/test`, `GET/POST /api/backup.yaml`, `POST /api/auth/{setup,login,logout,password}`.

## Bileşenler (React)
`StatusIcon`, `StatusPill`, `UptimeBars`, `Button` (primary/secondary/ghost/icon), `SegmentedControl`, `TextField` + `FieldError`, `Stepper`, `Disclosure`, `MonitorRow` (desktop/mobile), `FocusCard`, `HeroStatus`, `ResponseChart` (bantlı), `UptimeTriad`, `ExpiryRow`, `EventTimeline`, `CheckResult`, `SettingsGroup` / `SettingsRow`, `Panel` (dialog), `Logo`. Örnekler: `designs/Lucy Sistem.dc.html`.

## Tasarım token’ları
Tamamı `tokens.css` içinde (açık/koyu, `[data-theme]` + `prefers-color-scheme`). Özet:
- **Renk**: zemin/metin (`--bg --surface --hover --sep --sep-strong --ink --ink2`), durum (`--ok/-ink/-tint`, `--warn/-ink/-tint`, `--on-warn`, `--down/-ink/-tint`, `--unk/-ink`, `--none`, `--on-status`), etkileşim (`--accent --accent-ink --focus`). Değerler oklch; metin token’ları zeminlerinde AA.
- **Tipografi**: display 34, title 30, stat 22, headline 20, brand 17, body 15, control 14, meta 13, caption 12, axis 11; ağırlıklar 400/500/550/600/650; sistem yığını + mono.
- **Boşluk**: 2, 4, 6, 8, 12, 16, 20, 24, 32, 48, 64 px.
- **Köşe**: bar 1.5, sm 8, md 10, lg 12, card 16, xl 18, pill 999.
- **Gölge**: açık `0 1px 2px oklch(0 0 0/.04), 0 6px 24px oklch(0 0 0/.05)`; koyu `0 0 0 1px oklch(1 0 0/.05)`.
- **Hareket**: 150 / 180 / 220 ms.

## Varlıklar
- `assets/lucy-logo.svg` — 3c oturan kedi, `currentColor`, 24×24 viewBox. 16 px’e kadar okunur.
- `assets/favicon.svg` — aynı şekil, sistem temasına göre renk değiştirir.
- İkon glifleri (✓ ✕ ! ?, ok, +) basit SVG path’ler; `designs/*` içinde birebir mevcut. Ek ikon kütüphanesi gerekmez.

## Dosyalar
- `designs/Lucy.dc.html` — tüm ekran ve durumlar (tur 1–4).
- `designs/Lucy Sistem.dc.html` — token ve bileşen sayfası.
- `designs/LucyHome2.dc.html` — ana ekran (seçilen: `variant="focus"`), `LucyHome.dc.html` — ilk sürüm (boş durum ve internet kesik referansı).
- `designs/LucyLogin.dc.html`, `LucyDetail.dc.html`, `LucyEditor.dc.html`, `LucySettings.dc.html`.
- `designs/support.js` — yalnızca HTML önizleme çalışma zamanı; uygulamaya taşınmaz.
- `tokens.css` — üretimde kullanılabilir token dosyası.
