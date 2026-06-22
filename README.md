# WhatsLite

A **lightweight, efficient** desktop WhatsApp client for Linux — **no Chromium, no WebView**. The UI is a
**native pure-Go [Gio](https://gioui.org) application** running the
[whatsmeow](https://github.com/tulir/whatsmeow) engine (multi-device WhatsApp Web protocol over WebSocket)
**in the same process** — one binary, no IPC bridge, no embedded browser. Targets a RAM footprint **on par
with a native macOS app** and **3–6× lighter than WhatsApp Web**.

> Stack: **Gio (pure-Go GPU UI) + whatsmeow engine + SQLite (pure-Go, FTS5), all in-process.** Media flows
> in-memory (bytes → image, no media-server); voice = libopus (cgo), video = libmpv (cgo); WhatsApp SVG
> icons via oksvg; colors match WhatsApp Web exactly.
>
> The frontend was rewritten three times — Svelte/Wails (WebKitGTK), then Qt6/QML, now **Gio**. The
> **Svelte/Wails** app stays in the repo (`main.go`, `frontend/`) as the design reference + AUR target; the
> **Qt6/QML** frontend has been removed. The primary/packaged app is **Gio** (`cmd/whatslite-gio`).

Run: `./cmd/whatslite-gio/run.sh` (real session) or `run.sh demo` (static UI). Headless UI render for
audits: `cmd/gio-shot` / `tools/snap-gio.sh`. Capture the live app: see [`docs/ui-capture.md`](./docs/ui-capture.md).

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
| **Engine (BE)** | Go + whatsmeow + SQLite (`modernc.org/sqlite`, pure-Go) — **in-process** |
| **Frontend (FE)** | [Gio](https://gioui.org) — native pure-Go immediate-mode GPU UI (`cmd/whatslite-gio`) |
| **Render** | Gio GPU backend (OpenGL/Vulkan) — **no WebView, no bundled Chromium** |
| **Media** | in-memory (bytes→image, no media-server); voice libopus, video libmpv (cgo); SVG icons via oksvg |
| **Storage** | SQLite for session/keys/messages; media as files (not in the DB) |
| **Reference FE** | Svelte/Wails (WebKitGTK) kept in `main.go` + `frontend/` (design source; AUR target) |

- Local data: `~/.local/share/whatslite/` · media cache: `~/.cache/whatslite/` (XDG).
- The key differentiator is the **lean architecture** (local-first, media-as-file instead of base64 in the
  DB, evicted cache, bounded message retention, no telemetry) to offset the WebView overhead. Details in
  `PRODUCT-BRIEF.md` §12.3.

## Install

### Arch Linux / CachyOS (AUR)

Once published to the AUR, an AUR helper installs everything (dependencies, build, binary, desktop entry,
icon) in one command:

```sh
yay -S whatslite            # stable (latest tagged release)
yay -S whatslite-git        # bleeding edge (latest commit)
```

Or build locally without a helper (each package lives in its own directory):

```sh
cd packaging/aur/whatslite      # or packaging/aur/whatslite-git
makepkg -si                     # pulls deps, builds, installs
```

`pacman -R whatslite` (or `whatslite-git`) uninstalls it. The two conflict — install one.

### Any distro (Flatpak)

A Flatpak bundles its own WebKitGTK + glibc, so it runs anywhere — including older distros the prebuilt
binary can't reach. Build/install it from [`packaging/flatpak/`](./packaging/flatpak/) (experimental,
self-hosted for now):

```sh
flatpak-builder --user --install --force-clean build-flatpak \
  packaging/flatpak/io.github.muhammadmishbakhuzzuhail.WhatsLite.yml
```

### Other distros

Build from source (below), or use the prebuilt binary from
[Releases](https://github.com/muhammadmishbakhuzzuhail/whatslite/releases) if your distro meets the
[compatibility](#compatibility) requirements.

## Build prerequisites (Linux)

The primary app is **Gio** (`cmd/whatslite-gio`). Requires **Go** + Gio's GPU/windowing libs +
**libopus** (voice) + **libmpv** (video). On Arch/CachyOS:

```sh
sudo pacman -S --needed go pkgconf opus mpv \
  libglvnd libxkbcommon wayland libx11 libxcursor vulkan-icd-loader
```

(Debian/Ubuntu: `golang-go pkg-config build-essential libopus-dev libmpv-dev libgl1-mesa-dev \
libegl1-mesa-dev libwayland-dev libxkbcommon-dev libx11-dev libxcursor-dev libvulkan-dev`.)

The legacy **Svelte/Wails** reference frontend additionally needs **WebKitGTK 4.1 + GTK3** and the
**Wails CLI** (`webkit2gtk-4.1 gtk3`; `go install …/wails/v2/cmd/wails@latest`).

## Build & run

```sh
# Primary: Gio app (engine in-process). Real WhatsApp session:
./cmd/whatslite-gio/run.sh
# …or static demo data (UI without network):
./cmd/whatslite-gio/run.sh demo
# manual: go build -o whatslite-gio ./cmd/whatslite-gio && ./whatslite-gio

# Headless UI render (audit/screenshots, no GPU window needed):
go build -o gio-shot ./cmd/gio-shot && ./gio-shot out.png app-chat

# debug CLI (engine only, no UI; static binary):
CGO_ENABLED=0 go build -o walite-cli ./cmd/walite-cli && ./walite-cli

# legacy reference frontend (Svelte/Wails, needs WebKitGTK 4.1):
wails build -tags "webkit2_41 netgo"   # output at ./build/bin/whatslite
```

> `netgo` (pure-Go DNS) is recommended to avoid the CGo getaddrinfo resolver clashing with C runtimes.

On first launch a **QR screen** appears in the window. Scan it via:
**WhatsApp on your phone → Linked Devices → Link a device.**
The session and messages are stored locally, so subsequent runs don't require scanning again.

Verbose mode (debug logs): `WALITE_DEBUG=1 wails dev`

## Compatibility

The app links **WebKitGTK 4.1** (`libwebkit2gtk-4.1.so.0`) + GTK3 and is **not statically linkable**
(Wails uses cgo). So there are two independent requirements to run a **prebuilt binary**:

1. **WebKitGTK 4.1** present (not the older 4.0), and
2. **glibc ≥ 2.34** (the released x86_64 binary is built on an older toolchain to keep this floor low).

| Distro | Prebuilt 4.1 binary | Build from source |
|---|---|---|
| Arch / CachyOS, Manjaro | ✅ | ✅ |
| Ubuntu 22.04 / 24.04, Mint 21+, Pop!_OS 22.04+ | ✅ | ✅ |
| Debian 12 (bookworm) / 13 (trixie) | ✅ | ✅ |
| Fedora 40+ | ✅ | ✅ |
| openSUSE Leap 15.6 / Tumbleweed | ✅ | ✅ |
| Ubuntu 20.04, Debian 11 | ❌ (no 4.1, glibc too old) | ❌ |
| RHEL / Rocky / Alma 8 & 9 | ❌ (WebKitGTK is 4.0-only) | ❌ as 4.1 |

**Building from source works on any distro** that packages `webkit2gtk` (4.0 *or* 4.1) + GTK3 + Go — pick
the matching Wails tag (`webkit2_41` for 4.1; omit it for 4.0). For **older / EL distros or "just works
anywhere"**, a **Flatpak** (bundles its own WebKitGTK + glibc via the GNOME runtime) is the universal path —
a manifest is in [`packaging/flatpak/`](./packaging/flatpak/) (build it locally / self-host; experimental).
**ARM64 (aarch64)** is supported but needs a separate native build (no binary shipped yet).

> The `.AppImage` route is unreliable for WebKitGTK apps and is not published — use Flatpak for portability.

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

Copyright (C) 2026 Muhammad Mishbakhuz Zuhail. Licensed under **GPL-3.0** — see [`LICENSE`](./LICENSE)
and [`COPYRIGHT`](./COPYRIGHT); contributors are listed in [`AUTHORS`](./AUTHORS).

Forks and redistribution are welcome under the GPL: you may copy and modify, **but you must keep the
copyright and license notices, stay GPL, and provide source** — you may not relicense it or claim it as
your own work.

This project links [whatsmeow](https://github.com/tulir/whatsmeow),
which is **MPL-2.0**; MPL-2.0 §3.3 explicitly permits distributing the larger work under the GPL, so the
combination is license-compatible. Third-party components and their licenses are listed in
[`NOTICE.md`](./NOTICE.md).
```
