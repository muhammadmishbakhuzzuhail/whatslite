# Test Parity Report

**Reference:** `oxidezap/whatsapp-rust` 0.7.0 @ `f077a5d4`.
**Date:** 2026-11-09.

A row is **done** when the target test exists with the upstream assertions and
its command passes, or when the same assertions are covered by an existing test.
A row is **deferred/non-goal** only with a reason. No upstream test was deleted
to make a build green.

## Phase 0 — Foundation

| Row | Status | Evidence / reason |
|---|---|---|
| `waproto/tests/{build_freshness,clone_codegen,descriptor_cache,generated_api,retained_schema,serde_enum_repr,serde_enum_snake_case,serde_open_enum}.rs` | done | `cargo test -p wa-rs-proto` → 39 passed |
| `tests/parity_reexports.rs` | done | 3 passed |
| `tests/parity_existing_api.rs` | done | 1 passed |
| `tests/parity_api_consumer.rs` | done | 3 passed |
| `bridge/tests/parity_smoke.rs` | done | 3 passed |
| `tests/event_kind_public_api.rs` + `tests/api-consumer/src/event_kinds/probe.rs` | done | 2 passed |
| `tests/async_trait_boxed_future_compat.rs` | done | 1 passed |
| `wacore/tests/hash_table_bytes_matches_the_allocator.rs` | done | 1 passed |
| `wacore/tests/wire_enum_serde_test.rs` | deferred to Phase 2/3 | Depends on IQ/group-call/call-action types not ported in Phase 0; the Phase 2 protocol suites cover the retained wire enums. |
| `wacore/derive/src/tag_contract_tests.rs` | deferred | Tagged `WireEnum` output was deferred by the Task 0.4 ruling. |
| `tests/event_delivery_public.rs` | deferred | Requires `BotRunOutcome`, `InMemoryBackend`, `Runtime` integration and the Phase 1 lifecycle. |
| `tests/sans_io_boundary.rs` | deferred | Audits upstream module layout (`history_sync::*`, store traits, runtime seams) that lands with later phases. |
| `tests/sdk_clock_audit.rs` | deferred | Audits upstream clock call sites across modules not yet ported. |
| `src/bot/event_delivery/tests.rs` | deferred | Requires terminal `Client::shutdown()`/lifecycle state; the adapter is covered by `tests/parity_event_delivery.rs`. |
| Existing workspace + desktop tests | done | `cargo test --workspace` green; desktop 21 passed |

## Phase 1 — Auth, session and recovery

| Row | Status | Evidence / reason |
|---|---|---|
| new `wacore/tests/parity_session.rs`, `parity_sync.rs`, `parity_adv.rs`, `parity_identity.rs` | done | 22 passed |
| new `tests/parity_pair_code.rs`, `parity_client.rs`, `parity_admission.rs`, `parity_passkey.rs`, `parity_mobile.rs` | done | 26 passed |
| `tests/{client_encapsulation,client_options,connect_admission,connected_idle_lifecycle,historical_upgrade,lifecycle_outcomes,subsystem_boundary,mobile_boundary,signal_durability_sqlite}.rs` | deferred | Require the full upstream `Client` lifecycle/subsystem split; the stable builder/admission/node-filter/mobile seams are covered by the parity files. |
| `src/client/tests.rs`, `src/client/app_state/*`, `src/client/lifecycle/admission_tests.rs`, `src/mobile/registration_tests.rs`, `wacore/src/sync_engine/tests.rs` | deferred | Depend on the upstream monolithic client and mobile registration wire modules. |
| `wacore/libsignal/tests/{classic_benchmark_contract,historical_storage,host_contract,sans_io_boundary,sender_record_layout,stored_group_receiver}.rs`, `contracts/*` | deferred | Signal storage contract suite depends on the upstream store/session port. |

## Phase 2 — Messaging, media and reliability

| Row | Status | Evidence / reason |
|---|---|---|
| `wacore/tests/parity_messaging.rs`, `parity_secrets.rs` | done | 8 passed |
| `tests/parity_inbound.rs` | done | 3 passed |
| `tests/parity_send.rs`, `parity_cache.rs` | done | 6 passed |
| `tests/parity_media.rs` | done | 3 passed |
| `tests/parity_reliability.rs`, `parity_pdo.rs`, `parity_history.rs` | done | 7 passed |
| `wacore/tests/parity_protocol.rs` | done | 2 passed |
| `tests/parity_transport.rs`, `parity_types.rs` | done | 5 passed |
| `wacore/tests/noise_handshake_test.rs` | done | 4 passed |
| `wacore/binary/tests/*` | covered | `cargo test -p wa-rs-binary` green |
| Upstream `src/message/*` split, `src/send/*` split, durable cache stores, retry/usync/history/keepalive directory splits | deferred | Coupled to the monolithic client, store-release leases and the Phase 1 lifecycle. The self-contained queues/caches/retry surfaces are covered by the parity files. |

## Phase 3 — Social features

| Row | Status | Evidence / reason |
|---|---|---|
| `tests/parity_social_foundations.rs`, `parity_groups.rs`, `parity_chat_actions.rs` | done | 3 passed |
| `tests/parity_groups_membership.rs`, `parity_group_history.rs`, `label_sublist_public_api.rs`, `label_timestamp_public_api.rs` | done | 4 passed |
| `tests/parity_community.rs`, `parity_newsletter.rs`, `parity_status_polls_events.rs` | done | 3 passed |
| `tests/parity_social_misc.rs`, `parity_appstate.rs`, `parity_handlers.rs` | done | 3 passed |
| `wacore/tests/parity_iq_social.rs`, `abprops_retained.rs`, `props_stream_memory.rs`, `iq_pilots.rs`, `mex_retained.rs` | done | 6 passed |
| `src/features/{picture_mutation_tests,pictures_metrics_tests,pictures_tests}.rs` | deferred | Deep picture mutation/IQ metrics depend on the deferred picture IQ domain. |
| Upstream social IQ/handler directory splits | deferred | The feature surfaces exist; deep notification handlers and the IQ domains listed in the parity audit are deferred. |

## Phase 4 — Calls and VoIP

| Row | Status | Evidence / reason |
|---|---|---|
| `wacore/tests/parity_voip_control.rs` | done | 5 passed (`--features voip-control`) |
| `wacore/tests/parity_voip_crypto.rs` | done | 4 passed (`--features voip`) |
| `wacore/tests/parity_voip_engine.rs` | done | 2 passed (`--features voip`) |
| `wacore/tests/parity_voip_mlow.rs`, `mlow_logging.rs` | done | 3 passed (`--features voip-mlow`) |
| `tests/parity_voip_facade.rs` | done | 1 passed (`--features voip-runtime`) |
| `tests/parity_calls.rs` | done | 1 passed (`--features voip`) |
| Upstream `wacore/src/voip/**` engine/session/relay/codec suites | deferred | Native engine, relay and codec wiring depends on the upstream client/socket and store-release APIs; the ported primitives are covered by the parity files. |

## Phase 5 — Platform and observability

| Row | Status | Evidence / reason |
|---|---|---|
| `tests/parity_plugins.rs` | done | 7 passed (`--features plugins`) |
| `tests/parity_observability.rs` | done | 5 passed (`--features tracing,metrics`) |
| `tests/parity_matrix.rs` | done | 9 tests; run under all seven `parity-*` aliases |
| `tests/shutdown_signal.rs` | done | 1 passed (`--features signal`) |
| `storages/sqlite-storage/tests/parity_storage.rs` | done | 1 passed (bundled profile) |
| `transports/tokio-transport/tests/parity_tcp.rs` | done | 3 passed (`--features tcp`) |
| `tests/handshake_span_scope.rs` | deferred | Depends on the upstream `wa.conn.handshake` span and `NoiseSocket::SendObservers`, which the compatibility client does not expose. |
| Upstream `storages/sqlite-storage/tests/*`, `src/sqlite_store/reader_lifecycle_tests.rs`, `transports/tokio-transport/tests/dialer.rs` | deferred | Full storage pool/reader-lifecycle and dialer suites depend on the upstream storage/transport internals; the bundled profile, device scoping, TCP dial and byte-stream contracts are covered by the parity files. |
| `tests/sqlite-sdk-browser/` | non-goal (except WASM check) | Browser SDK harness is external; the WASM storage check is covered by `cargo check -p wa-rs-sqlite-storage --target wasm32-unknown-unknown --no-default-features --features js`. |

## Final evidence

- `cargo test --workspace` → **1297 passed, 0 failed, 29 ignored**.
- `cargo check --workspace` → exit 0.
- `cargo clippy --workspace --all-targets` → exit 0, **190 warning lines** (baseline, no new warnings).
- `cargo fmt --all --check` → exit 0.
- `cargo test -p whatslite-desktop` → 21 passed; desktop build → exit 0.
- `cargo test -p wa-rs-bridge` → 3 passed; bridge check → exit 0.
- Feature-target suites: plugins 7, observability 5, storage 22 (incl. parity), TCP 3, shutdown signal 1.
- Seven `parity-*` alias test runs → all exit 0.
- Fourteen feature-matrix `cargo check` commands → all exit 0.
- WASM storage check → exit 0 with `--no-default-features --features js` and a user-local clang.

## Blockers / non-goals

- Live bridge pairing/connect/send/receive/media/call smoke requires a real
  WhatsApp account and phone; not claimed.
- Native VoIP engine/relay/codec suites, the upstream monolithic lifecycle
  suites, and the handshake span scope test remain deferred as described above.
