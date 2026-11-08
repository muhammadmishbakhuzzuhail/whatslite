# wa-rs Parity with whatsapp-rust 0.7.0 — Design

## Intent

Bring the full runtime feature set of `oxidezap/whatsapp-rust` 0.7.0 into
`homun-app/wa-rs` 0.2.0, preserving the current repository tree and keeping
`bridge/` and `desktop/` compiling throughout.

## Goals

- Full runtime parity: auth, passkey, mobile sessions, messaging, media,
  groups, communities, newsletters, status, polls, events, stickers, business,
  bots, labels, favorites, quick replies, comments, calls/VoIP, app-state and
  history sync, ADV, durability, plugins, metrics, tracing, runtime
  abstraction, and platform adapters.
- Keep the current top-level tree.
- Keep runtime code in Rust. `.proto`, `.sql`, `.json`, `.yml` and Dockerfiles
  are data/schema/tooling.
- Keep `bridge/` and `desktop/` compiling.
- Keep parity features opt-in behind feature flags.

## Non-goals

- Porting upstream CI, Dockerfile, oracle tools, benchmark harnesses, or test
  vector generators into runtime crates.
- Adding JavaScript, Python, Kotlin, or Java runtime code.
- Maintaining a second upstream backend alongside wa-rs.
- Proving real WhatsApp server compatibility from synthetic tests.

## Constraints

- Current toolchain: Rust 1.99.1. Upstream MSRV is 1.95.
- SQLite and TLS/ring may remain C/asm-backed. “Full Rust” means
  application/library code, not zero C dependencies.
- `waproto/` and the `wa-rs-proto` crate name stay. The codegen backend may
  change internally.
- `voip`, `voip-mlow`, `mobile-core`, `passkey`, `plugins`, `metrics` and
  `tracing` remain opt-in.
- `bridge/` and `desktop/` are consumers, not forks of the core API.

## Selected Approach

**A. Strangler port per subsystem.**

Port upstream modules into the existing tree one subsystem at a time, behind
feature flags and compatibility shims. Keep the current `wa-rs` API working
while adding upstream APIs under `src/features`.

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
│  │  ├─ runtime.rs
│  │  ├─ time.rs
│  │  ├─ stats.rs
│  │  ├─ telemetry.rs
│  │  ├─ diagnostics.rs
│  │  ├─ adv/
│  │  ├─ identity_verification.rs
│  │  ├─ session_engine/
│  │  ├─ sync_engine/
│  │  ├─ message_processing.rs
│  │  ├─ message_edit.rs
│  │  ├─ msg_secret.rs
│  │  ├─ reaction.rs
│  │  ├─ media_retry.rs
│  │  ├─ poll.rs
│  │  ├─ sticker_pack.rs
│  │  ├─ webp.rs
│  │  ├─ zip.rs
│  │  ├─ voip/
│  │  ├─ voip_control/
│  │  └─ iq/
│  ├─ appstate/
│  ├─ binary/
│  ├─ derive/
│  ├─ libsignal/
│  └─ noise/
├─ waproto/
├─ src/
│  ├─ features/
│  ├─ client/
│  ├─ message/
│  ├─ download/
│  ├─ send/
│  ├─ handlers/
│  └─ plugins/
├─ bridge/
├─ desktop/
├─ storages/
├─ transports/
└─ http_clients/
```

No new top-level runtime crate is introduced.

### Protobuf backend

`waproto/` currently uses prost. Upstream uses buffa view/owned types. The
recommended path is to switch `waproto/` internally to buffa while keeping the
crate name and path.

Reason: upstream feature modules depend on buffa view types and
`buffa-build` descriptor handling. Rewriting every upstream module to prost
would be slower and harder to keep in sync.

Fallback: keep prost and adapt upstream modules one by one. This is a
higher-cost fallback, not the default.

### Compatibility

- Keep existing `wa-rs` public modules and re-exports.
- Add upstream-style modules under `src/features` behind feature flags.
- Add deprecated adapters where an upstream API replaces an existing one.
- Keep `bridge/` on the default feature set.
- Keep `desktop/` frontend-only.

### Feature flags

Root `wa-rs` umbrellas:

- `parity-core`
- `parity-messaging`
- `parity-social`
- `parity-voip`
- `parity-mobile`
- `parity-plugins`
- `parity-observability`
- `parity-wasm`

Core `wa-rs-core` sub-features:

- `parity-runtime`
- `parity-session`
- `parity-messaging`
- `parity-observability`
- `parity-mobile`
- `parity-voip` (umbrella)
- `voip-control`
- `voip`
- `voip-mlow`

The two levels are not interchangeable. Module tests use the narrow core
feature (`--features voip-control`); consumer and phase gates use the root
umbrella (`--features parity-voip`). Umbrellas are additive:
`parity-messaging` implies `parity-core`, `parity-social` implies
`parity-messaging`, and the remaining umbrellas imply `parity-core`.

Default features keep current behavior; parity features are opt-in until a
phase is complete.

## Phases

1. **Phase 0 — Foundation:** feature flags, runtime/time/stats/telemetry/
   diagnostics, buffa waproto, event expansion, compatibility tests.
2. **Phase 1 — Auth/session:** session/sync engines, ADV, identity, passkey,
   pair-code lifecycle, client lifecycle, offline resume, mobile core.
3. **Phase 2 — Messaging/media/reliability:** message processing, edits,
   reactions, retry, receipts, PDO, media streaming/retry/reupload, WebP, ZIP,
   sticker pack, durability, inbound queue.
4. **Phase 3 — Social:** advanced groups, communities, newsletters, status,
   polls, events, stickers, business, bots, labels, favorites, quick replies,
   comments, chat actions, IQ/notification coverage.
5. **Phase 4 — Calls/VoIP:** voip-control, voip, voip-mlow, call handlers, call
   log, call links, screen sharing, MLOW/Opus, H.264 frames.
6. **Phase 5 — Platform/observability:** plugins, metrics/tracing/diagnostics,
   SQLite bundled/WASM/mobile, TCP transport, WASM consumer tests.

## Risks

| Risk | Mitigation |
|---|---|
| buffa migration | Phase 0 only; compatibility tests; prost fallback |
| bridge API breakage | compatibility facade; bridge smoke tests each phase |
| upstream external codegen (`whatspec`/`unwasm`) | vendor generated Rust; never runtime dependency |
| feature explosion | opt-in flags and isolated consumer builds |
| type/proto conflicts | adapters and deprecated re-exports |
| synthetic tests only | protocol fixtures and bridge integration tests |
| tree drift | new modules only inside existing crates |

## Definition of Done

- All upstream runtime feature modules exist under the current tree.
- `cargo build`, `cargo test --workspace` and `cargo check --workspace` pass.
- `bridge/` and `desktop/` compile on the default feature set.
- Each parity phase has tests and a documented feature flag.
- No JS/Python/Kotlin/Java runtime code is introduced.
- Remaining non-Rust artifacts are only data/schema/build tooling.

## Approved Decisions

1. Switch `waproto` internally from prost to buffa, keeping the crate name and
   path. Fallback to prost only if the Phase 0 gate cannot be met.
2. Exclude upstream dev tools, CI, Dockerfile and oracle harnesses from runtime
   parity. Generated Rust is vendored; the generators stay out of runtime
   crates.
3. Keep the current `wa-rs` public API as a compatibility facade while adding
   upstream-style APIs under `src/features`.
