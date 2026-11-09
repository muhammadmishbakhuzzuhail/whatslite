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
