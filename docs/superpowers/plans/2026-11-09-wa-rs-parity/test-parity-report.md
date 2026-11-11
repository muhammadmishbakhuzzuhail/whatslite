# Test Parity Report

Status for the Phase 0 test port matrix. A row is `done` when the target test
file exists with the upstream assertions and its command passes, or when the
same assertions are covered by an existing test. Deferred rows name the phase
that supplies the APIs they depend on.

## Phase 0

| Row | Status | Evidence / reason |
|---|---|---|
| `waproto/tests/{build_freshness,clone_codegen,descriptor_cache,generated_api,retained_schema,serde_enum_repr,serde_enum_snake_case,serde_open_enum}.rs` | done | `cargo test -p wa-rs-proto` → 39 passed |
| `tests/parity_reexports.rs` | done | 3 passed |
| `tests/parity_existing_api.rs` | done | 1 passed |
| `tests/parity_api_consumer.rs` | done | 3 passed |
| `bridge/tests/parity_smoke.rs` | done | 3 passed (`/health`, `/status`, `/pair/start`, `/send` validation) |
| `tests/event_kind_public_api.rs` + `tests/api-consumer/src/event_kinds/probe.rs` | done | 2 passed |
| `tests/async_trait_boxed_future_compat.rs` | done | 1 passed; adapted to the current `Contacts` signature |
| `wacore/tests/hash_table_bytes_matches_the_allocator.rs` | done | 1 passed |
| `wacore/tests/wire_enum_serde_test.rs` | deferred to Phase 2/3 | Depends on `iq::usync`, `types::group_call`, `types::call::{CallAction, CallActionTag, VideoState}`, `MembershipRequestMethod`, and `MessageCategory`, which are not ported in Phase 0. |
| `wacore/derive/src/tag_contract_tests.rs` | deferred to Phase 2 | Tagged `WireEnum` output was deferred by the Task 0.4 ruling; the contract test invokes `rustc` on tagged-mode generated code. |
| `tests/event_delivery_public.rs` | deferred to Phase 1 | Requires `BotRunOutcome`, `wacore::store::in_memory::InMemoryBackend`, `Runtime` integration, and the Phase 1 client lifecycle. |
| `tests/sans_io_boundary.rs` | deferred to Phase 1/2 | Audits the upstream module layout (`wacore::history_sync::*`, `wacore::store::traits::MsgSecretEntry`, runtime seams) that lands with later phases. |
| `tests/sdk_clock_audit.rs` | deferred to Phase 1/2 | Audits upstream clock call sites across modules not yet ported. |
| `src/bot/event_delivery/tests.rs` | deferred to Phase 1 | Requires terminal `Client::shutdown()`/`shutdown_signal()` and `Client::stats` lifecycle state; the Phase 0 adapter is covered by `tests/parity_event_delivery.rs` instead. |
| Existing workspace tests and desktop tests | done | `cargo test --workspace` → 1032 passed, 0 failed, 28 ignored; desktop → 21 passed |

## Fixtures and support

Copied to the same paths: `tests/fixtures/`, `tests/support/`,
`wacore/tests/support/`, `wacore/tests/fixtures/`,
`wacore/appstate/tests/fixtures/`, `wacore/libsignal/tests/vectors/`,
`wacore/src/voip/testdata/`, and `wacore/src/voip/mlow/testdata/`.

## Deferred rows are not deletions

No upstream test is deleted. Deferred rows keep their upstream file available
in `reference/whatsapp-rust` and are re-opened by the phase that ports their
dependencies. The Phase 0 gate only requires the rows marked done above.

## Phase 1

| Row | Status | Evidence / reason |
|---|---|---|
| new `wacore/tests/parity_session.rs`, `parity_sync.rs`, `parity_adv.rs`, `parity_identity.rs` | done | 22 passed across the four files |
| new `tests/parity_pair_code.rs`, `parity_client.rs`, `parity_admission.rs`, `parity_passkey.rs`, `parity_mobile.rs` | done | 26 passed across the five files (feature-gated rows run with their feature) |
| `tests/{client_encapsulation,client_options,connect_admission,connected_idle_lifecycle,historical_upgrade,lifecycle_outcomes,subsystem_boundary,mobile_boundary,signal_durability_sqlite}.rs` | deferred | Require the full upstream `Client` lifecycle/subsystem split; Phase 1 ports the stable builder, admission, node-filter and mobile seams, and the existing client tests remain green. |
| `src/client/tests.rs`, `src/client/app_state/{download_memory_tests,future_layout_tests}.rs`, `src/client/lifecycle/admission_tests.rs`, `src/mobile/registration_tests.rs`, `wacore/src/sync_engine/tests.rs` | deferred | Depend on the upstream monolithic client and mobile registration wire modules; the Phase 1 pure engines are covered by the new parity files. |
| `wacore/libsignal/tests/{classic_benchmark_contract,historical_storage,host_contract,sans_io_boundary,sender_record_layout,stored_group_receiver}.rs`, `wacore/libsignal/tests/contracts/*` | deferred | Signal storage contract suite belongs to the Phase 2 signal/storage port. |

Phase 1 gate automated commands all pass. Live bridge pairing/reconnect smoke
still requires a WhatsApp account/phone and is not claimed.

## Phase 2

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
| Upstream message/send/cache/retry/usync/history/keepalive directory splits | deferred | Coupled to the monolithic client, store-release leases and the Phase 1 lifecycle. The self-contained queues/caches/retry surfaces are covered by the parity files. |

## Phase 3

| Row | Status | Evidence / reason |
|---|---|---|
| `tests/parity_social_foundations.rs`, `parity_groups.rs`, `parity_chat_actions.rs` | done | 3 passed |
| `tests/parity_groups_membership.rs`, `parity_group_history.rs`, `label_sublist_public_api.rs`, `label_timestamp_public_api.rs` | done | 4 passed |
| `tests/parity_community.rs`, `parity_newsletter.rs`, `parity_status_polls_events.rs` | done | 3 passed |
| `tests/parity_social_misc.rs`, `parity_appstate.rs`, `parity_handlers.rs` | done | 3 passed |
| `wacore/tests/parity_iq_social.rs`, `abprops_retained.rs`, `props_stream_memory.rs`, `iq_pilots.rs`, `mex_retained.rs` | done | 6 passed |
| `src/features/{picture_mutation_tests,pictures_metrics_tests,pictures_tests}.rs` | deferred | Deep picture mutation/IQ metrics depend on the deferred picture IQ domain. |
| Upstream social IQ/handler directory splits | deferred | Feature surfaces exist; deep notification handlers and several IQ domains are deferred. |

## Phase 4

| Row | Status | Evidence / reason |
|---|---|---|
| `wacore/tests/parity_voip_control.rs` | done | 5 passed with `--features voip-control` |
| `wacore/tests/parity_voip_crypto.rs` | done | 4 passed with `--features voip` (RTP/RTCP/STUN/SFrame/SRTP/FEC) |
| `wacore/tests/parity_voip_engine.rs` | done | 2 passed with `--features voip` (sans-IO engine/driver) |
| `wacore/tests/parity_voip_mlow.rs` + `wacore/tests/mlow_logging.rs` | done | 2 + 1 passed with `--features voip-mlow` |
| `tests/parity_voip_facade.rs` | done | 1 passed with `--features voip-runtime` |
| `tests/parity_calls.rs` | done | 1 passed with `--features voip` |
| upstream `wacore/src/voip/**` engine/session/relay/codec suites | deferred / non-goal for this pass | The full native engine, relay and codec wiring depends on the upstream monolithic client/socket and store-release APIs. The ported primitives and sans-IO engine are covered by the Phase 4 parity files; no upstream test was deleted. |

## Phase 5

| Row | Status | Evidence / reason |
|---|---|---|
| `tests/parity_plugins.rs` | done | 7 passed with `--features plugins` |
| `tests/parity_observability.rs` | done | 5 passed with `--features tracing,metrics` |
| `tests/parity_matrix.rs` | done | 9 tests; all seven `parity-*` alias runs exit 0 |
| `tests/shutdown_signal.rs` | done | 1 passed with `--features signal` |
| `storages/sqlite-storage/tests/parity_storage.rs` | done | 1 passed (bundled profile) |
| `transports/tokio-transport/tests/parity_tcp.rs` | done | 3 passed with `--features tcp` |
| `tests/handshake_span_scope.rs` | deferred | Depends on the upstream `wa.conn.handshake` span and `NoiseSocket::SendObservers`, not exposed by the compatibility client. |
| Upstream storage/transport reader-lifecycle and dialer suites | deferred | Depend on upstream storage/transport internals; bundled profile, device scoping, TCP dial and byte-stream contracts are covered. |
| `tests/sqlite-sdk-browser/` | non-goal except WASM check | Browser SDK harness is external; the WASM storage check is covered by the `wasm32` check below. |

Final evidence (revalidated 2026-11-11): workspace 1953 passed/0 failed/30 ignored;
fmt clean; workspace check green; bridge 3 passed; desktop 21 passed (previous
gate); 7/7 parity alias test runs; 14/14 feature-matrix checks; WASM storage
check green with `--no-default-features --features js`. Live bridge
pairing/send/receive/media smoke remains blocked on a real WhatsApp account/phone.

