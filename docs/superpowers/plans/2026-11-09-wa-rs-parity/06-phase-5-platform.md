# Phase 5 — Platform and observability

> Part of the [wa-rs Parity Plan](README.md). Read the overview first for
> global constraints, the feature contract, crate aliases, the gate protocol,
> and the test parity contract.
>
> Test parity obligations: [Test Parity Matrix](07-test-parity-matrix.md).

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

- [x] **Step 1: Write failing tests** for plugin registration, capability checks, lifecycle, event delivery and IQ reply.
- [x] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --features plugins --test parity_plugins
```

- [x] **Step 3: Port the modules** from `reference/whatsapp-rust/src/plugins/` and `src/client/extension_lifecycle.rs`.
- [x] **Step 4: Verify**

```bash
cargo test -p wa-rs --features plugins --test parity_plugins
cargo check -p wa-rs --features plugins
cargo check -p wa-rs --features client-lifecycle
```

- [x] **Step 5: Commit** with `feat(plugins): port plugin host and client lifecycle`.

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

- [x] **Step 1: Write failing tests** for metric emission, tracing span creation and memory reports.
- [x] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --features tracing,metrics --test parity_observability
```

- [x] **Step 3: Port the modules** from `reference/whatsapp-rust/wacore/src/telemetry.rs`, `stats.rs`, `src/logging.rs`, `src/client/device_memo_stats.rs`, and the observability parts of `src/client/accessors.rs`.
- [x] **Step 4: Verify**

```bash
cargo test -p wa-rs --features tracing,metrics --test parity_observability
cargo check -p wa-rs --features tracing
cargo check -p wa-rs --features metrics
cargo check -p wa-rs --features tracing-pii
```

- [x] **Step 5: Commit** with `feat(observability): port metrics tracing and diagnostics`.

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

- [x] **Step 1: Write failing tests** for bundled storage, WASM storage contract and TCP transport.
- [x] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs-sqlite-storage --test parity_storage
cargo test -p wa-rs-tokio-transport --features tcp --test parity_tcp
```

- [x] **Step 3: Port the feature changes** from upstream storage and transport manifests.
- [x] **Step 4: Verify**

```bash
cargo test -p wa-rs-sqlite-storage --test parity_storage
cargo test -p wa-rs-tokio-transport --features tcp --test parity_tcp
cargo check -p wa-rs-sqlite-storage --features bundled-sqlite
cargo check -p wa-rs-tokio-transport --features tcp
cargo check -p wa-rs --features sqlite-storage-bundled
cargo check -p wa-rs --features mobile-transport-native
cargo check -p wa-rs --features tokio-runtime
```

- [x] **Step 5: Commit** with `feat(platform): port storage profiles TCP transport and runtime implementation`.

### Task 5.4: Add the feature matrix and WASM consumer checks

**Files:**
- Create: `tests/parity_matrix.rs`
- Modify: `Cargo.toml`
- Test: `tests/parity_matrix.rs`

**Interfaces:**
- Produces: documented feature matrix and consumer checks.

- [x] **Step 1: Write the matrix test** covering the parity aliases and the canonical features.
- [x] **Step 2: Run the matrix and verify expected results.**
- [x] **Step 3: Fix feature leaks and missing gates.**
- [x] **Step 4: Run all matrix commands.**

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

- [x] **Step 4b: Run the default-features test matrix** for the parity aliases that can run natively:

```bash
cargo test -p wa-rs --features parity-core
cargo test -p wa-rs --features parity-messaging
cargo test -p wa-rs --features parity-social
cargo test -p wa-rs --features parity-voip
cargo test -p wa-rs --features parity-mobile
cargo test -p wa-rs --features parity-plugins
cargo test -p wa-rs --features parity-observability
```

`parity-wasm` is check-only on native; its test run happens on the wasm target.

- [x] **Step 4a: Run the WASM check** (requires `rustup target add wasm32-unknown-unknown`):

```bash
cargo check -p wa-rs-sqlite-storage --target wasm32-unknown-unknown --features js
```

- [x] **Step 5: Commit** with `test: add parity feature matrix`.

### Task 5.5: Final parity audit

**Files:**
- Create: `reference/design/plans/parity-report.md`
- Modify: `desktop/README.md` only if build instructions change
- Test: full workspace + bridge smoke

**Interfaces:**
- Produces: module-by-module parity table and known gaps.

- [x] **Step 1: Write the audit checklist** comparing upstream `src/features`, `wacore/src`, `wacore/src/iq`, `wacore/src/voip`, and `wacore/src/voip_control` to the ported tree.
- [x] **Step 2: Run the audit and list every missing module or method.**
- [x] **Step 3: Close remaining gaps or record them as explicit non-goals.**
- [x] **Step 4: Run the final gate.**
- [x] **Step 5: Commit** with `docs: add parity audit`.

### Task 5.6: Test parity audit

**Files:**
- Create: `reference/design/plans/test-parity-report.md`
- Test: full workspace + feature matrix + bridge + desktop

**Interfaces:**
- Produces: test-by-test parity table and known non-goals.

- [x] **Step 1: Build the report from the Test Port Matrix.** Every row gets a status: `ported`, `non-goal`, or `blocked`.
- [x] **Step 2: Run every command in the matrix.**
- [x] **Step 3: Compare the final test count with the baseline.** List every non-goal and the reason.
- [x] **Step 4: Fix gaps or record them.**
- [x] **Step 5: Commit** with `docs: add test parity audit`.

### Phase 5 Gate

- [x] `cargo build -p whatslite-desktop`
- [x] `cargo test --workspace`
- [x] `cargo check --workspace`
- [x] `cargo test -p whatslite-desktop`
- [x] `cargo test -p wa-rs-bridge`
- [x] `cargo test -p wa-rs --features plugins --test parity_plugins`
- [x] `cargo test -p wa-rs --features plugins`
- [x] `cargo test -p wa-rs --features tracing,metrics --test parity_observability`
- [x] `cargo test -p wa-rs --features tracing,metrics`
- [x] `cargo test -p wa-rs --features passkey`
- [x] `cargo test -p wa-rs --features mobile-core`
- [x] `cargo test -p wa-rs-sqlite-storage --test parity_storage`
- [x] `cargo test -p wa-rs-sqlite-storage`
- [x] `cargo test -p wa-rs-tokio-transport --features tcp --test parity_tcp`
- [x] `cargo test -p wa-rs-tokio-transport --features tcp`
- [x] `cargo check -p wa-rs --no-default-features --features parity-core`
- [x] `cargo check -p wa-rs --features parity-voip`
- [x] `cargo check -p wa-rs --features parity-mobile`
- [x] `cargo check -p wa-rs --features parity-plugins`
- [x] `cargo check -p wa-rs --features sqlite-storage-bundled`
- [x] `cargo check -p wa-rs --features mobile-transport-native`
- [x] `cargo check -p wa-rs-sqlite-storage --target wasm32-unknown-unknown --no-default-features --features js` (user-local clang; default-features bundled SQLite is native-only)
- [x] `cargo check -p wa-rs-bridge`
- [ ] Bridge pair/connect/send/receive/media/call smoke tests — blocked on a real WhatsApp account/phone; HTTP router smoke is green.
- [x] Parity report complete
- [x] Test parity report complete

---

Previous: [Phase 4](05-phase-4-voip.md) | Next: [Test Parity Matrix](07-test-parity-matrix.md)
