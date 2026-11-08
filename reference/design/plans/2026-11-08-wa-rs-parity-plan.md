# wa-rs Parity with whatsapp-rust 0.7.0 — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Port the full runtime feature set of `oxidezap/whatsapp-rust` 0.7.0 into `homun-app/wa-rs` 0.2.0 without changing the current tree and without breaking `bridge/` or `desktop/`.

**Architecture:** Strangler migration per subsystem. Each upstream module is ported into the existing crate boundary (`wacore/`, `waproto/`, `src/`, `storages/`, `transports/`, `http_clients/`) behind a `parity-*` feature flag. The existing `wa-rs` API stays as a compatibility facade; upstream-style APIs are added under `src/features`. Each phase must build, test and pass its gate before the next starts.

**Tech Stack:** Rust 2024, Tokio, SQLite/Diesel, buffa (recommended) or prost fallback, rustls, GPUI desktop consumer, `bridge/` HTTP sidecar.

**Spec:** `reference/design/plans/2026-11-08-wa-rs-parity-design.md`

## Global Constraints

- Keep the current top-level tree; do not add a new runtime crate.
- All runtime code is Rust. `.proto`, `.sql`, `.json`, `.yml`, Dockerfiles and test vectors are data/schema/tooling, not runtime languages.
- SQLite and TLS/ring may remain C/asm-backed.
- Rust 1.99.1 current; upstream MSRV 1.95 is compatible.
- `bridge/` and `desktop/` must compile at every phase gate.
- Parity features are opt-in behind `parity-*` flags.
- Do not port upstream CI, Dockerfile, oracle tools, benchmark harnesses, or generated test-vector tooling into runtime crates.
- Do not maintain a dual backend.
- Every task ends with a commit. Every phase ends with the phase gate.

## Prerequisites

Run these once before Task 0.1. Phase gates call `cargo fmt` and `cargo clippy`, so
a missing component is a blocker, not a warning.

- [ ] Install the Rust components:

```bash
rustup component add rustfmt clippy
cargo fmt --version
cargo clippy --version
```

- [ ] Provide the SQLite linker used by the workspace tests:

```bash
mkdir -p /tmp/opencode/sqlite
ln -sf /lib64/libsqlite3.so.0 /tmp/opencode/sqlite/libsqlite3.so
export LD_LIBRARY_PATH="/tmp/opencode/sqlite:${LD_LIBRARY_PATH:-}"
```

- [ ] Verify the baseline before touching code:

```bash
cargo test -p wa-rs-sqlite-storage
cargo test --workspace
```

Expected: the baseline is green. If it is not, fix the environment first and
record the failure; do not start Phase 0 on a red baseline.

- [ ] Confirm the reference checkout is at the pinned upstream commit:

```bash
git -C reference/whatsapp-rust rev-parse --short HEAD
```

Expected: `f077a5d`. Ported files are copied from this commit, not from a moving
branch.

## Gate Protocol

Every phase gate runs this standard set first, from the repo root:

1. `cargo fmt --all -- --check`
2. `cargo clippy --workspace --all-targets`
3. `cargo test --workspace`
4. `cargo check --workspace`
5. `cargo build -p whatslite-desktop`
6. `cargo check -p wa-rs-bridge`

Then the phase-specific commands listed in that phase's gate. Phase-specific
commands are additions, never replacements. Clippy must not add new warnings;
record any pre-existing warning count in the phase commit message.

The bridge smoke test is manual: pair, connect, send, receive. Record the
result in the phase commit message.

## Review Focus

These are the failure modes most likely to bite a user even when the module tests pass. Each line has a test in the owning task.

1. **Bridge pairing/send/receive breaks after the proto switch** — a user can still pair, send and receive through `wa-bridge`.
2. **Feature-flag combinations fail to compile** — `default`, `parity-core`, `parity-messaging`, `parity-social`, `parity-voip`, `parity-mobile`, `parity-plugins` each build independently.
3. **Event enum ordering/interest bits break persisted state** — existing event kinds keep their discriminants and new kinds are appended.
4. **Session/Signal state is lost after offline/reconnect** — resume, retry and durability hooks preserve sessions.
5. **WASM/mobile builds accidentally pull Tokio/native dependencies** — isolated no-default-feature builds link or fail for the expected reason only.

---

## File Structure

### Phase 0

- Modify: `Cargo.toml`, `wacore/Cargo.toml`, `wacore/src/lib.rs`, `waproto/Cargo.toml`, `waproto/build.rs`, `waproto/src/lib.rs`, `src/lib.rs`, `bridge/Cargo.toml`
- Create: `wacore/src/runtime.rs`, `wacore/src/time.rs`, `wacore/src/stats.rs`, `wacore/src/telemetry.rs`, `wacore/src/diagnostics.rs`
- Create tests: `wacore/tests/parity_phase0.rs`, `waproto/tests/proto_roundtrip.rs`, `bridge/tests/parity_smoke.rs`

### Phase 1

- Create: `wacore/src/adv/`, `wacore/src/identity_verification.rs`, `wacore/src/session_engine/`, `wacore/src/sync_engine/`
- Create: `src/client/lifecycle.rs`, `src/client/session_state.rs`, `src/client/offline_resume.rs`, `src/client/node_io.rs`, `src/client/builder.rs`, `src/client/iq_ops.rs`
- Create: `src/passkey/`, `src/mobile/`, `src/mobile_session.rs`
- Create tests: `wacore/tests/parity_adv.rs`, `tests/parity_auth.rs`

### Phase 2

- Create: `wacore/src/message_processing.rs`, `wacore/src/message_edit.rs`, `wacore/src/msg_secret.rs`, `wacore/src/reaction.rs`, `wacore/src/media_retry.rs`, `wacore/src/webp.rs`, `wacore/src/zip.rs`, `wacore/src/sticker_pack.rs`
- Create: `src/message/commit_batch.rs`, `src/message/dispatch.rs`, `src/message/durability.rs`, `src/message/live_retention.rs`, `src/message/raw_pending.rs`, `src/message/receive.rs`, `src/message/retry.rs`, `src/message/special.rs`
- Create: `src/inbound_queue.rs`, `src/flush_scope.rs`, `src/signal_flush.rs`, `src/resend_rate_limiter.rs`, `src/sender_key_device_cache.rs`
- Create: `src/download/budget.rs`, `src/download/memory.rs`, `src/media.rs`
- Create tests: `wacore/tests/parity_messaging.rs`, `tests/parity_media.rs`

### Phase 3

- Create/modify: `src/features/{groups,community,newsletter,status,polls,events,stickers,business,bots,labels,favorites,quick_replies,comments,chat_actions,pictures,profile,app_state_settings,app_state_resync}.rs`
- Create/modify: `wacore/src/iq/{abprops,bot,business,community_create,contextual,devices,join_shapes,newsletter,profile,targets,usync/query}.rs`
- Create/modify: `src/handlers/notification/{companion_reg,device,groups,privacy_business,profile}.rs`
- Create tests: `tests/parity_social.rs`, `wacore/tests/parity_iq_social.rs`

### Phase 4

- Create: `wacore/src/voip/`, `wacore/src/voip_control/`, `src/voip/`, `src/voip_control/`, `src/handlers/call/`, `src/features/call_log.rs`
- Create tests: `wacore/tests/parity_voip_control.rs`, `tests/parity_calls.rs`

### Phase 5

- Create: `src/plugins/`, `src/plugins/metrics.rs` (or `plugins/metrics` inside `src`), `src/logging.rs`, `src/shutdown.rs`, `src/runtime_impl.rs`
- Modify: `storages/sqlite-storage/Cargo.toml`, `storages/sqlite-storage/src/`, `transports/tokio-transport/Cargo.toml`, `transports/tokio-transport/src/`
- Create tests: `tests/parity_platform.rs`, `storages/sqlite-storage/tests/parity_storage.rs`

---

## Phase 0 — Foundation

**Phase goal:** create the compatibility and build foundation. No user-facing feature is added.

### Task 0.1: Add parity feature flags

**Files:**
- Modify: `Cargo.toml`
- Modify: `wacore/Cargo.toml`
- Modify: `src/lib.rs`
- Modify: `bridge/Cargo.toml`
- Test: `cargo check` feature matrix

**Interfaces:**
- Produces: root features `parity-core`, `parity-messaging`, `parity-social`, `parity-voip`, `parity-mobile`, `parity-plugins`, `parity-observability`, `parity-wasm`.
- Produces: `wacore` features `parity-runtime`, `parity-session`, `parity-messaging`, `parity-observability`, `parity-voip`, `parity-mobile`, plus the VoIP sub-features `voip-control`, `voip`, `voip-mlow`.

**Feature naming contract.** Two levels exist and they are not interchangeable:

| Layer | Feature | Purpose |
|---|---|---|
| Root `wa-rs` | `parity-core` | runtime, clock, stats, telemetry, diagnostics, session/sync engines |
| Root `wa-rs` | `parity-messaging` | message pipeline, media, retry, durability |
| Root `wa-rs` | `parity-social` | groups, communities, newsletters, status, polls, events, business, bots |
| Root `wa-rs` | `parity-voip` | umbrella that enables the three core VoIP sub-features |
| Root `wa-rs` | `parity-mobile` | mobile registration and primary session |
| Root `wa-rs` | `parity-plugins` | plugin host and lifecycle |
| Root `wa-rs` | `parity-observability` | metrics, tracing, diagnostics export |
| Root `wa-rs` | `parity-wasm` | WASM storage and randomness |
| Core `wa-rs-core` | `parity-runtime` | runtime and clock abstraction |
| Core `wa-rs-core` | `parity-session` | ADV, identity verification, session and sync engines |
| Core `wa-rs-core` | `parity-messaging` | message processing, edits, secrets, reactions, media retry; implies `parity-session` |
| Core `wa-rs-core` | `parity-observability` | core-side telemetry hooks |
| Core `wa-rs-core` | `parity-mobile` | mobile core types |
| Core `wa-rs-core` | `parity-voip` | umbrella = `voip-control` + `voip` + `voip-mlow` |
| Core `wa-rs-core` | `voip-control` | neutral call-control traits only |
| Core `wa-rs-core` | `voip` | SRTP/SFrame/WAHKDF media crypto, implies `voip-control` |
| Core `wa-rs-core` | `voip-mlow` | PCM/MLOW codec and video frames, implies `voip` |

Rules:

- Tests of an isolated core module use the narrow core feature, e.g.
  `cargo test -p wa-rs-core --features voip-control`.
- Consumer and phase-gate checks use the root umbrella, e.g.
  `cargo check -p wa-rs --features parity-voip`.
- Umbrellas are additive and imply the phase they depend on:
  `parity-messaging` implies `parity-core`, `parity-social` implies
  `parity-messaging`, `parity-voip`/`parity-mobile`/`parity-plugins`/
  `parity-observability`/`parity-wasm` imply `parity-core`.
- Never reference a feature that does not exist yet in a `Cargo.toml` edge; add
  the edge in the task that introduces the module.

- [ ] **Step 1: Add the feature declarations**

Declare every flag above in root `Cargo.toml` and `wacore/Cargo.toml`. In this
task they are empty shells (`parity-core = []`, `voip-control = []`, ...); each
later task adds its own dependency edges. Add the naming contract as a short
comment block above `[features]` so the two levels are not confused later.

- [ ] **Step 2: Verify the default build is unchanged**

Run:

```bash
cargo check -p wa-rs --no-default-features
cargo check -p wa-rs --no-default-features --features parity-core
cargo check -p whatslite-desktop
```

Expected: all three commands succeed.

- [ ] **Step 3: Commit**

```bash
git add Cargo.toml wacore/Cargo.toml src/lib.rs bridge/Cargo.toml
git commit -m "build: add parity feature flags"
```

### Task 0.2: Port runtime, time, stats, telemetry and diagnostics

**Files:**
- Create: `wacore/src/runtime.rs`
- Create: `wacore/src/time.rs`
- Create: `wacore/src/stats.rs`
- Create: `wacore/src/telemetry.rs`
- Create: `wacore/src/diagnostics.rs`
- Modify: `wacore/src/lib.rs`
- Modify: `wacore/Cargo.toml`
- Test: `wacore/tests/parity_phase0.rs`

**Interfaces:**
- Produces: `wacore::runtime::Runtime` with the upstream `sleep`, `spawn`, `now` contract.
- Produces: `wacore::time::Clock` and `InstanceClock` compatible with upstream call sites.
- Produces: `wacore::stats::AllocMeter` and `MemoryReport`.
- Produces: `wacore::telemetry` hooks gated by `parity-observability`.
- Produces: `wacore::diagnostics::DiagnosticEvent`.

- [ ] **Step 1: Write the failing test**

Create `wacore/tests/parity_phase0.rs` with tests:

```rust
#[test]
fn runtime_clock_is_monotonic() { /* construct test clock, assert no backward jump */ }

#[test]
fn alloc_meter_records_scoped_allocations() { /* enter scope, assert bytes > 0 */ }

#[test]
fn diagnostics_are_serializable() { /* serde round-trip of a diagnostic event */ }
```

- [ ] **Step 2: Run it to verify failure**

Run:

```bash
cargo test -p wa-rs-core --features parity-runtime,parity-observability --test parity_phase0
```

Expected: FAIL with unresolved `wacore::runtime`, `wacore::time`, `wacore::stats`.

- [ ] **Step 3: Port the modules**

Copy the public API and behavior from `reference/whatsapp-rust/wacore/src/runtime.rs`, `time.rs`, `stats.rs`, `telemetry.rs`, and `diagnostics.rs`. Keep upstream signatures and feature gates. Adapt crate names to `wa_rs_core`.

- [ ] **Step 4: Run the test to verify pass**

Run:

```bash
cargo test -p wa-rs-core --features parity-runtime,parity-observability --test parity_phase0
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add wacore/src/runtime.rs wacore/src/time.rs wacore/src/stats.rs wacore/src/telemetry.rs wacore/src/diagnostics.rs wacore/src/lib.rs wacore/Cargo.toml wacore/tests/parity_phase0.rs
git commit -m "feat(core): port runtime time stats telemetry and diagnostics"
```

### Task 0.3: Switch `waproto` to buffa

**Files:**
- Modify: `waproto/Cargo.toml`
- Modify: `waproto/build.rs`
- Modify: `waproto/src/lib.rs`
- Modify: `wacore/Cargo.toml`
- Modify: `src/` call sites
- Modify: `bridge/` call sites
- Test: `waproto/tests/proto_roundtrip.rs`
- Test: `bridge/tests/parity_smoke.rs`

**Interfaces:**
- Produces: buffa-generated Rust types under the existing `wa_rs_proto::whatsapp` path.
- Produces: compatibility conversions for the existing prost-shaped types used by `bridge`.

- [ ] **Step 1: Write the failing round-trip test**

Create `waproto/tests/proto_roundtrip.rs`:

```rust
#[test]
fn message_round_trips_through_buffa() { /* encode/decode a Message with text and key */ }

#[test]
fn adv_identity_view_decodes() { /* decode ADVSignedDeviceIdentity bytes */ }
```

- [ ] **Step 2: Run it to verify failure**

Run:

```bash
cargo test -p wa-rs-proto --test proto_roundtrip
```

Expected: FAIL because the buffa types/codegen are not present.

- [ ] **Step 3: Port the codegen**

Replace the prost build with the upstream `buffa-build` pipeline. Keep the crate name `wa-rs-proto` and path `waproto/`. Update `wacore`, `src`, and `bridge` imports. Add `From`/`TryFrom` adapters where the old API exposed prost structs.

- [ ] **Step 4: Run the tests to verify pass**

Run:

```bash
cargo test -p wa-rs-proto --test proto_roundtrip
cargo check -p wa-rs-bridge
cargo check -p whatslite-desktop
```

Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add waproto wacore/Cargo.toml src bridge
git commit -m "refactor(proto): switch waproto to buffa"
```

### Task 0.4: Expand event model and delivery

**Files:**
- Modify: `wacore/src/types/events.rs`
- Create: `src/bot/event_delivery.rs`
- Modify: `src/bot.rs`
- Test: `wacore/tests/parity_events.rs`

**Interfaces:**
- Produces: upstream event variants and `EventKind` discriminants.
- Produces: `EventDelivery` modes and hook registration.

- [ ] **Step 1: Write the failing test**

Create `wacore/tests/parity_events.rs`:

```rust
#[test]
fn existing_event_kinds_keep_their_discriminants() { /* assert old values */ }

#[test]
fn new_event_kinds_are_appended() { /* assert new values > old max */ }

#[test]
fn event_serde_round_trips() { /* serde round-trip for representative events */ }
```

- [ ] **Step 2: Run it to verify failure**

```bash
cargo test -p wa-rs-core --test parity_events
```

Expected: FAIL for missing variants/delivery.

- [ ] **Step 3: Port the event enum and delivery**

Copy upstream `wacore/src/types/events.rs` and `src/bot/event_delivery.rs`. Preserve old discriminants. Add compatibility conversions for old event names.

- [ ] **Step 4: Run the test to verify pass**

```bash
cargo test -p wa-rs-core --test parity_events
cargo check -p wa-rs-bridge
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add wacore/src/types/events.rs src/bot/event_delivery.rs src/bot.rs wacore/tests/parity_events.rs
git commit -m "feat(events): expand event model and delivery"
```

### Phase 0 Gate

- [ ] `cargo test -p wa-rs-core --features parity-runtime,parity-observability --test parity_phase0`
- [ ] `cargo test -p wa-rs-core --test parity_events`
- [ ] `cargo test -p wa-rs-proto --test proto_roundtrip`
- [ ] `cargo build -p whatslite-desktop`
- [ ] `cargo test --workspace`
- [ ] `cargo check --workspace`
- [ ] `cargo check -p wa-rs --no-default-features --features parity-core`
- [ ] `cargo check -p wa-rs-bridge`
- [ ] Bridge smoke test: pair, connect, send, receive

Commit the phase as a merge/series only when every box is checked.

---

## Phase 1 — Auth, session and recovery

**Phase goal:** upstream auth/session lifecycle, passkey and mobile core.

### Task 1.1: Port session and sync engines

**Files:**
- Create: `wacore/src/session_engine/`
- Create: `wacore/src/sync_engine/`
- Modify: `wacore/src/lib.rs`
- Modify: `wacore/Cargo.toml`
- Test: `wacore/tests/parity_session.rs`

**Interfaces:**
- Produces: `SessionEngine`, `SessionTransition`, `SyncEngine`, `SyncAdmission`.
- Consumes: `runtime::Runtime`, `time::Clock`, `store::traits`.

- [ ] **Step 1: Write failing tests** for session transitions, offline resume and sync admission.
- [ ] **Step 2: Run `cargo test -p wa-rs-core --features parity-session --test parity_session` and verify failure.**
- [ ] **Step 3: Port upstream `session_engine` and `sync_engine` modules and adapt imports.**
- [ ] **Step 4: Run the tests and `cargo check -p wa-rs --features parity-core`.**
- [ ] **Step 5: Commit** with `feat(core): port session and sync engines`.

### Task 1.2: Port ADV and identity verification

**Files:**
- Create: `wacore/src/adv/`
- Create: `wacore/src/identity_verification.rs`
- Create: `wacore/src/client_profile.rs`
- Modify: `wacore/src/iq/prekeys.rs`
- Modify: `wacore/src/iq/usync.rs`
- Test: `wacore/tests/parity_adv.rs`

**Interfaces:**
- Produces: `AdvValidation`, `validate_adv_with_identity_key`, identity-change handling.
- Consumes: buffa-generated `ADVSignedDeviceIdentityView`.

- [ ] **Step 1: Write failing tests** for valid/invalid/no-account-key ADV, hosted/companion prefixes, and identity change.
- [ ] **Step 2: Run `cargo test -p wa-rs-core --features parity-session --test parity_adv` and verify failure.**
- [ ] **Step 3: Port upstream `adv`, `identity_verification`, `client_profile` and adapt USync/device-list parsing.**
- [ ] **Step 4: Run the tests and `cargo check -p wa-rs --features parity-core`.**
- [ ] **Step 5: Commit** with `feat(core): port ADV and identity verification`.

### Task 1.3: Pair-code lifecycle and passkey

**Files:**
- Modify: `src/pair_code.rs`
- Create: `src/passkey/`
- Modify: `src/bot.rs`
- Modify: `src/handlers/notification/companion_reg.rs`
- Test: `tests/parity_auth.rs`

**Interfaces:**
- Produces: pair-code refresh, error, cancel, exhausted events.
- Produces: passkey request/confirmation/error events and flow.

- [ ] **Step 1: Write failing tests** for pair-code refresh/cancel/exhausted and passkey event mapping.
- [ ] **Step 2: Run `cargo test -p wa-rs --features parity-core --test parity_auth` and verify failure.**
- [ ] **Step 3: Port upstream pair-code lifecycle and passkey flow; keep old `pair_with_code` signature as a wrapper.**
- [ ] **Step 4: Run the tests and `cargo check -p wa-rs-bridge`.**
- [ ] **Step 5: Commit** with `feat(auth): port pair-code lifecycle and passkey`.

### Task 1.4: Client lifecycle, offline resume and node I/O

**Files:**
- Create: `src/client/lifecycle.rs`
- Create: `src/client/session_state.rs`
- Create: `src/client/offline_resume.rs`
- Create: `src/client/node_io.rs`
- Create: `src/client/builder.rs`
- Create: `src/client/iq_ops.rs`
- Modify: `src/client.rs`
- Test: `tests/parity_client.rs`

**Interfaces:**
- Produces: `ClientOptions`, `ConnectStage`, `ConnectError`, `NodeFilter`, `NodeWaiter<T>`.
- Preserves: existing `Client::new`, `run`, `connect`, `disconnect`, `send_node`.

- [ ] **Step 1: Write failing tests** for connect stages, node filter/waiter, and offline resume.
- [ ] **Step 2: Run `cargo test -p wa-rs --features parity-core --test parity_client` and verify failure.**
- [ ] **Step 3: Port upstream client modules and wire the old API through compatibility wrappers.**
- [ ] **Step 4: Run the tests, `cargo check -p wa-rs --features parity-core`, and bridge smoke test.**
- [ ] **Step 5: Commit** with `feat(client): port lifecycle offline resume and node I/O`.

### Task 1.5: Mobile core

**Files:**
- Create: `src/mobile/`
- Create: `src/mobile_session.rs`
- Create: `wacore/src/mobile/`
- Modify: `Cargo.toml` (`parity-mobile`)
- Test: `tests/parity_mobile.rs`

**Interfaces:**
- Produces: mobile registration, primary login, companion hosting.
- Gate: only compiled with `parity-mobile`.

- [ ] **Step 1: Write failing tests** for registration request/response and primary session binding.
- [ ] **Step 2: Run `cargo test -p wa-rs --features parity-mobile --test parity_mobile` and verify failure.**
- [ ] **Step 3: Port upstream mobile modules and feature-gate them.**
- [ ] **Step 4: Run the feature test and `cargo check -p wa-rs --features parity-mobile`.**
- [ ] **Step 5: Commit** with `feat(mobile): port registration and primary sessions`.

### Phase 1 Gate

- [ ] `cargo test -p wa-rs-core --features parity-session --test parity_session --test parity_adv`
- [ ] `cargo test -p wa-rs --features parity-core --test parity_auth --test parity_client`
- [ ] `cargo test -p wa-rs --features parity-mobile --test parity_mobile`
- [ ] `cargo check -p wa-rs --features parity-core`
- [ ] `cargo check -p wa-rs --features parity-mobile`
- [ ] `cargo check -p wa-rs-bridge`
- [ ] `cargo build -p whatslite-desktop`
- [ ] Bridge pairing and reconnect smoke test

---

## Phase 2 — Messaging, media and reliability

**Phase goal:** upstream message pipeline, retry/durability, media streaming and recovery.

### Task 2.1: Message processing, dispatch and durability

**Files:**
- Create: `wacore/src/message_processing.rs`
- Create: `src/message/commit_batch.rs`
- Create: `src/message/dispatch.rs`
- Create: `src/message/durability.rs`
- Create: `src/message/live_retention.rs`
- Create: `src/message/raw_pending.rs`
- Create: `src/message/receive.rs`
- Create: `src/message/special.rs`
- Modify: `src/message.rs`
- Test: `wacore/tests/parity_messaging.rs`

**Interfaces:**
- Produces: `EncType`, `MessageProcessing`, `CommitBatch`, `DurabilityHook`.
- Consumes: `session_engine`, `sync_engine`.

- [ ] **Step 1: Write failing tests** for inbound dispatch, commit ordering, live retention and raw pending.
- [ ] **Step 2: Run `cargo test -p wa-rs-core --features parity-messaging --test parity_messaging` and verify failure.**
- [ ] **Step 3: Port upstream message processing and `src/message` modules.**
- [ ] **Step 4: Run tests and `cargo check -p wa-rs --features parity-messaging`.**
- [ ] **Step 5: Commit** with `feat(message): port processing dispatch and durability`.

### Task 2.2: Edits, reactions and message secrets

**Files:**
- Create: `wacore/src/message_edit.rs`
- Create: `wacore/src/msg_secret.rs`
- Create: `wacore/src/reaction.rs`
- Create: `src/features/message_edit.rs`
- Create: `src/features/reaction.rs`
- Modify: `src/client.rs`
- Test: `wacore/tests/parity_edit_reaction.rs`

**Interfaces:**
- Produces: `EncryptedEdit`, `SecretEncrypted`, `send_reaction`, `decrypt_secret_encrypted`.
- Preserves: existing `Client::edit_message` and `bot::MessageContext::edit_message`.

- [ ] **Step 1: Write failing tests** for edit envelope, secret edit, reaction send/raw and legacy fallback.
- [ ] **Step 2: Run `cargo test -p wa-rs-core --features parity-messaging --test parity_edit_reaction` and verify failure.**
- [ ] **Step 3: Port the modules and add compatibility wrappers.**
- [ ] **Step 4: Run the tests and `cargo check -p wa-rs-bridge`.**
- [ ] **Step 5: Commit** with `feat(message): port edits reactions and secrets`.

### Task 2.3: Retry, receipts, PDO and rate limiting

**Files:**
- Create: `src/retry_conformance_tests.rs`
- Create: `src/resend_rate_limiter.rs`
- Create: `src/sender_key_device_cache.rs`
- Modify: `src/retry.rs`, `src/receipt.rs`, `src/pdo.rs`
- Test: `tests/parity_reliability.rs`

**Interfaces:**
- Produces: `RetryRequestOptions`, `RetryRequestOutcome`, `mark_message_played`, PDO fetch history.
- Consumes: `StanzaRejection`, `NackReason`.

- [ ] **Step 1: Write failing tests** for retry conformance, played receipts, PDO history and rate limit.
- [ ] **Step 2: Run `cargo test -p wa-rs --features parity-messaging --test parity_reliability` and verify failure.**
- [ ] **Step 3: Port upstream retry/receipt/PDO/rate-limit modules.**
- [ ] **Step 4: Run the tests and bridge smoke test.**
- [ ] **Step 5: Commit** with `feat(reliability): port retry receipts PDO and rate limiting`.

### Task 2.4: Media streaming, retry/reupload and cache

**Files:**
- Create: `src/media.rs`
- Create: `src/download/budget.rs`
- Create: `src/download/memory.rs`
- Create: `src/features/media_reupload.rs`
- Create: `wacore/src/media_retry.rs`
- Create: `wacore/src/webp.rs`
- Create: `wacore/src/zip.rs`
- Create: `wacore/src/sticker_pack.rs`
- Modify: `src/download.rs`, `src/upload.rs`, `src/mediaconn.rs`
- Test: `tests/parity_media.rs`

**Interfaces:**
- Produces: `MediaDownloader`, `DownloadParams`, `UploadOptions`, `MediaReupload`.
- Consumes: `MediaConn`, `HttpClient`, `Runtime`.

- [ ] **Step 1: Write failing tests** for streaming upload/download, retry/reupload, cache budget, WebP and sticker pack.
- [ ] **Step 2: Run `cargo test -p wa-rs --features parity-messaging --test parity_media` and verify failure.**
- [ ] **Step 3: Port upstream media modules and keep old `download`/`upload` wrappers.**
- [ ] **Step 4: Run the tests and `cargo check -p wa-rs --features parity-messaging`.**
- [ ] **Step 5: Commit** with `feat(media): port streaming retry and cache`.

### Task 2.5: Inbound queue, flush scope and offline resume

**Files:**
- Create: `src/inbound_queue.rs`
- Create: `src/flush_scope.rs`
- Create: `src/signal_flush.rs`
- Modify: `src/client/offline_resume.rs`
- Test: `tests/parity_offline.rs`

**Interfaces:**
- Produces: inbound queue admission, batch-safe signal flush, offline resume.
- Consumes: `DurabilityHook`, `SyncEngine`.

- [ ] **Step 1: Write failing tests** for queued inbound order, flush after reconnect and no duplicate delivery.
- [ ] **Step 2: Run `cargo test -p wa-rs --features parity-messaging --test parity_offline` and verify failure.**
- [ ] **Step 3: Port upstream queue/flush/offline modules.**
- [ ] **Step 4: Run the tests and bridge offline/online smoke test.**
- [ ] **Step 5: Commit** with `feat(reliability): port inbound queue and offline resume`.

### Phase 2 Gate

- [ ] `cargo test -p wa-rs-core --features parity-messaging --test parity_messaging --test parity_edit_reaction`
- [ ] `cargo test -p wa-rs --features parity-messaging --test parity_reliability --test parity_media --test parity_offline`
- [ ] `cargo check -p wa-rs --features parity-messaging`
- [ ] `cargo check -p wa-rs-bridge`
- [ ] `cargo build -p whatslite-desktop`
- [ ] Bridge send/receive/media smoke test

---

## Phase 3 — Social features

**Phase goal:** groups, communities, newsletters, status, polls, events, stickers, business, bots, labels, favorites, quick replies, comments and chat actions.

### Task 3.1: Advanced groups and chat actions

**Files:**
- Modify: `src/features/groups.rs`
- Create: `src/features/chat_actions.rs`
- Create: `src/features/labels.rs`
- Create: `src/features/favorites.rs`
- Create: `src/features/quick_replies.rs`
- Modify: `wacore/src/iq/groups.rs`, `wacore/src/iq/join_shapes.rs`
- Test: `tests/parity_groups.rs`

**Interfaces:**
- Produces: `GroupError`, `GroupLookupResult`, `GroupOverview`, `MembershipRequest`, `ClearChatOptions`, `StarredMessages`.
- Preserves: existing `Groups` basic methods.

- [ ] **Step 1: Write failing tests** for join by invite, membership approve/reject, locked/announce/ephemeral, history share, labels/favorites/quick replies.
- [ ] **Step 2: Run `cargo test -p wa-rs --features parity-social --test parity_groups` and verify failure.**
- [ ] **Step 3: Port upstream groups/chat actions/labels/favorites/quick replies and IQ modules.**
- [ ] **Step 4: Run the tests and bridge group smoke test.**
- [ ] **Step 5: Commit** with `feat(groups): port advanced groups and chat actions`.

### Task 3.2: Communities

**Files:**
- Create: `src/features/community.rs`
- Create: `wacore/src/iq/community_create.rs`
- Create: `wacore/src/iq/join_shapes.rs`
- Test: `tests/parity_community.rs`

**Interfaces:**
- Produces: `Community`, `CreateCommunityOptions`, `CreateSubgroupOptions`, `LinkSubgroupOptions`, `CommunitySubgroup`.

- [ ] **Step 1: Write failing tests** for create/deactivate, subgroup create/link/unlink, participant counts and join.
- [ ] **Step 2: Run `cargo test -p wa-rs --features parity-social --test parity_community` and verify failure.**
- [ ] **Step 3: Port upstream community modules and IQ types.**
- [ ] **Step 4: Run the tests and `cargo check -p wa-rs --features parity-social`.**
- [ ] **Step 5: Commit** with `feat(community): port communities and subgroups`.

### Task 3.3: Newsletters and channels

**Files:**
- Create: `src/features/newsletter.rs`
- Create: `wacore/src/iq/newsletter.rs`
- Modify: `src/handlers/notification.rs`
- Test: `tests/parity_newsletter.rs`

**Interfaces:**
- Produces: `Newsletter`, `NewsletterMetadata`, `NewsletterMessage`, `NewsletterAdminInfo`.

- [ ] **Step 1: Write failing tests** for list/get/create/join/leave/update/admin/messages/reactions/polls.
- [ ] **Step 2: Run `cargo test -p wa-rs --features parity-social --test parity_newsletter` and verify failure.**
- [ ] **Step 3: Port upstream newsletter modules and notification handlers.**
- [ ] **Step 4: Run the tests and `cargo check -p wa-rs --features parity-social`.**
- [ ] **Step 5: Commit** with `feat(newsletter): port newsletters and channels`.

### Task 3.4: Status, polls and events

**Files:**
- Create: `src/features/status.rs`
- Create: `src/features/polls.rs`
- Create: `src/features/events.rs`
- Create: `wacore/src/poll.rs`
- Test: `tests/parity_status_polls_events.rs`

**Interfaces:**
- Produces: `Status`, `StatusSendOptions`, `Polls`, `CreatedPoll`, `Events`, `EventCreationParams`.

- [ ] **Step 1: Write failing tests** for status send/revoke/privacy, poll create/vote/decrypt/aggregate and event create/respond.
- [ ] **Step 2: Run `cargo test -p wa-rs --features parity-social --test parity_status_polls_events` and verify failure.**
- [ ] **Step 3: Port upstream status/polls/events modules and IQ types.**
- [ ] **Step 4: Run the tests and `cargo check -p wa-rs --features parity-social`.**
- [ ] **Step 5: Commit** with `feat(social): port status polls and events`.

### Task 3.5: Stickers, business, bots, comments and profile

**Files:**
- Create: `src/features/stickers.rs`
- Create: `src/features/business.rs`
- Create: `src/features/bots.rs`
- Create: `src/features/comments.rs`
- Create: `src/features/pictures.rs`
- Create: `src/features/profile.rs`
- Create: `wacore/src/iq/{bot,business,profile}.rs`
- Test: `tests/parity_social_misc.rs`

**Interfaces:**
- Produces: `Business`, `Catalog`, `Order`, `Bots`, `Comments`, `Pictures`, `Profile`.
- Preserves: existing contact/profile picture calls.

- [ ] **Step 1: Write failing tests** for catalog/order, bot list, comments, profile picture and text status.
- [ ] **Step 2: Run `cargo test -p wa-rs --features parity-social --test parity_social_misc` and verify failure.**
- [ ] **Step 3: Port upstream modules and IQ types.**
- [ ] **Step 4: Run the tests and `cargo check -p wa-rs --features parity-social`.**
- [ ] **Step 5: Commit** with `feat(social): port stickers business bots and comments`.

### Task 3.6: IQ/notification coverage and event parity

**Files:**
- Modify: `wacore/src/iq/mod.rs`
- Modify: `src/handlers/notification/mod.rs`
- Modify: `src/handlers/notification/{companion_reg,device,groups,privacy_business,profile}.rs`
- Test: `wacore/tests/parity_iq_social.rs`

**Interfaces:**
- Produces: all upstream IQ domains and notification types.
- Produces: 80-variant event surface.

- [ ] **Step 1: Write failing conformance tests** for each new IQ domain and notification type.
- [ ] **Step 2: Run the test and verify failure.**
- [ ] **Step 3: Port the remaining IQ and notification modules.**
- [ ] **Step 4: Run tests and `cargo check -p wa-rs-bridge`.**
- [ ] **Step 5: Commit** with `feat(iq): complete social IQ and notification coverage`.

### Phase 3 Gate

- [ ] `cargo test -p wa-rs --features parity-social --test parity_groups --test parity_community --test parity_newsletter`
- [ ] `cargo test -p wa-rs --features parity-social --test parity_status_polls_events --test parity_social_misc`
- [ ] `cargo test -p wa-rs-core --test parity_iq_social`
- [ ] `cargo check -p wa-rs --features parity-social`
- [ ] `cargo check -p wa-rs-bridge`
- [ ] `cargo build -p whatslite-desktop`
- [ ] Event surface test: 80 variants present

---

## Phase 4 — Calls and VoIP

**Phase goal:** upstream call control and media stack.

### Task 4.1: VoIP control boundary

**Files:**
- Create: `wacore/src/voip_control/`
- Create: `src/voip_control/`
- Modify: `Cargo.toml` (`parity-voip`)
- Test: `wacore/tests/parity_voip_control.rs`

**Interfaces:**
- Produces: neutral call-control traits and registry.
- Gate: core-level `voip-control` only; root `parity-voip` also enables it.

- [ ] **Step 1: Write failing tests** for call registry and control-plane boundary.
- [ ] **Step 2: Run `cargo test -p wa-rs-core --features voip-control --test parity_voip_control` and verify failure.**
- [ ] **Step 3: Port upstream `voip_control` modules.**
- [ ] **Step 4: Run the feature test and `cargo check -p wa-rs-core --features voip-control`.**
- [ ] **Step 5: Commit** with `feat(voip): port control boundary`.

### Task 4.2: VoIP media crypto

**Files:**
- Create: `wacore/src/voip/`
- Modify: `Cargo.toml` (`voip`)
- Test: `wacore/tests/parity_voip.rs`

**Interfaces:**
- Produces: SRTP, SFrame, WAHKDF, relay session traits.
- Gate: `voip` feature only.

- [ ] **Step 1: Write failing tests** for SRTP/SFrame vectors and relay session.
- [ ] **Step 2: Run `cargo test -p wa-rs-core --features voip --test parity_voip` and verify failure.**
- [ ] **Step 3: Port upstream `voip` media modules.**
- [ ] **Step 4: Run the feature test.**

```bash
cargo test -p wa-rs-core --features voip --test parity_voip
cargo check -p wa-rs-core --features voip
```

- [ ] **Step 5: Commit** with `feat(voip): port media crypto and relay`.

### Task 4.3: MLOW/Opus and video frames

**Files:**
- Create: `wacore/src/voip/mlow/`
- Modify: `Cargo.toml` (`voip-mlow`)
- Test: `wacore/tests/parity_voip_mlow.rs`

**Interfaces:**
- Produces: PCM/MLOW adapter, audio codec, H.264 frame boundary.

- [ ] **Step 1: Write failing vector tests** for MLOW encode/decode and DTX.
- [ ] **Step 2: Run `cargo test -p wa-rs-core --features voip-mlow --test parity_voip_mlow` and verify failure.**
- [ ] **Step 3: Port MLOW/Opus/video modules.**
- [ ] **Step 4: Run the test and the two checks below.**

```bash
cargo test -p wa-rs-core --features voip-mlow --test parity_voip_mlow
cargo check -p wa-rs --features parity-voip
```

- [ ] **Step 5: Commit** with `feat(voip): port MLOW and video frames`.

### Task 4.4: Call handlers, call log and call links

**Files:**
- Create: `src/handlers/call/`
- Create: `src/features/call_log.rs`
- Create: `src/voip/`
- Modify: `wacore/src/types/call.rs`
- Test: `tests/parity_calls.rs`

**Interfaces:**
- Produces: incoming/missed/ended events, call log, call links, screen sharing.

- [ ] **Step 1: Write failing tests** for incoming call, group call, call ended elsewhere, call log and call link.
- [ ] **Step 2: Run `cargo test -p wa-rs --features parity-voip --test parity_calls` and verify failure.**
- [ ] **Step 3: Port upstream call handlers and call log.**
- [ ] **Step 4: Run the test and the consumer check below.**

```bash
cargo test -p wa-rs --features parity-voip --test parity_calls
cargo check -p wa-rs --features parity-voip
```

- [ ] **Step 5: Commit** with `feat(calls): port handlers call log and call links`.

### Phase 4 Gate

- [ ] `cargo test -p wa-rs-core --features voip-control --test parity_voip_control`
- [ ] `cargo test -p wa-rs-core --features voip --test parity_voip`
- [ ] `cargo test -p wa-rs-core --features voip-mlow --test parity_voip_mlow`
- [ ] `cargo test -p wa-rs --features parity-voip --test parity_calls`
- [ ] `cargo check -p wa-rs --features parity-voip`
- [ ] `cargo check -p wa-rs-bridge` (default features)
- [ ] `cargo build -p whatslite-desktop`

---

## Phase 5 — Platform and observability

**Phase goal:** plugins, metrics/tracing/diagnostics, WASM/mobile storage, TCP transport and final parity audit.

### Task 5.1: Plugin host and lifecycle

**Files:**
- Create: `src/plugins/`
- Create: `src/client/extension_lifecycle.rs`
- Modify: `Cargo.toml` (`parity-plugins`)
- Test: `tests/parity_plugins.rs`

**Interfaces:**
- Produces: `ClientPlugin`, `UntypedClientPlugin`, plugin host config.

- [ ] **Step 1: Write failing tests** for plugin registration, lifecycle and event delivery.
- [ ] **Step 2: Run `cargo test -p wa-rs --features parity-plugins --test parity_plugins` and verify failure.**
- [ ] **Step 3: Port upstream plugin host and lifecycle.**
- [ ] **Step 4: Run the test and `cargo check -p wa-rs --features parity-plugins`.**
- [ ] **Step 5: Commit** with `feat(plugins): port plugin host and lifecycle`.

### Task 5.2: Metrics, tracing and diagnostics

**Files:**
- Create: `src/logging.rs`
- Create: `src/client/device_memo_stats.rs`
- Modify: `wacore/src/stats.rs`, `wacore/src/telemetry.rs`
- Test: `tests/parity_observability.rs`

**Interfaces:**
- Produces: `metrics` and `tracing` features, diagnostic events, memory reports.

- [ ] **Step 1: Write failing tests** for metric emission and tracing span creation.
- [ ] **Step 2: Run `cargo test -p wa-rs --features parity-observability --test parity_observability` and verify failure.**
- [ ] **Step 3: Port observability modules behind `parity-observability`.**
- [ ] **Step 4: Run the test and `cargo check -p wa-rs --features parity-observability`.**
- [ ] **Step 5: Commit** with `feat(observability): port metrics tracing and diagnostics`.

### Task 5.3: SQLite bundled/WASM/mobile and TCP transport

**Files:**
- Modify: `storages/sqlite-storage/Cargo.toml`
- Modify: `storages/sqlite-storage/src/`
- Modify: `transports/tokio-transport/Cargo.toml`
- Modify: `transports/tokio-transport/src/`
- Test: `storages/sqlite-storage/tests/parity_storage.rs`

**Interfaces:**
- Produces: `bundled-sqlite`, `js`, `mobile`, `wasm-test` features.
- Produces: `tcp-transport` feature.

- [ ] **Step 1: Write failing tests** for bundled storage, WASM storage contract and TCP transport.
- [ ] **Step 2: Run `cargo test -p wa-rs-sqlite-storage --test parity_storage` and verify failure.**
- [ ] **Step 3: Port storage/transport feature changes.**
- [ ] **Step 4: Run the native checks.**

```bash
cargo test -p wa-rs-sqlite-storage --test parity_storage
cargo check -p wa-rs-sqlite-storage --features bundled-sqlite
cargo check -p wa-rs-tokio-transport --features tcp-transport
```

WASM target checks are part of Task 5.4, not this task.
- [ ] **Step 5: Commit** with `feat(platform): port storage and transport profiles`.

### Task 5.4: WASM consumer and final feature matrix

**Files:**
- Create: `tests/parity_matrix.rs`
- Modify: `Cargo.toml`
- Test: `tests/parity_matrix.rs`

**Interfaces:**
- Produces: documented feature matrix and consumer checks.

- [ ] **Step 1: Write the matrix test** for default, parity-core, parity-messaging, parity-social, parity-voip, parity-mobile, parity-plugins.
- [ ] **Step 2: Run the matrix and verify expected failures/successes.**
- [ ] **Step 3: Fix feature leaks and missing gates.**
- [ ] **Step 4: Run all matrix commands.**

```bash
cargo check -p wa-rs --no-default-features
cargo check -p wa-rs --no-default-features --features parity-core
cargo check -p wa-rs --no-default-features --features parity-messaging
cargo check -p wa-rs --no-default-features --features parity-social
cargo check -p wa-rs --no-default-features --features parity-voip
cargo check -p wa-rs --no-default-features --features parity-mobile
cargo check -p wa-rs --no-default-features --features parity-plugins
cargo check -p wa-rs --no-default-features --features parity-observability
cargo check -p wa-rs --no-default-features --features parity-wasm
```

- [ ] **Step 4a: Run the WASM link check** (requires `rustup target add wasm32-unknown-unknown`):

```bash
cargo check -p wa-rs-sqlite-storage --target wasm32-unknown-unknown --features js
```

- [ ] **Step 5: Commit** with `test: add parity feature matrix`.

### Task 5.5: Final parity audit

**Files:**
- Create: `reference/design/plans/parity-report.md`
- Modify: `desktop/README.md` only if build instructions change
- Test: full workspace + bridge smoke

**Interfaces:**
- Produces: module-by-module parity table and known gaps.

- [ ] **Step 1: Write the audit checklist** comparing upstream `src/features`, `wacore/src`, and `wacore/src/iq` to the ported tree.
- [ ] **Step 2: Run the audit and list every missing module or method.**
- [ ] **Step 3: Close remaining gaps or record them as explicit non-goals.**
- [ ] **Step 4: Run the final gate.**
- [ ] **Step 5: Commit** with `docs: add parity audit`.

### Phase 5 Gate

- [ ] `cargo build -p whatslite-desktop`
- [ ] `cargo test --workspace`
- [ ] `cargo check --workspace`
- [ ] `cargo test -p wa-rs --features parity-plugins --test parity_plugins`
- [ ] `cargo test -p wa-rs --features parity-observability --test parity_observability`
- [ ] `cargo test -p wa-rs-sqlite-storage --test parity_storage`
- [ ] `cargo check -p wa-rs --no-default-features --features parity-core`
- [ ] `cargo check -p wa-rs --features parity-voip`
- [ ] `cargo check -p wa-rs --features parity-mobile`
- [ ] `cargo check -p wa-rs --features parity-plugins`
- [ ] `cargo check -p wa-rs-sqlite-storage --features bundled-sqlite`
- [ ] `cargo check -p wa-rs-tokio-transport --features tcp-transport`
- [ ] `cargo check -p wa-rs-sqlite-storage --target wasm32-unknown-unknown --features js`
- [ ] `cargo check -p wa-rs-bridge`
- [ ] Bridge pair/connect/send/receive/media/call smoke tests
- [ ] Parity report complete

---

## Final Definition of Done

- [ ] All upstream runtime feature modules exist under the current tree.
- [ ] `cargo build`, `cargo test --workspace`, `cargo check --workspace` pass.
- [ ] `bridge/` and `desktop/` compile on default features.
- [ ] Feature matrix passes.
- [ ] Event surface matches upstream 80 variants.
- [ ] No JS/Python/Kotlin/Java runtime code.
- [ ] Parity report lists any remaining explicit non-goals.

## Execution Handoff

Plan complete and saved to `reference/design/plans/2026-11-08-wa-rs-parity-plan.md` (repo) and `/home/muhammadmishbakhuzzuhail/.opencode/plan/2026-11-08-wa-rs-parity-plan.md` (working copy).

Recommended execution: **subagent-driven**, because the phases have many independent module tasks but share interfaces (`runtime`, `time`, `buffa` types, `Event`, `SessionEngine`). A fresh implementer per task and a fresh reviewer per phase gate reduces cross-module regressions.

Before implementation, review this plan and confirm it captures what you want.
