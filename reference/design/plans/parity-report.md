# wa-rs Parity Audit

**Reference:** `oxidezap/whatsapp-rust` 0.7.0, pinned at `f077a5d4`.
**Target:** `homun-app/wa-rs` 0.2.0 in this workspace.
**Date:** 2026-11-09.

This audit compares the upstream runtime surface with the ported tree. The
strategy is a strangler port: upstream modules and feature names are added
alongside the existing compatibility facade. No existing public API was removed
and no upstream test was deleted to make a build green.

Legend: **ported** = functional parity; **partial** = public contract plus the
self-contained behavior, with runtime wiring deferred; **deferred** = upstream
module depends on a subsystem not yet ported; **non-goal** = explicitly outside
this port.

## Core (`wacore/`)

| Area | Upstream | Status | Notes |
|---|---|---|---|
| Telemetry | `wacore/src/telemetry.rs` | ported | Metric names, counters, histograms, `Timer`, `describe()`, no-op fallback. `metrics` feature wired. |
| Stats | `wacore/src/stats.rs` | ported | `SessionStats`, `StatsSnapshot`, `CollectionStats`, `AllocMeter`/`AllocSnapshot`, storage/transport/HTTP resource reports, `HeapSize`, task instrumentation. |
| Binary protocol | `wacore/binary` | ported | Encoder/decoder, JID, marshal, diagnostics, `tracing-pii` feature shell. |
| Noise | `wacore/noise` | ported | Handshake and transport primitives; `noise_handshake_test` green. |
| Signal | `wacore/libsignal` | ported | Existing vector/contract suites retained. |
| App state | `wacore/appstate` | ported | Existing app-state sync and fixtures retained. |
| IQ domains | `wacore/src/iq` | partial | 18 files. `props.rs` covers upstream `abprops.rs`; `usync.rs` covers `pilots.rs`. Deferred domains: `bot`, `business`, `community_create`, `devices`, `join_shapes`, `newsletter`, `profile`, `targets`, `mex_operations`, `mex_operations_unlisted`, `contextual` (+ tests). |
| VoIP primitives | `wacore/src/voip` | partial | RTP, RTCP, STUN, SFrame, SRTP, XOR FEC, MLOW/Opus framing, sans-IO engine. Deferred: `audio`, `codec_probe`, `demux`, `driver`, `e2e_srtp`, `group_audio`, `group_media`, `h264`, `hbh_srtp`, `opus_packet`, `session`, `tap`, `warp`. |
| VoIP control | `wacore/src/voip_control` | partial | One module exposes the neutral boundary types (`MediaSessionSpec`, `MediaCommand`, `CallEvent`, relay parsing, registry, scope/ports) instead of the upstream 19-file split. The directory split is deferred. |

## Root crate (`src/`)

| Area | Upstream | Status | Notes |
|---|---|---|---|
| Feature modules | `src/features` (38 files) | partial | 32 modules. Missing real modules: `media_reupload`, `message_edit`, `reaction` (the last two exist as `wacore` modules). Missing in-crate test files: `picture_mutation_tests`, `pictures_metrics_tests`, `pictures_tests`. |
| Plugins | `src/plugins/{mod,events,iq_reply}.rs` | ported (contract) | Manifest validation/ordering, capabilities, host registry/stats, event router with exact topics and overflow policies, IQ interceptor/reply grant. Runtime install callbacks remain owned by the client lifecycle. |
| Client lifecycle | `src/client/extension_lifecycle.rs` | ported (contract) | `ClientLifecycle`, `ConnectionScope`, `ConnectionScopeState`, cancellation signal and idempotent transitions. |
| Observability | `src/logging.rs`, `src/client/device_memo_stats.rs` | ported | Safe stanza summary, logging init, tracing span helper, device-memo stats, `MemoryReport`/`ResourceReport`/subsystem collections. |
| Shutdown | `src/shutdown.rs` | ported | SIGINT + SIGTERM on Unix, Ctrl+C elsewhere; `signal` feature. |
| Runtime | `src/runtime_impl.rs` | ported | `TokioRuntime`; `tokio-runtime` re-export. |
| Calls | `src/features/call_log.rs`, `src/handlers/call.rs`, `src/voip` | partial | Call log/group-call/call-event types, handler seam, high-level facade types. Full engine/session/group-media runtime deferred. |
| Social surface | `src/features/*` | partial | Groups, contacts, blocking, chat state, presence, MEX, TCToken plus creation, group history, stanza/retry, signal inspection, rotate key, chat actions, labels, favorites, quick replies, community, newsletter, status, polls, events, stickers, business, bots, comments, pictures, profile, app-state settings/resync. Deep IQ/handler wiring deferred. |

## Storage and transport

| Area | Status | Notes |
|---|---|---|
| SQLite profiles | ported | Default is `bundled-sqlite`; `js`, `mobile`, `instance-clock`, `wasm-test` features exist. System SQLite remains the `--no-default-features` fallback. |
| WASM storage | ported (check) | `cargo check -p wa-rs-sqlite-storage --target wasm32-unknown-unknown --no-default-features --features js` passes with a user-local clang. The default-features variant is invalid by design because bundled native SQLite cannot target wasm. |
| TCP transport | ported | `tcp` feature; `TokioTcpTransportFactory` and `from_byte_stream` with byte-stream read loop, disconnect and event channel. |
| Tokio runtime | ported | `TokioRuntime` plus `shutdown_signal`. |

## Feature matrix

All canonical and parity aliases resolve. `parity-*` aliases only add edges;
they do not appear in `#[cfg]` inside ported code.

| Command | Result |
|---|---|
| `cargo check -p wa-rs --no-default-features` | pass |
| `cargo check -p wa-rs --no-default-features --features parity-{core,messaging,social,voip,mobile,plugins,observability,wasm}` | pass (8/8) |
| `cargo check -p wa-rs --features {plugins,passkey,mobile-core,voip,tracing,metrics}` | pass |
| `cargo test -p wa-rs --features parity-{core,messaging,social,voip,mobile,plugins,observability}` | pass (7/7) |
| `cargo check -p wa-rs --features {sqlite-storage-bundled,mobile-transport-native}` | pass |
| WASM storage check | pass with `--no-default-features --features js` |

## Known gaps

1. **Live bridge smoke** — pairing/connect/send/receive/media/call smoke needs a
   real WhatsApp account and phone. HTTP router smoke is green.
2. **Deep runtime splits** — the upstream monolithic `Client` lifecycle,
   subsystem supervision, store-release leases and send fan-out are not replaced.
   The compatibility facade remains authoritative.
3. **VoIP engine/relay/codecs** — native relay (DTLS/SCTP/datachannel), H.264,
   audio pipeline, group media and full session/driver wiring are deferred.
4. **Social IQ/handler internals** — the feature surfaces exist, but deep
   notification handlers and several IQ domains listed above are deferred.
5. **`handshake_span_scope` test** — depends on the upstream `wa.conn.handshake`
   span and `NoiseSocket::SendObservers`, which the compatibility client does
   not expose. Recorded as deferred in the test parity report.
6. **Feature deviations** — `tokio-runtime` is a marker because Tokio is a
   non-optional root dependency; `mobile-core` omits the unused `dep:subtle`
   edge; `voip-libopus`/`voip-relay-native` omit the optional Opus/rtc
   dependencies because the codecs/relay are not ported yet.

## Gate evidence

- `cargo fmt --all --check` → exit 0.
- `cargo clippy --workspace --all-targets` → exit 0, 190 warning lines
  (baseline; no new warnings).
- `cargo test --workspace` → 1297 passed, 0 failed, 29 ignored.
- `cargo check --workspace` → exit 0.
- `cargo test -p whatslite-desktop` → 21 passed; `cargo build -p whatslite-desktop` → exit 0.
- `cargo test -p wa-rs-bridge` → 3 passed; `cargo check -p wa-rs-bridge` → exit 0.
- Feature-target suites: plugins 7, observability 5, storage parity 1, TCP parity 3, shutdown signal 1, all green.
- Phase 4 VoIP suites remain green under the aligned feature chain.
