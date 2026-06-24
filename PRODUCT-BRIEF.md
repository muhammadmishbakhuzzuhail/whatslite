# Product Brief — WhatsLite (lightweight WhatsApp desktop client for Linux)

> Product-direction document. Output of the PROJECT OVERVIEW + BRAINSTORM + TARGET MARKET sessions.
> Status: **draft v2** · Date: 2026-06-02 · **Stack (final)**: a single pure-Go binary using the
> **Gio immediate-mode GUI toolkit (gioui.org)**, with the whatsmeow + SQLite engine running **in-process**.
> Full rationale in Section 12.

---

## 0. TL;DR

A WhatsApp desktop client for Linux that's **lightweight & efficient**, built on **whatsmeow** (the WhatsApp
Web multi-device protocol directly over WebSocket). The UI = a **native Go immediate-mode GUI** drawn with the
**Gio toolkit (gioui.org)** — **no browser engine, no Chromium, no WebView at all** — so it's lighter than the
native macOS app, and far lighter than WhatsApp Web/Windows. It fills the gap left by the absence of an official
WhatsApp app on Linux.

- **Not** a commercial product — a **low-profile community open-source project**.
- **Not** full parity with the macOS app — calls & payments are **impossible** (protocol limits).
- **But** it mirrors WhatsApp macOS's **everyday features + UI/UX** — a faithful, custom-drawn look-alike of the desktop layout.
- Realistic lightness target: **tens of MB loaded** (well below native macOS, far below WhatsApp Web).
- **Stack: a single pure-Go binary — whatsmeow + SQLite engine in-process + a Gio native UI** (see Section 12).
- **Main differentiator = a lean, native architecture** with no web stack to drag along (see Section 12.3).

---

## 1. Problem

- **WhatsApp Web** (in a browser) is wasteful: ~300–500 MB idle, **1–2 GB** loaded — because it carries a full browser engine.
- **The official WhatsApp Windows app (late 2025)** turned into a **WebView2/Chromium** wrapper → still heavy
  (~250–450 MB idle, ~0.8–1.5 GB loaded). Many users were disappointed.
- **Linux has no official WhatsApp desktop app at all** → users are forced to use WhatsApp Web
  (Chromium again) or a third-party Electron wrapper (heavy again).

**Key insight:** the weight of WhatsApp desktop is **not** from its features, but from the **BUNDLED browser
engine** (Electron/WebView2 each haul in their own Chromium + Node). Proof: the **native macOS** app is ~3–4×
lighter than the Windows version. **The lesson for us:** drop the browser entirely — **no Chromium, no system
WebView, no web layer at all** — and draw the UI natively with **Gio**, a pure-Go immediate-mode toolkit. That
sheds the full weight of a browser engine while still giving us a faithful WhatsApp-style look. What little
weight remains is plain native rendering, which we keep small through **architectural discipline** (Section 12.3).

---

## 2. Solution

Engine = whatsmeow (Go, MPL-2.0 licensed, already handles media & Signal encryption, small RAM footprint).
Frontend = a **native immediate-mode UI** drawn with **Gio (gioui.org)**, running **in-process** with the
engine. A single Go binary holds everything — one process, one window, no IPC bridge, no WebView, no Node. A
TUI is an optional bonus for later.

**Why native (Gio)?** Three deciding reasons: (1) **no bundled or borrowed browser engine** — the lightest
possible footprint, the project's whole reason for existing; (2) **one language, one process** — the UI and the
whatsmeow engine are the same Go program, no shell, no IPC, no sidecar; (3) a faithful WhatsApp-style desktop
layout can be **drawn directly** with full control over rendering. The cost: the look is hand-built rather than
borrowed from CSS — paid down by careful component work (Section 12.3).

### Two separate axes: "lightweight" vs "complete" (a core concept to understand)

```
LIGHTWEIGHT? ← determined by whether a browser engine is present at all
             → no Chromium, no WebView — native Gio rendering → below native macOS,
               far below WhatsApp Web/Windows  ✅

COMPLETE?    ← determined by PROTOCOL ACCESS (having Meta's code vs not)
             → WE CANNOT match macOS (no calls, etc.)  ❌
```

The macOS app is light **because it doesn't bundle a browser** (which we beat by having no browser at all)
**and** complete **because it's a Meta app** (which we can't mimic). We get half of that luck — and that half
(lightweight + a faithful look) **is exactly what the market needs most.**

---

## 3. Selling points (in priority order)

1. **Lightweight & efficient** — below the native macOS app, **far leaner than WhatsApp Web/Windows**
   (which bundle Chromium). Plus a ~30–50 MB binary & **a single process**, not 5–10. This is the headline.
2. **Linux-first** — filling a gap, not competing with an official app.
3. **Lean, native architecture** — single-process, local-first, virtualized, no telemetry/background service (Section 12.3).
4. **Open-source & auditable** — important for an app that holds your private messages.
5. **Keyboard-first / scriptable** — relevant to the terminal persona.

---

## 4. Target metrics (estimates; MUST be re-measured before becoming public claims)

| Metric | Web/Chrome | Windows (WebView2) | macOS (Catalyst) | **Target Linux (us)** |
|---|---|---|---|---|
| Idle RAM | ~300–500 MB | ~250–450 MB | ~120–250 MB | **~30–80 MB** |
| Fully loaded RAM | ~1.0–2.0 GB | ~0.8–1.5 GB | ~300–600 MB | **~120–250 MB** |
| Browser engine **bundled** | Yes | Yes | No | **No browser at all (native Gio rendering)** |
| Process count | 5–10+ | 4–8 | 1–2 | **1 (a single Go binary)** |
| Install size | (browser) | ~150–300 MB | ~150–250 MB | **~30–50 MB (1 self-contained binary)** |

> ⚠️ The Web/Windows/macOS columns are **representative estimates** and vary widely by machine/number of chats.
> With no browser engine to carry, our target is **below native macOS** — the upside of a native Gio UI. Before
> this becomes public material, **measure it directly** (PSS) with a consistent methodology.

### Anatomy of the weight (where the RAM goes & our strategy)

| Component | Chromium (Web/Win) | macOS native | **Us (native Gio)** |
|---|---|---|---|
| Browser render engine | 150–400 MB (Blink, **bundled**) ⚠️ | 0 (borrows from the OS) | **0 — no browser; Gio draws via the GPU directly** ✅ |
| JS engine + heap | 100–300 MB (V8) ⚠️ | 0 | **0 — no JS, no DOM** ✅ |
| Extra runtime (Node) | 30–80 MB | 0 | **0 — UI + engine = one compiled Go binary** ✅ |
| Media cache (decoded) | 100–500 MB | controlled | **we control it** (files on disk, LRU) ✅ |
| Message/chat state | 20–80 MB | 10–40 MB | **10–40 MB** (local-first SQLite) ✅ |
| Telemetry/background svc | yes | yes | **0** ✅ |

**Our difference vs Chromium-based:** **no browser engine, no JS heap, no Node** → sheds the bulk of the weight.
**Our difference vs native macOS:** we carry no browser at all, so the only meaningful RAM is the rows below the
render engine, which we shrink through the discipline in rows 4–6 (Section 12.3). The biggest risk is still the
**media cache**.

---

## 5. Feature analysis (whatsmeow support)

Legend: ✅ supported · 🟡 partial/needs work · ❌ impossible (protocol limit) · UI weight: 🟢 light / 🟡 medium / 🔴 needs discipline

| Feature | whatsmeow | Weight | Notes |
|---|---|---|---|
| **Core messaging** |
| Text, emoji | ✅ | 🟢 | Core |
| Reply/quote, reactions, edit, delete/revoke, mentions | ✅ | 🟢 | |
| Disappearing/ephemeral | ✅ | 🟢 | |
| View-once | 🟡 | 🟢 | Respect the semantics |
| Scheduled messages | ❌* | 🟢 | Not in the protocol; can be emulated locally |
| **Media** |
| Images | ✅ | 🔴 | Decoding = the #1 RAM source |
| Video | ✅ | 🔴 | Needs a player (libmpv/ffmpeg) |
| Documents | ✅ | 🟢 | |
| Voice notes (PTT), audio | ✅ | 🟡 | Opus |
| Static stickers | ✅ | 🟡 | WebP |
| Animated stickers | 🟡 | 🔴 | Animated WebP, expensive to render |
| GIF | 🟡 | 🔴 | MP4 |
| Static location, contacts (vcard) | ✅ | 🟢 | |
| Live location | 🟡 | 🟡 | |
| Link preview | 🟡 | 🟡 | We generate it ourselves |
| **Groups & communities** |
| Groups: read/send/create/leave/invite link | ✅ | 🟢–🟡 | |
| Groups: manage members/admins | ✅ | 🟡 | |
| Communities | 🟡 | 🔴 | Partial |
| Channels (newsletter) | 🟡 | 🟡 | Read |
| **Social & organization** |
| View/post Status | ✅ | 🟡 | |
| Polls (create & vote) | ✅ | 🟢 | |
| Chat list | ✅ | 🟢 | |
| History (history sync) | 🟡 | 🟡 | **Only as much as the server sends at link time**, not the full history |
| Pin/archive/mute, mark unread | ✅ | 🟢 | App-state sync |
| Message search | 🟡 | 🟡 | **Local** — our own index (SQLite FTS) |
| Read receipts, typing, presence | ✅ | 🟢 | |
| Profile photo/info, block | ✅ | 🟢 | |
| **Account** |
| QR pairing/link device, multi-device | ✅ | 🟢 | |
| Multi-account | ✅ | 🟡 | |
| Alerts | ✅ | 🟢 | Sound alerts only (no desktop notifications, by design) |
| **IMPOSSIBLE (protocol limits)** |
| Voice/video/group call | ❌ | — | Can only *detect* an incoming call, can't make one |
| Screen sharing | ❌ | — | |
| Payments / WhatsApp Pay | ❌ | — | |

\* not an encryption limit, just absent from the WhatsApp Web protocol.

**Summary:** of ~48 features, **~37 ARE doable**, **~5 are IMPOSSIBLE** (all = call/pay). **~85–90% of everyday functionality can be mirrored.**

### The "heavy" parts that need discipline (in order)

1. 🔴 **Image/video cache in RAM** — the #1 risk. → decode on-demand, release on leaving the viewport, cache to disk not RAM.
2. 🔴 **Video & GIF playback** — → delegate to libmpv/ffmpeg / a system app.
3. 🔴 **Animated stickers** — → cap the fps, static first.
4. 🟡 **Initial history sync** — → store it in SQLite, don't hold it in RAM.
5. 🟡 **Naive search** — → an FTS index on disk.

**The golden rule:** *Lightweight isn't about having few features — it's about discipline around media & memory.*

---

## 6. Will it be "exactly like" macOS? (Expectations)

**Similar in function & style — YES, that's the goal. 100% identical — no (and it doesn't need to be).**

Product decision: **copy WhatsApp macOS's everyday features + UI/UX style** as a **fixed look** (sidebar+chat
layout, rounded bubbles, spacing, UX flow). This is legitimate and common — Telegram/Discord/Spotify all keep a
fixed *brand* look that doesn't follow the OS theme, and Linux users use them without issue.

- **Everyday functionality (text, media, groups, status, etc.): ON PAR.** ~85–90% of daily needs are met.
- **UI/UX style: A FAITHFUL LOOK-ALIKE.** The UI is custom-drawn in Gio to closely mirror the WhatsApp desktop
  layout — sidebar+chat, rounded bubbles, spacing, UX flow. Fixed look, built-in light/dark.
- **Lightness: BELOW native macOS** (we carry no browser engine at all). And **far below** WhatsApp Web/Windows,
  and we win on **disk footprint, process count, no-telemetry, lean native architecture**.
- **What STAYS different / lacking:**
  - Not Meta's official assets — we draw our own widgets and icons (can be very close, but not copying Meta's files).
  - Calls/pay are gone (impossible), history sync is limited, some 🟡 features are partial, and depend on protocol changes.

> **Target:** *"WhatsApp macOS's everyday functionality + a faithful look-alike UI/UX, lighter than native, far
> below Chromium"*. What we mirror = layout, style, flow (hand-drawn in Gio). What we DON'T = calls (impossible).

---

## 7. Roadmap & estimates (solo/community scale)

Architecture: **the engine (whatsmeow + state + SQLite) is a self-contained pure Go package** (ready to become a
daemon/headless), with the **Gio native UI running in-process** in the same binary — no IPC, no bridge. The UI
is a **native GUI from the start** (WhatsApp macOS-style look). A TUI is an optional bonus, well behind.

| Phase | Scope | Effort | Target RAM |
|---|---|---|---|
| **v0.1 — Minimal daily-driver** | whatsmeow engine + basic Gio UI; QR pairing, chat list, send/receive text, receive media (click→open externally), reply, reactions, read receipts, typing, alerts, basic history (SQLite); light/dark | a few weeks–2 months | ~40–100 MB |
| **v0.2 — Media & groups** | inline images + thumbnails, voice notes, documents, full groups, mentions, edit/delete | ~1 month+ | ~80–150 MB |
| **v0.3 — Social & organization** | Status, polls, pin/archive/mute, local search (FTS), profile photos, blocking | a few weeks+ | ~100–200 MB |
| **v0.4 — Advanced** | stickers (static→animated), inline video (libmpv), GIF, multi-account, link preview, Channels (read) | ~1 month+ | ~120–250 MB |
| **v0.x — Experimental** | Communities (partial), live location, daemon/headless, TUI, CLI integration | gradual | — |
| **❌ NEVER** | Voice/video calls, screen share, payments | — | — |

- Through **v0.4 (macOS-style + everyday features minus calls):** ~6–12 months solo, faster with contributors.
- The longest & hardest part = the **engine foundation + basic GUI of v0.1**, not the later features.

---

## 8. Target market & positioning

- **Goal = a low-profile community FOSS project, NOT commercial.** Evident from the constraints:
  1. It violates WhatsApp's ToS; users' numbers are at risk of being **banned** → can't be sold soundly.
  2. Meta's protocol shifts constantly → eternal maintenance burden (fine for a community, suicide for an SLA-bound product).
  3. High visibility = an invitation to bans → **a low profile is a survival strategy.**
- **The "market" = a community, not customers.** Success metrics: GitHub stars, contributors, AUR/Flathub/nixpkgs packages,
  HN/r/linux discussion — **not** MAU/revenue.
- **Primary persona:** Linux/terminal power users (who like lightweight, native, FOSS, keyboard-first, and tolerate rough edges).
- **Linux-first is right.** Windows = a **distribution bonus** later, **not** an equal second target
  (Windows users generally don't tolerate fiddly setup/bans and don't value FOSS — not our persona).

**One-sentence positioning:**
> *"A lightweight, efficient WhatsApp client for Linux — no browser engine, lighter than the native macOS app —
> for people who care about their RAM. Open-source, used at your own risk."*

The disclaimer ("at your own risk") = **part of the positioning**, not a footnote. It's both honest and a filter for the right users.

---

## 9. Risks & limitations (don't ignore these)

| Risk | Notes / mitigation |
|---|---|
| **Number banned by Meta** | Real. A clear disclaimer in the README. Consider a **separate dev number** for testing to keep the project alive (separate from the courage to use a main number for actual usage). |
| **Protocol changes** | Depends on whatsmeow's update pace. 🟡 features may shift. |
| **Limited history sync** | Can't get the full old history on first link — set user expectations. |
| **Takedown/DMCA** | Could happen someday. A consequence of the project's nature. |
| **Bus factor / solo maintainer** | If the maintainer's number is banned → velocity dies. → a separate dev number. |

---

## 10. LOCKED decisions

- ✅ **No browser engine at all** — no Chromium, no system WebView. The UI is drawn natively with **Gio**
  for the lightest possible footprint and a single-process design; see Section 12.
- ✅ Linux-first; Windows = a distribution bonus later.
- ✅ A low-profile community FOSS project; the "market" = a community.
- ✅ Voice/video calls & payments = a **permanent non-goal** (impossible, protocol limits).
- ✅ Not full feature parity; the anchor = **"everyday functionality + a faithful macOS look-alike UI/UX + lighter than native"**.
- ✅ A **self-contained pure-Go architecture** (engine = a pure Go package, ready to become a daemon; UI runs in-process).
- ✅ Build: **engine first → native Gio UI as the primary frontend → TUI as a later bonus**.
- ✅ UI stack = **a single pure-Go binary: Gio (gioui.org) UI + whatsmeow/SQLite engine in-process**. Electron/Tauri/Wails+WebView/GTK/Qt are **rejected** (Section 12).
- ✅ Style = **WhatsApp macOS-style look, no user theming**; built-in light/dark only.
- ✅ **Differentiator = a lean, native architecture** (Section 12.3) with no web stack to drag along.
- ✅ v0.1 scope = **a minimal "everyday text chat"** (listed in Section 7).
- ✅ v0.1 media strategy = **click → open in an external app** (inline deferred to v0.2).
- ✅ Packaging = **AUR + Flatpak + AppImage** (portable across all distros, X11 + Wayland).

## 11. Still to be decided (left for implementation)

1. **Final project name & license:** the project ships as **GPL-3.0** (community protection). whatsmeow is
   **MPL-2.0**, which is GPL-compatible — MPL-2.0 §3.3 permits distributing the larger work under the GPL.
2. **RAM measurement methodology** to validate the claims in Section 4 before they become public material.
3. **ToS/legal stance in the README:** how explicit to make the ban-risk disclaimer.
4. **Heavy media strategy (v0.2+):** libmpv vs ffmpeg vs a system app for video/GIF.

---

## 12. Tech Stack (locked)

### 12.1 Stack

```
Engine     : Go + whatsmeow + SQLite (modernc.org/sqlite, pure-Go)
Architecture : engine = a pure Go package (ready to become a headless daemon),
             with the Gio UI running in-process → one binary, one process,
             reusable by a TUI/daemon
Frontend   : Gio (gioui.org) — a pure-Go immediate-mode GUI, drawn directly
             via the GPU; in-process with the engine; TUI = a later bonus
Render     : native Gio (OpenGL/Vulkan via the OS). NO browser, NO WebView,
             NO Chromium — no web layer at all.
Build      : go build ./cmd/whatslite-gio → a single self-contained binary
Style      : look inspired by WhatsApp macOS, custom-drawn in Gio.
             No user theming. Built-in light/dark.
Packaging  : AUR (Arch/CachyOS) + Flatpak (all distros) + AppImage (optional)
Display    : X11 + Wayland (Gio supports both natively)
Dependencies : the usual Gio system libs (e.g. libwayland/X11, GPU drivers);
             NO webkit2gtk, NO Chromium, NO Node/npm
Rejected   : Electron (bundles Chromium+Node → heavy, against the goal)
             Wails / system WebView (still hauls in WebKitGTK + a web layer;
                    a browser engine we don't want)
             Tauri (a Rust shell + the Go engine as a sidecar = 2 toolchains
                    + an IPC boundary, zero gain)
             native GTK/Qt (forces the DE look; extra C deps)
```

> Historical note: earlier drafts of this brief explored a web UI (HTML/CSS/JS in the system WebView via Wails)
> and, briefly, a Qt6/QML frontend. **Both have been fully removed.** The shipping client is Gio-only.

### 12.2 Why Gio (not Wails/WebView, not Tauri, not Electron)

| Reason | Explanation |
|---|---|
| **No browser engine at all** | Pure native rendering — no Chromium, no WebKit, no WebView; the lightest possible footprint |
| **One language, one process** | UI + whatsmeow engine = the same Go program, in-process; no shell, no sidecar, no IPC (unlike Wails/Tauri) |
| **No second runtime** | No Node, no JS, no DOM → one self-contained ~30–50 MB binary |
| **Full control of the look** | The WhatsApp desktop layout is drawn directly in Gio — exact spacing, bubbles, animations |
| **Portable** | Gio targets X11 & Wayland natively; no webkit2gtk/gtk web stack to depend on |

### 12.3 Architectural advantages & optimizations — THE MAIN DIFFERENTIATOR ⭐

Going native (Gio) means we carry **no browser engine at all** — and we keep the rest small **through
architecture**, not by hoping for a lighter toolkit. This is the project's technical selling point.

**A. Leaner than WhatsApp Web/Windows (Chromium-based):**
- **No browser engine** — Gio draws the UI directly via the GPU; zero MB of Chromium/WebKit in our binary.
- **No Node/JS runtime** — the whole program = compiled Go, not a 30–80 MB Node process plus a JS heap.
- **No DOM, no web framework** — native widgets instead of React + a fat bundle.
- **Zero telemetry / analytics / background service** — the official app runs background services; we don't.
- **One binary, a single process** vs Chromium's 5–10 processes.

**B. Going below native macOS (discipline on the data side — the dominant RAM share):**
- **Virtualized message list** — Gio's immediate-mode model only lays out visible rows; thousands of messages ≠
  thousands of retained widgets, keeping the heap small.
- **Local-first SQLite** — the chat list & messages are read from the local DB, **paginated ~50** + lazy-loaded on scroll;
  the full history never enters RAM.
- **Media = files on disk, the DB stores the path** — thumbnails are lazy, full images decoded on-demand and
  released on leaving the viewport; a **bounded LRU cache** of decoded textures.
- **Video/GIF/animated stickers → delegated** to libmpv / a system app; codecs aren't pulled into the binary.
- **Event-driven (delta) updates** — a new message updates the model and invalidates a frame, **not** a reload
  of the whole view; minimal allocation & re-layout.
- **Internal assets**: SVG/vector icons (small), wallpaper drawn directly, **system fonts** (no bundled megabytes of fonts).

**C. Frugal when idle / hidden:**
- Gio is event-driven — with no input or events, no frames are drawn; when hidden, only the **Go WebSocket** stays
  alive (a cheap engine).
- No wasteful polling; the UI only wakes on an event or interaction.

**An honest, marketable claim:** *"lighter than the native macOS app, far leaner than WhatsApp Web, a ~30–50 MB
binary, a single process, zero telemetry"*. All numbers **must be measured (PSS)** before becoming public material.

### 12.4 Storage (SQLite) — locked, designed to be light

**Why it's required:** WhatsApp = a **device-local + E2E** model, the server is only a **relay** — not a cloud
like Telegram. The server does **not** serve old message history on-demand. So sessions, keys, and history
**must** live locally, or else: re-scan the QR every launch + an always-empty chat list. SQLite = the lightest
persistence option (embedded, one file, no server).

**Principle:** *a small, lean DB, media in separate files, RAM kept in check by pagination.*

| Data | Location | Notes |
|---|---|---|
| Session + encryption keys | SQLite (whatsmeow's built-in store) | Required; managed automatically by the library |
| Messages + chat metadata | SQLite (`app.db`) | Lightweight text, on disk |
| Search | FTS5 (a SQLite feature) | No extra dependency |
| **Media** | **Files in a cache folder; the DB stores only the *path*** | ⭐ The DB never bloats |

**Three key decisions to keep it light:**
1. **Media = files on disk, not blobs in the DB** (the DB only stores a reference).
2. **The media cache is bounded (LRU)** — old files are dropped automatically when the size cap is exceeded.
3. **Messages are read incrementally (pagination)** — load the ~50 most recent, lazy-load on scroll; the full history never goes to RAM.

**Details:**
- **Location (XDG):** data in `~/.local/share/<app>/`, media cache in `~/.cache/<app>/`.
- **SQLite:** WAL mode + a bounded `cache_size` → SQLite itself stays RAM-frugal.
- **Encryption at rest:** for v0.1, **file permissions (0600)** are enough; SQLCipher = optional later (avoid premature weight/complexity).
