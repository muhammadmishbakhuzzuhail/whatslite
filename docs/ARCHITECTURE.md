# WhatsApp Lite — Arsitektur Target (proper, ringan, Linux-optimized)

Tujuan: mirip ~100% WhatsApp (Web/macOS) **dan lebih optimal di Linux**. Lean:
engine Go/whatsmeow, UI web (Svelte) di WebKitGTK via Wails. Dokumen ini =
acuan kebenaran arsitektur + roadmap berfase.

---

## 0. Prinsip
- **DB simpan teks/metadata kecil; media = FILE di disk** (path di DB). Tak ada
  bytes besar / base64 di DB.
- **Lazy + cache + eviction** di setiap jalur berat (media, foto profil, render).
- **Event-driven**: engine emit → store persist → UI react. Idempoten.
- **Linux-native** di titik yang penting (notifikasi D-Bus, single-instance, XDG).

---

## 1. Lapisan

```
┌─ Frontend (Svelte di WebKitGTK) ── virtualized list, lazy media, state tipis
│
├─ App/Service (Go, Wails bindings) ── orkestrasi engine↔store↔UI,
│                                       media-server, notifikasi, dedup
├─ Engine (Go, whatsmeow)           ── protokol WA, koneksi, stream event
│
└─ Store (SQLite, modernc)          ── skema ternormalisasi, FTS5, WAL, file-refs
```

Aturan: tipe whatsmeow tak bocor keluar engine; UI tak tahu SQL; media tak
pernah jadi base64 di DB.

---

## 2. Store (SQLite) — skema v2

Tabel:
- `chats(jid PK, name, last_text, last_ts, last_sender, last_from_me, unread,
  pinned, muted, archived)`
- `messages(chat_jid, id, sender, push_name, text, kind, ts, from_me,
  quoted_id, quoted_sender, quoted_text,
  media_path, media_mime, media_w, media_h, thumb_path, PRIMARY KEY(chat_jid,id))`
  - **media_path/thumb_path = FILE relatif** (bukan blob). `media` proto base64
    TETAP disimpan HANYA sampai diunduh (lalu boleh dikosongkan), atau di tabel
    terpisah `media_blob(chat_jid,id,proto)` agar tabel messages tetap ramping.
- `messages_fts` (FTS5 virtual, content=messages) → **search isi pesan cepat**
  (ganti LIKE-scan O(n)).
- Index: `(chat_jid, ts)` (ada), `(ts)` utk global.

Optimasi:
- **WAL + busy_timeout** (ada). Pertimbangkan koneksi baca terpisah (read pool)
  agar baca tak terblok tulis — modernc 1 writer; baca bisa paralel via DB
  kedua handle read-only.
- **RecomputeSummaries jangan tiap GetChats** (sekarang O(chats) tiap refresh).
  Pindah ke: update incremental saat SaveMessage (sudah upsert last_*), +
  recompute sekali saat startup migrasi. Hilangkan dari GetChats.

---

## 3. Pipeline Media (inti "ringan")

- **Terima**: simpan thumbnail JPEG (kecil) → **file** `media/th/<id>.jpg`,
  simpan `thumb_path`. Simpan proto media (utk unduh nanti) di `media_blob`.
- **Tampil**: UI render `thumb_path` dulu (instan, blur-up). Saat mendekati
  viewport (IntersectionObserver) → minta `/media/<chat>/<id>` →
  asset-server cache-first (sudah) → simpan `media/full/<id>.<ext>` →
  set `media_path`. Swap thumb→full mulus.
- **Eviction**: sweeper periodik — hapus file `full/` tertua bila total >
  cap (mis. 512MB) berdasar atime/akses. thumb kecil boleh tetap.
- **Foto profil**: sama — **file-cache** `avatars/<jid>.jpg` + path di tabel
  `contacts_local(jid, name, avatar_path, avatar_fetched_ts)`. Lazy (chat
  terlihat), refresh TTL (mis. 24 jam). Hentikan eager `RequestPhotos` semua →
  ganti lazy per-baris terlihat (kurangi ratusan request saat start).

Hasil: DB ramping, memori rendah (URL file bukan data-URI), re-open tak
re-download, disk terjaga (eviction).

---

## 4. Sinkronisasi pesan

- **Awal**: history blob whatsmeow → store (ada).
- **Live**: event → store (ada).
- **Lama (on-demand)**: scroll atas habis lokal + online →
  `BuildHistorySyncRequest(oldestMsgInfo, N)` ke device sendiri → balasan
  masuk via `OnHistorySync` → store → UI reload. (belum; Fase 3)
- **Idempoten**: semua upsert by (chat,id). Revoke → MarkDeleted (ada).
- **Pagination lokal**: ada (`ListMessagesBefore`).

---

## 5. Performa Frontend (Linux/WebKitGTK)

- **Virtualisasi list**: render hanya pesan terlihat (windowing). Chat ribuan
  pesan tetap ringan. (belum)
- **Unload**: simpan di `allMessages` hanya chat aktif + beberapa terakhir;
  buang yang lama dari memori JS. (belum)
- **Lazy media** via IntersectionObserver — JANGAN auto-load semua (berat).
  Load saat dekat viewport. (sekarang auto semua → ubah)
- **CSS**: hindari properti Blink-only (sudah pakai clip-path utk ekor).
- **WebKitGTK flags**: compositing/DMABUF sudah ditangani; `file://` media via
  asset-server (sudah).

---

## 6. Integrasi Linux-native ("lebih kompleks/optimal")

- **Notifikasi**: **D-Bus `org.freedesktop.Notifications`** dari Go (bukan web
  Notification API yg tak andal di WebKitGTK). Klik notif → fokus app + buka
  chat. (belum)
- **Single-instance**: lock file / D-Bus name → relaunch fokus jendela lama.
- **Tray** (opsional): libappindicator / StatusNotifierItem — minimize-to-tray,
  badge unread.
- **.desktop entry** + ikon → integrasi menu app, autostart opsional.
- **XDG dirs** (sudah: ~/.local/share). Cache di ~/.cache/whatsapp-lite (media
  evictable) — pisahkan dari data.
- **Wayland/X11**: jalan di keduanya (Wayland native + Xwayland fallback).

---

## 7. Fitur — gap & perbaikan

Ada-BE-UI-belum: voice record, buat grup, info grup, edit profil, blokir,
**search results panel**, **@mention** (warna+klik→profil+autocomplete
@→list+Semua+Meta AI).
Belum-sama-sekali: notifikasi desktop, strip pinned-message, video/doc auto.
Perbaiki: reaksi (toggle ✓ sudah), media size (✓), deleted placeholder (✓).
Di luar scope lean: calls, status/stories, channels/communities penuh.

---

## 8. Roadmap berfase

- **Fase 1 — Fondasi Store/Media** (ringan & benar):
  skema v2 (file refs + FTS5 + media_blob terpisah), thumbnail→file,
  foto profil→file+lazy, **eviction cache**, hapus RecomputeSummaries dari
  GetChats. → DB ramping, memori turun, search cepat.
- **Fase 2 — Performa FE**: virtualisasi list, lazy media (IntersectionObserver),
  unload memori. → ringan di chat besar.
- **Fase 3 — Sinkronisasi**: on-demand history dari HP, reconnect/backoff.
- **Fase 4 — Linux-native**: notifikasi D-Bus, single-instance, .desktop, (tray).
- **Fase 5 — Fitur**: @mention, voice record, group create/info, profil edit,
  blokir, search UI.
- **Fase 6 — Poles visual 100%**: verifikasi native tiap layar vs WhatsApp.

Tiap fase: build + verifikasi (snap utk visual; native utk perilaku) sebelum lanjut.
