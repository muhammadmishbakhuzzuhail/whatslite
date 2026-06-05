# Product Brief — WhatsApp Lite (klien WhatsApp desktop Linux ringan)

> Dokumen arah produk. Hasil sesi PROJECT OVERVIEW + BRAINSTORM + TARGET MARKET.
> Status: **draft v2** · Tanggal: 2026-06-02 · **Stack diubah**: dari Gio (native custom-drawn) ke
> **web (HTML/CSS/JS) di WebView sistem via Wails (Go)**. Alasan lengkap di Bagian 12.

---

## 0. TL;DR

Klien WhatsApp desktop untuk Linux yang **ringan & efisien**, dibangun di atas **whatsmeow** (protokol
WhatsApp Web multi-device langsung via WebSocket). UI = **web (HTML/CSS/JS)** dirender di **WebView sistem
(WebKitGTK), bukan Chromium yang dibundel** — jadi setara ringan dengan app macOS native, dan jauh di bawah
WhatsApp Web/Windows. Mengisi kekosongan tidak adanya app WhatsApp resmi di Linux.

- **Bukan** produk komersial — proyek **open-source komunitas profil-rendah**.
- **Bukan** parity penuh dengan app macOS — call & pembayaran **mustahil** (batas protokol).
- **Tapi** meniru **fitur harian + UI/UX** WhatsApp macOS — kini **bisa pixel-identik** karena UI = web (CSS).
- Target ke-ringan-an realistis: **~120–250 MB termuat** (setara macOS native, ~3–6× lebih ringan dari WhatsApp Web).
- **Stack: Go (whatsmeow) + Wails + WebView sistem + HTML/CSS/JS** (lihat Bagian 12).
- **Pembeda utama = arsitektur lean & teroptimasi** yang menutup selisih memori WebView (lihat Bagian 12.3).

---

## 1. Problem

- **WhatsApp Web** (di browser) boros: ~300–500 MB idle, **1–2 GB** termuat — karena membawa mesin browser penuh.
- **WhatsApp Windows app resmi (akhir 2025)** berubah jadi wrapper **WebView2/Chromium** → tetap berat
  (~250–450 MB idle, ~0.8–1.5 GB termuat). Banyak pengguna kecewa.
- **Linux tidak punya app WhatsApp desktop resmi sama sekali** → pengguna terpaksa pakai WhatsApp Web
  (Chromium lagi) atau wrapper Electron pihak ketiga (berat lagi).

**Insight kunci:** beratnya WhatsApp desktop **bukan** dari fiturnya, melainkan dari **mesin browser yang
DIBUNDEL** (Electron/WebView2 mengangkut Chromium + Node sendiri). Bukti: app **macOS native** ~3–4× lebih
ringan dari versi Windows. **Pelajaran untuk kita:** kita boleh memakai UI web, **asal jangan membundel
Chromium** — pakai **WebView yang sudah ada di sistem (WebKitGTK)**. Itu membuang ~70–80% beban Chromium
sambil tetap memberi look identik & kemudahan web. Sisa beban (mesin WebKit sistem ~80–150 MB) kita tutup
lewat **disiplin arsitektur** (Bagian 12.3).

---

## 2. Solusi

Engine = whatsmeow (Go, lisensi MIT, sudah menangani media & enkripsi Signal, jejak RAM kecil). Frontend =
**web (HTML/CSS/JS)** dirender di **WebView sistem (WebKitGTK)** lewat **Wails** (shell Go). Satu binary Go
berisi engine + shell; UI web di-embed. TUI = bonus opsional belakangan.

**Kenapa pindah dari native (Gio) ke web?** Tiga alasan menentukan: (1) **look identik WhatsApp** hampir
mustahil digambar tangan di Gio, tapi mudah di CSS — WhatsApp Web sendiri memang web; (2) UI = **skill web
maintainer**, bukan immediate-mode level rendah; (3) ada **feedback loop** (buka di browser, lihat langsung).
Ongkosnya: RAM lebih tinggi dari Gio — ditutup oleh arsitektur (Bagian 12.3).

### Dua sumber "ringan" vs "lengkap" (konsep inti yang harus dipahami)

```
RINGAN?   ← ditentukan oleh apakah mesin browser DIBUNDEL
          → WebView sistem (bukan Chromium bundel) → setara macOS native,
            jauh di bawah WhatsApp Web/Windows  ✅

LENGKAP?  ← ditentukan oleh AKSES PROTOKOL (punya kode Meta vs tidak)
          → KITA TIDAK BISA setara macOS (tak ada call dll)  ❌
```

App macOS ringan **karena tak membundel browser** (bisa kita tiru via WebView sistem) **dan** lengkap **karena
itu app Meta** (tak bisa kita tiru). Kita kebagian separuh keberuntungan itu — dan separuh itu (ringan + look
mudah ditiru) **justru yang paling dibutuhkan pasar.**

---

## 3. Nilai jual (urut prioritas)

1. **Ringan & efisien** — setara app macOS native, **~3–6× lebih hemat dari WhatsApp Web/Windows** (yang
   membundel Chromium). Plus binary ~30–50 MB & **satu proses tambahan**, bukan 5–10 proses. Ini headline.
2. **Linux-first** — mengisi kekosongan, bukan bersaing dengan app resmi.
3. **Arsitektur lean & teroptimasi** — local-first, virtualized, no telemetry/background-service (Bagian 12.3).
4. **Open-source & dapat diaudit** — penting untuk app yang memegang pesan pribadi.
5. **Keyboard-first / scriptable** — relevan untuk persona terminal.

---

## 4. Metrik target (perkiraan; WAJIB diukur ulang sebelum jadi klaim publik)

| Metrik | Web/Chrome | Windows (WebView2) | macOS (Catalyst) | **Target Linux (kita)** |
|---|---|---|---|---|
| RAM idle | ~300–500 MB | ~250–450 MB | ~120–250 MB | **~120–200 MB** |
| RAM termuat penuh | ~1.0–2.0 GB | ~0.8–1.5 GB | ~300–600 MB | **~200–400 MB** |
| Mesin browser **dibundel** | Ya | Ya | Tidak | **Tidak (pakai WebKitGTK sistem)** |
| Jumlah proses | 5–10+ | 4–8 | 1–2 | **2 (Go + WebView)** |
| Ukuran instalasi | (browser) | ~150–300 MB | ~150–250 MB | **~30–50 MB (1 binary, WebKit numpang sistem)** |

> ⚠️ Angka kolom Web/Windows/macOS adalah **perkiraan representatif**, sangat bervariasi per mesin/jumlah chat.
> Target kita kini **setara macOS native** (bukan lagi "puluhan MB" seperti rencana Gio) — konsekuensi sadar
> dari memilih UI web. Sebelum jadi materi publik, **ukur langsung** (PSS) dengan metodologi konsisten.

### Anatomi berat (di mana RAM pergi & strategi kita)

| Komponen | Chromium (Web/Win) | macOS native | **Kita (WebKit sistem)** |
|---|---|---|---|
| Mesin render browser | 150–400 MB (Blink, **dibundel**) ⚠️ | 0 (numpang OS) | **~80–150 MB (WebKit, numpang sistem)** |
| JS engine + heap | 100–300 MB (V8) ⚠️ | 0 | **kecil — JS lean, no framework berat** ✅ |
| Runtime tambahan (Node) | 30–80 MB | 0 | **0 — engine = Go terkompilasi** ✅ |
| Cache media (decoded) | 100–500 MB | kontrol | **kita kontrol** (file di disk, LRU) ✅ |
| State pesan/chat | 20–80 MB | 10–40 MB | **10–40 MB** (local-first SQLite) ✅ |
| Telemetry/background svc | ada | ada | **0** ✅ |

**Beda kita vs Chromium-based:** mesin render **numpang sistem** (bukan dibundel) + **tanpa Node** + **JS
lean** → buang ~60–75% beban. **Beda kita vs macOS native:** hanya selisih mesin WebKit (~80–150 MB), yang
kita perkecil lewat disiplin di baris 2–6 (Bagian 12.3). Risiko terbesar tetap **cache media**.

---

## 5. Analisa fitur (dukungan whatsmeow)

Legenda: ✅ didukung · 🟡 sebagian/perlu kerja · ❌ mustahil (batas protokol) · berat UI: 🟢 ringan / 🟡 sedang / 🔴 perlu disiplin

| Fitur | whatsmeow | Berat | Catatan |
|---|---|---|---|
| **Pesan inti** |
| Teks, emoji | ✅ | 🟢 | Inti |
| Reply/quote, reaksi, edit, hapus/revoke, mention | ✅ | 🟢 | |
| Disappearing/ephemeral | ✅ | 🟢 | |
| View-once | 🟡 | 🟢 | Hormati semantik |
| Pesan terjadwal | ❌* | 🟢 | Bukan protokol; bisa diemulasi lokal |
| **Media** |
| Gambar | ✅ | 🔴 | Decode = sumber RAM #1 |
| Video | ✅ | 🔴 | Butuh pemutar (libmpv/ffmpeg) |
| Dokumen | ✅ | 🟢 | |
| Voice note (PTT), audio | ✅ | 🟡 | Opus |
| Stiker statis | ✅ | 🟡 | WebP |
| Stiker animasi | 🟡 | 🔴 | WebP animasi, render mahal |
| GIF | 🟡 | 🔴 | MP4 |
| Lokasi statis, kontak (vcard) | ✅ | 🟢 | |
| Live location | 🟡 | 🟡 | |
| Link preview | 🟡 | 🟡 | Kita generate sendiri |
| **Grup & komunitas** |
| Grup: baca/kirim/buat/keluar/invite link | ✅ | 🟢–🟡 | |
| Grup: kelola anggota/admin | ✅ | 🟡 | |
| Communities | 🟡 | 🔴 | Sebagian |
| Channels (newsletter) | 🟡 | 🟡 | Baca |
| **Sosial & organisasi** |
| Lihat/post Status | ✅ | 🟡 | |
| Polling (buat & vote) | ✅ | 🟢 | |
| Daftar chat | ✅ | 🟢 | |
| Riwayat (history sync) | 🟡 | 🟡 | **Hanya sebatas yang server kirim saat link**, bukan seluruh riwayat |
| Pin/arsip/mute, tandai belum dibaca | ✅ | 🟢 | App-state sync |
| Search pesan | 🟡 | 🟡 | **Lokal** — index sendiri (SQLite FTS) |
| Read receipt, typing, presence | ✅ | 🟢 | |
| Foto profil/info, blokir | ✅ | 🟢 | |
| **Akun** |
| Pairing QR/link device, multi-device | ✅ | 🟢 | |
| Multi-akun | ✅ | 🟡 | |
| Notifikasi desktop | ✅ | 🟢 | notify-send/dunst |
| **MUSTAHIL (batas protokol)** |
| Voice/video/group call | ❌ | — | Hanya bisa *deteksi* panggilan masuk, tak bisa lakukan |
| Screen sharing | ❌ | — | |
| Pembayaran / WhatsApp Pay | ❌ | — | |

\* bukan batas enkripsi, tapi tak ada di protokol WhatsApp Web.

**Ringkasan:** dari ~48 fitur, **~37 BISA**, **~5 MUSTAHIL** (semua = call/pay). **Fungsi harian ~85–90% bisa ditiru.**

### Bagian "berat" yang harus didisiplinkan (diurutkan)

1. 🔴 **Cache gambar/video di RAM** — risiko #1. → decode on-demand, lepas saat keluar viewport, cache ke disk bukan RAM.
2. 🔴 **Pemutaran video & GIF** — → delegasikan ke libmpv/ffmpeg / app sistem.
3. 🔴 **Stiker animasi** — → batasi fps, statis dulu.
4. 🟡 **History sync awal** — → simpan ke SQLite, jangan tahan di RAM.
5. 🟡 **Search naif** — → index FTS di disk.

**Prinsip emas:** *Ringan bukan karena fitur sedikit, tapi karena disiplin soal media & memori.*

---

## 6. Apakah hasilnya akan "sama persis" seperti macOS? (Ekspektasi)

**Mirip secara fungsi & gaya — YA itu tujuannya. Identik 100% — tidak (dan tak perlu).**

Keputusan produk: **menyalin fitur harian + gaya UI/UX WhatsApp macOS** sebagai **look tetap** (layout
sidebar+chat, bubble rounded, spacing, alur UX). Ini sah & umum — Telegram/Discord/Spotify pun punya look
*brand* tetap yang tidak ikut tema OS, dan Linux user memakainya tanpa masalah.

- **Fungsi harian (teks, media, grup, status, dll): SETARA.** ~85–90% kebutuhan sehari-hari terpenuhi.
- **Gaya UI/UX: BISA PIXEL-IDENTIK.** Karena UI = web (HTML/CSS), look WhatsApp (yang memang web) dapat ditiru
  persis — bukan sekadar "mirip rasa" seperti rencana Gio dulu. Look tetap, light/dark bawaan.
- **Ke-ringan-an: SETARA macOS native** (bukan lebih baik — itu konsekuensi WebView). Tapi **jauh di bawah**
  WhatsApp Web/Windows, dan kita unggul di **footprint disk, jumlah proses, no-telemetry, arsitektur lean**.
- **Yang TETAP berbeda / kurang:**
  - Bukan aset resmi Meta — kita tulis ulang CSS/markup sendiri (boleh sangat mirip, bukan menyalin file Meta).
  - Call/pay hilang (mustahil), history sync terbatas, beberapa fitur 🟡 parsial, tergantung perubahan protokol.

> **Target:** *"fungsi harian + UI/UX WhatsApp macOS yang (nyaris) pixel-identik, ringan setara native, jauh di
> bawah Chromium"*. Yang ditiru = layout, style, alur (kini murah via CSS). Yang TIDAK = call (mustahil).

---

## 7. Roadmap & estimasi (skala solo/komunitas)

Arsitektur: **engine (whatsmeow + state + SQLite) terpisah dari frontend web.** Engine = package Go murni
(siap jadi daemon/headless), frontend = HTML/CSS/JS di WebView, dijembatani Wails. Frontend = **GUI web sejak
awal** (look ala WhatsApp macOS). TUI = bonus opsional, jauh di belakang.

| Fase | Cakupan | Effort | Target RAM |
|---|---|---|---|
| **v0.1 — Daily-driver minimal** | Engine whatsmeow + UI web dasar; pairing QR, daftar chat, kirim/terima teks, terima media (klik→buka eksternal), reply, reaksi, read receipt, typing, notifikasi, riwayat dasar (SQLite); light/dark | beberapa minggu–2 bulan | ~120–180 MB |
| **v0.2 — Media & grup** | Gambar inline + thumbnail, voice note, dokumen, grup penuh, mention, edit/hapus | ~1 bulan+ | ~150–250 MB |
| **v0.3 — Sosial & organisasi** | Status, polling, pin/arsip/mute, search lokal (FTS), foto profil, blokir | beberapa minggu+ | ~180–300 MB |
| **v0.4 — Lanjutan** | Stiker (statis→animasi), video inline (libmpv), GIF, multi-akun, link preview, Channels (baca) | ~1 bulan+ | ~200–400 MB |
| **v0.x — Eksperimen** | Communities (sebagian), live location, daemon/headless, TUI, integrasi CLI | bertahap | — |
| **❌ TIDAK PERNAH** | Voice/video call, screen share, payments | — | — |

- Sampai **v0.4 (gaya + fitur harian macOS minus call):** ~6–12 bulan solo, lebih cepat dengan kontributor.
- Bagian terlama & tersulit = **fondasi engine + GUI dasar v0.1**, bukan fitur belakangan.

---

## 8. Target market & positioning

- **Tujuan = proyek FOSS komunitas profil-rendah, BUKAN komersial.** Terbukti dari batasan:
  1. Melanggar ToS WhatsApp; nomor pengguna berisiko **banned** → tak bisa dijual secara sehat.
  2. Protokol Meta berubah-ubah → beban maintenance abadi (wajar untuk komunitas, bunuh diri untuk produk ber-SLA).
  3. Visibilitas tinggi = undangan banned → **profil rendah = strategi bertahan hidup.**
- **"Market" = komunitas, bukan pelanggan.** Metrik sukses: GitHub stars, kontributor, paket AUR/Flathub/nixpkgs,
  diskusi HN/r/linux — **bukan** MAU/revenue.
- **Persona utama:** power user Linux/terminal (suka ringan, native, FOSS, keyboard-first, toleran rough edges).
- **Linux-first sudah tepat.** Windows = **bonus distribusi** belakangan, **bukan** target kedua setara
  (pengguna Windows umumnya tak toleran setup ribet/banned & tak menghargai FOSS — bukan persona kita).

**Positioning satu kalimat:**
> *"Klien WhatsApp Linux yang ringan & efisien — tanpa membundel Chromium, seringan app macOS native —
> untuk orang yang sayang RAM-nya. Open-source, dipakai dengan risiko sendiri."*

Disclaimer ("risiko sendiri") = **bagian dari positioning**, bukan footnote. Sekaligus jujur & menyaring pengguna yang tepat.

---

## 9. Risiko & batasan (jangan diabaikan)

| Risiko | Catatan / mitigasi |
|---|---|
| **Banned nomor oleh Meta** | Nyata. Disclaimer jelas di README. Pertimbangkan **nomor dev terpisah** untuk testing demi kelangsungan proyek (terpisah dari keberanian pakai nomor utama untuk pemakaian). |
| **Perubahan protokol** | Bergantung pada laju update whatsmeow. Fitur 🟡 bisa bergeser. |
| **History sync terbatas** | Tak dapat seluruh riwayat lama saat pertama link — set ekspektasi pengguna. |
| **Takedown/DMCA** | Mungkin terjadi suatu saat. Konsekuensi dari sifat proyek. |
| **Bus factor / solo maintainer** | Jika nomor maintainer banned → velocity mati. → nomor dev terpisah. |

---

## 10. Keputusan yang DIKUNCI

- ✅ **Tanpa membundel Chromium** — pakai **WebView sistem (WebKitGTK)**. (Revisi dari "tanpa webview sama
  sekali": UI web di WebView sistem dipilih demi look identik + skill web maintainer; lihat Bagian 12.)
- ✅ Linux-first; Windows = bonus distribusi belakangan.
- ✅ Proyek FOSS komunitas profil-rendah; "market" = komunitas.
- ✅ Voice/video call & pembayaran = **non-goal permanen** (mustahil, batas protokol).
- ✅ Bukan parity fitur penuh; jangkar = **"fungsi harian + UI/UX macOS (nyaris pixel-identik) + ringan setara native"**.
- ✅ Arsitektur **engine + frontend terpisah** (engine = package Go murni, siap jadi daemon).
- ✅ Frontend: **engine dulu → UI web (Wails) sebagai frontend utama → TUI bonus belakangan**.
- ✅ Stack UI = **web (HTML/CSS/JS) + Wails (shell Go) + WebKitGTK**. Gio/Electron/Tauri/GTK/Qt **ditolak** (Bagian 12).
- ✅ Style = **look ala WhatsApp macOS, tidak ada theming user**; light/dark bawaan saja.
- ✅ **Pembeda = arsitektur lean & teroptimasi** (Bagian 12.3) untuk menutup overhead WebView.
- ✅ Cakupan v0.1 = **minimal "chat teks harian"** (daftar di Bagian 7).
- ✅ Strategi media v0.1 = **klik → buka di app eksternal** (inline ditunda ke v0.2).
- ✅ Packaging = **AUR + Flatpak + AppImage** (portabel semua distro, X11 + Wayland).

## 11. Yang masih perlu diputuskan (sisa untuk implementasi)

1. **Nama proyek final & lisensi:** MIT (ikut whatsmeow) vs GPL (proteksi komunitas).
2. **Metodologi ukur RAM** untuk memvalidasi klaim di Bagian 4 sebelum jadi materi publik.
3. **Sikap ToS/legal di README:** seberapa eksplisit disclaimer ban-risk.
4. **Framework UI web:** vanilla JS vs framework ringan (Preact/Svelte yang ter-compile). Default: seminimal mungkin.
5. **Strategi media berat (v0.2+):** libmpv vs ffmpeg vs app sistem untuk video/GIF.
6. **Wails v2 vs v3 vs bare `webview_go`:** Wails untuk DX/binding; bare-webview kalau mau paling tipis.

---

## 12. Tech Stack (terkunci)

### 12.1 Stack

```
Engine     : Go + whatsmeow + SQLite (modernc.org/sqlite, pure-Go)
Arsitektur : engine = package Go murni (siap jadi daemon headless),
             terpisah dari frontend → bisa dipakai ulang GUI/TUI/daemon
Shell      : Wails (Go ↔ WebView sistem; binding Go↔JS, bundling, dev-server)
Frontend   : web — HTML/CSS/JS (vanilla atau framework ringan ter-compile)
             di-embed dalam binary; TUI = bonus opsional jauh di belakang
Render     : WebView sistem — WebKitGTK (Linux). TIDAK membundel Chromium.
Style      : look terinspirasi WhatsApp macOS (kini bisa nyaris pixel-identik
             via CSS). Tidak ada theming user. Light/dark bawaan.
Packaging  : AUR (Arch/CachyOS) + Flatpak (semua distro) + AppImage (opsional)
Display    : X11 + Wayland (lewat GTK host Wails)
Dependensi : webkit2gtk + gtk3 (ada di semua distro mainstream)
Ditolak    : Electron (bundel Chromium+Node → berat, lawan tujuan)
             Tauri (sama-sama WebView sistem, TAPI shell Rust + engine Go
                    jadi sidecar = 2 toolchain + batas IPC, nol untung)
             Gio (custom-drawn: look identik nyaris mustahil, no feedback loop,
                  bukan skill web) — stack lama, ditinggalkan
             GTK/Qt native (memaksa look DE; bukan skill web)
```

### 12.2 Kenapa Wails + web (bukan Gio, bukan Tauri, bukan Electron)

| Alasan | Penjelasan |
|---|---|
| **Look identik WhatsApp** | UI = CSS; WhatsApp Web memang web → ditiru persis, bukan digambar tangan |
| **Pakai skill web maintainer** | HTML/CSS/JS, bukan immediate-mode level rendah; bisa di-preview di browser |
| **Satu bahasa untuk shell+engine** | Wails = shell Go → whatsmeow hidup di proses yang sama, tanpa sidecar/IPC (beda dari Tauri yang Rust) |
| **Tak membundel browser** | Pakai WebKitGTK sistem → buang ~70–80% beban Chromium (beda dari Electron) |
| **Tanpa runtime kedua** | Engine = Go terkompilasi, bukan Node → satu binary ~30–50 MB |
| **Portabel** | webkit2gtk+gtk3 ada di semua distro mainstream; X11 & Wayland |

### 12.3 Keunggulan & optimasi arsitektur — PEMBEDA UTAMA ⭐

WebView menaruh kita ~80–150 MB di atas Gio (mesin WebKit). **Kita tutup selisih itu — dan unggul atas app
resmi — lewat arsitektur**, bukan dengan berharap toolkit lebih ringan. Inilah nilai jual teknis proyek.

**A. Lebih ramping dari WhatsApp Web/Windows (Chromium-based):**
- **WebView sistem, bukan Chromium dibundel** — render numpang OS; nol MB browser di binary kita.
- **Tanpa Node/runtime JS server** — "backend" = Go terkompilasi, bukan proses Node 30–80 MB.
- **JS frontend lean** — vanilla atau framework ter-compile (Preact/Svelte), **bukan** React + bundle gemuk.
- **Nol telemetry / analytics / background service** — app resmi menjalankan layanan latar; kita tidak.
- **Satu binary, ~2 proses** (Go + WebView) vs 5–10 proses Chromium.

**B. Menutup selisih vs macOS native (disiplin di sisi data — bagian RAM yang dominan):**
- **Virtualized message list** — hanya node DOM yang terlihat yang ada; ribuan pesan ≠ ribuan elemen.
  Ini menjaga heap WebKit tetap kecil (DOM besar = pembunuh memori web #1).
- **Local-first SQLite** — daftar chat & pesan dibaca dari DB lokal, **paginasi ~50** + lazy-load saat scroll;
  tak pernah seluruh riwayat masuk RAM.
- **Media = file di disk, DB simpan path** — **tidak** base64 di DOM (pembunuh memori #2); thumbnail lazy,
  gambar penuh on-demand, dilepas saat keluar viewport; **cache LRU** berbatas.
- **Video/GIF/stiker animasi → delegasi** ke libmpv / app sistem; codec tak ditarik ke WebView.
- **Update event-driven (delta)** — pesan baru dikirim Go→JS sebagai satu event & disisipkan, **bukan** reload
  seluruh view; minim alokasi & re-layout.
- **Aset internal**: ikon SVG (kecil), wallpaper via CSS, **font sistem** (tak membundel megabyte font).

**C. Hemat saat idle / tersembunyi:**
- Saat window disembunyikan: hentikan render/animasi UI; yang hidup cukup **WebSocket Go** (engine murah).
- Tak ada polling boros; UI hanya bangun saat ada event atau interaksi.

**Klaim jujur yang boleh dipasarkan:** *"seringan app macOS native, 3–6× lebih hemat dari WhatsApp Web,
binary ~30–50 MB, dua proses, nol telemetry"* — **bukan** "puluhan MB" (itu mimpi Gio yang sudah kita lepas).
Semua angka **wajib diukur (PSS)** sebelum jadi materi publik.

### 12.4 Penyimpanan (SQLite) — terkunci, dirancang ringan

**Kenapa wajib:** WhatsApp = model **device-local + E2E**, server hanya **relay** — bukan cloud seperti
Telegram. Server **tidak** menyajikan riwayat pesan lama on-demand. Maka sesi, kunci, dan riwayat **harus**
hidup lokal, atau: scan QR ulang tiap buka + daftar chat selalu kosong. SQLite = opsi persistensi paling
ringan (embedded, satu file, tanpa server).

**Prinsip:** *DB kecil & ramping, media di file terpisah, RAM dijaga dengan paginasi.*

| Data | Lokasi | Catatan |
|---|---|---|
| Sesi + kunci enkripsi | SQLite (store bawaan whatsmeow) | Wajib; dikelola otomatis library |
| Pesan + metadata chat | SQLite (`app.db`) | Teks ringan, di disk |
| Search | FTS5 (fitur SQLite) | Tanpa dependensi tambahan |
| **Media** | **File di folder cache; DB simpan *path* saja** | ⭐ DB tak pernah membengkak |

**Tiga keputusan kunci agar ringan:**
1. **Media = file di disk, bukan blob di DB** (DB hanya simpan referensi).
2. **Cache media dibatasi (LRU)** — file lama dibuang otomatis saat lewat batas ukuran.
3. **Pesan dibaca bertahap (paginasi)** — muat ~50 pesan terakhir, lazy-load saat scroll; tak pernah seluruh riwayat ke RAM.

**Detail:**
- **Lokasi (XDG):** data di `~/.local/share/<app>/`, cache media di `~/.cache/<app>/`.
- **SQLite:** mode WAL + `cache_size` dibatasi → SQLite sendiri hemat RAM.
- **Enkripsi at-rest:** v0.1 cukup **permission file (0600)**; SQLCipher = opsional nanti (hindari beban/kompleksitas dini).
