# 🦀 system-critters

Terminalinde çalışan minik bir **sistem monitörü**. Bilgisayarının CPU, RAM,
disk ve ağ durumunu canlı gösterir; en altta ise küçük ASCII balıkların yüzdüğü
bir **akvaryum** vardır. En güzel yanı: balıklar rastgele yüzmez, bilgisayarının
o anki durumuna göre davranır.

```
🦀 system-critters                         ⏱ 10d 11h · Windows 11 Pro
CPU  ██████████░░░░░░░░░░░░░░░  28.8%   TEMP 54°C
RAM  ████████████████░░░░░░░░░  56.0%   8.6 GiB / 15.2 GiB
DISK █████████████░░░░░░░░░░░░  51.9%   121 GiB / 234 GiB
NET  ↓ 850 KiB/s   ↑ 120 KiB/s
╭─────────────────────────────────────────────────────────────╮
│              ><(((º>                                         │
│  <><                        ><>                              │
│         ><(((º>                          °                   │
│                    <º)))><          °                        │
│       (   (                                    (             │
│▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒│
╰─────────────────────────────────────────────────────────────╯
q ile çık  ·  balık hızı = CPU  ·  sürü = RAM  ·  baloncuk = ağ
```

## Balıklar ne anlatıyor?

Akvaryum aslında bir grafik gibi çalışır:

| Gördüğün şey | Anlamı |
|--------------|--------|
| 🐟 Balıklar hızlı yüzüyor | CPU (işlemci) çok çalışıyor |
| 🐌 Balıklar yavaş / uykulu | Bilgisayar boşta |
| 🐠 Akvaryum kalabalık | RAM (bellek) dolu |
| 🫧 Bol baloncuk | Ağdan çok veri iniyor/gidiyor |
| 🔴 Balıklar kırmızıya döndü | CPU %85'i geçti, sistem zorlanıyor |

## Nasıl çalıştırırım?

Uygulama **tek dosya** olarak gelir; kurulum, yükleme, bağımlılık yoktur.
İndir, çift tıkla ya da terminalden çalıştır. Çıkmak için **`q`** tuşuna bas.

### Yol 1 — Hazır dosyayı indir (en kolay, hiçbir şey kurmadan)

1. Bu sayfanın sağındaki **Releases** bölümüne git.
2. İşletim sistemine uygun dosyayı indir:
   - Windows → `system-critters-windows-amd64.exe`
   - macOS (Apple M1/M2/M3) → `system-critters-macos-arm64`
   - macOS (Intel) → `system-critters-macos-amd64`
   - Linux → `system-critters-linux-amd64`
3. Çalıştır:

   **Windows (PowerShell):**
   ```powershell
   .\system-critters-windows-amd64.exe
   ```
   **macOS / Linux (Terminal):**
   ```bash
   chmod +x ./system-critters-*        # bir kez: çalıştırma izni ver
   ./system-critters-*
   ```

> 💡 macOS "bilinmeyen geliştirici" uyarısı verirse: dosyaya sağ tıkla → **Aç**.

### Yol 2 — Kaynaktan çalıştır (Go kuruluysa)

Bilgisayarında [Go](https://go.dev/dl/) (sürüm 1.24+) varsa:

```bash
git clone https://github.com/<kullanici-adin>/system-critters.git
cd system-critters
go run .
```

## Neler gösteriliyor?

| Ölçüm | Açıklama |
|-------|----------|
| **CPU** | İşlemci kullanımı (%) |
| **RAM** | Kullanılan / toplam bellek |
| **DISK** | Ana diskin doluluğu |
| **NET** | Anlık indirme (↓) ve yükleme (↑) hızı |
| **TEMP** | CPU sıcaklığı — *okunabiliyorsa* |
| **Uptime / OS** | Üst satırda: açık kalma süresi ve işletim sistemi |

> ⚠️ **Sıcaklık neden bazen `N/A`?**
> CPU sıcaklığı her bilgisayarda okunamaz. Linux'ta genelde çalışır; **Windows'ta
> çoğu makinede ekstra sürücü olmadan boş gelir**, macOS'ta yönetici izni ister.
> Okunamadığında uygulama çökmez, sadece `N/A` yazar. Bu normaldir.

## Kısayollar

| Tuş | İşlev |
|-----|-------|
| `q` | çıkış |
| `Esc` veya `Ctrl+C` | çıkış |

## Kendine göre değiştir 🎨

Dosyalar sade ve yorumlu. Hızlıca oynayabileceğin yerler:

- **Balık renkleri:** `aquarium.go` → `fishPalette`
- **Yeni balık türü:** `aquarium.go` → `fishSprite()` içine sağa/sola bakan iki
  ASCII şekil ekle, sonra `addFish()` içindeki `rng.Intn(2)` sayısını artır.
- **Yaratık davranışı:** `aquarium.go` → `Tank.Update()` (CPU/RAM/ağ değerlerine
  göre hız, renk, sayı ayarlanır).
- **Yenilenme hızı:** `model.go` → `frameRate` (animasyon) ve `statsRate` (ölçüm).

## Kendin derlemek istersen

Tüm platformlar için tek komutla binary üretir (`dist/` klasörüne):

```bash
./build.sh      # macOS / Linux
./build.ps1     # Windows (PowerShell)
```

## Proje yapısı

```
main.go        giriş noktası
model.go       ekran düzeni ve güncelleme döngüsü
stats.go       sistem bilgisi okuma (gopsutil)
aquarium.go    akvaryum: balık, baloncuk, yosun
ui.go          göstergeler, renkler, biçimlendirme
smoke_test.go  testler
```

## Kullanılan kütüphaneler

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — terminal arayüzü
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) — renk ve düzen
- [gopsutil](https://github.com/shirou/gopsutil) — sistem bilgisi

## Gizlilik

Uygulama sistem değerlerini **yalnızca okur ve ekranda gösterir**. Hiçbir veriyi
internete göndermez, hiçbir dosyaya kaydetmez.

## Lisans

MIT — bkz. [LICENSE](LICENSE).
