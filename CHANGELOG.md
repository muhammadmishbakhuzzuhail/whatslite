# Changelog

All notable changes to this project are documented here.
The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Changed
- **Frontend rewritten to [Gio](https://gioui.org)** — a native pure-Go immediate-mode GPU UI running
  the whatsmeow engine **in the same process**. This replaces the v0.1.0 Wails/WebKitGTK client. The
  intermediate Qt6/QML frontend and the original Svelte/Wails frontend have both been **fully removed**:
  no WebView, no Chromium, no Node/npm, no `frontend/` build step. The shipping app is a single Gio
  binary (`cmd/whatslite-gio`).
- Performance: media decodes are downscaled to display size before GPU upload; caches are byte-budgeted
  instead of item-count-capped; thumbnails use a disk-first tier with LRU eviction; the chat wallpaper
  is pre-rendered once. Chat summaries update incrementally on save instead of an O(chats) recompute
  per refresh.

### Added
- Voice notes (record + playback), animated stickers, GIFs, polls, location, contacts, view-once and
  disappearing messages, translate, multi-select — all ported to the Gio UI.
- "Diteruskan" (forwarded) label on message bubbles (schema v10).
- Blocked-contacts list under Settings → Privacy; Calls privacy (`calladd`) wired.
- Group admin UI: set group photo, join-request approval, admin-only add-member toggle, invite-link reset.
- Export chat transcript to a `.txt` file.
- Mute-duration picker (8 hours / 1 day / 1 week / always).
- Read receipts are now sent when a chat is opened (blue ticks + cross-device read sync).
- Build version stamped into the binary via `-ldflags -X main.version` (logged at startup, shown in
  Settings, exposed to the UI via `App.Version()`); defaults to `dev` for un-stamped local builds.
- Versioned AUR package `whatslite` (tracks tagged releases) alongside `whatslite-git`.
- Flatpak manifest + AppStream metainfo (`packaging/flatpak/`) for universal cross-distro install
  (Freedesktop runtime + bundled libmpv; no WebKit needed).
- Release CI workflow (build + publish binary on version tag), pinned to ubuntu-22.04 for a wide
  glibc floor. Compatibility matrix documented in the README.

### Removed
- Voice/video **call buttons** in the chat header — whatsmeow has no WebRTC, so they never worked.
  Call *history* (rail tab + log) stays.
- Per-message desktop notifications (could spawn one OS process per message on a large reconnect
  backlog); sound alerts only, by design.

## [0.1.0] - 2026-06-14

First public release as **WhatsLite** — a lightweight WhatsApp desktop client for Linux
(Go + whatsmeow + Wails + WebKitGTK), **with no bundled Chromium**.

### Added
- Messages: text, @mention, media (photo/video/document), voice notes (ogg/opus), stickers
  (static & animated), GIFs (Tenor), location, contacts (vCard), polls, view-once, disappearing.
- Message actions: reply/quote, private reply, forward (single & bulk), reactions, edit, delete,
  star, pin, message info, translate, multi-select.
- Chats: archive, mute, pin, mark read/unread, FTS5 search, starred-messages panel.
- Groups, Channels, Communities, Status (text + media), profiles, privacy + blocking.
- Scheduled messages, reminders, chat folders/filters, app lock (PIN), themes, i18n in 73 languages.
- Group typing indicator + a **voice-recording** indicator ("recording audio…") in both the sidebar
  row and the chat header; outbound typing/recording presence is sent (throttled).
- **Undecryptable messages** show a "Waiting for this message…" placeholder, replaced by the real
  message once the automatic re-request resolves.
- **Experimental message-list virtualization** behind a Settings toggle (default off) for very long chats.
- whatsmeow hardening: `AutomaticMessageRerequestFromPhone`, `EnableDecryptedEventBuffer`,
  `UseRetryMessageStore`; `StreamReplaced` and `TemporaryBan` surfaced to the user; keepalive-timeout
  fast-reconnect.
- Lean architecture: media-as-file, evicting cache (~512 MB cap), bounded message retention, no telemetry.
- FOSS scaffolding: `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`, `NOTICE.md`
  (third-party licenses), issue/PR templates, and an AUR `PKGBUILD` (`whatslite-git`).

### Performance
- SQLite tuning (`synchronous=NORMAL`, `mmap`, in-memory temp) on both databases.
- `loading=lazy`/`decoding=async` on chat images; off-screen GIFs pause via IntersectionObserver;
  the message-list scroll handler is coalesced to one run per animation frame.
- Presence subscribed only for the ~30 most-recent 1:1 chats on connect (plus the open chat and the
  Contacts panel) instead of every chat — less connect-time IQ traffic and battery use.

### Notes
- Documentation is in English. License is **GPL-3.0**; whatsmeow is **MPL-2.0** (GPL-compatible via
  MPL-2.0 §3.3); third-party licenses are listed in `NOTICE.md`.
- **Unofficial** client using the WhatsApp Web protocol — violates WhatsApp's ToS; the linked number
  may be banned. Use at your own risk.

[Unreleased]: https://github.com/muhammadmishbakhuzzuhail/whatslite/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/muhammadmishbakhuzzuhail/whatslite/releases/tag/v0.1.0
