# Kebijakan Keamanan

## Lingkup

Project ini menyimpan **sesi WhatsApp dan kunci enkripsi end-to-end** secara lokal di
`~/.local/share/whatsapp-lite/` (SQLite). Celah yang membocorkan/menyalahgunakan data ini serius.

Yang **dalam lingkup**:

- Kebocoran sesi/kunci/pesan lokal (mis. permission file salah, path traversal saat simpan media).
- Eksekusi kode dari konten pesan masuk (XSS di WebView dari pesan/notifikasi/link-preview).
- Kebocoran rahasia ke log atau ke jaringan (telemetri tak diharapkan).

Yang **di luar lingkup**: risiko ban akun oleh Meta (itu konsekuensi ToS yang sudah didokumentasikan,
bukan bug), dan kerentanan di whatsmeow/Wails upstream (laporkan ke proyek masing-masing).

## Cara melapor

**Jangan buka issue publik untuk celah keamanan.** Sebaliknya:

- Pakai **GitHub Security Advisories** (tab *Security → Report a vulnerability*) pada repo ini, atau
- Kirim email privat ke maintainer.

Sertakan: langkah reproduksi, versi/commit, dan dampak. Mohon beri waktu wajar untuk perbaikan
sebelum publikasi (disclosure terkoordinasi).

## Tidak ada jaminan

Lihat [Disclaimer di README](./README.md#-disclaimer-baca-dulu). Software disediakan **tanpa jaminan apa pun**.
Memakai protokol tak-resmi berisiko diban; gunakan dengan risiko sendiri.
