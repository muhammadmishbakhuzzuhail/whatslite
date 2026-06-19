# Scope & Limitations

This document records the current scope of the Qt 6 / QML frontend, items that
are not yet implemented, and operational notes.

## Design parity

The visual design is reimplemented in QML and mirrors the project's design
tokens (colors, spacing, radii) defined in `frontend/src/styles/app.css`. Because
QML and CSS use different rendering models, the styling is reproduced rather than
shared verbatim with the previous web frontend.

## Feature coverage

The Qt frontend wires **all applicable engine methods** used by the previous
Svelte frontend (119 of 121). Onboarding, chat management, compose, status,
channels, communities, group administration, profile, privacy, reminders,
scheduled messages, and application settings are connected through dedicated
views, context menus, and the conversation overflow menu.

Two methods are intentionally not wired because they manage the Wails window and
are handled natively by Qt instead:

- `Quit` — the Qt application manages its own lifecycle.
- `SetUnreadBadge` — taskbar/badge integration is handled by the Qt platform.

Some actions currently use placeholder inputs (e.g. example location/poll/group
values) where a dedicated input dialog is still to be added; the engine call
itself is wired and functional.

## Not yet implemented

| Item | Notes |
|------|-------|
| Voice & video calls | Out of scope at the engine level — whatsmeow does not provide WebRTC. |
| Media attachment (non-document) | Sending images/video/audio via a file picker is pending; document attachment is implemented. |
| Packaging | Flatpak / AUR packaging for the headless engine and the Qt binary is pending. |
| Pixel-level animation polish | Transitions and micro-interactions are functional; finer animation tuning is ongoing. |

## Localization

The UI defaults to English with runtime language switching (see the README).
Core and primary surfaces are localized via `i18n/<code>.json`; a few secondary
menu labels are still being moved into the dictionaries. Adding a new language
is a matter of providing a `<code>.json` file with the same keys — missing keys
fall back to English.

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
