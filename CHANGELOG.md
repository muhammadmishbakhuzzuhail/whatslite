# Changelog

Semua perubahan penting pada project ini didokumentasikan di sini.
Format mengikuti [Keep a Changelog](https://keepachangelog.com/id/1.1.0/),
dan project ini memakai [Semantic Versioning](https://semver.org/lang/id/).

## [Unreleased]

### Added
- Dokumen komunitas FOSS: `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`, `CHANGELOG.md`,
  serta template issue/PR.

### Changed
- Module path Go diubah dari `whatsapp-lite` ke `github.com/muhammadmishbakhuzzuhail/whatsapp-lite`
  agar dapat di-`go install`/vendor.
- README disinkronkan dengan kondisi nyata: GIF lewat **Tenor** (bukan Giphy), i18n **73 bahasa**,
  stiker animasi didukung, klaim "virtualized list" dan "background-removal ML" dihapus
  (fitur tersebut tidak/tidak lagi ada).

## [0.1.0] - 2026-06-14

Rilis publik pertama. Klien WhatsApp desktop Linux yang ringan (Go + whatsmeow + Wails + WebKitGTK),
**tanpa membundel Chromium**.

### Added
- Pesan: teks, media (foto/video/dokumen), voice note (ogg/opus), stiker (statis & animasi),
  GIF (Tenor), lokasi, kontak (vCard), polling, view-once, disappearing.
- Aksi pesan: balas/kutip, balas-pribadi, teruskan (satu & massal), reaksi, edit, hapus,
  bintangi, sematkan, info pesan, terjemah, pilih-banyak.
- Chat: arsip, bisukan, sematkan, tandai baca/belum, pencarian FTS5, panel pesan berbintang.
- Grup, Channels, Komunitas, Status (teks + media), profil, privasi + blokir.
- Pesan terjadwal, pengingat, folder/filter chat, kunci-aplikasi (PIN), tema, i18n 73 bahasa.
- Arsitektur lean: media-as-file, cache ter-evict, retensi pesan terbatas, no telemetry.

[Unreleased]: https://github.com/muhammadmishbakhuzzuhail/whatsapp-lite/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/muhammadmishbakhuzzuhail/whatsapp-lite/releases/tag/v0.1.0
