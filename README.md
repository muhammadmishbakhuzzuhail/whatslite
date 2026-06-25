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
> The frontend was rewritten three times — first Svelte/Wails (WebKitGTK), then Qt6/QML, now **Gio**. Both
> earlier frontends have been **fully removed**: there is no WebView, no Chromium, no Node/npm, and no
> `frontend/` build step. The shipping app is a single pure-Go **Gio** binary (`cmd/whatslite-gio`).

Run: `go run ./cmd/whatslite-gio`. Headless UI render for
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
**Communities** (sub-groups), **profile** (name/about/photo), **privacy** (last-seen/photo/status/groups, read
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

- Local data: `~/.local/share/whatslite/` · media cache: `~/.cache/whatslite/` (XDG).
- The key differentiator is the **lean architecture** (native pure-Go GPU UI with no WebView/Chromium,
  local-first, media-as-file instead of base64 in the DB, evicted cache, bounded message retention, no
  telemetry). Details in `PRODUCT-BRIEF.md` §12.3.

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

A Flatpak bundles its own runtime + glibc, so it runs anywhere — including older distros the prebuilt
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

The app is **Gio** (`cmd/whatslite-gio`). Requires **Go** + Gio's GL/Wayland/X11 libs +
**libopus** (voice) + **libmpv** (video). No WebKitGTK, no GTK3, no Node/npm. On Arch/CachyOS:

```sh
sudo pacman -S --needed go mesa wayland libxkbcommon libx11 libxcursor libxfixes \
  vulkan-icd-loader opus mpv pkgconf
```

(Debian/Ubuntu: `golang-go pkg-config build-essential libopus-dev libmpv-dev libgl1-mesa-dev \
libegl1-mesa-dev libwayland-dev libxkbcommon-dev libxkbcommon-x11-dev libx11-dev libx11-xcb-dev \
libxcursor-dev libxfixes-dev libvulkan-dev`.)

## Build & run

```sh
# The Gio app (engine in-process). Real WhatsApp session:
go run ./cmd/whatslite-gio
# …or build a binary named `whatslite`:
go build -o whatslite ./cmd/whatslite-gio && ./whatslite

# Headless UI render (audit/screenshots, no GPU window needed):
go build -o gio-shot ./cmd/gio-shot && ./gio-shot out.png app-chat

# debug CLI (engine only, no UI; static binary):
CGO_ENABLED=0 go build -o walite-cli ./cmd/walite-cli && ./walite-cli
```

On first launch a **QR screen** appears in the window. Scan it via:
**WhatsApp on your phone → Linked Devices → Link a device.**
The session and messages are stored locally, so subsequent runs don't require scanning again.

Verbose mode (debug logs): `WALITE_DEBUG=1 go run ./cmd/whatslite-gio`

## Compatibility

The app is a Gio GPU UI that links **OpenGL/EGL + Wayland/X11** libs and **libopus**/**libmpv** (cgo), so it
is **not statically linkable**. No WebKitGTK or GTK3 is required. To run a **prebuilt binary** you need:

1. A working **OpenGL/EGL** stack and a **Wayland or X11** session (Mesa drivers), and
2. **glibc ≥ 2.34** (the released x86_64 binary is built on an older toolchain to keep this floor low).

| Distro | Prebuilt binary | Build from source |
|---|---|---|
| Arch / CachyOS, Manjaro | ✅ | ✅ |
| Ubuntu 22.04 / 24.04, Mint 21+, Pop!_OS 22.04+ | ✅ | ✅ |
| Debian 12 (bookworm) / 13 (trixie) | ✅ | ✅ |
| Fedora 40+ | ✅ | ✅ |
| openSUSE Leap 15.6 / Tumbleweed | ✅ | ✅ |
| Ubuntu 20.04, Debian 11 | ❌ (glibc too old) | ✅ |
| RHEL / Rocky / Alma 8 & 9 | ❌ (glibc too old) | ✅ |

**Building from source works on any distro** that packages Mesa/GL + Wayland/X11 dev libs + libopus +
libmpv + Go. For **older / EL distros or "just works anywhere"**, a **Flatpak** (bundles its own runtime +
glibc) is the universal path — a manifest is in [`packaging/flatpak/`](./packaging/flatpak/) (build it
locally / self-host; experimental). **ARM64 (aarch64)** is supported but needs a separate native build (no
binary shipped yet).

> CI ([`.github/workflows/build.yml`](./.github/workflows/build.yml)) runs `go vet ./...`, `go test ./...`,
> and builds `./cmd/whatslite-gio` + `./cmd/gio-shot`.

## Limitations (not possible client-side)

- **Per-recipient read list for old messages** — `Message info` is only populated for messages sent *after*
  the feature was active (receipts are collected live). whatsmeow does not expose the companion↔primary
  protocol to pull historical logs. Aggregate ticks (✓✓) for old messages *are* recovered via
  `WebMessageInfo.Status` from history sync.
- **Voice/video calls** — whatsmeow has no WebRTC (call *history* is shown; placing/answering calls is not possible, so no call buttons are drawn).
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
