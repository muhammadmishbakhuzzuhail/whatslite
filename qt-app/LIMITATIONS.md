# Scope & Limitations

This document records the current scope of the Qt 6 / QML frontend, items that
are not yet implemented, and operational notes.

## Design parity

The visual design is reimplemented in QML and mirrors the project's design
tokens (colors, spacing, radii) defined in `frontend/src/styles/app.css`. Because
QML and CSS use different rendering models, the styling is reproduced rather than
shared verbatim with the previous web frontend.

## Feature coverage

The Qt frontend is being brought to parity with the previous Svelte frontend
incrementally. The architecture, navigation shell, and core messaging flows are
in place; engine actions are wired progressively. Areas still in progress:

- Full onboarding (`MyQR`, phone link, logout)
- History pagination (load older messages)
- Chat management (read/unread, pin, mute, archive, delete, clear)
- Compose extras (reply/quote, edit, mentions, location, poll, contact)
- Status posting & viewers
- Channel and community actions
- Group administration
- Profile editing, privacy block list, reminders, and scheduled messages
- Application settings (proxy, retention, storage usage)

## Not yet implemented

| Item | Notes |
|------|-------|
| Voice & video calls | Out of scope at the engine level — whatsmeow does not provide WebRTC. |
| Media attachment (non-document) | Sending images/video/audio via a file picker is pending; document attachment is implemented. |
| Packaging | Flatpak / AUR packaging for the headless engine and the Qt binary is pending. |
| Pixel-level animation polish | Transitions and micro-interactions are functional; finer animation tuning is ongoing. |

## Requirements

- **Qt 6.9 or newer** is required for color emoji (COLRv1). Older Qt 6 releases
  build and run but may render emoji without color.

## Operational notes

- **Bridge socket.** The engine exposes its bridge on a Unix domain socket
  (`bridge.sock`) created with owner-only permissions. The media server binds to
  the loopback interface. Both are intended for the local user only.
- **Single source of truth.** All state lives in the engine; the frontend caches
  only what it receives. If a view looks stale, it can be refreshed by
  re-issuing the corresponding request — the engine remains authoritative.

## Compatibility with the previous frontend

The Svelte/Wails frontend remains in the repository and fully functional. The
engine broadcasts events to both frontends (dual-emit), so the two can run
against the same engine and the previous build can be used as a fallback.
