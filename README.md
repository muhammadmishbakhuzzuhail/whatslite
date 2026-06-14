# WhatsApp Lite

A **lightweight, efficient** desktop WhatsApp client for Linux — **without bundling Chromium**. The UI
runs in the **system WebView (WebKitGTK)**, not a browser shipped alongside the app like
Electron/WhatsApp Web. Built on [whatsmeow](https://github.com/tulir/whatsmeow) (the multi-device
WhatsApp Web protocol, straight over WebSocket). Targets a RAM footprint **on par with a native macOS
app** and **3–6× lighter than WhatsApp Web**.

> Stack: **web (HTML/CSS/JS + Svelte) + Wails (Go shell) + WebKitGTK**. whatsmeow engine + SQLite storage
> (pure-Go, FTS5). Media is stored as files (not base64 in the DB), the cache is evicted, avatars load lazily.

See [`PRODUCT-BRIEF.md`](./PRODUCT-BRIEF.md) for product direction and
[`docs/ARCHITECTURE.md`](./docs/ARCHITECTURE.md) for architecture.

## ✅ Features

**Messages**: text, @mention, photo/video/document, **voice notes** (ogg/opus), **stickers** (static &
animated), **GIFs** (Tenor), **location**, **contacts** (vCard), **polls**, **view-once photos**,
**disappearing messages**.

**Message actions**: reply/quote, private-reply (group → DM), forward (single & **bulk**), emoji
reactions, edit, delete (for-me / for-everyone), copy, **star**, **pin in chat**, **message info** (sent/
delivered/read ticks + per-recipient read list), **translate** (auto-detect, 25 languages), **multi-select**
(bulk).

**Chats**: pin, mute, **archive** (+ archive panel), mark unread/read, delete, **full-text message search
(FTS5)**, **starred messages** (panel).

**Groups**: info + members, **admin** (rename/change photo, add/remove members, promote/demote, **invite
link**), create group, leave.

**Other**: **Status** (text + photo/video, tap-through viewer), **Channels** (follow/feed/mute),
**Communities** (sub-groups), **profile** (name/about), **privacy** (last-seen/photo/status/groups, read
receipts, **block list**), **sound alerts** (no desktop notifications — intentional, see below), media
lightbox, app lock (PIN), light/dark theme, **i18n (73 languages — ID/EN/ES hand-curated, the rest
machine-translated)**.

---

## ⚠️ Disclaimer (read first)

- This is an **UNOFFICIAL** app and is **NOT affiliated** with WhatsApp or Meta.
- It uses the WhatsApp Web protocol via whatsmeow — this **violates WhatsApp's Terms of Service (ToS)**.
- **The number you link MAY BE BANNED by Meta.** Use **at your own risk**.
- Consider using a spare number, especially during development/testing.
- Provided **with no warranty of any kind**.

> **No desktop notifications by design.** The app never sends OS/desktop notifications (sound alerts only).
> An earlier per-message `notify-send` implementation could spawn one OS process per incoming message and,
> on a large offline backlog replayed at reconnect, exhaust system resources. It was removed deliberately.

---

## Stack

| Layer | Choice |
|---|---|
| **Engine (BE)** | Go + whatsmeow + SQLite (`modernc.org/sqlite`, pure-Go) |
| **Shell** | Wails (Go ↔ system WebView) |
| **Frontend (FE)** | HTML / CSS / JS + Svelte (embedded in the binary) |
| **Render** | WebKitGTK (system WebView, **not** a bundled Chromium) |
| **Storage** | SQLite for session/keys/messages; media as files (not in the DB) |

- Local data: `~/.local/share/whatsapp-lite/` · media cache: `~/.cache/whatsapp-lite/` (XDG).
- The key differentiator is the **lean architecture** (local-first, media-as-file instead of base64 in the
  DB, evicted cache, bounded message retention, no telemetry) to offset the WebView overhead. Details in
  `PRODUCT-BRIEF.md` §12.3.

## Build prerequisites (Linux)

Requires **Go**, **WebKitGTK + GTK3**, and the **Wails CLI**. On Arch/CachyOS:

```sh
sudo pacman -S --needed go webkit2gtk gtk3 pkgconf
go install github.com/wailsapp/wails/v2/cmd/wails@latest   # ensure $(go env GOPATH)/bin is on PATH
```

(Debian/Ubuntu equivalent: `golang-go libwebkit2gtk-4.0-dev libgtk-3-dev pkg-config build-essential`.)

Check the toolchain:

```sh
wails doctor
```

## Build & run

Build tags are **required** on Arch/CachyOS:
- `webkit2_41` → use WebKitGTK 4.1 (not 4.0).
- `netgo` → pure-Go DNS resolver (avoids the `free(): corrupted unsorted chunks` crash from the CGo
  getaddrinfo resolver clashing with WebKitGTK's C runtime).

```sh
# dev mode (UI hot-reload in the WebView):
wails dev -tags "webkit2_41 netgo"

# release build (single binary):
wails build -tags "webkit2_41 netgo"   # output at ./build/bin/whatsapp-lite
./build/bin/whatsapp-lite

# debug CLI (engine only, no UI; static binary):
CGO_ENABLED=0 go build -o walite-cli ./cmd/walite-cli
./walite-cli
```

On first launch a **QR screen** appears in the window. Scan it via:
**WhatsApp on your phone → Linked Devices → Link a device.**
The session and messages are stored locally, so subsequent runs don't require scanning again.

Verbose mode (debug logs): `WALITE_DEBUG=1 wails dev`

## Limitations (not possible client-side)

- **Per-recipient read list for old messages** — `Message info` is only populated for messages sent *after*
  the feature was active (receipts are collected live). whatsmeow does not expose the companion↔primary
  protocol to pull historical logs. Aggregate ticks (✓✓) for old messages *are* recovered via
  `WebMessageInfo.Status` from history sync.
- **Voice/video calls** — whatsmeow has no WebRTC.
- **Changing your own profile photo** — no whatsmeow API (*group* photos work).
- **Meta-curated sticker packs & AI stickers** — endpoints not exposed / require a generative model.

> Note: **"Who viewed my status"** is populated from `status@broadcast` receipts live (since the app
> started) — it may be empty if viewers watched while the app was offline.

## Contributing

PRs and issues welcome. Read [`CONTRIBUTING.md`](./CONTRIBUTING.md) (including the **lean philosophy** and
pre-PR checklist), [`CODE_OF_CONDUCT.md`](./CODE_OF_CONDUCT.md), and [`SECURITY.md`](./SECURITY.md) for
reporting vulnerabilities privately. Change history in [`CHANGELOG.md`](./CHANGELOG.md).

## License

GPL-3.0 — see [`LICENSE`](./LICENSE). This project links [whatsmeow](https://github.com/tulir/whatsmeow),
which is **MPL-2.0**; MPL-2.0 §3.3 explicitly permits distributing the larger work under the GPL, so the
combination is license-compatible. Third-party components and their licenses are listed in
[`NOTICE.md`](./NOTICE.md).
```
