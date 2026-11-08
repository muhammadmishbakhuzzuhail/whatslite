# wa-rs Parity with whatsapp-rust 0.7.0 — Design (v2)

## Intent

Bring the full runtime feature set of `oxidezap/whatsapp-rust` 0.7.0
(`f077a5d`) into `homun-app/wa-rs` 0.2.0, preserving the current repository
tree and keeping `bridge/` and `desktop/` compiling throughout.

## Goals

- Full runtime parity: auth, passkey, mobile sessions, messaging, media,
  groups, communities, newsletters, status, polls, events, stickers, business,
  bots, labels, favorites, quick replies, comments, calls/VoIP, app-state and
  history sync, ADV, durability, plugins, metrics, tracing, runtime
  abstraction, and platform adapters.
- Keep the current top-level tree.
- Keep runtime code in Rust. `.proto`, `.sql`, `.json`, `.yml` and Dockerfiles
  are data/schema/tooling.
- Keep `bridge/` and `desktop/` compiling at every phase gate.
- Keep upstream feature names as the canonical gates and expose `parity-*`
  umbrellas as aliases.

## Non-goals

- Porting upstream CI, Dockerfile, oracle tools, benchmark harnesses, or test
  vector generators into runtime crates.
- Adding JavaScript, Python, Kotlin, or Java runtime code.
- Maintaining a second upstream backend alongside wa-rs.
- Proving real WhatsApp server compatibility from synthetic tests.
- Porting `bench-harness`, `test-support`, `session-simulator`, `dhat-heap` or
  `bench-internals` features.

## Constraints

- Current toolchain: Rust 1.99.1. Upstream MSRV is 1.95.
- SQLite and TLS/ring may remain C/asm-backed. “Full Rust” means
  application/library code, not zero C dependencies.
- `waproto/` and the `wa-rs-proto` package name stay. The codegen backend
  changes internally to buffa.
- `voip`, `voip-mlow`, `mobile-core`, `passkey`, `plugins`, `metrics` and
  `tracing` remain opt-in.
- `bridge/` and `desktop/` are consumers, not forks of the core API.

## Selected Approach

**A. Strangler port per subsystem.**

Port upstream modules into the existing tree one subsystem at a time, behind
upstream feature names and compatibility shims. Keep the current `wa-rs` API
working while adding upstream APIs under `src/features`.

Rejected alternatives:

- **B. Rebase onto upstream 0.7.0** — fastest, but changes the public API and
  tree and makes phased review and bridge compatibility hard.
- **C. Dual-stack adapter** — duplicates state, Signal sessions, protobuf types
  and runtime ownership.

## Architecture

### Tree mapping

```text
whatslite/
├─ wacore/
│  ├─ src/
│  │  ├─ runtime.rs, runtime/{clocked,crypto}.rs
│  │  ├─ time.rs, time/clock.rs
│  │  ├─ stats.rs, telemetry.rs, diagnostics.rs, sync_marker.rs
│  │  ├─ adv/, client/, session_engine/, sync_engine/
│  │  ├─ message_processing.rs, message_edit.rs, msg_secret.rs
│  │  ├─ reaction.rs, media_retry.rs, webp.rs, zip.rs, sticker_pack.rs
│  │  ├─ bot_message.rs, comment.rs, messages.rs, secret_enc_addon.rs
│  │  ├─ poll.rs, voip/, voip_control/, iq/
│  │  └─ ...
│  ├─ appstate/
│  ├─ binary/
│  ├─ derive/
│  ├─ libsignal/
│  └─ noise/
├─ waproto/
├─ src/
│  ├─ features/               # 34 non-test feature modules
│  ├─ client/                 # lifecycle, builder, node I/O, iq ops, ...
│  ├─ message/                # commit batch, dispatch, durability, ...
│  ├─ send/                   # actions, requests, group repair, tctoken
│  ├─ download/, media.rs, upload.rs
│  ├─ handlers/               # basic, iq, message, receipt, call, notification
│  ├─ plugins/
│  ├─ types/                  # admission, durability, enc handler
│  └─ ...
├─ bridge/
├─ desktop/
├─ storages/
├─ transports/
└─ http_clients/
```

No new top-level runtime crate is introduced.

### Protobuf backend

`waproto/` currently uses prost. Upstream uses buffa 0.9.2 view/owned types.
Switch `waproto/` internally to buffa while keeping the package name and path.

The upstream pipeline includes `whatsapp.desc`, `whatsapp.desc.sha256`,
`build_support/`, `build/`, `api.snapshot`, `signal-storage.snapshot`, and a
large `src/lib.rs` compatibility layer. These are part of the port, not
optional extras.

Fallback: keep prost and adapt upstream modules one by one. This is a
higher-cost fallback, not the default.

### Crate names and aliases

Upstream code uses `wacore::`, `wacore_binary::`, `waproto::`,
`wacore_derive::`, `wacore_appstate::`, `wacore_noise::`,
`wacore_libsignal::`, and `whatsapp_rust::`. The current tree uses
`wa_rs_core::`, `wa_rs_binary::`, `wa_rs_proto::`, and so on.

The migration uses Cargo dependency renaming plus self-aliases:

- Root and consumer `Cargo.toml` depend on the packages under upstream names:
  `wacore = { package = "wa-rs-core", ... }`,
  `wacore_binary = { package = "wa-rs-binary", ... }`,
  `waproto = { package = "wa-rs-proto", ... }`.
- Each crate adds `extern crate self as <upstream_name>;`.
- Existing imports are mechanically renamed once in Phase 0.

This lets ported upstream files keep their original `use wacore::...` paths.

### Feature contract

Upstream feature names are canonical. `parity-*` features are additive
umbrellas and aliases; they never replace the upstream gates.

Root canonical features (runtime scope):

- `sqlite-storage`, `sqlite-storage-bundled`, `tokio-transport`,
  `tokio-runtime`, `tokio-native`, `ureq-client`, `signal`,
  `danger-skip-tls-verify`
- `instance-clock`, `legacy-session-interop`, `client-lifecycle`, `plugins`,
  `passkey`, `mobile-core`, `mobile-transport-native`,
  `generated-group-notifications`, `tracing`, `metrics`, `tracing-pii`, `js`
- `voip-runtime`, `voip-control`, `voip-engine-wacore`, `voip-relay-native`,
  `voip`, `voip-encoded`, `voip-mlow`, `voip-libopus`
- `debug-snapshots`

Core canonical features:

- `mobile`, `instance-clock`, `legacy-session-interop`, `tracing`, `metrics`,
  `tracing-pii`, `js`, `voip-control`, `voip`, `voip-mlow`,
  `debug-snapshots`

Parity aliases:

```toml
parity-core = ["instance-clock", "legacy-session-interop"]
parity-messaging = ["parity-core"]
parity-social = ["parity-messaging"]
parity-voip = ["voip"]
parity-mobile = ["mobile-core"]
parity-plugins = ["plugins", "client-lifecycle"]
parity-observability = ["tracing", "metrics"]
parity-wasm = ["js"]
```

Rules:

- Ported code keeps the upstream `#[cfg(feature = "...")]` attributes.
- `parity-*` only adds edges; it does not rename gates.
- Core modules are always compiled, matching upstream. Only `mobile`,
  `voip-control`, `voip`, `voip-mlow`, `tracing`, `metrics`, `js` and
  `instance-clock` gate optional behavior.
- Excluded features: `bench-harness`, `test-support`, `session-simulator`,
  `dhat-heap`, `bench-internals`.

### Compatibility

- Keep existing `wa-rs` public modules and re-exports.
- Add upstream-style modules under `src/features`.
- Add deprecated adapters where an upstream API replaces an existing one.
- Keep `bridge/` on the default feature set until Phase 5.
- Keep `desktop/` frontend-only.

## Phases

1. **Phase 0 — Foundation:** crate aliases, feature contract, runtime/time/
   stats/telemetry/diagnostics/sync_marker, `WireEnum`, buffa waproto, event
   model and delivery, compatibility tests.
2. **Phase 1 — Auth/session:** `wacore/src/client`, session/sync engines, ADV,
   identity, companion registration, shortcake, passkey, pair-code lifecycle,
   client lifecycle/builder/node I/O/offline resume, admission and durability
   types, mobile core.
3. **Phase 2 — Messaging/media/reliability:** message processing, edits,
   secrets, reactions, `src/message`, `src/send`, retry, receipts, PDO,
   history sync, request/usync, media streaming/retry/reupload, WebP, ZIP,
   sticker pack, durability, inbound queue, caches.
4. **Phase 3 — Social:** all 34 `src/features` modules, including `creation`,
   `group_history`, `mex`, `tctoken`, `stanza`, `signal`, `rotate_key`, and
   upgrades to `blocking`, `chatstate`, `contacts`, `presence`; IQ domains,
   notification handlers, and `src/handlers` coverage.
5. **Phase 4 — Calls/VoIP:** `voip-control`, `voip`, `voip-mlow`,
   `voip-runtime`, `voip-engine-wacore`, `voip-relay-native`,
   `voip-encoded`, `voip-libopus`, call handlers, call log, call links,
   screen sharing, MLOW/Opus, H.264 frames.
6. **Phase 5 — Platform/observability:** plugins, client lifecycle,
   metrics/tracing/diagnostics, SQLite bundled/WASM/mobile, TCP transport,
   feature matrix, parity audit.

## Risks

| Risk | Mitigation |
|---|---|
| buffa migration | Phase 0 only; descriptor and snapshot tests; prost fallback |
| crate name mismatch | Cargo dependency aliases plus self-aliases in Phase 0 |
| bridge API breakage | compatibility facade; bridge smoke tests each phase |
| upstream external codegen (`whatspec`/`unwasm`) | vendor generated Rust; never runtime dependency |
| feature explosion | upstream names plus isolated parity umbrella checks |
| type/proto conflicts | adapters and deprecated re-exports |
| synthetic tests only | protocol fixtures and bridge integration tests |
| tree drift | new modules only inside existing crates |
| interface drift | every task names the exact upstream type, not an invented one |

## Definition of Done

- All upstream runtime feature modules exist under the current tree.
- `cargo build`, `cargo test --workspace` and `cargo check --workspace` pass.
- `bridge/` and `desktop/` compile on the default feature set.
- Each parity phase has tests and a documented feature gate.
- Event surface matches upstream: 80 `EventKind` and 80 `Event` variants.
- No JS/Python/Kotlin/Java runtime code is introduced.
- Remaining non-Rust artifacts are only data/schema/build tooling.

## Approved Decisions

1. Switch `waproto` internally from prost to buffa, keeping the package name
   and path. Fallback to prost only if the Phase 0 gate cannot be met.
2. Exclude upstream dev tools, CI, Dockerfile and oracle harnesses from runtime
   parity. Generated Rust is vendored; the generators stay out of runtime
   crates.
3. Keep the current `wa-rs` public API as a compatibility facade while adding
   upstream-style APIs under `src/features`.
4. Use upstream feature names as canonical; `parity-*` features are additive
   aliases.
5. Keep core modules always compiled, matching upstream; only optional
   modules and dependencies are feature-gated.
6. Use Cargo dependency aliases and `extern crate self as ...` so ported
   upstream files keep their original crate paths.
