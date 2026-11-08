# wa-rs Parity with whatsapp-rust 0.7.0 — Implementation Plan (v2)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Port the full runtime feature set of `oxidezap/whatsapp-rust` 0.7.0 (`f077a5d`) into `homun-app/wa-rs` 0.2.0 without changing the current tree and without breaking `bridge/` or `desktop/`.

**Architecture:** Strangler migration per subsystem. Port upstream modules into the existing crate boundary (`wacore/`, `waproto/`, `src/`, `storages/`, `transports/`, `http_clients/`) using upstream feature names as the canonical gates. `parity-*` features are additive aliases. Core modules stay always compiled, matching upstream; only optional modules and dependencies are gated. The existing `wa-rs` API remains a compatibility facade.

**Tech Stack:** Rust 2024, Tokio, SQLite/Diesel, buffa 0.9.2, rustls, GPUI desktop consumer, `bridge/` HTTP sidecar.

**Spec:** `reference/design/plans/2026-11-08-wa-rs-parity-design.md`

## Global Constraints

- Keep the current top-level tree; do not add a new runtime crate.
- All runtime code is Rust. `.proto`, `.sql`, `.json`, `.yml`, Dockerfiles and test vectors are data/schema/tooling, not runtime languages.
- SQLite and TLS/ring may remain C/asm-backed.
- Rust 1.99.1 current; upstream MSRV 1.95 is compatible.
- `bridge/` and `desktop/` must compile at every phase gate.
- Upstream feature names are canonical. `parity-*` features are aliases and never replace upstream gates.
- Core modules (`runtime`, `time`, `stats`, `session_engine`, `sync_engine`, `message_processing`, ...) are always compiled. Do not add `parity-*` gates to them.
- Use Cargo dependency aliases and `extern crate self as ...` so ported upstream files keep `wacore::`, `wacore_binary::`, `waproto::` paths.
- Do not port upstream CI, Dockerfile, oracle tools, benchmark harnesses, or generated test-vector tooling.
- Excluded features: `bench-harness`, `test-support`, `session-simulator`, `dhat-heap`, `bench-internals`.
- Every task ends with a commit. Every phase ends with the phase gate.

## Prerequisites

- [ ] Install the Rust components:

```bash
rustup component add rustfmt clippy
cargo fmt --version
cargo clippy --version
```

- [ ] Provide the SQLite linker used by the workspace tests until Task 5.3 switches the default to bundled SQLite:

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

Expected: the baseline is green. If it is not, fix the environment first and record the failure; do not start Phase 0 on a red baseline.

- [ ] Confirm the reference checkout is at the pinned upstream commit:

```bash
git -C reference/whatsapp-rust rev-parse --short HEAD
```

Expected: `f077a5d`. Ported files are copied from this commit, not from a moving branch.

- [ ] Confirm buffa is available locally:

```bash
ls ~/.cargo/registry/cache/*/buffa-0.9.2.crate
ls ~/.cargo/registry/cache/*/buffa-build-0.9.2.crate
```

## Gate Protocol

Every phase gate runs this standard set first, from the repo root:

1. `cargo fmt --all -- --check`
2. `cargo clippy --workspace --all-targets`
3. `cargo test --workspace`
4. `cargo check --workspace`
5. `cargo build -p whatslite-desktop`
6. `cargo check -p wa-rs-bridge`

Then the phase-specific commands listed in that phase's gate. Phase-specific commands are additions, never replacements. Clippy must not add new warnings; record any pre-existing warning count in the phase commit message.

The bridge smoke test is manual: pair, connect, send, receive. Record the result in the phase commit message.

## Feature Contract

### Root canonical features

These are the upstream names. They are the only gates used inside ported code.

```toml
[features]
# Existing compatibility features
debug-snapshots = ["wacore/debug-snapshots"]
danger-skip-tls-verify = ["wa-rs-tokio-transport?/danger-skip-tls-verify"]
ureq-client = ["dep:wa-rs-ureq-http"]
tokio-transport = ["dep:wa-rs-tokio-transport"]
tokio-runtime = ["dep:tokio"]
tokio-native = ["tokio-runtime", "tokio/rt-multi-thread"]
signal = ["tokio-runtime", "tokio/signal"]
sqlite-storage = ["wa-rs-sqlite-storage"]
sqlite-storage-bundled = ["sqlite-storage", "wa-rs-sqlite-storage/bundled-sqlite"]

# Runtime optional features
instance-clock = ["wacore/instance-clock", "wa-rs-sqlite-storage?/instance-clock"]
legacy-session-interop = ["wacore/legacy-session-interop"]
client-lifecycle = []
plugins = ["client-lifecycle"]
passkey = []
mobile-core = ["wacore/mobile", "instance-clock", "dep:subtle", "wa-rs-sqlite-storage?/mobile"]
mobile-transport-native = ["mobile-core", "tokio-transport", "wa-rs-tokio-transport/tcp"]
generated-group-notifications = []
js = ["wacore/js", "wa-rs-sqlite-storage?/js"]
tracing = ["dep:tracing", "wacore/tracing"]
metrics = ["wacore/metrics"]
tracing-pii = ["wacore/tracing-pii", "wacore_binary/tracing-pii"]

# VoIP features
voip-control = ["wacore/voip-control"]
voip-runtime = ["voip-control", "voip-engine-wacore"]
voip-engine-wacore = ["voip-control", "wacore/voip"]
voip-relay-native = ["voip-runtime", "tokio-runtime", "tokio/net", "dep:rtc-dtls", "dep:rtc-sctp", "dep:rtc-datachannel", "dep:rtc-shared"]
voip = ["voip-mlow", "voip-libopus", "voip-relay-native"]
voip-encoded = ["voip-runtime"]
voip-mlow = ["voip-runtime", "wacore/voip-mlow"]
voip-libopus = ["voip-encoded", "dep:opus"]

# Parity aliases
parity-core = ["instance-clock", "legacy-session-interop"]
parity-messaging = ["parity-core"]
parity-social = ["parity-messaging"]
parity-voip = ["voip"]
parity-mobile = ["mobile-core"]
parity-plugins = ["plugins", "client-lifecycle"]
parity-observability = ["tracing", "metrics"]
parity-wasm = ["js"]
```

### Core canonical features

```toml
[features]
default = []
debug-snapshots = []
instance-clock = []
legacy-session-interop = ["wacore_libsignal/legacy-session-interop"]
tracing = ["dep:tracing"]
metrics = ["dep:metrics"]
tracing-pii = ["wacore_binary/tracing-pii"]
js = ["getrandom/wasm_js"]
mobile = []
voip-control = []
voip = ["voip-control", "dep:aes-gcm", "dep:zerocopy"]
voip-mlow = ["voip"]
```

### Rules

- Ported files keep `#[cfg(feature = "plugins")]`, `#[cfg(feature = "mobile-core")]`, `#[cfg(feature = "voip-control")]`, and so on.
- `parity-*` only adds edges. It never appears in a `#[cfg]` inside ported code.
- Core modules are always compiled. Only `mobile`, `voip-control`, `voip`, `voip-mlow`, `tracing`, `metrics`, `js` and `instance-clock` gate optional behavior.
- Never reference a feature that does not exist yet in a `Cargo.toml` edge; add the edge in the task that introduces the module.
- Task 0.2 declares every feature key as an empty shell. Later tasks add the dependency edges shown above.

## Crate Alias Strategy

### Cargo dependency aliases

Root `Cargo.toml`, `bridge/Cargo.toml`, `storages/sqlite-storage/Cargo.toml`, `transports/tokio-transport/Cargo.toml`, and `http_clients/ureq-client/Cargo.toml` depend on the workspace packages under upstream names:

```toml
wacore = { package = "wa-rs-core", path = "./wacore", version = "0.2.0" }
wacore_appstate = { package = "wa-rs-appstate", path = "./wacore/appstate", version = "0.2.0" }
wacore_binary = { package = "wa-rs-binary", path = "./wacore/binary", version = "0.2.0" }
wacore_derive = { package = "wa-rs-derive", path = "./wacore/derive", version = "0.2.0" }
wacore_libsignal = { package = "wa-rs-libsignal", path = "./wacore/libsignal", version = "0.2.0" }
wacore_noise = { package = "wa-rs-noise", path = "./wacore/noise", version = "0.2.0" }
waproto = { package = "wa-rs-proto", path = "./waproto", version = "0.2.0" }
```

`wacore/Cargo.toml` uses the same aliases for its sibling crates. Paths are adjusted relative to the owning crate.

### Self aliases

Add one line to each crate root so upstream `use` paths resolve inside the crate:

```rust
// wacore/src/lib.rs
extern crate self as wacore;

// wacore/binary/src/lib.rs
extern crate self as wacore_binary;

// waproto/src/lib.rs
extern crate self as waproto;

// wacore/derive/src/lib.rs
extern crate self as wacore_derive;

// wacore/appstate/src/lib.rs
extern crate self as wacore_appstate;

// wacore/noise/src/lib.rs
extern crate self as wacore_noise;

// wacore/libsignal/src/lib.rs
extern crate self as wacore_libsignal;

// src/lib.rs
extern crate self as whatsapp_rust;
```

### Mechanical rename

In Phase 0 Task 0.1, rename every existing `wa_rs_core::`, `wa_rs_binary::`, `wa_rs_proto::`, `wa_rs_derive::`, `wa_rs_appstate::`, `wa_rs_noise::`, and `wa_rs_libsignal::` path to its upstream alias. This is a one-time mechanical change; later tasks write upstream paths directly.

## Module Inventory

This is the completeness checklist. A phase is not done while a runtime module in its area is missing.

| Area | Upstream count | Phase | Notes |
|---|---:|---|---|
| `wacore/src` top-level | 46 `.rs` files | 0–4 | Includes `bot_message`, `client`, `comment`, `companion_reg`, `event`, `messages`, `reporting_token`, `secret_enc_addon`, `sync_marker`, `download`, `upload` |
| `wacore/src/iq` | 33 `.rs` files + `usync/` | 3 | Includes `mex_operations`, `mex_operations_unlisted`, `pilots`, `pilot_conformance`, `targets` |
| `wacore/src/types` | 13 `.rs` files + `message_ref/` | 0–4 | Includes `call.rs`, `group_call.rs`, `message_secret.rs`, `wire_enums.rs` |
| `wacore/src/voip` | 20 files + `mlow/`, `sframe/`, `testdata/` | 4 | Exclude `testdata/` from runtime crates; vendor only required fixtures |
| `wacore/src/voip_control` | 19 files | 4 | Boundary types and traits |
| `src/features` | 34 non-test modules | 3 | All 34 must exist and be re-exported from `src/features/mod.rs` |
| `src/client` | 28 `.rs` files + `app_state/`, `lifecycle/`, `tests/` | 1 | Includes `accessors`, `adapters`, `context_impl`, `device_registry`, `device_topology`, `group_notifications`, `interceptor`, `lid_pn`, `member_index`, `messaging`, `notification_scope`, `sender_keys`, `sessions`, `subsystem` |
| `src/message` | 11 modules | 2 | Includes `msg_secret.rs` |
| `src/send` | 5 modules | 2 | `actions`, `group_repair`, `requests`, `tctoken_lifecycle`, `mod` |
| `src/handlers` | 12 `.rs` files + `call/`, `notification/` | 1–4 | Includes `basic`, `chatstate`, `ib`, `iq`, `message`, `presence`, `receipt`, `router`, `traits` |
| `src/types` | 10 modules | 1–2 | Admission, durability, enc handler, group notification facts |
| `src/cache*` | 4 modules | 2 | `cache`, `cache_config`, `cache_store`, `portable_cache` |
| `src/pdo` | `pdo.rs` + tests | 2 | Port runtime logic and in-crate tests |
| `src/history_sync` | `history_sync.rs` + tests | 2 | Port runtime logic and in-crate tests |
| `src/request` | `request.rs` + tests | 2 | Port runtime logic and in-crate tests |
| `src/usync` | `usync.rs` + tests | 2 | Port runtime logic and in-crate tests |
| `src/keepalive` | `keepalive.rs` + tests | 2 | Port runtime logic and in-crate tests |
| `src/lid_pn_cache` | `lid_pn_cache.rs` + `implementation.rs` | 2 | Port runtime logic |
| `src/socket` | 3 modules | 2 | Port runtime logic |
| `src/store` | 11 modules | 2 | Port runtime logic |
| `src/plugins` | 3 modules | 5 | `events`, `iq_reply`, `mod` |
| `src/mobile` | 2 modules + tests | 1 | `primary`, `registration` |
| `src/passkey` | 2 modules | 1 | `flow`, `mod` |
| `src/voip` | 14 files + `facade/`, `transport/` | 4 | Includes `audio`, `direct`, `driver`, `events`, `registry`, `session`, `state`, `video` |
| `src/voip_control` | 2 modules | 4 | `mod`, `wacore_backend` |

Excluded from runtime crates: `bench_support/`, `test_support/`, `test_utils.rs`, `reexports_test.rs`, `request_context_tests.rs`, `benches/`, `tools/`, `whatspec/`, `unwasm/`, CI files, Dockerfiles, and generated test-vector tooling.

## Phase 0 — Foundation

**Phase goal:** create the build, naming and compatibility foundation. No user-facing feature is added.

### Task 0.1: Add crate aliases and rename imports

**Files:**
- Modify: `Cargo.toml`, `wacore/Cargo.toml`, `wacore/binary/Cargo.toml`, `wacore/derive/Cargo.toml`, `wacore/appstate/Cargo.toml`, `wacore/noise/Cargo.toml`, `wacore/libsignal/Cargo.toml`, `waproto/Cargo.toml`, `bridge/Cargo.toml`, `storages/sqlite-storage/Cargo.toml`, `transports/tokio-transport/Cargo.toml`, `http_clients/ureq-client/Cargo.toml`
- Modify: `src/lib.rs`, `wacore/src/lib.rs`, `wacore/binary/src/lib.rs`, `waproto/src/lib.rs`, `wacore/derive/src/lib.rs`, `wacore/appstate/src/lib.rs`, `wacore/noise/src/lib.rs`, `wacore/libsignal/src/lib.rs`
- Modify: every `.rs` file using `wa_rs_core::`, `wa_rs_binary::`, `wa_rs_proto::`, `wa_rs_derive::`, `wa_rs_appstate::`, `wa_rs_noise::`, `wa_rs_libsignal::`
- Test: `cargo check --workspace`

**Interfaces:**
- Produces: dependency names `wacore`, `wacore_binary`, `waproto`, `wacore_derive`, `wacore_appstate`, `wacore_noise`, `wacore_libsignal`.
- Produces: self aliases in every crate root.
- Preserves: package names, folder tree, and public crate names for external consumers.

- [ ] **Step 1: Rename the dependency keys**

Rename the existing `[workspace.dependencies]` keys and every member manifest entry from `wa-rs-core`, `wa-rs-binary`, `wa-rs-proto`, and the other sibling packages to the upstream aliases. Keep the package names unchanged. Do not keep both keys for the same package; Cargo rejects duplicate package dependencies. Update every `workspace = true` reference to the new key.

- [ ] **Step 2: Add the self aliases**

Add `extern crate self as ...;` to each crate root that does not already have one. `wacore/src/lib.rs` already has `extern crate self as wa_rs_core;`; the mechanical rename converts it to `wacore`, so do not add a duplicate. Add `extern crate self as whatsapp_rust;` to `src/lib.rs`.

- [ ] **Step 3: Rename the imports**

Run:

```bash
rg -l 'wa_rs_(core|binary|proto|derive|appstate|noise|libsignal)' \
  --glob '*.rs' --glob '!reference/**' --glob '!target/**' | xargs sed -i \
  -e 's/wa_rs_core/wacore/g' \
  -e 's/wa_rs_binary/wacore_binary/g' \
  -e 's/wa_rs_proto/waproto/g' \
  -e 's/wa_rs_derive/wacore_derive/g' \
  -e 's/wa_rs_appstate/wacore_appstate/g' \
  -e 's/wa_rs_noise/wacore_noise/g' \
  -e 's/wa_rs_libsignal/wacore_libsignal/g'
```

Do not touch `reference/whatsapp-rust`; it is the pinned upstream checkout.

Do not rename the `wa-rs` package itself or the `wa_rs` crate name. Only the internal sibling crates are renamed.

- [ ] **Step 4: Verify**

```bash
cargo check --workspace
cargo build -p whatslite-desktop
```

Expected: both succeed.

- [ ] **Step 5: Commit**

```bash
git add Cargo.toml wacore waproto bridge storages transports http_clients src
git commit -m "build: alias workspace crates to upstream names"
```

### Task 0.2: Declare the feature contract

**Files:**
- Modify: `Cargo.toml`
- Modify: `wacore/Cargo.toml`
- Modify: `storages/sqlite-storage/Cargo.toml`
- Modify: `transports/tokio-transport/Cargo.toml`
- Modify: `src/lib.rs`
- Test: feature matrix

**Interfaces:**
- Produces: root features from the Feature Contract section.
- Produces: core features from the Feature Contract section.
- Produces: `parity-core`, `parity-messaging`, `parity-social`, `parity-voip`, `parity-mobile`, `parity-plugins`, `parity-observability`, `parity-wasm`.

- [ ] **Step 1: Declare every feature key as an empty shell**

Add every canonical feature and every parity alias to the root manifest. Add the core features to `wacore/Cargo.toml`. Add the storage features `bundled-sqlite`, `js`, `mobile`, `wasm-test` and the transport feature `tcp` as empty shells. Do not add dependency edges yet.

- [ ] **Step 2: Add the parity aliases**

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

- [ ] **Step 3: Verify the default build is unchanged**

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
cargo check -p whatslite-desktop
```

Expected: all succeed because every key is an empty shell.

- [ ] **Step 4: Commit**

```bash
git add Cargo.toml wacore/Cargo.toml storages/sqlite-storage/Cargo.toml transports/tokio-transport/Cargo.toml src/lib.rs
git commit -m "build: declare upstream and parity feature contract"
```

### Task 0.3: Port runtime, time, stats, telemetry, diagnostics and sync markers

**Files:**
- Create: `wacore/src/runtime.rs`, `wacore/src/runtime/clocked.rs`, `wacore/src/runtime/crypto.rs`
- Create: `wacore/src/time.rs`, `wacore/src/time/clock.rs`
- Create: `wacore/src/stats.rs`
- Create: `wacore/src/telemetry.rs`
- Create: `wacore/src/diagnostics.rs`
- Create: `wacore/src/sync_marker.rs`
- Modify: `wacore/src/lib.rs`
- Modify: `wacore/Cargo.toml`
- Test: `wacore/tests/parity_runtime.rs`, `wacore/tests/parity_stats.rs`

**Interfaces:**
- Produces: `Runtime`, `CryptoRuntime`, `ClockedRuntime`, `BoxFuture`, `Spawnable`, `AbortHandle`, `ShutdownNotifier`, `ShutdownSignal`, `timeout`, `wait_for_shutdown`.
- Produces: `TimeProvider`, `MonotonicProvider`, `Clock`, `ClockReading`, `ClockError`, `GlobalClock`, `VirtualClock`, `Instant`, `AtomicInstant`, `now_millis`, `now_secs`, `now_utc`, `from_secs`, `from_millis`.
- Produces: `SessionStats`, `StatsSnapshot`, `UnkeyableDevice`, `AllocMeter`, `AllocSnapshot`, `HeapSize`, `CollectionStats`, `TaskInstrument`, `MeteredFuture`, `CpuMeter`, `InstrumentedRuntime`.
- Produces: `telemetry` metric functions and `Timer`; `ErrorSummary`; `MaybeSend`, `MaybeSendSync`.

- [ ] **Step 1: Write the failing tests**

Create `wacore/tests/parity_runtime.rs` and `wacore/tests/parity_stats.rs` covering monotonic clock behavior, `VirtualClock::advance`, runtime shutdown, allocation metering, and `ErrorSummary` formatting.

- [ ] **Step 2: Run them and verify failure**

```bash
cargo test -p wa-rs-core --test parity_runtime --test parity_stats
```

Expected: FAIL with unresolved modules and types.

- [ ] **Step 3: Port the modules**

Copy the upstream files listed above. Keep the upstream `#[cfg(feature = "instance-clock")]`, `#[cfg(feature = "tracing")]`, and `#[cfg(feature = "metrics")]` attributes. Add `dep:tracing`, `dep:metrics`, `portable-atomic`, `event-listener`, and any missing dependencies to `wacore/Cargo.toml`.

- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs-core --test parity_runtime --test parity_stats
cargo test -p wa-rs-core --features instance-clock --test parity_runtime
cargo test -p wa-rs-core --features metrics,tracing --test parity_runtime
cargo check -p wa-rs-core
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add wacore/src/runtime.rs wacore/src/runtime wacore/src/time.rs wacore/src/time wacore/src/stats.rs wacore/src/telemetry.rs wacore/src/diagnostics.rs wacore/src/sync_marker.rs wacore/src/lib.rs wacore/Cargo.toml wacore/tests/parity_runtime.rs wacore/tests/parity_stats.rs
git commit -m "feat(core): port runtime time stats telemetry diagnostics and sync markers"
```

### Task 0.4: Add the WireEnum derive macro

**Files:**
- Modify: `wacore/derive/src/lib.rs`
- Modify: `wacore/src/lib.rs`
- Test: `wacore/tests/parity_wire_enum.rs`

**Interfaces:**
- Produces: `WireEnum` derive with `wire`, `wire_alias`, `wire_default`, `wire_fallback` attributes.
- Produces: `as_str()`, `code()`, `wire_tag()`, `tag_name()`, `TryFrom<&str>`, `Default`.
- Preserves: existing `StringEnum` derive and all 12 current uses.

- [ ] **Step 1: Write the failing test**

Create `wacore/tests/parity_wire_enum.rs` with a small enum deriving `WireEnum` and assertions for `as_str`, `code`, `wire_tag`, `TryFrom<&str>`, and default fallback.

- [ ] **Step 2: Run it and verify failure**

```bash
cargo test -p wa-rs-core --test parity_wire_enum
```

Expected: FAIL because `WireEnum` is not exported.

- [ ] **Step 3: Port the macro**

Copy upstream `WireEnum` from `reference/whatsapp-rust/wacore/derive/src/lib.rs` into `wacore/derive/src/lib.rs`. Keep `StringEnum` in place. Re-export both from `wacore/src/lib.rs`.

Upstream uses `syn 3.0`; the current derive crate uses `syn 2.0`. Upgrade `wacore/derive/Cargo.toml` to `syn = { version = "3.0", features = ["full"] }` and run the existing derive tests. If `syn 3` breaks `ProtocolNode`, `EmptyNode` or `StringEnum`, port the upstream versions of those three macros as well.

- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs-core --test parity_wire_enum
cargo test -p wa-rs-core --features instance-clock
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add wacore/derive/src/lib.rs wacore/src/lib.rs wacore/tests/parity_wire_enum.rs
git commit -m "feat(core): add WireEnum derive alongside StringEnum"
```

### Task 0.5: Switch waproto to buffa

This task is split into three commits. Do not combine them.

#### Task 0.5a: Port the buffa codegen pipeline

**Files:**
- Modify: `waproto/Cargo.toml`, `waproto/build.rs`, `waproto/src/lib.rs`
- Create: `waproto/build_support/`, `waproto/build/`, `waproto/src/whatsapp.desc`, `waproto/src/whatsapp.desc.sha256`, `waproto/api.snapshot`, `waproto/signal-storage.snapshot`
- Test: `waproto/tests/proto_roundtrip.rs`, `waproto/tests/build_freshness.rs`

**Interfaces:**
- Produces: buffa-generated types under `waproto::whatsapp`.
- Produces: view types such as `MessageView`, `ADVSignedDeviceIdentityView`, `ADVSignedKeyIndexListView`.
- Produces: `waproto` features `serde-deserialize`, `serde-snake-case`, `serde-enum-repr`.

- [ ] **Step 1: Write the failing tests**

Create `waproto/tests/proto_roundtrip.rs` and `waproto/tests/build_freshness.rs`.

- [ ] **Step 2: Run them and verify failure**

```bash
cargo test -p wa-rs-proto --test proto_roundtrip --test build_freshness
```

Expected: FAIL because buffa codegen is not present.

- [ ] **Step 3: Port the pipeline**

Copy the upstream `waproto` build pipeline and compatibility layer. Keep the package name `wa-rs-proto` and path `waproto/`. Do not vendor `whatspec` or `unwasm`; the committed descriptor and snapshots are the source of truth.

- [ ] **Step 4: Verify codegen**

```bash
cargo test -p wa-rs-proto --test proto_roundtrip --test build_freshness
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add waproto
git commit -m "refactor(proto): port buffa codegen pipeline"
```

#### Task 0.5b: Migrate call sites to buffa types

**Files:**
- Modify: `wacore/src/`, `src/`, `bridge/src/main.rs`
- Test: `cargo check -p wa-rs-core`, `cargo check -p wa-rs`, `cargo check -p wa-rs-bridge`

**Interfaces:**
- Produces: buffa-shaped proto access across `wacore`, `src`, and `bridge`.
- Preserves: `bridge` message construction through `waproto::whatsapp`.

- [ ] **Step 1: Migrate `wacore`**

Replace prost-specific field access, `Box<T>` wrapping, and `Default` construction with the buffa equivalents. Work in this order and run `cargo check -p wa-rs-core` after each group:

1. `wacore/src/proto_helpers.rs`
2. `wacore/src/send.rs`
3. `wacore/src/types/`
4. `wacore/src/store/`
5. `wacore/src/download.rs`, `wacore/src/history_sync.rs`
6. The remaining `wacore/src/` call sites.

This is the largest task in Phase 0. Do not batch it with unrelated changes.

- [ ] **Step 2: Migrate `src`**

Update `src/bot.rs`, `src/appstate_sync.rs`, `src/message.rs`, `src/send.rs`, and the feature modules that construct proto messages.

- [ ] **Step 3: Migrate `bridge`**

`bridge/src/main.rs` constructs `wa::Message` and `wa::message::ExtendedTextMessage`. Update it to the buffa constructors.

- [ ] **Step 4: Verify**

```bash
cargo check -p wa-rs-core
cargo check -p wa-rs
cargo check -p wa-rs-bridge
cargo check -p whatslite-desktop
```

Expected: all succeed.

- [ ] **Step 5: Commit**

```bash
git add wacore src bridge
git commit -m "refactor(proto): migrate call sites to buffa types"
```

#### Task 0.5c: Add proto compatibility tests

**Files:**
- Create: `waproto/tests/generated_api.rs`, `waproto/tests/retained_schema.rs`, `waproto/tests/serde_enum_repr.rs`
- Test: `cargo test -p wa-rs-proto`

- [ ] **Step 1: Port the upstream proto tests** that validate the generated API, retained schema fields, and serde enum representation.
- [ ] **Step 2: Run them and fix the compatibility layer** until they pass.
- [ ] **Step 3: Verify**

```bash
cargo test -p wa-rs-proto
cargo check -p wa-rs-bridge
```

- [ ] **Step 4: Commit**

```bash
git add waproto/tests
git commit -m "test(proto): cover buffa generated API and retained schema"
```

### Task 0.6: Port the event model and delivery

**Files:**
- Modify: `wacore/src/types/events.rs`
- Create: `wacore/src/event.rs`
- Modify: `src/bot.rs`
- Create: `src/bot/event_delivery.rs`, `src/bot/event_delivery/`
- Test: `wacore/tests/parity_events.rs`, `tests/parity_event_delivery.rs`

**Interfaces:**
- Produces: `EventKind` with 80 variants and `Event` with 80 variants.
- Produces: `EventInterest`, `EventHandler`, `CoreEventBus`, `Subscription`, `ChannelEventHandler`, `ChannelEventStats`, `LazyHistorySync`.
- Produces: `EventDelivery`, `EventDeliveryStats`, `CallbackEventHandler`.
- Preserves: current `Event` names used by `bridge` and `src/bot.rs`.

- [ ] **Step 1: Write the failing tests**

Create `wacore/tests/parity_events.rs` with a count assertion for 80 `EventKind` and 80 `Event` variants, serde round-trips, and interest-bit tests. Create `tests/parity_event_delivery.rs` for delivery modes and hook registration.

- [ ] **Step 2: Run them and verify failure**

```bash
cargo test -p wa-rs-core --test parity_events
cargo test -p wa-rs --test parity_event_delivery
```

Expected: FAIL because `EventKind`, `CoreEventBus`, and `EventDelivery` are missing.

- [ ] **Step 3: Port the event model**

Copy upstream `wacore/src/types/events.rs` and `wacore/src/event.rs`. Add the current `Event` variants that upstream no longer has as compatibility adapters if `bridge` still uses them. Port `src/bot/event_delivery.rs` and `src/bot/event_delivery/`.

- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs-core --test parity_events
cargo test -p wa-rs --test parity_event_delivery
cargo check -p wa-rs-bridge
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add wacore/src/types/events.rs wacore/src/event.rs src/bot.rs src/bot/event_delivery.rs src/bot/event_delivery wacore/tests/parity_events.rs tests/parity_event_delivery.rs
git commit -m "feat(events): port 80-variant event model and delivery"
```

### Phase 0 Gate

- [ ] `cargo test -p wa-rs-core --test parity_runtime --test parity_stats --test parity_wire_enum --test parity_events`
- [ ] `cargo test -p wa-rs-proto`
- [ ] `cargo test -p wa-rs --test parity_event_delivery`
- [ ] `cargo build -p whatslite-desktop`
- [ ] `cargo test --workspace`
- [ ] `cargo check --workspace`
- [ ] `cargo check -p wa-rs --no-default-features --features parity-core`
- [ ] `cargo check -p wa-rs-bridge`
- [ ] Bridge smoke test: pair, connect, send, receive

---

## Phase 1 — Auth, session and recovery

**Phase goal:** upstream auth/session lifecycle, passkey and mobile core.

### Task 1.1: Port the core client context, session engine and sync engine

**Files:**
- Modify: `wacore/src/client.rs`, `wacore/src/client/context.rs`
- Create: `wacore/src/session_engine/{mod,exchange,lifecycle,publication,retry}.rs`
- Create: `wacore/src/sync_engine/{mod,app_state,history}.rs`
- Create: `wacore/src/shortcake.rs`
- Modify: `wacore/src/lib.rs`
- Test: `wacore/tests/parity_session.rs`, `wacore/tests/parity_sync.rs`

**Interfaces:**
- Produces: `RegisteredExchange`, `SendingExchange`, `AwaitingExchange`, `CompletedExchange`.
- Produces: `WorkerGeneration`, `AttemptTicket`, `TransportTicket`, `AuthenticationTicket`, `CleanupTicket`, `DriveLease`, `DriveKind`, `SupervisorPhase`, `DriveState`, `TransportPhase`, `AuthenticationPhase`, `ReconnectPolicy`, `DisconnectIntent`, `DesiredConnection`, `StopReason`, `NextConnection`, `BackoffRequest`, `LifecycleError`, `LifecycleSnapshot`.
- Produces: `SyncOutcome`, `SyncScope`, `ScopeLost`, `SyncHolder`, `ReservationSkip`, `SyncSettles`, `ReservationWait`, `BatchedSyncRequest`, `StandDown`, `BatchedSyncOutcome`, `CriticalSyncPlan`, `RetryBudget`, `HistoryRecovery`, `HistoryAdmission`, `HistoryReceiptPoint`, `HistoryPublication`, `HistoryGeneration`, `HistoryActivitySnapshot`, `HistoryActivity`.
- Consumes: `runtime::Runtime`, `time::Clock`, `store::traits`.

- [ ] **Step 1: Write failing tests** for session transitions, reconnect policy, sync admission, history admission and publication.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs-core --test parity_session --test parity_sync
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/wacore/src/client/`, `session_engine/`, `sync_engine/` and `shortcake.rs`. Keep the upstream module paths and type names.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs-core --test parity_session --test parity_sync
cargo check -p wa-rs-core
```

- [ ] **Step 5: Commit** with `feat(core): port client context session and sync engines`.

### Task 1.2: Port ADV, identity verification, companion registration and client profile

**Files:**
- Create: `wacore/src/adv.rs`, `wacore/src/adv/`
- Create: `wacore/src/identity_verification.rs`
- Modify: `wacore/src/companion_reg.rs`
- Create: `wacore/src/client_profile.rs`
- Modify: `wacore/src/iq/prekeys.rs`, `wacore/src/iq/usync.rs`
- Test: `wacore/tests/parity_adv.rs`, `wacore/tests/parity_identity.rs`

**Interfaces:**
- Produces: `AdvValidation`, `validate_adv_with_identity_key`, `IdentityChange`.
- Produces: `CompanionReg`, `ClientProfile`.
- Consumes: buffa `ADVSignedDeviceIdentityView`, `ADVSignedKeyIndexListView`.

- [ ] **Step 1: Write failing tests** for valid/invalid/no-account-key ADV, hosted/companion prefixes, identity change, and companion registration parsing.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs-core --test parity_adv --test parity_identity
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/wacore/src/adv.rs`, `adv/`, `identity_verification.rs`, `companion_reg.rs`, `client_profile.rs`, and the USync/device-list parsing changes.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs-core --test parity_adv --test parity_identity
cargo check -p wa-rs-core
```

- [ ] **Step 5: Commit** with `feat(core): port ADV identity companion registration and client profile`.

### Task 1.3: Port pair-code lifecycle, passkey and companion notifications

**Files:**
- Modify: `src/pair_code.rs`
- Create: `src/passkey/mod.rs`, `src/passkey/flow.rs`
- Replace: `src/handlers/notification.rs` with `src/handlers/notification/mod.rs`
- Create: `src/handlers/notification/companion_reg.rs`
- Modify: `src/bot.rs`
- Modify: `Cargo.toml` (`passkey`)
- Test: `tests/parity_pair_code.rs`, `tests/parity_passkey.rs`

**Interfaces:**
- Produces: `PairError`, `PairCodeRejection`, `pair_with_code`, `cancel_pair_code`.
- Produces: `UserVerification`, `AssertionRequest`, `Assertion`, `PasskeyError`, `PasskeyAuthenticator`, `CallbackAuthenticator`, `parse_request_options`, `build_webauthn_assertion_json`, `set_passkey_authenticator`, `send_passkey_response`, `send_passkey_confirmation`.
- Gate: `passkey`.

- [ ] **Step 1: Write failing tests** for pair-code refresh/cancel/exhausted and passkey request/response/confirmation.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --test parity_pair_code
cargo test -p wa-rs --features passkey --test parity_passkey
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/src/pair_code.rs`, `src/passkey/`, and `src/handlers/notification/companion_reg.rs`. Keep the existing `pair_with_code` signature as a wrapper.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --test parity_pair_code
cargo test -p wa-rs --features passkey --test parity_passkey
cargo check -p wa-rs-bridge
```

- [ ] **Step 5: Commit** with `feat(auth): port pair-code lifecycle and passkey`.

### Task 1.4: Port client lifecycle, builder, node I/O, offline resume and IQ ops

**Files:**
- Modify: `src/client.rs`
- Create: `src/client/{lifecycle.rs,builder.rs,node_io.rs,offline_resume.rs,session_state.rs,iq_ops.rs,accessors.rs,adapters.rs,app_state.rs,device_topology.rs,durability_probe_id.rs,group_notifications.rs,interceptor.rs,member_index.rs,messaging.rs,notification_scope.rs,subsystem.rs}`
- Modify: `src/client/{context_impl.rs,device_registry.rs,lid_pn.rs,sender_keys.rs,sessions.rs}`
- Create: `src/client/app_state/`, `src/client/lifecycle/`, `src/client/tests.rs`, `src/client/tests/`
- Create: `src/types/{connect_admission.rs,decrypt_failure_admission.rs,durability_hook.rs,group_notification_durability.rs,group_notification_facts.rs,history_sync_admission.rs,history_sync_durability.rs,retry_admission.rs}`
- Modify: `src/types/{enc_handler.rs,mod.rs}`
- Test: `tests/parity_client.rs`, `tests/parity_admission.rs`

**Interfaces:**
- Produces: `ClientOptions`, `ClientBuilder`, `ClientBuild`, `ClientBuilderError`, `ConnectStage`, `ConnectError`, `NodeFilter`, `NodeWaiter<T>`, `ClientError`, `SignalMaintenanceError`.
- Produces: `ConnectAdmission`, `DecryptFailureAdmission`, `DecryptFailureContext`, `DecryptFailureDecision`, `InboundDurabilityHook`, `HistorySyncCaptureHook`, `HistorySyncAdmission`, `HistorySyncDecision`, `HistorySyncMetadata`, `RetryAdmission`, `RetryContext`, `RetryDecision`.
- Preserves: existing `Client::new`, `run`, `connect`, `disconnect`, `send_node`.

- [ ] **Step 1: Write failing tests** for connect stages, node filter/waiter, offline resume, admission decisions and durability hooks.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --test parity_client --test parity_admission
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/src/client/`, `src/types/`, and the upstream `src/client.rs`. Keep the old API as compatibility wrappers.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --test parity_client --test parity_admission
cargo check -p wa-rs
cargo check -p wa-rs-bridge
```

- [ ] **Step 5: Commit** with `feat(client): port lifecycle builder node IO offline resume and admissions`.

### Task 1.5: Port mobile registration, primary session and mobile API

**Files:**
- Create: `wacore/src/mobile/{mod,environment,primary,registration,registration_wire,session,token}.rs`
- Create: `src/mobile/{primary.rs,registration.rs}`
- Create: `src/mobile_session.rs`
- Create: `src/features/mobile_api.rs`
- Modify: `Cargo.toml` (`mobile-core`, `mobile-transport-native`)
- Modify: `wacore/Cargo.toml` (`mobile`)
- Test: `tests/parity_mobile.rs`

**Interfaces:**
- Produces: `MobileConfigurationError`, `RegistrationEndpoint`, `TcpEndpoint`, `MobileTrust`, `EnvironmentBinding`, `MobileEnvironment`.
- Produces: `PendingCompanion`, `AcknowledgedCompanion`, `VerifiedCompanion`.
- Produces: `MobileAppStatePolicy`, `MobileEphemeralKeyProvider`, `MobileLoginAttempt`, `MobileLoginOptionsProvider`, `RegisteredMobileSession`, `RegistrationLoginPhoneIdMapping`, `PrimarySelfIdentityProfile`, `PrimaryDeviceIdentityPolicy`, `MobileMediaHttpProfile`, `validate_registration_login_profile`.
- Gate: `mobile-core` (root), `mobile` (core).

- [ ] **Step 1: Write failing tests** for registration request/response, primary session binding, environment binding and mobile app-state policy.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --features mobile-core --test parity_mobile
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/wacore/src/mobile/`, `src/mobile/`, `src/mobile_session.rs`, and `src/features/mobile_api.rs`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --features mobile-core --test parity_mobile
cargo check -p wa-rs --features mobile-core
```

- [ ] **Step 5: Commit** with `feat(mobile): port registration primary session and mobile API`.

### Phase 1 Gate

- [ ] `cargo test -p wa-rs-core --test parity_session --test parity_sync --test parity_adv --test parity_identity`
- [ ] `cargo test -p wa-rs --test parity_pair_code --test parity_client --test parity_admission`
- [ ] `cargo test -p wa-rs --features passkey --test parity_passkey`
- [ ] `cargo test -p wa-rs --features mobile-core --test parity_mobile`
- [ ] `cargo check -p wa-rs --features parity-core`
- [ ] `cargo check -p wa-rs --features parity-mobile`
- [ ] `cargo check -p wa-rs-bridge`
- [ ] `cargo build -p whatslite-desktop`
- [ ] Bridge pairing and reconnect smoke test

---

## Phase 2 — Messaging, media and reliability

**Phase goal:** upstream message pipeline, send pipeline, retry/durability, media streaming and recovery.

### Task 2.1: Port core message processing and secrets

**Files:**
- Create: `wacore/src/message_processing.rs`
- Create: `wacore/src/message_edit.rs`
- Create: `wacore/src/msg_secret.rs`
- Create: `wacore/src/reaction.rs`
- Create: `wacore/src/media_retry.rs`
- Create: `wacore/src/webp.rs`
- Create: `wacore/src/zip.rs`
- Create: `wacore/src/sticker_pack.rs`
- Create: `wacore/src/bot_message.rs`
- Create: `wacore/src/comment.rs`
- Modify: `wacore/src/messages.rs`
- Modify: `wacore/src/reporting_token.rs`
- Create: `wacore/src/secret_enc_addon.rs`
- Modify: `wacore/src/lib.rs`
- Test: `wacore/tests/parity_messaging.rs`, `wacore/tests/parity_secrets.rs`

**Interfaces:**
- Produces: `EncType`, `EncNodeInfo`, `CategorizedEncNodes`, `categorize_enc_nodes`, `ProtocolMessageInfo`, `DecryptedMessageResult`, `process_decrypted_plaintext`.
- Produces: `EncryptedEdit`, `SecretEncrypted`, `SecretEncKind`, `decrypt_secret_encrypted`, `decrypt_secret_encrypted_with_fallback`.
- Produces: `BotMessage`, `Comment`, `SecretEncAddon`.

- [ ] **Step 1: Write failing tests** for enc-node categorization, decrypted plaintext routing, edit envelopes, secret encryption/decryption and reaction payloads.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs-core --test parity_messaging --test parity_secrets
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/wacore/src/message_processing.rs`, `message_edit.rs`, `msg_secret.rs`, `reaction.rs`, `media_retry.rs`, `webp.rs`, `zip.rs`, `sticker_pack.rs`, `bot_message.rs`, `comment.rs`, `messages.rs`, `secret_enc_addon.rs`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs-core --test parity_messaging --test parity_secrets
cargo check -p wa-rs-core
```

- [ ] **Step 5: Commit** with `feat(core): port message processing edits secrets reactions and stickers`.

### Task 2.2: Port the inbound pipeline, durability and inbound queue

**Files:**
- Modify: `src/message.rs`
- Create: `src/message/{commit_batch.rs,dispatch.rs,durability.rs,history_durability_tests.rs,live_retention.rs,msg_secret.rs,raw_pending.rs,receive.rs,retry.rs,special.rs,tests.rs}`
- Create: `src/msg_secret_buffer.rs`
- Create: `src/inbound_queue.rs`
- Create: `src/flush_scope.rs`
- Create: `src/signal_flush.rs`
- Test: `tests/parity_inbound.rs`

**Interfaces:**
- Produces: `InboundDurabilityHook`, `HistorySyncCaptureHook`, `InboundMessage`, `MessageBatch`, `BatchOrigin`.
- Produces: `SecretFlushReport`, `DrainOutcome`.
- Consumes: `types::durability_hook`, `types::history_sync_durability`.

- [ ] **Step 1: Write failing tests** for commit ordering, live retention, raw pending replay, queued inbound order, flush after reconnect and no duplicate delivery.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --test parity_inbound
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/src/message/`, `src/inbound_queue.rs`, `src/flush_scope.rs`, `src/signal_flush.rs`, and `src/msg_secret_buffer.rs`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --test parity_inbound
cargo check -p wa-rs
```

- [ ] **Step 5: Commit** with `feat(message): port inbound pipeline durability and queue`.

### Task 2.3: Port the send pipeline, cache and error surface

**Files:**
- Modify: `src/send.rs`
- Create: `src/send/{mod.rs,actions.rs,requests.rs,group_repair.rs,tctoken_lifecycle.rs}`, `src/send/tests/`
- Create: `src/error.rs`
- Create: `src/cache.rs`, `src/cache_config.rs`, `src/cache_store.rs`, `src/portable_cache.rs`, `src/portable_cache/`
- Create: `src/pending_device_sync.rs`
- Create: `src/sender_key_device_cache.rs`
- Create: `src/resend_rate_limiter.rs`
- Test: `tests/parity_send.rs`, `tests/parity_cache.rs`

**Interfaces:**
- Produces: `SendError`, `SendOptions`, `EditOptions`, `SendResult`, `PinDuration`, `RevokeType`, `SendRequest`, `EditRequest`.
- Produces: `ServerRejection`, `Sources`, `ErrorChainExt`.
- Produces: `CacheConfig`, `CacheStores`, `CacheStore`, `Freshness`, `OriginalMessageResolver`.
- Produces: `SecretFlushReport`.

- [ ] **Step 1: Write failing tests** for direct/group send, edit, revoke, pin, keep, group repair, cache freshness and error chaining.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --test parity_send --test parity_cache
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/src/send/`, `src/error.rs`, `src/cache.rs`, `src/cache_config.rs`, `src/cache_store.rs`, `src/portable_cache.rs`, `src/pending_device_sync.rs`, `src/sender_key_device_cache.rs`, `src/resend_rate_limiter.rs`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --test parity_send --test parity_cache
cargo check -p wa-rs-bridge
```

- [ ] **Step 5: Commit** with `feat(send): port send pipeline cache and error surface`.

### Task 2.4: Port media streaming, retry/reupload and cache budgets

**Files:**
- Create: `src/media.rs`
- Modify: `src/download.rs`
- Create: `src/download/{budget.rs,memory.rs}`
- Modify: `src/upload.rs`
- Modify: `wacore/src/download.rs`
- Modify: `wacore/src/upload.rs`
- Create: `src/features/media_reupload.rs`
- Test: `tests/parity_media.rs`

**Interfaces:**
- Produces: `ImageOptions`, `VideoOptions`, `DocumentOptions`, `AudioOptions`.
- Produces: `DownloadParams`, `MediaDownloadError`, `ClientDownloadError`, `MediaDownloader`, `DownloadLimits`, `DownloadPolicyError`, `MediaByteDomain`.
- Produces: `UploadResponse`, `UploadOptions`, `MediaReupload`, `MediaReuploadRequest`, `MediaReuploadPhase`, `MediaReuploadError`.

- [ ] **Step 1: Write failing tests** for streaming upload/download, retry/reupload, cache budget, WebP and sticker pack fetch.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --test parity_media
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/src/media.rs`, `src/download.rs`, `src/download/`, `src/upload.rs`, and `src/features/media_reupload.rs`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --test parity_media
cargo check -p wa-rs
```

- [ ] **Step 5: Commit** with `feat(media): port streaming retry reupload and cache budgets`.

### Task 2.5: Port reliability, PDO, history, request/usync, keepalive, LID and store

**Files:**
- Modify: `src/receipt.rs`, `src/retry.rs`
- Modify: `src/pdo.rs`
- Create: `src/pdo/`
- Modify: `src/history_sync.rs`
- Create: `src/history_sync/`
- Modify: `src/request.rs`
- Create: `src/request/`
- Modify: `src/usync.rs`
- Create: `src/usync/`
- Modify: `src/keepalive.rs`
- Create: `src/keepalive/`
- Modify: `src/lid_pn_cache.rs`
- Create: `src/lid_pn_cache/`
- Modify: `src/socket/{mod.rs,error.rs,noise_socket.rs}`
- Modify: `src/store/`
- Test: `tests/parity_reliability.rs`, `tests/parity_pdo.rs`, `tests/parity_history.rs`

**Interfaces:**
- Produces: `RetryRequestOptions`, `RetryRequestOutcome`, `RetryRequestError`, `StanzaRejection`, `StanzaResponseError`, `MessageRetransmission`.
- Produces: `mark_message_played`, `mark_as_played`.
- Produces: `HistorySync`, `HistorySyncNotification`, `UsyncQuery`, `UsyncProtocol`.
- Produces: `LidPnEntry`, `LearningSource`, `PersistenceManager`, `DeviceCommand`, `SqliteStore`.

- [ ] **Step 1: Write failing tests** for retry conformance, played receipts, PDO history, history sync admission, usync parsing, keepalive and LID/PN learning.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --test parity_reliability --test parity_pdo --test parity_history
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/src/receipt.rs`, `retry.rs`, `pdo/`, `history_sync/`, `request/`, `usync/`, `keepalive/`, `lid_pn_cache/`, `socket/`, and `store/`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --test parity_reliability --test parity_pdo --test parity_history
cargo check -p wa-rs-bridge
```

- [ ] **Step 5: Commit** with `feat(reliability): port retry receipts PDO history request usync and store`.

### Task 2.6: Port core protocol, handshake, transport and shared utility modules

**Files:**
- Modify: `wacore/src/net.rs`, `wacore/src/version.rs`, `wacore/src/xml.rs`
- Create: `wacore/src/serde_helpers.rs`
- Modify: `wacore/src/handshake/`
- Replace: `wacore/src/protocol.rs` with `wacore/src/protocol/`
- Modify: `wacore/src/ib.rs`, `wacore/src/pair.rs`, `wacore/src/pair_code.rs`, `wacore/src/prekeys.rs`, `wacore/src/request.rs`, `wacore/src/send.rs`, `wacore/src/usync.rs`, `wacore/src/history_sync.rs`, `wacore/src/proto_helpers.rs`, `wacore/src/stanza/`, `wacore/src/store/`
- Modify: `wacore/src/types/{jid.rs,lid_pn.rs,message.rs,mod.rs,presence.rs,spam_report.rs,user.rs}`
- Create: `wacore/src/types/{message_ref.rs,message_secret.rs,wire_enums.rs}`, `wacore/src/types/message_ref/`
- Modify: `src/handshake.rs`, `src/http.rs`, `src/jid_utils.rs`, `src/mediaconn.rs`, `src/spam_report.rs`, `src/sync_task.rs`, `src/unified_session.rs`, `src/version.rs`
- Create: `src/retry_conformance_tests.rs`
- Test: `tests/parity_protocol.rs`, `tests/parity_transport.rs`

**Interfaces:**
- Produces: `NoiseHandshake`, `NoiseError`, `HandshakeError`, `Transport`, `TransportFactory`, `HttpClient`, `MediaConn`, `JidUtils`, `SpamFlow`, `SpamReportRequest`, `SpamReportResult`, `SyncTask`, `UnifiedSession`.
- Preserves: existing `wa_rs::handshake`, `wa_rs::http`, `wa_rs::mediaconn`, `wa_rs::spam_report` APIs.

- [ ] **Step 1: Write failing tests** for handshake state transitions, transport framing, media connection refresh, spam reporting and unified session behavior.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --test parity_protocol --test parity_transport
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/wacore/src/net.rs`, `serde_helpers.rs`, `version.rs`, `xml.rs`, `handshake/`, `protocol/`, `ib.rs`, `pair.rs`, `pair_code.rs`, `prekeys.rs`, `request.rs`, `send.rs`, `usync.rs`, `history_sync.rs`, `proto_helpers.rs`, `stanza/`, `store/`, the listed `types/` modules, and the matching root `src/` files.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --test parity_protocol --test parity_transport
cargo check -p wa-rs-core
cargo check -p wa-rs
```

- [ ] **Step 5: Commit** with `feat(core): port protocol handshake transport and shared utilities`.

### Phase 2 Gate

- [ ] `cargo test -p wa-rs-core --test parity_messaging --test parity_secrets`
- [ ] `cargo test -p wa-rs --test parity_inbound --test parity_send --test parity_cache --test parity_media`
- [ ] `cargo test -p wa-rs --test parity_reliability --test parity_pdo --test parity_history`
- [ ] `cargo test -p wa-rs --test parity_protocol --test parity_transport`
- [ ] `cargo check -p wa-rs-core`
- [ ] `cargo check -p wa-rs --features parity-messaging`
- [ ] `cargo check -p wa-rs-bridge`
- [ ] `cargo build -p whatslite-desktop`
- [ ] Bridge send/receive/media smoke test

---

## Phase 3 — Social features

**Phase goal:** all 34 upstream `src/features` modules, their IQ domains, notification handlers and event surface.

### Task 3.1: Port social foundations and upgrade existing feature modules

**Files:**
- Modify: `src/features/mod.rs`, `src/lib.rs`, `wacore/src/iq/mex.rs`
- Create: `src/features/creation.rs`
- Create: `src/features/group_history.rs`
- Modify: `src/features/mex.rs`
- Modify: `src/features/tctoken.rs`
- Create: `src/features/stanza.rs`
- Create: `src/features/signal.rs`
- Create: `src/features/rotate_key.rs`
- Modify: `src/features/blocking.rs`
- Modify: `src/features/chatstate.rs`
- Modify: `src/features/contacts.rs`
- Modify: `src/features/presence.rs`
- Test: `tests/parity_social_foundations.rs`, `wacore/tests/parity_iq_social.rs`

**Interfaces:**
- Produces: `creation_types!` and the `CreatedPoll`, `CreatedEvent`, `PollRef`, `EventRef` types.
- Produces: `GroupHistorySkipReason`, `GroupHistoryRetryToken`, `GroupHistoryShareOutcome`.
- Produces: `Mex`, `MexError`, `MexOperation`, `MexRequest`, `MexDoc`, `CappingStatus`.
- Produces: `TcToken`, `TcTokenError`, `TcTokenEntry`, `ReceivedTcToken`.
- Produces: `StanzaRejection`, `StanzaResponseError`, `RetryRequestOptions`, `RetryRequestOutcome`, `MessageRetransmission`.
- Produces: `Signal`, `SignalError`, `SignalSessionInfo`, `SignalSessionMigration`.
- Preserves: existing `Blocking`, `Chatstate`, `Contacts`, `Presence` method names.

- [ ] **Step 1: Write failing tests** for creation helpers, group-history tokens, MEX requests, TCToken lifecycle, stanza rejection mapping, signal sessions and rotate-key.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --test parity_social_foundations
cargo test -p wa-rs-core --test parity_iq_social
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/src/features/{creation,group_history,mex,tctoken,stanza,signal,rotate_key,blocking,chatstate,contacts,presence}.rs` and the matching `wacore/src/iq/` domains.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --test parity_social_foundations
cargo test -p wa-rs-core --test parity_iq_social
cargo check -p wa-rs
```

- [ ] **Step 5: Commit** with `feat(social): port social foundations and upgrade existing features`.

### Task 3.2: Port advanced groups, chat actions, labels, favorites and quick replies

**Files:**
- Modify: `src/features/mod.rs`, `src/features/groups.rs`
- Create: `src/features/chat_actions.rs`
- Create: `src/features/labels.rs`
- Create: `src/features/favorites.rs`
- Create: `src/features/quick_replies.rs`
- Modify: `wacore/src/iq/groups.rs`
- Create: `wacore/src/iq/join_shapes.rs`
- Test: `tests/parity_groups.rs`, `tests/parity_chat_actions.rs`

**Interfaces:**
- Produces: `Groups`, `GroupError`, `GroupLookupResult`, `GroupOverview`, `GroupOverviewResult`, `GroupMetadata`, `GroupParticipant`, `MembershipRequest`, `GroupJoinError`, `GroupHistoryAddResult`.
- Produces: `ChatActions`, `ClearChatOptions`, `StarredMessages`, `MediaDeletion`, `SyncActionMessageRange`.
- Produces: `Labels`, `Favorites`, `QuickReplies`.

- [ ] **Step 1: Write failing tests** for join by invite, membership approve/reject, locked/announce/ephemeral, history share, labels, favorites and quick replies.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --test parity_groups --test parity_chat_actions
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/src/features/groups.rs`, `chat_actions.rs`, `labels.rs`, `favorites.rs`, `quick_replies.rs` and the matching IQ domains.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --test parity_groups --test parity_chat_actions
cargo check -p wa-rs
```

- [ ] **Step 5: Commit** with `feat(groups): port advanced groups chat actions labels favorites and quick replies`.

### Task 3.3: Port communities and newsletters

**Files:**
- Modify: `src/features/mod.rs`, `wacore/src/types/newsletter.rs`
- Create: `src/features/community.rs`
- Create: `src/features/newsletter.rs`
- Create: `wacore/src/iq/community_create.rs`
- Create: `wacore/src/iq/newsletter.rs`
- Test: `tests/parity_community.rs`, `tests/parity_newsletter.rs`

**Interfaces:**
- Produces: `Community<'a>`, `CommunityError`, `CreateCommunityOptions`, `CreateCommunityResult`, `CreateSubgroupOptions`, `LinkSubgroupOptions`, `CommunitySubgroup`, `SubgroupFailure`, `SubgroupVisibility`.
- Produces: `Newsletter<'a>`, `NewsletterError`, `NewsletterMetadata`, `NewsletterMessage`, `NewsletterAdminInfo`, `NewsletterRole`, `NewsletterState`.

- [ ] **Step 1: Write failing tests** for community create/deactivate, subgroup create/link/unlink, newsletter list/get/create/join/leave/update/admin/messages.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --test parity_community --test parity_newsletter
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/src/features/community.rs`, `newsletter.rs` and the matching IQ domains.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --test parity_community --test parity_newsletter
cargo check -p wa-rs
```

- [ ] **Step 5: Commit** with `feat(social): port communities and newsletters`.

### Task 3.4: Port status, polls and events

**Files:**
- Modify: `src/features/mod.rs`
- Create: `src/features/status.rs`
- Create: `src/features/polls.rs`
- Create: `src/features/events.rs`
- Create: `wacore/src/poll.rs`
- Test: `tests/parity_status_polls_events.rs`

**Interfaces:**
- Produces: `Status`, `StatusSendOptions`, `StatusPrivacySetting`.
- Produces: `Polls`, `PollError`, `CreatedPoll`, `PollRef`, `PollOptionResult`, `PollVoteCiphertext`.
- Produces: `Events`, `EventCreationParams`, `CreatedEvent`, `EventRef`, `EventResponseType`.

- [ ] **Step 1: Write failing tests** for status send/revoke/privacy, poll create/vote/decrypt/aggregate, and event create/respond/cancel.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --test parity_status_polls_events
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/src/features/status.rs`, `polls.rs`, `events.rs`, `creation.rs` and `wacore/src/poll.rs`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --test parity_status_polls_events
cargo check -p wa-rs
```

- [ ] **Step 5: Commit** with `feat(social): port status polls and events`.

### Task 3.5: Port stickers, business, bots, comments, pictures, profile and app state

**Files:**
- Modify: `src/features/mod.rs`
- Create: `src/features/stickers.rs`
- Create: `src/features/business.rs`
- Create: `src/features/bots.rs`
- Create: `src/features/comments.rs`
- Create: `src/features/pictures.rs`
- Create: `src/features/profile.rs`
- Create: `src/features/app_state_settings.rs`
- Create: `src/features/app_state_resync.rs`
- Modify: `src/appstate_sync.rs`
- Create: `src/appstate_known_verbs.rs`
- Create: `wacore/src/appstate_sync.rs`, `wacore/src/appstate_sync/`
- Create: `wacore/src/iq/{bot,business,profile}.rs`
- Test: `tests/parity_social_misc.rs`, `tests/parity_appstate.rs`

**Interfaces:**
- Produces: `Stickers`, `Business`, `Catalog`, `Collection`, `Order`, `Bots`, `Comments`, `Pictures`, `Profile`.
- Produces: `AppStateSettings`, `AppStateResyncMode`, `AppStateResyncReport`.
- Preserves: existing contact/profile picture calls.

- [ ] **Step 1: Write failing tests** for catalog/order, bot list, comments, profile picture, text status, app-state settings/resync and app-state sync.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --test parity_social_misc --test parity_appstate
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/src/features/{stickers,business,bots,comments,pictures,profile,app_state_settings,app_state_resync}.rs`, `src/appstate_sync.rs`, `wacore/src/appstate_sync.rs`, `wacore/src/appstate_sync/`, and the matching IQ domains.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --test parity_social_misc --test parity_appstate
cargo check -p wa-rs
```

- [ ] **Step 5: Commit** with `feat(social): port stickers business bots comments pictures profile and app state`.

### Task 3.6: Port IQ domains, notification handlers and handler coverage

**Files:**
- Modify: `wacore/src/iq/mod.rs`
- Modify: `wacore/src/iq/{blocklist,chatstate,contacts,dirty,keepalive,mediaconn,node,passive,prekeys,privacy,props,spam_report,spec,tctoken}.rs`
- Create: `wacore/src/iq/{abprops,bot,business,contextual,devices,mex_operations,mex_operations_unlisted,pilots,profile,targets}.rs`
- Modify: `wacore/src/iq/usync.rs`
- Create: `wacore/src/iq/usync/`
- Modify: `src/handlers/{basic.rs,chatstate.rs,ib.rs,iq.rs,message.rs,receipt.rs,router.rs,traits.rs}`
- Create: `src/handlers/presence.rs`
- Modify: `src/handlers/mod.rs`
- Modify: `src/handlers/notification/{mod.rs,companion_reg.rs}`
- Create: `src/handlers/notification/{device.rs,groups.rs,privacy_business.rs,profile.rs}`
- Test: extend `wacore/tests/parity_iq_social.rs`; create `tests/parity_handlers.rs`

**Interfaces:**
- Produces: every upstream IQ domain and notification type.
- Produces: 80-variant event surface exercised through handlers.
- Preserves: existing handler entry points used by `bridge`.

- [ ] **Step 1: Write failing conformance tests** for each IQ domain and notification type.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs-core --test parity_iq_social
cargo test -p wa-rs --test parity_handlers
```

- [ ] **Step 3: Port the remaining IQ and notification modules.**
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs-core --test parity_iq_social
cargo test -p wa-rs --test parity_handlers
cargo check -p wa-rs-bridge
```

- [ ] **Step 5: Commit** with `feat(iq): complete social IQ notification and handler coverage`.

### Phase 3 Gate

- [ ] `cargo test -p wa-rs --test parity_social_foundations --test parity_groups --test parity_chat_actions`
- [ ] `cargo test -p wa-rs --test parity_community --test parity_newsletter --test parity_status_polls_events`
- [ ] `cargo test -p wa-rs --test parity_social_misc --test parity_appstate --test parity_handlers`
- [ ] `cargo test -p wa-rs-core --test parity_iq_social`
- [ ] `cargo check -p wa-rs --features parity-social`
- [ ] `cargo check -p wa-rs-bridge`
- [ ] `cargo build -p whatslite-desktop`
- [ ] Event surface test: 80 `EventKind` and 80 `Event` variants

---

## Phase 4 — Calls and VoIP

**Phase goal:** upstream call control and media stack, including the runtime, engine, relay and codec features.

### Task 4.1: Port the VoIP control boundary

**Files:**
- Create: `wacore/src/voip_control/{mod.rs,app_data.rs,audio_format.rs,control.rs,engine_bridge.rs,events.rs,fake_backend.rs,group.rs,kdf.rs,media_stats.rs,ports.rs,registry.rs,relay_parse.rs,resident_session.rs,signaling.rs,signaling_plan.rs,spec_build.rs,ssrc.rs,transport.rs}`
- Create: `src/voip_control/{mod.rs,wacore_backend.rs}`
- Modify: `Cargo.toml` (`voip-control`)
- Modify: `wacore/Cargo.toml` (`voip-control`)
- Test: `wacore/tests/parity_voip_control.rs`

**Interfaces:**
- Produces: `GroupControl`, `NormalizedGroupControl`, `GroupRawEpoch`, `VideoControl`, `VideoControlSender`, `VideoControlReceiver`, `PeerAnswer`.
- Produces: `CallEvent`, `MediaGroupEpoch`, `MediaSessionKey`, `MediaAudioSpec`, `MediaKeyframeUrgency`, `VideoFecConfig`, `MediaVideoUpgradeToken`, `MediaGroupTransition`, `MediaCodecDecisionSource`, `MediaSilenceReason`, `MediaEncodedFrame`, `MediaRtcpReportBlock`, `MediaRtcpFeedback`, `MediaSessionSpec`, `MediaDirectPeer`, `MediaGroupSpec`, `MediaCommand`, `MediaGroupControlKind`, `MediaCloseReason`, `MediaSetupError`.
- Produces: `AudioSource`, `AudioSink`, `EncodedAudioSource`, `EncodedAudioSink`, `VideoFrame`, `TimedVideoFrame`, `VideoInput`.
- Gate: `voip-control`.

- [ ] **Step 1: Write failing tests** for call registry, control-plane boundary, group control normalization, media setup errors and relay parsing.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs-core --features voip-control --test parity_voip_control
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/wacore/src/voip_control/` and `src/voip_control/`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs-core --features voip-control --test parity_voip_control
cargo check -p wa-rs-core --features voip-control
cargo check -p wa-rs --features voip-control
```

- [ ] **Step 5: Commit** with `feat(voip): port control boundary`.

### Task 4.2: Port the VoIP engine, media crypto and MLOW

**Files:**
- Create: `wacore/src/voip/{mod.rs,audio.rs,codec_probe.rs,demux.rs,driver.rs,e2e_srtp.rs,engine.rs,fec.rs,group_audio.rs,group_media.rs,h264.rs,hbh_srtp.rs,opus_packet.rs,rtcp.rs,rtp.rs,session.rs,sframe.rs,stun.rs,tap.rs,warp.rs}`
- Create: `wacore/src/voip/mlow/`, `wacore/src/voip/sframe/`
- Modify: `Cargo.toml` (`voip`, `voip-mlow`, `voip-libopus`, `voip-encoded`)
- Modify: `wacore/Cargo.toml` (`voip`, `voip-mlow`)
- Test: `wacore/tests/parity_voip.rs`, `wacore/tests/parity_voip_mlow.rs`

**Interfaces:**
- Produces: `AudioConfig`, `OpusMlowPacketError`, `depacketize_opus_from_mlow`, `packetize_opus_for_mlow`.
- Produces: SRTP, SFrame, WAHKDF, RTP/RTCP, H.264, FEC, STUN, relay session traits.
- Gate: `voip`, `voip-mlow`, `voip-libopus`, `voip-encoded`.

- [ ] **Step 1: Write failing tests** for SRTP/SFrame vectors, MLOW encode/decode, DTX, H.264 frame boundaries and relay session behavior.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs-core --features voip --test parity_voip
cargo test -p wa-rs-core --features voip-mlow --test parity_voip_mlow
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/wacore/src/voip/`, excluding `testdata/` unless a runtime test needs a fixture.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs-core --features voip --test parity_voip
cargo test -p wa-rs-core --features voip-mlow --test parity_voip_mlow
cargo check -p wa-rs-core --features voip-mlow
```

- [ ] **Step 5: Commit** with `feat(voip): port engine media crypto and MLOW`.

### Task 4.3: Port the VoIP runtime, facade and native relay

**Files:**
- Create: `src/voip/{mod.rs,audio.rs,direct.rs,direct_client.rs,direct_control.rs,direct_ingress.rs,driver.rs,events.rs,registry.rs,session.rs,state.rs,video.rs}`
- Create: `src/voip/facade/`, `src/voip/facade.rs`, `src/voip/transport/`
- Modify: `Cargo.toml` (`voip-runtime`, `voip-engine-wacore`, `voip-relay-native`)
- Test: `tests/parity_voip_facade.rs`

**Interfaces:**
- Produces: `CallEvents`, `AcceptCall`, `CallHandle`, `CallLinkCall`, `CallTermination`, `GroupBoundCall`, `OutgoingCall`, `OutgoingGroupCall`, `VIDEO_UPGRADE_TIMEOUT`.
- Produces: `TimedVideoFrame`, `VideoFrame`, `VideoSink`, `VideoSource`, `AudioSink`, `AudioSource`, `EncodedAudioSink`, `EncodedAudioSource`.
- Produces: `CallError`, `Voip`.
- Gate: `voip-runtime`, `voip-engine-wacore`, `voip-relay-native`.

- [ ] **Step 1: Write failing tests** for outgoing/incoming calls, call links, group calls, screen sharing, video upgrade and call termination.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --features voip-runtime --test parity_voip_facade
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/src/voip/`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --features voip-runtime --test parity_voip_facade
cargo check -p wa-rs --features voip
cargo check -p wa-rs --features voip-encoded
cargo check -p wa-rs --features voip-libopus
```

- [ ] **Step 5: Commit** with `feat(voip): port runtime facade and native relay`.

### Task 4.4: Port call handlers, call log and call types

**Files:**
- Create: `src/handlers/call.rs`, `src/handlers/call/`
- Create: `src/features/call_log.rs`
- Modify: `wacore/src/types/call.rs`
- Create: `wacore/src/types/group_call.rs`
- Modify: `wacore/src/types/mod.rs`
- Test: `tests/parity_calls.rs`

**Interfaces:**
- Produces: incoming/missed/ended call events, call log, call links and screen-sharing events.
- Produces: `CallLog`, `GroupCallUpdate`, `CallEvent`.

- [ ] **Step 1: Write failing tests** for incoming call, group call, call ended elsewhere, call log and call link.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --features voip --test parity_calls
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/src/handlers/call.rs`, `src/handlers/call/`, `src/features/call_log.rs`, and `wacore/src/types/call.rs`, `group_call.rs`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --features voip --test parity_calls
cargo check -p wa-rs --features voip
cargo check -p wa-rs-bridge
```

- [ ] **Step 5: Commit** with `feat(calls): port handlers call log and call types`.

### Phase 4 Gate

- [ ] `cargo test -p wa-rs-core --features voip-control --test parity_voip_control`
- [ ] `cargo test -p wa-rs-core --features voip --test parity_voip`
- [ ] `cargo test -p wa-rs-core --features voip-mlow --test parity_voip_mlow`
- [ ] `cargo test -p wa-rs --features voip-runtime --test parity_voip_facade`
- [ ] `cargo test -p wa-rs --features voip --test parity_calls`
- [ ] `cargo check -p wa-rs --features parity-voip`
- [ ] `cargo check -p wa-rs --features voip-encoded`
- [ ] `cargo check -p wa-rs --features voip-libopus`
- [ ] `cargo check -p wa-rs-bridge`
- [ ] `cargo build -p whatslite-desktop`

---

## Phase 5 — Platform and observability

**Phase goal:** plugins, metrics/tracing/diagnostics, SQLite bundled/WASM/mobile, TCP transport, feature matrix and final parity audit.

### Task 5.1: Port the plugin host and client lifecycle

**Files:**
- Create: `src/plugins/{mod.rs,events.rs,iq_reply.rs}`
- Create: `src/client/extension_lifecycle.rs`
- Modify: `Cargo.toml` (`plugins`, `client-lifecycle`)
- Modify: `src/lib.rs`
- Test: `tests/parity_plugins.rs`

**Interfaces:**
- Produces: `ClientPlugin`, `UntypedClientPlugin`, `PluginCapability`, `PluginContext`, `PluginEvents`, `PluginEventRouter`, `PluginIqInterceptor`, `PluginIqReply`, `PluginManifest`, `PluginState`, `PluginHealth`, `PluginHostConfig`, `PluginHostStats`.
- Produces: `ClientLifecycle`, `ConnectionScope`, `ConnectionScopeState`.
- Gate: `plugins`, `client-lifecycle`.

- [ ] **Step 1: Write failing tests** for plugin registration, capability checks, lifecycle, event delivery and IQ reply.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --features plugins --test parity_plugins
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/src/plugins/` and `src/client/extension_lifecycle.rs`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --features plugins --test parity_plugins
cargo check -p wa-rs --features plugins
cargo check -p wa-rs --features client-lifecycle
```

- [ ] **Step 5: Commit** with `feat(plugins): port plugin host and client lifecycle`.

### Task 5.2: Port metrics, tracing and diagnostics

**Files:**
- Modify: `wacore/src/telemetry.rs`, `wacore/src/stats.rs`
- Create: `src/logging.rs`
- Create: `src/client/device_memo_stats.rs`
- Modify: `Cargo.toml` (`tracing`, `metrics`, `tracing-pii`)
- Modify: `wacore/Cargo.toml` (`tracing`, `metrics`, `tracing-pii`)
- Test: `tests/parity_observability.rs`

**Interfaces:**
- Produces: metric emission, tracing spans, `Timer`, `SessionStats`, `StatsSnapshot`, `AllocMeter`, `MemoryReport`, `ResourceReport`, `SubsystemCollection`, `SubsystemMemory`.
- Gate: `tracing`, `metrics`, `tracing-pii`.

- [ ] **Step 1: Write failing tests** for metric emission, tracing span creation and memory reports.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --features tracing,metrics --test parity_observability
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/wacore/src/telemetry.rs`, `stats.rs`, `src/logging.rs`, `src/client/device_memo_stats.rs`, and the observability parts of `src/client/accessors.rs`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --features tracing,metrics --test parity_observability
cargo check -p wa-rs --features tracing
cargo check -p wa-rs --features metrics
cargo check -p wa-rs --features tracing-pii
```

- [ ] **Step 5: Commit** with `feat(observability): port metrics tracing and diagnostics`.

### Task 5.3: Port storage profiles, TCP transport and runtime implementation

**Files:**
- Modify: `storages/sqlite-storage/Cargo.toml`, `storages/sqlite-storage/src/`
- Modify: `transports/tokio-transport/Cargo.toml`, `transports/tokio-transport/src/`
- Create: `src/runtime_impl.rs`
- Create: `src/shutdown.rs`
- Modify: `Cargo.toml` (`sqlite-storage-bundled`, `mobile-transport-native`, `tokio-runtime`)
- Test: `storages/sqlite-storage/tests/parity_storage.rs`, `transports/tokio-transport/tests/parity_tcp.rs`

**Interfaces:**
- Produces: storage features `bundled-sqlite`, `js`, `mobile`, `wasm-test`.
- Produces: transport feature `tcp`.
- Produces: `TokioRuntime`, `shutdown_signal`.
- Changes: storage default becomes `bundled-sqlite`, matching upstream. The system SQLite symlink remains a fallback for `--no-default-features`.

- [ ] **Step 1: Write failing tests** for bundled storage, WASM storage contract and TCP transport.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs-sqlite-storage --test parity_storage
cargo test -p wa-rs-tokio-transport --features tcp --test parity_tcp
```

- [ ] **Step 3: Port the feature changes** from upstream storage and transport manifests.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs-sqlite-storage --test parity_storage
cargo test -p wa-rs-tokio-transport --features tcp --test parity_tcp
cargo check -p wa-rs-sqlite-storage --features bundled-sqlite
cargo check -p wa-rs-tokio-transport --features tcp
cargo check -p wa-rs --features sqlite-storage-bundled
cargo check -p wa-rs --features mobile-transport-native
cargo check -p wa-rs --features tokio-runtime
```

- [ ] **Step 5: Commit** with `feat(platform): port storage profiles TCP transport and runtime implementation`.

### Task 5.4: Add the feature matrix and WASM consumer checks

**Files:**
- Create: `tests/parity_matrix.rs`
- Modify: `Cargo.toml`
- Test: `tests/parity_matrix.rs`

**Interfaces:**
- Produces: documented feature matrix and consumer checks.

- [ ] **Step 1: Write the matrix test** covering the parity aliases and the canonical features.
- [ ] **Step 2: Run the matrix and verify expected results.**
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
cargo check -p wa-rs --features plugins
cargo check -p wa-rs --features passkey
cargo check -p wa-rs --features mobile-core
cargo check -p wa-rs --features voip
cargo check -p wa-rs --features tracing,metrics
```

- [ ] **Step 4a: Run the WASM check** (requires `rustup target add wasm32-unknown-unknown`):

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

- [ ] **Step 1: Write the audit checklist** comparing upstream `src/features`, `wacore/src`, `wacore/src/iq`, `wacore/src/voip`, and `wacore/src/voip_control` to the ported tree.
- [ ] **Step 2: Run the audit and list every missing module or method.**
- [ ] **Step 3: Close remaining gaps or record them as explicit non-goals.**
- [ ] **Step 4: Run the final gate.**
- [ ] **Step 5: Commit** with `docs: add parity audit`.

### Phase 5 Gate

- [ ] `cargo build -p whatslite-desktop`
- [ ] `cargo test --workspace`
- [ ] `cargo check --workspace`
- [ ] `cargo test -p wa-rs --features plugins --test parity_plugins`
- [ ] `cargo test -p wa-rs --features tracing,metrics --test parity_observability`
- [ ] `cargo test -p wa-rs-sqlite-storage --test parity_storage`
- [ ] `cargo test -p wa-rs-tokio-transport --features tcp --test parity_tcp`
- [ ] `cargo check -p wa-rs --no-default-features --features parity-core`
- [ ] `cargo check -p wa-rs --features parity-voip`
- [ ] `cargo check -p wa-rs --features parity-mobile`
- [ ] `cargo check -p wa-rs --features parity-plugins`
- [ ] `cargo check -p wa-rs --features sqlite-storage-bundled`
- [ ] `cargo check -p wa-rs --features mobile-transport-native`
- [ ] `cargo check -p wa-rs-sqlite-storage --target wasm32-unknown-unknown --features js`
- [ ] `cargo check -p wa-rs-bridge`
- [ ] Bridge pair/connect/send/receive/media/call smoke tests
- [ ] Parity report complete

---

## Final Definition of Done

- [ ] All upstream runtime feature modules exist under the current tree.
- [ ] `cargo build`, `cargo test --workspace`, `cargo check --workspace` pass.
- [ ] `bridge/` and `desktop/` compile on default features.
- [ ] Feature matrix passes for every parity alias and canonical feature.
- [ ] Event surface matches upstream: 80 `EventKind` and 80 `Event` variants.
- [ ] No JS/Python/Kotlin/Java runtime code.
- [ ] Parity report lists any remaining explicit non-goals.

## Execution Handoff

Plan complete and saved to `reference/design/plans/2026-11-08-wa-rs-parity-plan.md` (repo) and `/home/muhammadmishbakhuzzuhail/.opencode/plan/2026-11-08-wa-rs-parity-plan.md` (working copy).

Recommended execution: **subagent-driven**, because the phases have many independent module tasks but share interfaces (`Runtime`, `Clock`, buffa types, `Event`, `session_engine`). A fresh implementer per task and a fresh reviewer per phase gate reduces cross-module regressions.

Before implementation, review this plan and confirm it captures what you want.
