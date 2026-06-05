# WhatsApp Lite

Klien WhatsApp desktop untuk Linux yang **ringan & efisien** — **tanpa membundel Chromium**. UI memakai
**WebView sistem (WebKitGTK)**, bukan browser yang ikut dibundel seperti Electron/WhatsApp Web. Dibangun di
atas [whatsmeow](https://github.com/tulir/whatsmeow) (protokol WhatsApp Web multi-device langsung via
WebSocket). Target jejak RAM **setara app macOS native** dan **3–6× lebih hemat dari WhatsApp Web**.

> Status: **migrasi stack.** Prototipe lama berbasis **Gio (native custom-drawn)** sedang diganti ke
> **web (HTML/CSS/JS) + Wails (shell Go) + WebKitGTK**, supaya UI bisa nyaris pixel-identik dengan WhatsApp
> macOS dan dikerjakan dengan skill web. Engine whatsmeow + storage SQLite tetap dipakai.

Lihat [`PRODUCT-BRIEF.md`](./PRODUCT-BRIEF.md) untuk arah produk, rasional stack, dan strategi optimasi.

---

## ⚠️ Disclaimer (baca dulu)

- Ini aplikasi **TIDAK RESMI** dan **TIDAK berafiliasi** dengan WhatsApp atau Meta.
- Menggunakan protokol WhatsApp Web melalui whatsmeow — ini **melanggar Ketentuan Layanan
  (ToS) WhatsApp**.
- **Nomor yang kamu tautkan BERISIKO DIBANNED oleh Meta.** Gunakan dengan **risiko sendiri**.
- Pertimbangkan memakai nomor cadangan, terutama saat pengembangan/pengujian.
- Disediakan **tanpa jaminan apa pun** (no warranty).

---

## Stack

| Lapisan | Pilihan |
|---|---|
| **Engine (BE)** | Go + whatsmeow + SQLite (`modernc.org/sqlite`, pure-Go) |
| **Shell** | Wails (Go ↔ WebView sistem) |
| **Frontend (FE)** | HTML / CSS / JS (di-embed dalam binary) |
| **Render** | WebKitGTK (WebView sistem, **bukan** Chromium dibundel) |
| **Penyimpanan** | SQLite untuk sesi/kunci/pesan; media sebagai file (bukan di DB) |

- Data lokal: `~/.local/share/whatsapp-lite/` · cache media: `~/.cache/whatsapp-lite/` (XDG).
- Pembeda utama = **arsitektur lean** (virtualized list, local-first, media-as-file, no telemetry) untuk
  menutup overhead WebView. Detail di `PRODUCT-BRIEF.md` §12.3.

## Prasyarat build (Linux)

Butuh **Go**, **WebKitGTK + GTK3**, dan **Wails CLI**. Di Arch/CachyOS:

```sh
sudo pacman -S --needed go webkit2gtk gtk3 pkgconf
go install github.com/wailsapp/wails/v2/cmd/wails@latest   # pastikan $(go env GOPATH)/bin di PATH
```

(Di Debian/Ubuntu padanannya: `golang-go libwebkit2gtk-4.0-dev libgtk-3-dev pkg-config build-essential`.)

Cek kesiapan toolchain:

```sh
wails doctor
```

## Build & jalankan

Tag build **wajib** di Arch/CachyOS:
- `webkit2_41` → pakai WebKitGTK 4.1 (bukan 4.0).
- `netgo` → resolver DNS murni-Go (hindari crash `free(): corrupted unsorted chunks`
  akibat resolver CGo getaddrinfo bentrok dengan runtime C WebKitGTK).

```sh
# mode dev (hot-reload UI di WebView):
wails dev -tags "webkit2_41 netgo"

# build rilis (binary tunggal):
wails build -tags "webkit2_41 netgo"   # hasil di ./build/bin/whatsapp-lite
./build/bin/whatsapp-lite

# CLI debug (engine saja, tanpa UI; binary statis):
CGO_ENABLED=0 go build -o walite-cli ./cmd/walite-cli
./walite-cli
```

Saat pertama dijalankan, **layar QR** muncul di jendela. Scan via:
**WhatsApp di HP → Perangkat Tertaut → Tautkan perangkat.**
Sesi & pesan tersimpan lokal, jadi run berikutnya tidak perlu scan ulang.

Mode verbose (debug log): `WALITE_DEBUG=1 wails dev`

## Bug yang diketahui

- **Daftar baca per-penerima untuk pesan lama tidak tersedia.** Pada modal *Info pesan*,
  bagian "Dibaca oleh / Tersampaikan ke" hanya terisi untuk pesan yang dikirim **setelah**
  fitur ini aktif (tanda terima dikumpulkan secara live). Pesan historis tidak punya data
  ini. **Penyebab:** whatsmeow tidak mengekspos protokol companion↔primary yang dipakai
  WhatsApp resmi untuk menarik log tanda terima dari HP. **Status: tidak bisa diperbaiki**
  di sisi klien (keterbatasan pustaka, bukan bug logika kita). Centang agregat (✓✓) pesan
  lama *sudah* diperbaiki via `WebMessageInfo.Status` dari history sync.

## Lisensi

GPL-3.0 (lihat berkas `LICENSE` — akan ditambahkan). whatsmeow (MIT) kompatibel dipakai di sini.
