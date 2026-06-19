# WhatsLite — Qt 6 / QML Frontend

A native **Qt 6 / QML** desktop frontend for WhatsLite. It connects to the
existing Go engine (whatsmeow + SQLite) over a lightweight local IPC bridge and
renders the full messaging experience with the Qt scene graph — no embedded
browser engine.

The Go engine is unchanged; only the presentation layer and the engine↔UI
bridge are new. See [`ARCHITECTURE.md`](./ARCHITECTURE.md) for design details and
[`LIMITATIONS.md`](./LIMITATIONS.md) for current scope.

## Highlights

- **Native rendering** via the Qt Quick scene graph (GPU-accelerated), with a
  modest memory footprint.
- **Virtualized timeline** — `ListView` with delegate recycling backed by a C++
  model; validated against 10,000 variable-height messages.
- **Clean engine/UI boundary** inspired by TDLib: the engine is the single
  source of truth; the UI sends typed requests and renders pushed state.
- **Light and dark themes**, with the palette aligned to the project design
  tokens.

## Features

| Area | Capability |
|------|-----------|
| Onboarding | QR link screen with live connection state |
| Navigation | Chats, Status, Channels, Communities, Starred, Calls, Contacts, Archived, Scheduled, Settings |
| Search | Full-text message search (FTS) |
| Messaging | Send text, stickers, GIFs, and documents; virtualized timeline; document/sticker/GIF bubbles |
| Message actions | React, star, save sticker, save GIF, forward, message info, reactions, delete |
| Collections | Saved-sticker and saved-GIF pickers |
| Detail views | Group info & members, contact profile, message info (receipts), reactions |
| Privacy & lock | Privacy settings, app lock (PIN) |
| Media | Full-screen media viewer (lightbox) |

## Languages

The UI defaults to **English** and can be switched at runtime (Settings →
Language). Translations are plain JSON dictionaries under `i18n/<code>.json`
(key → string), with English (`en`) as the source and fallback. **English**,
**Indonesian** (`id`), and **Spanish** (`es`) are bundled; add another language
by dropping a `<code>.json` file that uses the same keys.

## Architecture

```
┌──────────────────┐   NDJSON over Unix socket   ┌──────────────────────┐
│  Qt 6 / QML      │ ◄─────────────────────────► │  whatslite-engine    │
│  walite-qt       │   {@type, @extra, args}     │  (Go, headless)      │
│                  │   + wa:* event stream       │  whatsmeow + SQLite  │
│  WaEngineClient  │                             │  + IPC bridge        │
│  AppController   │   media over HTTP (loopback)│  + media server      │
│  JsonListModel   │ ◄───── /media /sticker ─────┤                      │
│  main.qml        │                             │                      │
└──────────────────┘                             └──────────────────────┘
```

| Component | Responsibility |
|-----------|----------------|
| `WaEngineClient` | `QLocalSocket` transport: request/response correlated by `@extra`, `wa:*` events surfaced as signals (TDLib-style: send from any thread, single receive loop) |
| `AppController` | Bridges QML and the engine; invokes engine methods, populates models, handles events. Generic `loadInto` / `loadDetail` helpers add new views without new C++ |
| `JsonListModel` | Generic list model (`QJsonArray` → QML role `m`), reused across all lists |
| `whatslite-engine` | Headless engine host (`cmd/whatslite-engine`): opens the bridge socket and media server |

## Requirements

- Qt **6.9+** (color emoji via COLRv1)
- CMake 3.16+
- A C++17 compiler

## Build

```sh
cmake -B build -S .
cmake --build build -j
```

## Run

```sh
# 1) Start the headless engine (from the repository root)
go build -o whatslite-engine ./cmd/whatslite-engine
./whatslite-engine &

# 2) Launch the Qt frontend against the engine's bridge socket
./build/walite-qt ~/.local/share/whatslite/bridge.sock
```

On first launch a QR screen appears — link it from **WhatsApp → Linked Devices →
Link a device**. The session is stored locally, so later launches reconnect
automatically.
