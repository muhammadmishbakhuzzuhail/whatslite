# Phase 0 — Foundation

> Part of the [wa-rs Parity Plan](README.md). Read the overview first for
> global constraints, the feature contract, crate aliases, the gate protocol,
> and the test parity contract.
>
> Test parity obligations: [Test Parity Matrix](07-test-parity-matrix.md).

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
- Produces: `wa_rs::wacore`, `wa_rs::wacore_binary`, `wa_rs::waproto` re-exports in `src/lib.rs`.
- Preserves: package names, folder tree, and public crate names for external consumers.

- [ ] **Step 1: Rename the dependency keys**

Rename the existing `[workspace.dependencies]` keys and every member manifest entry from `wa-rs-core`, `wa-rs-binary`, `wa-rs-proto`, and the other sibling packages to the upstream aliases. Keep the package names unchanged. Do not keep both keys for the same package; Cargo rejects duplicate package dependencies. Update every `workspace = true` reference to the new key.

- [ ] **Step 2: Add the self aliases**

Add `extern crate self as ...;` to each crate root that does not already have one. `wacore/src/lib.rs` already has `extern crate self as wa_rs_core;`; the mechanical rename converts it to `wacore`, so do not add a duplicate. Add `extern crate self as whatsapp_rust;` to `src/lib.rs`. Also add `pub use wacore; pub use wacore_binary; pub use waproto;` to `src/lib.rs` so external consumers can reach the sibling crates through the main crate.

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

Also declare the `wa-rs` binary target with `required-features = ["sqlite-storage", "tokio-transport", "ureq-client", "tokio-native"]`. The CLI always needs the native stack; without this, `cargo check -p wa-rs --no-default-features` fails on `src/main.rs` instead of checking the library.

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
- Modify: `wacore/tests/`, `tests/`, and every in-crate `#[cfg(test)]` module that constructs proto types
- Test: `cargo check -p wa-rs-core`, `cargo check -p wa-rs`, `cargo check -p wa-rs-bridge`, `cargo test --workspace`

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
cargo test --workspace
```

Expected: all succeed. The existing tests are part of the migration; do not leave a test on prost types.

- [ ] **Step 5: Commit**

```bash
git add wacore src bridge tests
git commit -m "refactor(proto): migrate call sites and tests to buffa types"
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

### Task 0.7: Port the test harness, fixtures and compatibility tests

**Files:**
- Create: `tests/support/`, `wacore/tests/support/`, `wacore/tests/fixtures/`, `wacore/appstate/tests/fixtures/`, `wacore/libsignal/tests/vectors/`, `wacore/src/voip/testdata/`, `wacore/src/voip/mlow/testdata/`
- Create: `tests/parity_reexports.rs`, `tests/parity_existing_api.rs`, `tests/parity_api_consumer.rs`, `bridge/tests/parity_smoke.rs`
- Create: `bridge/src/lib.rs` (exposes `build_router`, `AppState`, pure handlers)
- Modify: `bridge/src/main.rs` (thin binary using the library)
- Modify: `bridge/Cargo.toml` (lib + bin targets; dev-dependencies: `assert_cmd = "2"`, `predicates = "3"`, `tempfile = "3"`, `tower = { version = "0.5", features = ["util"] }`)
- Modify: `Cargo.toml` (dev-dependency `tempfile = "3"` for root integration tests)
- Test: `cargo test -p wa-rs --test parity_reexports --test parity_existing_api --test parity_api_consumer`, `cargo test -p wa-rs-bridge`, `cargo test -p whatslite-desktop`

**Interfaces:**
- Produces: `tests/parity_reexports.rs` asserting `wa_rs::wacore`, `wa_rs::wacore_binary`, `wa_rs::waproto`, `wa_rs::Client`, `wa_rs::features::*` are reachable.
- Produces: `tests/parity_existing_api.rs` asserting the current `Client::new`, `run`, `connect`, `disconnect`, `send_node`, `bot::Bot`, `pair_code::PairCodeOptions`, `store::SqliteStore` signatures still compile.
- Produces: `tests/parity_api_consumer.rs` with the Phase 0 public surface (`wa_rs::wacore`, `wa_rs::wacore_binary`, `wa_rs::waproto`, `wa_rs::Client`, `EventDelivery`); each later phase extends it with its public API (`ClientBuilder`, `Groups`, `SendRequest`, `MediaDownloader`, ...).
- Produces: `wa_rs_bridge::build_router`, `AppState`, and `AppState::for_test()` (`client: None`) so `bridge/tests/parity_smoke.rs` can call `/health`, `/status`, `/pair/start` validation and `/send` validation through `tower::ServiceExt::oneshot`.
- Preserves: bridge HTTP routes and response shapes; only the module boundary moves. Manual bridge pairing smoke remains.

- [ ] **Step 1: Write the failing compatibility tests**

Create the four test files. Assert the re-exports, the existing public signatures, the Phase 0 consumer surface, and the bridge HTTP surface.

- [ ] **Step 2: Run them and verify failure**

```bash
cargo test -p wa-rs --test parity_reexports --test parity_existing_api --test parity_api_consumer
cargo test -p wa-rs-bridge
```

Expected: FAIL because the aliases and the extracted bridge router are not present.

- [ ] **Step 3: Port the harness and fixtures**

Create `tests/support/` and copy `wacore/tests/support/`, `wacore/tests/fixtures/`, `wacore/appstate/tests/fixtures/`, `wacore/libsignal/tests/vectors/`, `wacore/src/voip/testdata/`, and `wacore/src/voip/mlow/testdata/`.

Port the Phase 0 rows from the Test Port Matrix: `waproto/tests/*`, `wacore/tests/wire_enum_serde_test.rs`, `wacore/derive/src/tag_contract_tests.rs`, `tests/{event_delivery_public,event_kind_public_api,sans_io_boundary,sdk_clock_audit,async_trait_boxed_future_compat}.rs`, `wacore/tests/hash_table_bytes_matches_the_allocator.rs`, and `src/bot/event_delivery/tests.rs`.

Move `AppState`, `Callback`, the handlers and `build_router` from `bridge/src/main.rs` into `bridge/src/lib.rs`; keep `main` as a thin binary so the HTTP surface is testable without a live WhatsApp connection.

- [ ] **Step 4: Run and verify pass**

```bash
cargo test -p wa-rs --test parity_reexports --test parity_existing_api --test parity_api_consumer
cargo test -p wa-rs-bridge
cargo test -p whatslite-desktop
cargo test --workspace
```

- [ ] **Step 5: Commit**

```bash
git add tests wacore/tests wacore/appstate/tests wacore/libsignal/tests wacore/src/voip/testdata wacore/src/voip/mlow/testdata bridge Cargo.toml
git commit -m "test: port harness fixtures and compatibility tests"
```

### Phase 0 Gate

- [ ] `cargo test -p wa-rs-core --test parity_runtime --test parity_stats --test parity_wire_enum --test parity_events`
- [ ] `cargo test -p wa-rs-core --features instance-clock --test parity_runtime`
- [ ] `cargo test -p wa-rs-core --features metrics,tracing --test parity_runtime`
- [ ] `cargo test -p wa-rs-proto`
- [ ] `cargo test -p wa-rs --test parity_event_delivery`
- [ ] `cargo test -p wa-rs --test parity_reexports --test parity_existing_api --test parity_api_consumer`
- [ ] `cargo test -p wa-rs-bridge`
- [ ] `cargo test -p whatslite-desktop`
- [ ] `cargo build -p whatslite-desktop`
- [ ] `cargo test --workspace`
- [ ] `cargo check --workspace`
- [ ] `cargo check -p wa-rs --no-default-features --features parity-core`
- [ ] `cargo check -p wa-rs-bridge`
- [ ] Bridge smoke test: pair, connect, send, receive

---

Previous: [Overview](README.md) | Next: [Phase 1](02-phase-1-auth-session.md)
