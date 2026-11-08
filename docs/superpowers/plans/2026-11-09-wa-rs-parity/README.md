# wa-rs Parity with whatsapp-rust 0.7.0 — Implementation Plan (v3, split)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Port the full runtime feature set of `oxidezap/whatsapp-rust` 0.7.0 (`f077a5d`) into `homun-app/wa-rs` 0.2.0 without changing the current tree and without breaking `bridge/` or `desktop/`.

**Architecture:** Strangler migration per subsystem. Port upstream modules into the existing crate boundary (`wacore/`, `waproto/`, `src/`, `storages/`, `transports/`, `http_clients/`) using upstream feature names as the canonical gates. `parity-*` features are additive aliases. Core modules stay always compiled, matching upstream; only optional modules and dependencies are gated. The existing `wa-rs` API remains a compatibility facade.

**Tech Stack:** Rust 2024, Tokio, SQLite/Diesel, buffa 0.9.2, rustls, GPUI desktop consumer, `bridge/` HTTP sidecar.

**Spec:** `docs/superpowers/specs/2026-11-09-wa-rs-parity-design.md`

## Plan Files

Read the overview first, then the phase you are executing. The test matrix is
the functional-parity contract for every phase.

- [Test Parity Matrix](07-test-parity-matrix.md)
- [Phase 0 — Foundation](01-phase-0-foundation.md)
- [Phase 1 — Auth, session and recovery](02-phase-1-auth-session.md)
- [Phase 2 — Messaging, media and reliability](03-phase-2-messaging-media.md)
- [Phase 3 — Social features](04-phase-3-social.md)
- [Phase 4 — Calls and VoIP](05-phase-4-voip.md)
- [Phase 5 — Platform and observability](06-phase-5-platform.md)
- [Design spec](../../specs/2026-11-09-wa-rs-parity-design.md)


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

## Review Focus

These are the failure modes most likely to bite a user even when the module tests pass. Each line has a test in the owning task or in the Test Port Matrix.

1. **Existing `wa-rs` API compatibility** — `bridge` can still pair, connect, send and receive; `desktop` still builds and its 21 tests pass after the crate aliases and buffa switch.
2. **Test parity** — every upstream runtime test is ported or explicitly listed as a non-goal; no test is deleted to make the build green.
3. **Feature-gated builds** — default, each `parity-*` alias, each canonical feature, and the wasm32 target build and test independently.
4. **Session and Signal durability** — reconnect, offline resume and storage restart preserve sessions and do not duplicate deliveries.
5. **Wire and event conformance** — 80 `EventKind`/`Event` variants, IQ fixtures, media crypto and VoIP vectors match upstream.

## Prerequisites

- [ ] Install the Rust components. If `rustup` is absent, install it user-local (no sudo) first:

```bash
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs -o /tmp/opencode/rustup-init.sh
sh /tmp/opencode/rustup-init.sh -y --profile minimal --default-toolchain 1.99.1
source "$HOME/.cargo/env"
rustup component add rustfmt clippy
cargo fmt --version
cargo clippy --version
```

- [ ] Provide the SQLite linker and runtime path. `LIBRARY_PATH` is required for `-lsqlite3`; `LD_LIBRARY_PATH` alone fails at link time:

```bash
mkdir -p /tmp/opencode/sqlite
ln -sf /lib64/libsqlite3.so.0 /tmp/opencode/sqlite/libsqlite3.so
export LIBRARY_PATH="/tmp/opencode/sqlite:${LIBRARY_PATH:-}"
export LD_LIBRARY_PATH="/tmp/opencode/sqlite:${LD_LIBRARY_PATH:-}"
```

- [ ] Verify the baseline before touching code:

```bash
cargo test -p wa-rs-sqlite-storage
cargo test --workspace
```

Expected: the baseline is green. If it is not, fix the environment first and record the failure; do not start Phase 0 on a red baseline.

- [ ] Verified on 2026-11-09:

  - `cargo fmt --all -- --check` → exit 0
  - `cargo clippy --workspace --all-targets` → exit 0, 7 pre-existing warnings (4 `wa-rs` lib, 1 `wa-rs` bin, 2 `wa-rs-bridge`)
  - `cargo test --workspace` → exit 0, 780 passed, 0 failed, 27 ignored
  - `cargo test -p wa-rs-sqlite-storage` → exit 0, 21 passed

  Phase gates must not add new clippy warnings; the 7 warnings above are the
  baseline to compare against.

- [ ] Confirm the reference checkout is at the pinned upstream commit:

```bash
git -C reference/whatsapp-rust rev-parse --short HEAD
```

Expected: `f077a5d`. Ported files are copied from this commit, not from a moving branch. `origin/fix/adv-primary-migration` is 4 commits ahead at `7df72129` with ADV/USync fixes and an expanded `src/usync/ordinary_recovery_tests.rs`; it is not merged. Stay on `f077a5d` and port the branch's tests when it lands on main.

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
6. `cargo test -p whatslite-desktop`
7. `cargo check -p wa-rs-bridge`
8. `cargo test -p wa-rs-bridge`

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

## Test Parity Contract

The migration is a port, not a redesign. The upstream test suite is the functional specification. Every upstream runtime test must be ported or explicitly listed as a non-goal; no test may be deleted to make the build green.

### Why this policy

“Wajib port” means every upstream runtime test is accounted for: ported,
covered by an equivalent existing test, or explicitly non-goal. It does not
mean copying all ~7,400 tests blindly.

- **With the policy:** behavioral parity is evidenced; edge cases in retry,
  session, media, VoIP and ADV/USync are protected; regressions surface in CI.
- **Without it:** migration is faster, but “functionally the same” is unproven;
  `bridge`/`desktop` can pass while protocol behavior diverges; edge cases are
  discovered in production.
- **Cost:** the upstream suite is large. Use `covered-by-existing` for duplicate
  behavior and `non-goal` for tests tied to excluded features or external
  harnesses; record every exception in the test parity report.

### Port rules

1. Copy the upstream test file to the target path. Rewrite only:
   - `whatsapp_rust::` → `wa_rs::`
   - `use whatsapp_rust as ...` → `use wa_rs as ...`
   - fixture paths that move with the file
   - test harness imports (`crate::test_utils`, `tests/support`, ...)
2. Keep `wacore::`, `wacore_binary::`, `waproto::` paths unchanged; the crate aliases make them resolve.
3. Do not change assertions, expected values, or test names. If an assertion must change, record the reason in `test-parity-report.md` and get it reviewed.
4. Keep the existing `wa-rs` tests. The final test count must be at least the baseline; any test removed must be listed with a reason.
5. Port test-only helpers (`src/test_utils.rs`, `wacore/noise/src/test_util.rs`, `tests/support/`, `wacore/tests/support/`, `storages/sqlite-storage/tests/support/`) even though they are not runtime code.
6. Excluded: `bench-harness`, `test-support`, `session-simulator`, `dhat-heap`, `bench-internals`, `bench_support/`, `benches/`, `tools/`, `whatspec/`, `unwasm/`, oracle harnesses, CI, Dockerfiles. Tests that require an excluded feature are either rewritten behind `#[cfg(test)]` or recorded as non-goals with the reason.

### Per-task obligation

Each task's Step 1 ports the upstream test files assigned to its phase in the Test Port Matrix, then adds the task-specific `parity_*` test. A task is not done while its matrix rows are unported.

### Test layers

- **L1 unit** — in-crate `#[cfg(test)]` modules and `*_tests.rs`.
- **L2 integration** — `tests/`, `wacore/tests/`, `waproto/tests/`, storage/transport tests.
- **L3 compatibility** — existing `wa-rs` API, `bridge`, `desktop`, `crates/whatslite-*`.
- **L4 conformance** — wire protocol, IQ, events, media, VoIP, Signal vectors.
- **L5 feature matrix** — default, each `parity-*` alias, each canonical feature, wasm32.

### Baseline and target

Record the baseline before Phase 0:

```bash
cargo test --workspace -- --list | wc -l
cargo test -p whatslite-desktop -- --list | wc -l
```

Verified 2026-11-09: 780 passed, 0 failed, 27 ignored in the workspace; 21 desktop tests. The target is the baseline plus every ported upstream runtime test. The final count is recorded in the parity report.

### Test commands

- Default: `cargo test --workspace`
- Desktop: `cargo test -p whatslite-desktop`
- Bridge HTTP: `cargo test -p wa-rs-bridge`
- Canonical features: the commands in each phase gate.
- Feature matrix: Task 5.4.

## Module Inventory

This is the completeness checklist. A phase is not done while a runtime module in its area is missing.

| Area | Upstream count | Phase | Notes |
|---|---:|---|---|
| `wacore/src` top-level | 46 `.rs` files | 0–4 | Includes `bot_message`, `client`, `comment`, `companion_reg`, `event`, `messages`, `reporting_token`, `secret_enc_addon`, `sync_marker`, `download`, `upload` |
| `wacore/src/iq` | 33 `.rs` files + `usync/` | 3 | Includes `mex_operations`, `mex_operations_unlisted`, `pilots`, `pilot_conformance`, `targets` |
| `wacore/src/types` | 13 `.rs` files + `message_ref/` | 0–4 | Includes `call.rs`, `group_call.rs`, `message_secret.rs`, `wire_enums.rs` |
| `wacore/src/voip` | 20 files + `mlow/`, `sframe/`, `testdata/` | 4 | Port runtime files; vendor `testdata/` only as test data, never as runtime code |
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

Excluded from runtime crates: `bench_support/`, `test_support/`, `benches/`, `tools/`, `whatspec/`, `unwasm/`, CI files, Dockerfiles, and generated test-vector tooling. Test-only files (`test_utils.rs`, `reexports_test.rs`, `request_context_tests.rs`, `tests/support/`) are ported under `#[cfg(test)]` and are not runtime code.

## Final Definition of Done

- [ ] All upstream runtime feature modules exist under the current tree.
- [ ] `cargo build`, `cargo test --workspace`, `cargo check --workspace` pass.
- [ ] `bridge/` and `desktop/` compile on default features.
- [ ] Feature matrix passes for every parity alias and canonical feature.
- [ ] Event surface matches upstream: 80 `EventKind` and 80 `Event` variants.
- [ ] All upstream runtime tests are ported or listed as non-goals; no test is deleted.
- [ ] Existing workspace tests still pass (baseline: 780 passed, 27 ignored).
- [ ] Desktop tests (21) still pass.
- [ ] Bridge HTTP tests pass.
- [ ] No JS/Python/Kotlin/Java runtime code.
- [ ] Parity report and test parity report list any remaining explicit non-goals.

## Execution Handoff

Plan complete and saved under `docs/superpowers/plans/2026-11-09-wa-rs-parity/`. The design spec is at `docs/superpowers/specs/2026-11-09-wa-rs-parity-design.md`.

Recommended execution: **subagent-driven**, because the phases have many independent module tasks but share interfaces (`Runtime`, `Clock`, buffa types, `Event`, `session_engine`). A fresh implementer per task and a fresh reviewer per phase gate reduces cross-module regressions.

Before implementation, review this plan and confirm it captures what you want.
