# Architecture

This document describes the design of the Qt 6 / QML frontend and the bridge
that connects it to the Go engine.

## Overview

WhatsLite keeps a strict separation between the **engine** and the **frontend**:

- **Engine** (`internal/engine`, `internal/storage`, `internal/app`): owns the
  WhatsApp protocol (whatsmeow), local storage (SQLite), media, and all
  application state. It is the single source of truth.
- **Frontend** (`qt-app`): a presentation layer that issues typed requests and
  renders state. It holds no protocol or persistence logic.

The two communicate over a small, asynchronous IPC bridge.

## Design influences

The frontend follows two well-established desktop patterns:

1. **Native rendering (Telegram Desktop).** Telegram Desktop renders its UI with
   Qt rather than an embedded web view, which keeps its memory and CPU profile
   low. This frontend applies the same principle: the Qt Quick scene graph draws
   the UI on the GPU, the message timeline is virtualized with delegate
   recycling, and animations run on Qt's own clock. A proof of concept confirmed
   that a `ListView` with `reuseItems` keeps only a small, bounded set of live
   delegates (~32) while scrolling a list of 10,000 variable-height items.

2. **Narrow async boundary (TDLib).** TDLib exposes the engine through a tiny
   asynchronous interface — send a request from any thread, receive responses and
   updates on one stream. This frontend mirrors that shape: a single bridge
   carries request/response traffic and a unified event stream.

## IPC bridge

The bridge (`internal/ipc`) speaks **newline-delimited JSON over a Unix domain
socket**.

### Message envelope

```jsonc
// Request  (frontend → engine)
{ "@type": "GetChats", "@extra": "7", "args": [] }

// Response (engine → frontend)
{ "@type": "Response", "@extra": "7", "result": [ /* ChatDTO[] */ ] }

// Error
{ "@type": "Error", "@extra": "7", "code": "call_failed", "message": "…" }

// Event    (engine → frontend, unsolicited)
{ "@type": "wa:message", "payload": "123@s.whatsapp.net" }
```

- `@type` on a request is the engine method name. Engine method dispatch is
  reflection-based, so every exported `App` method is reachable without
  per-method wiring.
- `@extra` is a caller-generated id echoed back on the matching response,
  enabling request/response correlation on the client.
- Events carry no `@extra`; their `@type` is one of the `wa:*` names.

### Media

Media is referenced by URL, not embedded in the message stream. The engine runs
a loopback HTTP server exposing `/media/<chat>/<id>`, `/avatar/<jid>`,
`/sticker/<hash>`, and `/savedgif/<hash>`. The frontend loads these as image
sources.

## Frontend components

| Component | Role |
|-----------|------|
| `WaEngineClient` | `QLocalSocket` client. `call(method, args, cb)` sends a request and resolves the callback on the correlated response; `wa:*` frames are emitted as `event(type, payload)` signals. |
| `AppController` | Connects QML to the engine: invokes methods, fills models, reacts to events. `loadInto(method, model)` and `loadDetail(method, arg)` provide generic, reusable data loading. |
| `JsonListModel` | A generic `QAbstractListModel` holding a `QJsonArray` and exposing each object to QML as the role `m`. Reused by every list view. |
| `main.qml` | The UI: navigation rail, sidebar views, conversation timeline, composer, pickers, and modals. |

## Migration strategy

The frontend is introduced alongside the existing Svelte/Wails frontend rather
than replacing it outright:

- The engine emits every UI event through a single `emit()` fan-out that
  delivers to **both** the Wails runtime (Svelte) **and** the IPC bridge (Qt).
- The Wails build and the headless engine build share the same `internal/app`
  code, so both frontends can run against the same engine.

This keeps the previous frontend fully functional during the transition and
allows reverting by simply running the Wails build.
