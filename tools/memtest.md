<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

# Protokol Uji RAM — WhatsLite (Gio)

Protokol singkat untuk mengukur pemakaian RAM aplikasi: kondisi **idle** vs
**saat dipakai**. Tujuannya memastikan memori naik wajar saat dipakai lalu
**plateau** (mendatar), bukan naik terus tanpa batas.

## Prasyarat

- Build aplikasi:

  ```bash
  go build -o whatslite-gio ./cmd/whatslite-gio
  ```

- Jalankan binarinya: `./whatslite-gio`
- Pengukuran RAM yang berarti **butuh display sungguhan** (jendela GUI benar-benar
  tampil — bukan headless) **dan sesi WhatsApp yang sudah ter-pairing**, supaya
  chat, media, sticker/GIF, dan video bisa benar-benar dimuat.

## Cara menjalankan monitor

Di terminal lain, pantau RSS proses secara berkala:

```bash
tools/memmon.sh 2
```

(interval 2 detik; default nama proses `whatslite-gio`). Tabel akan menampilkan
`elapsed_s  RSS_MB  VSZ_MB  peak=…` tiap sampel.

### Opsi pprof (detail heap Go)

Jalankan aplikasi dengan profil pprof aktif:

```bash
WLGIO_PPROF=1 ./whatslite-gio
```

lalu, di terminal lain:

```bash
go tool pprof http://localhost:6060/debug/pprof/heap
```

Untuk log memori runtime periodik (baris `[mem]` tiap 5 detik) di stdout aplikasi:

```bash
WLGIO_MEMLOG=1 ./whatslite-gio
```

(Keduanya bisa digabung: `WLGIO_PPROF=1 WLGIO_MEMLOG=1 ./whatslite-gio`.)

## Protokol pengukuran

Untuk setiap skenario, biarkan monitor berjalan dan **catat RSS** (dan `peak`)
setelah aktivitas stabil.

1. **IDLE baseline** — aplikasi terbuka, tidak ada chat dipilih, tunggu 60 detik.
   **catat RSS** (ini garis dasar idle).
2. **Chat dengan media** — buka satu chat yang berisi media, lalu scroll ke atas
   menembus ~100+ pesan. **catat RSS**.
3. **Picker sticker/GIF online** — buka picker sticker/GIF daring, scroll banyak
   hasil. **catat RSS**.
4. **Video** — buka lalu putar sebuah video. **catat RSS** (perhatikan lonjakan
   dari libmpv — lihat catatan cgo di bawah).
5. **Beberapa chat berfoto** — buka beberapa chat berbeda yang berisi foto.
   **catat RSS**.
6. **Idle lagi** — tinggalkan idle 60 detik dan lihat apakah RSS turun kembali.
   Karena cache bersifat FIFO dan dibatasi (capped), RSS seharusnya **plateau**
   (mendatar), bukan naik terus selamanya. **catat RSS**.

## Cara baca hasil

- **RSS = total memori proses** = Go heap + cgo (libmpv untuk video, opus untuk
  audio) + buffer GPU. Jadi RSS lebih besar dari sekadar Go heap.
- **Efisien** jika: idle rendah (orde **puluhan MB**), dan saat dipakai RSS
  **naik lalu PLATEAU** (tidak naik terus) — ini bukti cache cap bekerja.
- **Detail heap** lewat pprof: gunakan `top` (alokator teratas) dan
  `list <fungsi>` untuk melihat baris kode yang mengalokasikan paling banyak.

## Catatan bobot cgo

Bobot cgo yang sudah diketahui: **libmpv** (pemutaran video) adalah kontributor
terbesar saat video diputar. Lonjakan RSS pada skenario (4) sebagian besar
berasal dari sini, bukan dari Go heap — jadi cek dengan pprof bila ingin
memisahkan kontribusi Go heap dari cgo.
