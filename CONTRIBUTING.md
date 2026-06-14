# Berkontribusi

Terima kasih sudah tertarik berkontribusi! Project ini klien WhatsApp desktop Linux yang ringan
(Go + whatsmeow + Wails + WebKitGTK). Baca dulu [README](./README.md), [`PRODUCT-BRIEF.md`](./PRODUCT-BRIEF.md),
dan [`docs/ARCHITECTURE.md`](./docs/ARCHITECTURE.md) untuk memahami arah & arsitektur.

> ⚠️ **Baca [Disclaimer ToS di README](./README.md#-disclaimer-baca-dulu) lebih dulu.** Project ini memakai
> protokol WhatsApp Web tak-resmi. Nomor uji **berisiko diban**. Pakai nomor cadangan saat develop.

## Filosofi: tetap *lean*

Pembeda utama project ini adalah **ringan**. Sebelum menambah fitur/dependensi, tanyakan:

- Apakah bisa dilakukan tanpa dependensi baru? (binary saat ini ~25 MB — jaga tetap kecil)
- Apakah menambah berat runtime (RAM/disk) yang signifikan?
- Apakah memuat data besar ke frontend? (jangan kirim base64 media penuh ke WebView — pakai file/cache)

PR yang menambah dependensi berat (mis. ML/WASM besar, Electron-isme) kemungkinan ditolak kecuali
nilainya sepadan. Lihat riwayat: ONNX background-removal pernah dibuang (binary 50→25 MB).

## Prasyarat & build

Lihat [README §Prasyarat build](./README.md#prasyarat-build-linux). Ringkasnya (Arch/CachyOS):

```sh
sudo pacman -S --needed go webkit2gtk gtk3 pkgconf
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

```sh
# dev (hot-reload):
wails dev -tags "webkit2_41 netgo"
# build rilis:
wails build -tags "webkit2_41 netgo"
```

## Sebelum membuka PR

Jalankan semua ini (sama dengan CI di `.github/workflows/build.yml`):

```sh
go vet ./...
go test ./...
npm --prefix frontend ci
npm --prefix frontend run build      # harus 0 warning unused-CSS
wails build -tags "webkit2_41 netgo" # harus sukses
```

- **Tanpa dead code / CSS tak terpakai.** Build FE menandai unused-CSS — bersihkan.
- **Jangan commit rahasia.** API key, `*.db`, binary, dan `real-data.json` sudah di-`.gitignore` — jaga tetap begitu.
- **Hormati handler whatsmeow yang sinkron.** Kerja DB berat di event handler harus di-offload ke
  antrian `a.bg()` — handler jalan di socket loop, blocking = websocket drop.

## Gaya kode

- **Go**: ikuti `gofmt`/`go vet`. Komentar Indonesia konsisten dengan kode sekitar.
- **Svelte/CSS**: cocokkan idiom & kerapatan komentar file sekitarnya. WebKitGTK tidak punya
  `window.confirm()/prompt()` — pakai store `askConfirm`/`askPrompt` (ConfirmDialog/PromptDialog).
- **Commit**: Conventional Commits (`feat:`, `fix:`, `perf:`, `chore:`, `refactor:`, `docs:`).

## Lapor bug / minta fitur

Pakai [issue templates](./.github/ISSUE_TEMPLATE/). Untuk celah keamanan, **jangan** buka issue publik —
lihat [`SECURITY.md`](./SECURITY.md).

## Lisensi

Dengan berkontribusi, kamu setuju kontribusimu dilisensikan di bawah **GPL-3.0** (lihat [`LICENSE`](./LICENSE)).
