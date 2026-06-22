# WhatsLite — Target Architecture (proper, lightweight, Linux-optimized)

Goal: ~100% similar to WhatsApp (Web/macOS) **and more optimal on Linux**.

> **Current implementation (2026): the FE is [Gio](https://gioui.org), in-process.**
> The frontend was rewritten Svelte/Wails → Qt6/QML → **Gio**. The primary app
> (`cmd/whatslite-gio`) runs the whatsmeow engine + SQLite **in the same process** —
> no Wails shell, no WebView, no media-server HTTP, no IPC bridge. Media flows
> in-memory (`MediaBytes`/`AvatarBytes` → `image.Decode` → GPU texture). The UI
> **polls** the store/engine (`GetChats`/`GetMessages`/`QRCode`/`ChatSubtitle`) rather
> than reacting to emitted events. The Svelte/Wails description below is the original
> design (kept as the reference frontend); the Qt6/QML frontend was removed. The lean
> principles, schema, and roadmap below still apply.

This document is the source of truth for the architecture + a phased roadmap.

---

## 0. Principles
- **The DB stores small text/metadata; media = FILES on disk** (path in the DB). No
  large bytes / base64 in the DB.
- **Lazy + cache + eviction** on every heavy path (media, profile photos, rendering).
- **Event-driven**: engine emits → store persists → UI reacts. Idempotent.
- **Linux-native** where it matters (single-instance, XDG, sound alerts only — no desktop notifications, by design).

---

## 1. Layers

Current (Gio, in-process — one binary, no shell/bridge/media-server):
```
┌─ Frontend (Gio, pure-Go GPU)      ── widget.List virtualization, lazy in-memory
│                                       media (MediaBytes→texture), polls the store
├─ App/Service (Go, internal/app)   ── orchestrate engine↔store↔UI, dedup, sync
│                                       reconcile; exposes Get*/Send*/… methods
├─ Engine (Go, whatsmeow)           ── WA protocol, connection, event stream
│
└─ Store (SQLite, modernc)          ── normalized schema, FTS5, WAL, file-refs
```

Original / reference FE (Svelte in WebKitGTK via Wails — `main.go` + `frontend/`):
```
┌─ Frontend (Svelte in WebKitGTK) ── virtualized list, lazy media, thin state
├─ App/Service (Go, Wails bindings) ── + media-server (AssetServer ServeHTTP), alerts
├─ Engine / └─ Store                ── (shared with the Gio app)
```

Rules: whatsmeow types don't leak outside the engine; the UI doesn't know SQL; media never
becomes base64 in the DB.

---

## 2. Store (SQLite) — schema v2

Tables:
- `chats(jid PK, name, last_text, last_ts, last_sender, last_from_me, unread,
  pinned, muted, archived)`
- `messages(chat_jid, id, sender, push_name, text, kind, ts, from_me,
  quoted_id, quoted_sender, quoted_text,
  media_path, media_mime, media_w, media_h, thumb_path, PRIMARY KEY(chat_jid,id))`
  - **media_path/thumb_path = relative FILE** (not a blob). The base64 `media` proto
    is kept ONLY until it's downloaded (then it may be cleared), or moved to a
    separate `media_blob(chat_jid,id,proto)` table to keep the messages table lean.
- `messages_fts` (FTS5 virtual, content=messages) → **fast message-content search**
  (replaces the O(n) LIKE scan).
- Indexes: `(chat_jid, ts)` (exists), `(ts)` for global.

Optimizations:
- **WAL + busy_timeout** (exists). Consider a separate read connection (read pool)
  so reads aren't blocked by writes — modernc allows 1 writer; reads can run in parallel
  via a second read-only handle.
- **Don't RecomputeSummaries on every GetChats** (currently O(chats) per refresh).
  Move to: incremental updates on SaveMessage (last_* is already upserted), +
  a one-time recompute at startup migration. Remove it from GetChats.

---

## 3. Media pipeline (the core of "lightweight")

- **Receive**: save a (small) JPEG thumbnail → **file** `media/th/<id>.jpg`,
  store `thumb_path`. Save the media proto (for later download) in `media_blob`.
- **Display**: the UI renders `thumb_path` first (instant, blur-up). As it nears
  the viewport (IntersectionObserver) → request `/media/<chat>/<id>` →
  asset-server, cache-first (exists) → save `media/full/<id>.<ext>` →
  set `media_path`. Swap thumb→full smoothly.
- **Eviction**: a periodic sweeper — delete the oldest `full/` files when the total
  exceeds the cap (e.g. 512MB), based on atime/access. Small thumbnails may stay.
- **Profile photos**: same — **file cache** `avatars/<jid>.jpg` + path in the
  `contacts_local(jid, name, avatar_path, avatar_fetched_ts)` table. Lazy (visible
  chats), refreshed on a TTL (e.g. 24h). Stop eagerly `RequestPhotos`-ing everything →
  switch to lazy per-visible-row (cuts hundreds of requests at startup).

Result: a lean DB, low memory (file URLs not data URIs), no re-download on re-open,
disk kept in check (eviction).

---

## 4. Message sync

- **Initial**: whatsmeow history blob → store (exists).
- **Live**: event → store (exists).
- **Old (on-demand)**: scroll past the top of local history while online →
  `BuildHistorySyncRequest(oldestMsgInfo, N)` to your own device → reply
  arrives via `OnHistorySync` → store → UI reload. (not yet; Phase 3)
- **Idempotent**: all upserts by (chat,id). Revoke → MarkDeleted (exists).
- **Local pagination**: exists (`ListMessagesBefore`).

---

## 5. Frontend performance (Linux/WebKitGTK)

- **List virtualization**: render only visible messages (windowing). A chat with
  thousands of messages stays light. (not yet)
- **Unload**: keep only the active chat (plus a few recent ones) in `allMessages`;
  drop old ones from JS memory. (not yet)
- **Lazy media** via IntersectionObserver — DON'T auto-load everything (heavy).
  Load it near the viewport. (currently auto-loads everything → change this)
- **CSS**: avoid Blink-only properties (already using clip-path for tails).
- **WebKitGTK flags**: compositing/DMABUF already handled; `file://` media via the
  asset-server (done).

---

## 6. Linux-native integration ("more complex/optimal")

- **Alerts**: **sound alerts only (no desktop notifications, by design)** — a per-message
  notify-send implementation could exhaust resources on large offline backlogs, so it was
  intentionally removed.
- **Single-instance**: lock file / D-Bus name → relaunch focuses the existing window.
- **Tray** (optional): libappindicator / StatusNotifierItem — minimize-to-tray,
  unread badge.
- **.desktop entry** + icon → app-menu integration, optional autostart.
- **XDG dirs** (done: ~/.local/share). Cache in ~/.cache/whatslite (media
  evictable) — kept separate from data.
- **Wayland/X11**: runs on both (Wayland native + Xwayland fallback).

---

## 7. Features — gaps & fixes

Backend-done-UI-pending: voice recording, group creation, group info, profile editing, blocking,
**search results panel**, **@mention** (color + click→profile + @-autocomplete
→list + Everyone + Meta AI).
Not started at all: pinned-message strip, video/doc auto-handling.
Fixed: reactions (toggle ✓ done), media size (✓), deleted placeholder (✓).
Out of lean scope: calls, status/stories, full channels/communities.

---

## 8. Phased roadmap

- **Phase 1 — Store/Media foundation** (lightweight & correct):
  schema v2 (file refs + FTS5 + separate media_blob), thumbnail→file,
  profile photo→file+lazy, **cache eviction**, remove RecomputeSummaries from
  GetChats. → lean DB, lower memory, fast search.
- **Phase 2 — FE performance**: list virtualization, lazy media (IntersectionObserver),
  memory unload. → light on big chats.
- **Phase 3 — Sync**: on-demand history from the phone, reconnect/backoff.
- **Phase 4 — Linux-native**: single-instance, .desktop, (tray).
- **Phase 5 — Features**: @mention, voice recording, group create/info, profile edit,
  blocking, search UI.
- **Phase 6 — 100% visual polish**: verify each screen natively against WhatsApp.

Each phase: build + verify (snap for visuals; native for behavior) before moving on.
