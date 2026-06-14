# Changelog

All notable changes to this project are documented here.
The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added
- Group typing indicator + a **voice-recording** indicator ("recording audio…") shown in both the
  sidebar row and the chat header; outbound typing/recording presence is now actually sent (it was
  previously dead code) and throttled to avoid flooding the socket.
- **Undecryptable messages** now show a "Waiting for this message…" placeholder that is replaced by the
  real message once the automatic re-request resolves.
- **Experimental message-list virtualization** behind a Settings toggle (default off): renders only
  on-screen messages with measured-height spacers, for very long chats.
- FOSS community docs: `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`, `CHANGELOG.md`,
  `NOTICE.md` (third-party licenses), plus issue/PR templates.

### Changed
- All documentation translated to **English** and brought up to date for open-source release.
- Corrected license attribution: whatsmeow is **MPL-2.0** (was mislabeled MIT); the project remains
  **GPL-3.0**, which is compatible (MPL-2.0 §3.3).
- whatsmeow client hardening: `AutomaticMessageRerequestFromPhone`, `EnableDecryptedEventBuffer`, and
  `UseRetryMessageStore` enabled; `StreamReplaced` and `TemporaryBan` events now surfaced to the user;
  a keepalive-timeout fast-reconnect for chatty accounts.
- Performance: SQLite tuning (`synchronous=NORMAL`, `mmap`, in-memory temp) on both databases;
  `loading=lazy`/`decoding=async` on chat images; off-screen GIFs pause via IntersectionObserver;
  the message-list scroll handler is coalesced to one run per animation frame.
- Presence is no longer bulk-subscribed for every chat on connect — only the ~30 most-recent 1:1 chats
  (plus the open chat and the Contacts panel), reducing connect-time IQ traffic and battery use.
- Go module path changed from `whatsapp-lite` to `github.com/muhammadmishbakhuzzuhail/whatsapp-lite`
  so it can be `go install`-ed/vendored.

## [0.1.0] - 2026-06-14

First public release. A lightweight WhatsApp desktop client for Linux (Go + whatsmeow + Wails + WebKitGTK),
**with no bundled Chromium**.

### Added
- Messages: text, media (photo/video/document), voice notes (ogg/opus), stickers (static & animated),
  GIFs (Tenor), location, contacts (vCard), polls, view-once, disappearing.
- Message actions: reply/quote, private reply, forward (single & bulk), reactions, edit, delete,
  star, pin, message info, translate, multi-select.
- Chats: archive, mute, pin, mark read/unread, FTS5 search, starred-messages panel.
- Groups, Channels, Communities, Status (text + media), profiles, privacy + blocking.
- Scheduled messages, reminders, chat folders/filters, app lock (PIN), themes, i18n in 73 languages.
- Lean architecture: media-as-file, evicting cache, bounded message retention, no telemetry.

[Unreleased]: https://github.com/muhammadmishbakhuzzuhail/whatsapp-lite/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/muhammadmishbakhuzzuhail/whatsapp-lite/releases/tag/v0.1.0
