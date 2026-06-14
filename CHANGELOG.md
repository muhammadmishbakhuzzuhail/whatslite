# Changelog

All notable changes to this project are documented here.
The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added
- Build version stamped into the binary via `-ldflags -X main.version` (logged at startup, shown in
  Settings, exposed to the UI via `App.Version()`); defaults to `dev` for un-stamped local builds.
- Versioned AUR package `whatslite` (tracks tagged releases) alongside `whatslite-git`.
- Flatpak manifest + AppStream metainfo (`packaging/flatpak/`) for universal cross-distro install
  (GNOME runtime bundles WebKitGTK 4.1 + glibc).
- Release CI workflow (build + publish binary on version tag), pinned to ubuntu-22.04 for a wide
  glibc floor. Compatibility matrix documented in the README.

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
