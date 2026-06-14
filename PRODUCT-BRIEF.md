# Product Brief — WhatsApp Lite (lightweight WhatsApp desktop client for Linux)

> Product-direction document. Output of the PROJECT OVERVIEW + BRAINSTORM + TARGET MARKET sessions.
> Status: **draft v2** · Date: 2026-06-02 · **Stack changed**: from Gio (native custom-drawn) to
> **web (HTML/CSS/JS) in the system WebView via Wails (Go)**. Full rationale in Section 12.

---

## 0. TL;DR

A WhatsApp desktop client for Linux that's **lightweight & efficient**, built on **whatsmeow** (the WhatsApp
Web multi-device protocol directly over WebSocket). The UI = **web (HTML/CSS/JS)** rendered in the **system
WebView (WebKitGTK), not a bundled Chromium** — so it's as light as the native macOS app, and far lighter than
WhatsApp Web/Windows. It fills the gap left by the absence of an official WhatsApp app on Linux.

- **Not** a commercial product — a **low-profile community open-source project**.
- **Not** full parity with the macOS app — calls & payments are **impossible** (protocol limits).
- **But** it mirrors WhatsApp macOS's **everyday features + UI/UX** — and can now be **pixel-identical** because the UI = web (CSS).
- Realistic lightness target: **~120–250 MB loaded** (on par with native macOS, ~3–6× lighter than WhatsApp Web).
- **Stack: Go (whatsmeow) + Wails + system WebView + HTML/CSS/JS** (see Section 12).
- **Main differentiator = a lean, optimized architecture** that closes the WebView memory gap (see Section 12.3).

---

## 1. Problem

- **WhatsApp Web** (in a browser) is wasteful: ~300–500 MB idle, **1–2 GB** loaded — because it carries a full browser engine.
- **The official WhatsApp Windows app (late 2025)** turned into a **WebView2/Chromium** wrapper → still heavy
  (~250–450 MB idle, ~0.8–1.5 GB loaded). Many users were disappointed.
- **Linux has no official WhatsApp desktop app at all** → users are forced to use WhatsApp Web
  (Chromium again) or a third-party Electron wrapper (heavy again).

**Key insight:** the weight of WhatsApp desktop is **not** from its features, but from the **BUNDLED browser
engine** (Electron/WebView2 each haul in their own Chromium + Node). Proof: the **native macOS** app is ~3–4×
lighter than the Windows version. **The lesson for us:** we may use a web UI, **as long as we don't bundle
Chromium** — use **the WebView already present on the system (WebKitGTK)**. That sheds ~70–80% of Chromium's
weight while still giving an identical look and the convenience of the web. The remaining weight (the system
WebKit engine, ~80–150 MB) we close through **architectural discipline** (Section 12.3).

---

## 2. Solution

Engine = whatsmeow (Go, MPL-2.0 licensed, already handles media & Signal encryption, small RAM footprint).
Frontend = **web (HTML/CSS/JS)** rendered in the **system WebView (WebKitGTK)** via **Wails** (a Go shell). A
single Go binary holds the engine + shell; the web UI is embedded. A TUI is an optional bonus for later.

**Why move from native (Gio) to web?** Three deciding reasons: (1) an **identical WhatsApp look** is nearly
impossible to hand-draw in Gio but easy in CSS — WhatsApp Web is itself web; (2) the UI matches the
**maintainer's web skills**, not low-level immediate-mode rendering; (3) there's a **feedback loop** (open it in
a browser, see it instantly). The cost: higher RAM than Gio — closed by the architecture (Section 12.3).

### Two separate axes: "lightweight" vs "complete" (a core concept to understand)

```
LIGHTWEIGHT? ← determined by whether the browser engine is BUNDLED
             → system WebView (not a bundled Chromium) → on par with native macOS,
               far below WhatsApp Web/Windows  ✅

COMPLETE?    ← determined by PROTOCOL ACCESS (having Meta's code vs not)
             → WE CANNOT match macOS (no calls, etc.)  ❌
```

The macOS app is light **because it doesn't bundle a browser** (which we can mimic via the system WebView)
**and** complete **because it's a Meta app** (which we can't mimic). We get half of that luck — and that half
(lightweight + easy-to-mimic look) **is exactly what the market needs most.**

---

## 3. Selling points (in priority order)

1. **Lightweight & efficient** — on par with the native macOS app, **~3–6× leaner than WhatsApp Web/Windows**
   (which bundle Chromium). Plus a ~30–50 MB binary & **one extra process**, not 5–10 processes. This is the headline.
2. **Linux-first** — filling a gap, not competing with an official app.
3. **Lean, optimized architecture** — local-first, virtualized, no telemetry/background service (Section 12.3).
4. **Open-source & auditable** — important for an app that holds your private messages.
5. **Keyboard-first / scriptable** — relevant to the terminal persona.

---

## 4. Target metrics (estimates; MUST be re-measured before becoming public claims)

| Metric | Web/Chrome | Windows (WebView2) | macOS (Catalyst) | **Target Linux (us)** |
|---|---|---|---|---|
| Idle RAM | ~300–500 MB | ~250–450 MB | ~120–250 MB | **~120–200 MB** |
| Fully loaded RAM | ~1.0–2.0 GB | ~0.8–1.5 GB | ~300–600 MB | **~200–400 MB** |
| Browser engine **bundled** | Yes | Yes | No | **No (uses the system WebKitGTK)** |
| Process count | 5–10+ | 4–8 | 1–2 | **2 (Go + WebView)** |
| Install size | (browser) | ~150–300 MB | ~150–250 MB | **~30–50 MB (1 binary, WebKit borrowed from the system)** |

> ⚠️ The Web/Windows/macOS columns are **representative estimates** and vary widely by machine/number of chats.
> Our target is now **on par with native macOS** (no longer the "tens of MB" of the Gio plan) — a conscious
> consequence of choosing a web UI. Before this becomes public material, **measure it directly** (PSS) with a
> consistent methodology.

### Anatomy of the weight (where the RAM goes & our strategy)

| Component | Chromium (Web/Win) | macOS native | **Us (system WebKit)** |
|---|---|---|---|
| Browser render engine | 150–400 MB (Blink, **bundled**) ⚠️ | 0 (borrows from the OS) | **~80–150 MB (WebKit, borrowed from the system)** |
| JS engine + heap | 100–300 MB (V8) ⚠️ | 0 | **small — lean JS, no heavy framework** ✅ |
| Extra runtime (Node) | 30–80 MB | 0 | **0 — engine = compiled Go** ✅ |
| Media cache (decoded) | 100–500 MB | controlled | **we control it** (files on disk, LRU) ✅ |
| Message/chat state | 20–80 MB | 10–40 MB | **10–40 MB** (local-first SQLite) ✅ |
| Telemetry/background svc | yes | yes | **0** ✅ |

**Our difference vs Chromium-based:** the render engine **borrows from the system** (not bundled) + **no Node** +
**lean JS** → sheds ~60–75% of the weight. **Our difference vs native macOS:** only the WebKit-engine delta
(~80–150 MB), which we shrink through the discipline in rows 2–6 (Section 12.3). The biggest risk is still the
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
- **UI/UX style: CAN BE PIXEL-IDENTICAL.** Because the UI = web (HTML/CSS), the WhatsApp look (which is itself web)
  can be mirrored exactly — not just "similar in feel" like the old Gio plan. Fixed look, built-in light/dark.
- **Lightness: ON PAR with native macOS** (not better — that's a consequence of the WebView). But **far below**
  WhatsApp Web/Windows, and we win on **disk footprint, process count, no-telemetry, lean architecture**.
- **What STAYS different / lacking:**
  - Not Meta's official assets — we rewrite the CSS/markup ourselves (can be very similar, but not copying Meta's files).
  - Calls/pay are gone (impossible), history sync is limited, some 🟡 features are partial, and depend on protocol changes.

> **Target:** *"WhatsApp macOS's everyday functionality + (nearly) pixel-identical UI/UX, as light as native, far
> below Chromium"*. What we mirror = layout, style, flow (now cheap via CSS). What we DON'T = calls (impossible).

---

## 7. Roadmap & estimates (solo/community scale)

Architecture: **the engine (whatsmeow + state + SQLite) is separate from the web frontend.** Engine = a pure Go
package (ready to become a daemon/headless), frontend = HTML/CSS/JS in the WebView, bridged by Wails. The
frontend is a **web GUI from the start** (WhatsApp macOS-style look). A TUI is an optional bonus, well behind.

| Phase | Scope | Effort | Target RAM |
|---|---|---|---|
| **v0.1 — Minimal daily-driver** | whatsmeow engine + basic web UI; QR pairing, chat list, send/receive text, receive media (click→open externally), reply, reactions, read receipts, typing, alerts, basic history (SQLite); light/dark | a few weeks–2 months | ~120–180 MB |
| **v0.2 — Media & groups** | inline images + thumbnails, voice notes, documents, full groups, mentions, edit/delete | ~1 month+ | ~150–250 MB |
| **v0.3 — Social & organization** | Status, polls, pin/archive/mute, local search (FTS), profile photos, blocking | a few weeks+ | ~180–300 MB |
| **v0.4 — Advanced** | stickers (static→animated), inline video (libmpv), GIF, multi-account, link preview, Channels (read) | ~1 month+ | ~200–400 MB |
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
> *"A lightweight, efficient WhatsApp client for Linux — no bundled Chromium, as light as the native macOS app —
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

- ✅ **No bundled Chromium** — use the **system WebView (WebKitGTK)**. (Revised from "no webview at all":
  a web UI in the system WebView was chosen for an identical look + the maintainer's web skills; see Section 12.)
- ✅ Linux-first; Windows = a distribution bonus later.
- ✅ A low-profile community FOSS project; the "market" = a community.
- ✅ Voice/video calls & payments = a **permanent non-goal** (impossible, protocol limits).
- ✅ Not full feature parity; the anchor = **"everyday functionality + (nearly pixel-identical) macOS UI/UX + as light as native"**.
- ✅ A **separate engine + frontend** architecture (engine = a pure Go package, ready to become a daemon).
- ✅ Frontend: **engine first → web UI (Wails) as the primary frontend → TUI as a later bonus**.
- ✅ UI stack = **web (HTML/CSS/JS) + Wails (Go shell) + WebKitGTK**. Gio/Electron/Tauri/GTK/Qt are **rejected** (Section 12).
- ✅ Style = **WhatsApp macOS-style look, no user theming**; built-in light/dark only.
- ✅ **Differentiator = a lean, optimized architecture** (Section 12.3) to close the WebView overhead.
- ✅ v0.1 scope = **a minimal "everyday text chat"** (listed in Section 7).
- ✅ v0.1 media strategy = **click → open in an external app** (inline deferred to v0.2).
- ✅ Packaging = **AUR + Flatpak + AppImage** (portable across all distros, X11 + Wayland).

## 11. Still to be decided (left for implementation)

1. **Final project name & license:** the project ships as **GPL-3.0** (community protection). whatsmeow is
   **MPL-2.0**, which is GPL-compatible — MPL-2.0 §3.3 permits distributing the larger work under the GPL.
2. **RAM measurement methodology** to validate the claims in Section 4 before they become public material.
3. **ToS/legal stance in the README:** how explicit to make the ban-risk disclaimer.
4. **Web UI framework:** vanilla JS vs a lightweight compiled framework (Preact/Svelte). Default: as minimal as possible.
5. **Heavy media strategy (v0.2+):** libmpv vs ffmpeg vs a system app for video/GIF.
6. **Wails v2 vs v3 vs bare `webview_go`:** Wails for DX/bindings; bare-webview if we want the thinnest possible.

---

## 12. Tech Stack (locked)

### 12.1 Stack

```
Engine     : Go + whatsmeow + SQLite (modernc.org/sqlite, pure-Go)
Architecture : engine = a pure Go package (ready to become a headless daemon),
             separate from the frontend → reusable by a GUI/TUI/daemon
Shell      : Wails (Go ↔ system WebView; Go↔JS bindings, bundling, dev-server)
Frontend   : web — HTML/CSS/JS (vanilla or a lightweight compiled framework)
             embedded in the binary; TUI = an optional bonus well behind
Render     : system WebView — WebKitGTK (Linux). Does NOT bundle Chromium.
Style      : look inspired by WhatsApp macOS (can now be nearly pixel-identical
             via CSS). No user theming. Built-in light/dark.
Packaging  : AUR (Arch/CachyOS) + Flatpak (all distros) + AppImage (optional)
Display    : X11 + Wayland (via Wails's GTK host)
Dependencies : webkit2gtk + gtk3 (present on all mainstream distros)
Rejected   : Electron (bundles Chromium+Node → heavy, against the goal)
             Tauri (also a system WebView, BUT a Rust shell + the Go engine
                    as a sidecar = 2 toolchains + an IPC boundary, zero gain)
             Gio (custom-drawn: identical look nearly impossible, no feedback loop,
                  not a web skill) — the old stack, abandoned
             native GTK/Qt (forces the DE look; not a web skill)
```

### 12.2 Why Wails + web (not Gio, not Tauri, not Electron)

| Reason | Explanation |
|---|---|
| **Identical WhatsApp look** | UI = CSS; WhatsApp Web is itself web → mirrored exactly, not hand-drawn |
| **Uses the maintainer's web skills** | HTML/CSS/JS, not low-level immediate-mode; previewable in a browser |
| **One language for shell+engine** | Wails = a Go shell → whatsmeow lives in the same process, no sidecar/IPC (unlike Tauri's Rust) |
| **No bundled browser** | Uses the system WebKitGTK → sheds ~70–80% of Chromium's weight (unlike Electron) |
| **No second runtime** | Engine = compiled Go, not Node → one ~30–50 MB binary |
| **Portable** | webkit2gtk+gtk3 are on all mainstream distros; X11 & Wayland |

### 12.3 Architectural advantages & optimizations — THE MAIN DIFFERENTIATOR ⭐

The WebView puts us ~80–150 MB above Gio (the WebKit engine). **We close that gap — and beat the official
app — through architecture**, not by hoping for a lighter toolkit. This is the project's technical selling point.

**A. Leaner than WhatsApp Web/Windows (Chromium-based):**
- **System WebView, not a bundled Chromium** — rendering borrows from the OS; zero MB of browser in our binary.
- **No Node/JS server runtime** — the "backend" = compiled Go, not a 30–80 MB Node process.
- **Lean JS frontend** — vanilla or a compiled framework (Preact/Svelte), **not** React + a fat bundle.
- **Zero telemetry / analytics / background service** — the official app runs background services; we don't.
- **One binary, ~2 processes** (Go + WebView) vs Chromium's 5–10 processes.

**B. Closing the gap vs native macOS (discipline on the data side — the dominant RAM share):**
- **Virtualized message list** — only visible DOM nodes exist; thousands of messages ≠ thousands of elements.
  This keeps the WebKit heap small (a large DOM = the #1 web memory killer).
- **Local-first SQLite** — the chat list & messages are read from the local DB, **paginated ~50** + lazy-loaded on scroll;
  the full history never enters RAM.
- **Media = files on disk, the DB stores the path** — **not** base64 in the DOM (memory killer #2); thumbnails are lazy,
  full images on-demand, released on leaving the viewport; a **bounded LRU cache**.
- **Video/GIF/animated stickers → delegated** to libmpv / a system app; codecs aren't pulled into the WebView.
- **Event-driven (delta) updates** — a new message is sent Go→JS as a single event & inserted, **not** a reload
  of the whole view; minimal allocation & re-layout.
- **Internal assets**: SVG icons (small), wallpaper via CSS, **system fonts** (no bundled megabytes of fonts).

**C. Frugal when idle / hidden:**
- When the window is hidden: stop UI rendering/animation; only the **Go WebSocket** needs to be alive (a cheap engine).
- No wasteful polling; the UI only wakes on an event or interaction.

**An honest, marketable claim:** *"as light as the native macOS app, 3–6× leaner than WhatsApp Web, a ~30–50 MB
binary, two processes, zero telemetry"* — **not** "tens of MB" (that was the Gio dream we've let go). All numbers
**must be measured (PSS)** before becoming public material.

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
