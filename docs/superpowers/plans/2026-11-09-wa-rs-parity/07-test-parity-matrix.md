# Test Parity Matrix

> Part of the [wa-rs Parity Plan](README.md). This is the functional-parity
> contract: every upstream runtime test is ported, covered by an existing
> test, or explicitly listed as a non-goal.

## Test Port Matrix

Every row is a port obligation. A row is done when the target test file exists with unchanged assertions and its command passes, or when it is marked `covered-by-existing` with the current test path and the same assertions. Rows marked **non-goal** must be recorded with a reason in `test-parity-report.md`. Do not delete the existing test when a row is `covered-by-existing`.

| Phase | Upstream test files | Target | Notes |
|---|---|---|---|
| 0 | `waproto/tests/{build_freshness,clone_codegen,descriptor_cache,generated_api,retained_schema,serde_enum_repr,serde_enum_snake_case,serde_open_enum}.rs` | same paths | Rewrite imports only |
| 0 | `wacore/tests/wire_enum_serde_test.rs`, `wacore/derive/src/tag_contract_tests.rs` | same paths | `WireEnum` and derive contract |
| 0 | `tests/{event_delivery_public,event_kind_public_api,sans_io_boundary,sdk_clock_audit,async_trait_boxed_future_compat}.rs` | same paths | Events, runtime, API compatibility |
| 0 | `wacore/tests/hash_table_bytes_matches_the_allocator.rs`, `src/bot/event_delivery/tests.rs` | same paths | Stats and event delivery |
| 0 | new `tests/parity_reexports.rs`, `tests/parity_existing_api.rs`, `tests/parity_api_consumer.rs`, `bridge/tests/parity_smoke.rs` | new paths | Architecture compatibility |
| 0 | `tests/api-consumer/` | `tests/parity_api_consumer.rs` | Rewrite as a single integration test; do not add a workspace member |
| 1 | `tests/{client_encapsulation,client_options,connect_admission,connected_idle_lifecycle,historical_upgrade,lifecycle_outcomes,subsystem_boundary,mobile_boundary,signal_durability_sqlite}.rs` | same paths | Auth/session |
| 1 | `src/client/tests.rs`, `src/client/app_state/{download_memory_tests,future_layout_tests}.rs`, `src/client/lifecycle/admission_tests.rs`, `src/mobile/registration_tests.rs`, `wacore/src/sync_engine/tests.rs` | same paths | In-crate |
| 1 | `wacore/libsignal/tests/{classic_benchmark_contract,historical_storage,host_contract,sans_io_boundary,sender_record_layout,stored_group_receiver}.rs`, `wacore/libsignal/tests/contracts/*` | same paths | Signal contracts and vectors |
| 2 | `tests/{discrepancy_pocs,download_public_api,error_surface,handshake_integration,message_references,noise_frame_size_contract,queued_download_memory,report_coverage,store_release}.rs` | same paths | Messaging/media/reliability |
| 2 | `wacore/tests/{binary_protocol_test,context_info,jid_test,noise_handshake_test,retry_conformance,storage_contract_repro,stored_message_secret,usync_response_fixture,usync_stream_response}.rs` | same paths | Protocol conformance |
| 2 | `wacore/binary/tests/*`, `wacore/noise/tests/cert_chain_verify.rs` | same paths | Binary and Noise |
| 2 | `src/message/{tests,history_durability_tests}.rs`, `src/send/tests/*`, `src/pdo/regression_tests.rs`, `src/request_context_tests.rs`, `src/request/tracing_tests.rs`, `src/retry_conformance_tests.rs`, `src/store/release_tests.rs`, `src/usync/*_tests.rs`, `src/keepalive/*_tests.rs`, `src/handlers/notification/durability_tests.rs`, `wacore/src/send/tests.rs`, `wacore/src/types/message_ref/tests.rs` | same paths | In-crate |
| 3 | `tests/{ab_prop_watch_coverage,app_state_resync_future_compat,creation_errors,group_lookup_contract,iq_parse_errors,label_sublist_public_api,label_timestamp_public_api,mex_public_api,newsletter_public_api,picture_mutation,profile_picture_lookup,reachout_timelock_public_api}.rs` | same paths | Social |
| 3 | `wacore/tests/{abprops_retained,appstate_commit_contract,appstate_external_mutations_test,appstate_mac_test,iq_pilots,mex_retained,props_stream_memory,tc_token_contract}.rs`, `wacore/tests/support/*`, `wacore/src/iq/contextual_tests.rs`, `wacore/appstate/tests/fixtures/*` | same paths | IQ and app state |
| 3 | `src/features/{picture_mutation_tests,pictures_metrics_tests,pictures_tests}.rs`, `src/handlers/notification/reachout_tests.rs` | same paths | In-crate |
| 4 | `tests/{voip_call_events,voip_call_fixture,voip_media_seam,voip_neutral_consumer}.rs` | same paths | VoIP |
| 4 | `wacore/tests/{call_action_public,group_call_builders_test,incoming_call_builder_test,mlow_logging,voip_control_boundary}.rs`, `src/voip/direct_tests.rs`, `src/voip/facade/invited_group_tests.rs`, `src/handlers/call/identity_tests.rs`, `wacore/src/voip/mlow/quality_tests.rs` | same paths | In-crate |
| 5 | `tests/{handshake_span_scope,shutdown_signal}.rs` | same paths | Tracing and shutdown |
| 5 | `storages/sqlite-storage/tests/*`, `storages/sqlite-storage/src/sqlite_store/reader_lifecycle_tests.rs`, `transports/tokio-transport/tests/dialer.rs` | same paths | Storage and transport |
| 5 | new `tests/parity_plugins.rs`, `tests/parity_observability.rs`, `tests/parity_matrix.rs` | new paths | Feature-gated |
| all | `src/test_utils.rs`, `wacore/noise/src/test_util.rs` | same paths | Test-only helpers behind `#[cfg(test)]` |
| all | existing workspace tests (`wa-rs`, `crates/whatslite-*`, baseline 780 passed/27 ignored) and `desktop` tests (21) | same paths | Keep and run |

### Fixtures and support

- `tests/fixtures/` → `tests/fixtures/`
- `wacore/tests/fixtures/retry_metadata.json` → same
- `wacore/appstate/tests/fixtures/` → same
- `wacore/libsignal/tests/vectors/` → same
- `wacore/src/voip/testdata/` and `wacore/src/voip/mlow/testdata/` → same
- `storages/sqlite-storage/tests/support/storage_fixture.rs` → same
- `tests/support/` and `wacore/tests/support/` → same

### Excluded tests

- `tests/native_test_support_cfg.rs` — requires `test-support`.
- `tests/bench-integration/`, `tests/e2e/`, `tests/sqlite-interop/` — external harnesses.
- `tests/sqlite-sdk-browser/` — keep only as the Phase 5 WASM check.
- `tools/oracle-*`, `tools/whatspec-*`, `tools/unwasm-*` — excluded tooling.
- `bench_support.rs`, `bench_*` — benchmark-only.

### Reference watch

`origin/fix/adv-primary-migration` (`7df72129`) is 4 commits ahead of
`origin/main` (`f077a5d`) with ADV/USync migration fixes. When it merges,
port the expanded `src/usync/ordinary_recovery_tests.rs` and the
`src/usync.rs`, `src/prekeys.rs`, `src/client.rs`, and
`src/client/lifecycle.rs` changes before closing Phase 2.

### Task splits

The large tasks are split in the phase files:

- Phase 2: `2.6a` protocol/XML/version, `2.6b` handshake/transport/socket/store, `2.6c` types/message refs/prekeys/usync.
- Phase 3: `3.2a` group metadata/lookup/create, `3.2b` membership/approval/settings, `3.2c` history/pictures/reports, `3.2d` chat actions/labels/favorites/quick replies; `3.6a` core IQ, `3.6b` bot/business/profile/newsletter/community/usync, `3.6c` MEX/pilots/targets, `3.6d` handlers/notifications.
- Phase 4: `4.2a` RTP/SRTP/SFrame/STUN/FEC, `4.2b` MLOW/Opus/H.264/audio/video, `4.2c` engine/driver/session/group media.
