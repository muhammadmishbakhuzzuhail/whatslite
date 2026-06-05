# WhatsApp Lite

Klien WhatsApp desktop untuk Linux yang **ringan & efisien** — **tanpa membundel Chromium**. UI memakai
**WebView sistem (WebKitGTK)**, bukan browser yang ikut dibundel seperti Electron/WhatsApp Web. Dibangun di
atas [whatsmeow](https://github.com/tulir/whatsmeow) (protokol WhatsApp Web multi-device langsung via
WebSocket). Target jejak RAM **setara app macOS native** dan **3–6× lebih hemat dari WhatsApp Web**.

> Stack: **web (HTML/CSS/JS + Svelte) + Wails (shell Go) + WebKitGTK**. Engine whatsmeow + storage SQLite
> (pure-Go, FTS5). Media disimpan sbg file (bukan base64 di DB), cache ter-evict, avatar lazy.

Lihat [`PRODUCT-BRIEF.md`](./PRODUCT-BRIEF.md) untuk arah produk dan [`docs/ARCHITECTURE.md`](./docs/ARCHITECTURE.md) untuk arsitektur.

## ✅ Fitur

**Pesan**: teks, @mention, foto/video/dokumen, **voice note** (ogg/opus), **stiker** (+ hapus-background otomatis via ML in-browser), **GIF** (Giphy), **lokasi**, **kontak** (vCard), **polling**, **foto sekali-lihat (view-once)**, **pesan sementara (disappearing)**.

**Aksi pesan**: balas/kutip, balas-pribadi (grup→japri), teruskan (satu & **massal**), reaksi emoji, edit, hapus (untuk-saya / untuk-semua), salin, **bintangi**, **sematkan di chat**, **info pesan** (centang terkirim/sampai/dibaca + daftar baca per-penerima), **terjemah** (auto-detect, 25 bahasa), **pilih banyak** (bulk).

**Chat**: sematkan, bisukan, **arsipkan** (+ panel arsip), tandai belum/sudah dibaca, hapus, **pencarian isi pesan (FTS5)**, **pesan berbintang** (panel).

**Grup**: info + anggota, **admin** (ubah nama/foto, tambah/keluarkan anggota, promote/demote, **tautan undangan**), buat grup, keluar.

**Lainnya**: **Status** (teks + foto/video, viewer tap-through), **Channels** (ikuti/feed/mute), **Komunitas** (sub-grup), **profil** (nama/info), **privasi** (last-seen/foto/status/grup, tanda-baca, **daftar blokir**), notifikasi **desktop + suara**, lightbox media, kunci-aplikasi (PIN), tema terang/gelap, **i18n (ID/EN/ES)**.

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

## Belum diimplementasikan

Bisa dikerjakan (belum):
- **Voting polling** — polling bisa *dibuat & dikirim*, tapi polling masuk belum bisa di-vote / lihat hasil di UI (whatsmeow punya `BuildPollVote`).
- **Pratinjau tautan (link preview)** — URL belum di-render jadi kartu OG.
- **Render kaya untuk lokasi/kontak masuk** — saat ini label teks (📍/👤), belum peta/parse-vCard.
- **Drag-drop / tempel gambar** ke composer; tombol **simpan media** di lightbox.
- **Pencarian dalam satu chat** (navigasi antar-kecocokan); shortcut keyboard.
- **Wallpaper chat**, ekspor chat, pencarian/kategori di emoji picker.

## Keterbatasan (tidak bisa di sisi klien)

- **Daftar baca per-penerima untuk pesan lama** — `Info pesan` hanya terisi untuk pesan
  yang dikirim *setelah* fitur aktif (receipt dikumpulkan live). whatsmeow tak mengekspos
  protokol companion↔primary utk menarik log historis. Centang agregat (✓✓) pesan lama
  *sudah* diperbaiki via `WebMessageInfo.Status` dari history sync.
- **Panggilan suara/video** — whatsmeow tak punya WebRTC.
- **Ganti foto profil sendiri** — tak ada API di whatsmeow (foto *grup* bisa).
- **Paket stiker kurasi Meta & stiker AI** — endpoint tak diekspos / butuh model generatif.
- **Animated sticker** — WebKit tak meng-encode animated-webp (stiker statis saja).

## Lisensi

GPL-3.0 (lihat berkas `LICENSE` — akan ditambahkan). whatsmeow (MIT) kompatibel dipakai di sini.
