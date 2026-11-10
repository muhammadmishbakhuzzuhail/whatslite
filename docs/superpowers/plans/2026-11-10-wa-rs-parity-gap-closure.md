# wa-rs Parity Gap Closure Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close every remaining feature and test gap between the current `wa-rs` tree and the parity plan (`reference/whatsapp-rust` @ `f077a5d4`), without removing the compatibility facade or breaking `bridge/` and `desktop/`.

**Architecture:** Continue the strangler port. Add the missing upstream modules alongside the existing monolithic `Client`, `send`, `message`, `request`, `usync`, `pdo`, `history_sync`, `keepalive` and `voip` implementations; route new code through `PersistenceManager` and the existing feature gates. Each task lands one independently testable subsystem and keeps the old public API working.

**Tech Stack:** Rust 2024, Tokio, Diesel/SQLite, buffa 0.9.2, rustls, GPUI desktop consumer, `bridge/` HTTP sidecar.

**Spec:** [Parity README](2026-11-09-wa-rs-parity/README.md), [Phase 0](2026-11-09-wa-rs-parity/01-phase-0-foundation.md), [Phase 1](2026-11-09-wa-rs-parity/02-phase-1-auth-session.md), [Phase 2](2026-11-09-wa-rs-parity/03-phase-2-messaging-media.md), [Phase 3](2026-11-09-wa-rs-parity/04-phase-3-social.md), [Phase 4](2026-11-09-wa-rs-parity/05-phase-4-voip.md), [Phase 5](2026-11-09-wa-rs-parity/06-phase-5-platform.md), [Test Parity Matrix](2026-11-09-wa-rs-parity/07-test-parity-matrix.md).

## Global Constraints

- Keep the current top-level tree: `wacore/`, `waproto/`, `src/`, `storages/`, `transports/`, `http_clients/`, `bridge/`, `desktop/`.
- `reference/whatsapp-rust` stays read-only at `f077a5d4`; `origin/fix/adv-primary-migration` (`7df72129`) is a watch item only.
- Upstream feature names are canonical; `parity-*` aliases only add edges and never appear in `#[cfg]`.
- Never modify `Device` directly. Use `DeviceCommand` + `PersistenceManager::process_command()`; use `get_device_snapshot()` for reads.
- All I/O is Tokio. Blocking I/O and heavy CPU work go in `tokio::task::spawn_blocking`. Serialize per-chat operations with `Client::chat_locks`.
- Preserve the existing `wa-rs` API. Add upstream-style APIs alongside it.
- No dual backend. Do not delete a test to make a build green.
- Build policy: `.cargo/config.toml` sets `jobs = 4`; use per-package `cargo check -p <crate>` / `cargo test -p <crate> --test <name>` during iteration. Full `cargo test --workspace` and `cargo clippy --workspace --all-targets` are phase-gate only. Full test execution uses `-- --test-threads=4` when other apps are running.
- `bridge/` and `desktop/` must compile and their tests must pass at every gate.

## Audit Findings (2026-11-10)

Read-only audit of the six phase plans against the current tree:

| Phase | Plan status | Evidence |
|---|---|---|
| 0 — Foundation | complete (minor test fixtures) | All core files exist; only `waproto/tests/proto_roundtrip.rs` and an optional `tests/support/` directory are absent. Event delivery exists as `src/bot/event_delivery.rs`. |
| 1 — Auth/session | **partial** | 26 unchecked plan steps; 25 `Create` paths absent, including `src/client/{offline_resume,session_state,iq_ops,accessors,adapters,app_state,device_topology,durability_probe_id,group_notifications,interceptor,member_index,messaging,notification_scope,subsystem}.rs`, `src/client/{app_state,lifecycle,tests}/`, and `wacore/src/mobile/{registration,registration_wire,session,token}.rs`. Missing interfaces: `CompanionReg`, `SignalMaintenanceError`. |
| 2 — Messaging/media | **partial** | 41 unchecked plan steps; 36 `Create` paths absent, including the `src/message/*`, `src/send/*`, `src/request/`, `src/usync/`, `src/keepalive/`, `src/pdo/`, `src/history_sync/`, `src/lid_pn_cache/`, `src/portable_cache/` splits, `wacore/src/message_processing.rs`, `wacore/src/serde_helpers.rs`, `wacore/src/types/message_ref/`, `wacore/src/types/message_secret.rs`, and `src/features/media_reupload.rs`. Missing interfaces include `BotMessage`, `CategorizedEncNodes`, `DecryptedMessageResult`, `EncNodeInfo`, `EncType`, `EncryptedEdit`, `ProtocolMessageInfo`, `SecretEncAddon`, `SecretEncKind`, `SecretEncrypted`, `UsyncProtocol`, `UsyncQuery`, `ProtoHelpers`, `Xml`, `MessageSecret`, `SyncTask`. |
| 3 — Social | **partial** | 60 unchecked plan steps; 27 `Create` paths absent, including `wacore/src/iq/{abprops,bot,business,community_create,contextual,devices,join_shapes,mex_operations,mex_operations_unlisted,newsletter,pilots,profile,targets}.rs`, `wacore/src/iq/usync/`, `wacore/src/appstate_sync*`, `wacore/src/poll.rs`, `src/appstate_known_verbs.rs`, `src/handlers/notification/{device,groups,privacy_business,profile}.rs`, `src/handlers/presence.rs`. Missing interfaces include `CappingStatus`, `MexDoc`, `MexOperation`, `TcTokenError`, `GroupError`, `GroupLookupResult`, `GroupOverview`, `GroupOverviewResult`, `GroupEphemeralSettings`, `GroupJoinError`, `GroupParticipantDetails`, `JoinGroupResult`, `MembershipRequest`, `GroupPictureEntry`, `GroupProfilePicture`, `ReportedGroupMessage`, `ReportedGroupMessages`, `StarredMessages`. |
| 4 — Calls/VoIP | **partial** | 30 unchecked plan steps; 49 `Create` paths absent, including the 19-file `wacore/src/voip_control/` split, `src/voip_control/wacore_backend.rs`, `wacore/src/voip/{e2e_srtp,hbh_srtp,audio,codec_probe,demux,h264,opus_packet,driver,session,group_audio,group_media,tap,warp}.rs`, `wacore/src/voip/sframe/`, `src/voip/{audio,direct,direct_client,direct_control,direct_ingress,driver,events,facade,registry,session,state,video,transport}.rs`, `src/handlers/call/`, and `wacore/src/types/group_call.rs`. |
| 5 — Platform | contract complete, deep wiring partial | All plan files and named interfaces exist. Remaining deep gaps: plugin install/callback queue is not wired into `Client`; `ClientLifecycle` is not installed by the builder; `Client::memory_report`/`device_memo_stats` return defaults; storage `js` has no browser shim; transport `StreamDialer`/dialer tests are absent; `handshake_span_scope` is deferred. |

**Conclusion:** Phases 0 and 5 are usable as delivered, but Phases 1–4 are not feature-complete. The tasks below close the missing files, interfaces, tests, and deep wiring in dependency order.

## Review Focus

1. **Compatibility facade** — `bridge` can still pair, connect, send and receive; `desktop` builds and its 21 tests pass.
2. **No test deletion** — every deferred upstream test is either ported or explicitly recorded as a non-goal with a reason.
3. **Feature-gated builds** — default, every `parity-*` alias, canonical features, and the wasm32 storage check build independently.
4. **Session/Signal durability** — reconnect, offline resume and storage restart preserve sessions and do not duplicate deliveries.
5. **Wire/event conformance** — 80 `EventKind`/`Event` variants, IQ fixtures, media crypto and VoIP vectors match upstream.

---

### Task G1: Protocol, serde, message refs and core types

**Files:**
- Create: `wacore/src/serde_helpers.rs`
- Create: `wacore/src/types/message_ref.rs` and `wacore/src/types/message_ref/` (tests)
- Test: `wacore/tests/context_info.rs`
- Modify: `wacore/src/types/mod.rs`, `wacore/src/lib.rs`

**Interfaces:**
- Produces: `ProtoHelpers`, `Xml`, `MessageRef`.
- Preserves: existing `wacore::xml`, `wacore::types` exports.

**Scope ruling (2026-11-10):** `wacore/src/types/message_secret.rs`, `stored_message_secret`, `InMemoryBackend`/`MsgSecretStore`, the binary tests, and the USync stream fixtures are moved to G3/G6 because they depend on store/stream internals that do not exist yet. `waproto/tests/proto_roundtrip.rs` is removed: it does not exist upstream at `f077a5d4`.

- [ ] **Step 1: Write the failing tests** by porting the four upstream `wacore/tests/*` files and the binary test files. Keep assertions unchanged; rewrite imports to `wacore`/`wacore_binary`.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs-core --test context_info --test stored_message_secret --test usync_response_fixture --test usync_stream_response
cargo test -p wa-rs-binary
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/wacore/src/serde_helpers.rs`, `wacore/src/types/message_ref*`, `wacore/src/types/message_secret.rs`, and `wacore/binary/tests/`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs-core --test context_info --test stored_message_secret --test usync_response_fixture --test usync_stream_response
cargo test -p wa-rs-binary
cargo check -p wa-rs-core
```

- [ ] **Step 5: Commit**

```bash
git add wacore/src/serde_helpers.rs wacore/src/types/message_ref.rs wacore/src/types/message_ref wacore/src/types/message_secret.rs wacore/binary/tests wacore/tests/context_info.rs wacore/tests/stored_message_secret.rs wacore/tests/usync_response_fixture.rs wacore/tests/usync_stream_response.rs
git commit -m "feat(core): port serde helpers message refs and core types"
```

### Task G2: Client lifecycle, subsystem and offline resume

**Files:**
- Create: `src/client/{offline_resume,session_state,iq_ops,accessors,adapters,app_state,device_topology,durability_probe_id,group_notifications,interceptor,member_index,messaging,notification_scope,subsystem}.rs`
- Create: `src/client/app_state/`, `src/client/lifecycle/`, `src/client/tests.rs`, `src/client/tests/`
- Modify: `src/client.rs`, `src/client/{builder,lifecycle,node_io,context_impl,device_registry,lid_pn,sender_keys,sessions}.rs`
- Test: `tests/parity_client.rs`, `tests/parity_admission.rs`; add `tests/client_encapsulation.rs`, `tests/subsystem_boundary.rs` from upstream.

**Interfaces:**
- Produces: `SignalMaintenanceError`, `CompanionReg`, `Subsystem`, `OfflineResume`, `ClientAccessors`.
- Preserves: `ClientOptions`, `ClientBuilder`, `ClientBuild`, `ClientBuilderError`, `ConnectStage`, `ConnectError`, `NodeFilter`, `NodeWaiter<T>`, `ClientError`, `Client::new`, `run`, `connect`, `disconnect`, `send_node`.

- [ ] **Step 1: Write failing tests** for offline resume, subsystem boundaries, accessors, IQ ops, group notification facts, and durability probe IDs.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --test parity_client --test parity_admission --test client_encapsulation --test subsystem_boundary
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/src/client/` and `src/client.rs`. Keep the current `Client` as the compatibility entry point; route state changes through `PersistenceManager`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --test parity_client --test parity_admission --test client_encapsulation --test subsystem_boundary
cargo check -p wa-rs
cargo test -p wa-rs-bridge
```

- [ ] **Step 5: Commit** with `feat(client): port lifecycle subsystem and offline resume`.

### Task G3: Message processing and inbound pipeline

**Files:**
- Create: `wacore/src/message_processing.rs`
- Create: `wacore/src/store/in_memory.rs`, `wacore/src/types/message_secret.rs`
- Create: `src/message/{commit_batch,dispatch,durability,live_retention,msg_secret,raw_pending,receive,retry,special,tests}.rs`
- Create: `src/handlers/notification/durability_tests.rs`
- Modify: `src/message.rs`, `src/handlers/notification/mod.rs`
- Test: `src/message/tests.rs`, `wacore/tests/stored_message_secret.rs`, `tests/parity_inbound.rs`, `tests/parity_messaging.rs`

**Interfaces:**
- Produces: `BotMessage`, `CategorizedEncNodes`, `DecryptedMessageResult`, `EncNodeInfo`, `EncType`, `EncryptedEdit`, `ProtocolMessageInfo`, `SecretEncAddon`, `SecretEncKind`, `SecretEncrypted`, `MessageSecret`, `StoredMessageSecret`, `MsgSecretStore`, `InMemoryBackend`.
- Preserves: existing `Client::handle_message`, `Event` dispatch, and inbound durability hooks.

- [ ] **Step 1: Write failing tests** for batch commit/drain, decrypt result classification, message secrets, retry/special messages, live retention, and durability notification.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --test parity_inbound --test parity_messaging
cargo test -p wa-rs-core --test stored_message_secret
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/src/message/`, `src/handlers/notification/durability_tests.rs`, and `wacore/src/message_processing.rs`. Keep the existing 64-line `src/message.rs` as a facade until the split is green.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --test parity_inbound --test parity_messaging
cargo test -p wa-rs-core --test parity_messaging --test stored_message_secret
cargo check -p wa-rs
```

- [ ] **Step 5: Commit** with `feat(message): port inbound pipeline and message processing`.

### Task G4: Send pipeline, cache and device sync

**Files:**
- Create: `src/send/{mod,actions,requests,group_repair,tctoken_lifecycle}.rs`, `src/send/tests/`
- Create: `src/pending_device_sync.rs`, `src/resend_rate_limiter.rs`, `src/sender_key_device_cache.rs`, `src/portable_cache/`
- Modify: `src/send.rs`, `src/portable_cache.rs`, `src/client.rs`
- Test: `tests/parity_send.rs`, `tests/parity_cache.rs`

**Interfaces:**
- Preserves: `SendOptions`, `EditOptions`, `SendResult`, `PinDuration`, `SendRequest`, `EditRequest`, `SendError`, `ServerRejection`, `CacheConfig`, `CacheEntryConfig`, `CacheStores`, `CacheStore`, `Freshness`, `OriginalMessageResolver`, `PortableCache`.
- Produces: device-sync dedup and resend rate limiting used by the new send split.

- [ ] **Step 1: Write failing tests** for send actions/requests, group repair, TCToken lifecycle, device sync dedup, resend throttling and cache freshness.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --test parity_send --test parity_cache
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/src/send/`, `src/pending_device_sync.rs`, `src/resend_rate_limiter.rs`, `src/sender_key_device_cache.rs`, and `src/portable_cache/`. Keep `src/send.rs` as the compatibility facade.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --test parity_send --test parity_cache
cargo check -p wa-rs
cargo test -p wa-rs-bridge
```

- [ ] **Step 5: Commit** with `feat(send): port send pipeline cache and device sync`.

### Task G5: Media streaming, reupload and download budgets

**Files:**
- Create: `src/download/budget.rs`, `src/download/memory.rs`, `src/features/media_reupload.rs`
- Modify: `src/download.rs`, `src/upload.rs`, `src/features/mod.rs`
- Test: `tests/parity_media.rs`, `tests/download_public_api.rs`, `tests/queued_download_memory.rs`

**Interfaces:**
- Preserves: `ImageOptions`, `VideoOptions`, `DocumentOptions`, `AudioOptions`, `DownloadLimits`, `MediaByteDomain`, `DownloadParams`, `MediaDownloadError`, `ClientDownloadError`, `MediaDownloader`, `UploadOptions`, `MediaReupload*`.
- Produces: download budget/memory accounting and media reupload entry point.

- [ ] **Step 1: Write failing tests** for queued download memory, streaming budget enforcement, reupload and public download API.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --test parity_media --test download_public_api --test queued_download_memory
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/src/download/{budget,memory}.rs`, `src/features/media_reupload.rs`, and the upstream download/upload implementations. Keep the existing `download.rs`/`upload.rs` paths working.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --test parity_media --test download_public_api --test queued_download_memory
cargo check -p wa-rs
```

- [ ] **Step 5: Commit** with `feat(media): port streaming budgets and reupload`.

### Task G6: Reliability, PDO, history, request/usync, keepalive and LID

**Files:**
- Create: `src/{pdo,history_sync,request,usync,keepalive,lid_pn_cache}/`
- Create: `src/request_context_tests.rs`, `src/retry_conformance_tests.rs`, `src/store/release_tests.rs`, `src/usync/*_tests.rs`
- Create: `tests/handshake_integration.rs`, `tests/noise_frame_size_contract.rs`
- Create: `wacore/binary/tests/`, `wacore/tests/{usync_response_fixture,usync_stream_response}.rs`
- Modify: `src/{pdo,history_sync,request,usync,keepalive,lid_pn_cache}.rs`, `src/store/mod.rs`
- Test: `tests/parity_reliability.rs`, `tests/parity_pdo.rs`, `tests/parity_history.rs`

**Interfaces:**
- Produces: the directory-split public surfaces for PDO, history sync, request context, USync, keepalive, LID/PN cache and store release; `UsyncProtocol`, `UsyncQuery`, `SyncTask`, `NodeStream`, `IqStreamSpec`.
- Preserves: current retry/stanza-error/history-sync keys and `StoreRelease` behavior.

- [ ] **Step 1: Write failing tests** for request context, retry conformance, store release, USync fixtures, handshake integration and noise frame size.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --test parity_reliability --test parity_pdo --test parity_history
cargo test -p wa-rs-core --test noise_handshake_test
cargo test -p wa-rs-core --test usync_response_fixture --test usync_stream_response
cargo test -p wa-rs-binary
```

- [ ] **Step 3: Port the modules** from the corresponding upstream directories. Keep the existing monolithic modules as re-export facades.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --test parity_reliability --test parity_pdo --test parity_history
cargo test -p wa-rs-core --test noise_handshake_test --test usync_response_fixture --test usync_stream_response
cargo test -p wa-rs-binary
cargo check -p wa-rs
```

- [ ] **Step 5: Commit** with `feat(reliability): port pdo history request usync keepalive and lid`.

### Task G7: Mobile registration and primary session

**Files:**
- Create: `wacore/src/mobile/{registration,registration_wire,session,token}.rs`
- Create: `src/mobile/{primary,registration,registration_tests}.rs`
- Modify: `wacore/src/mobile/mod.rs`, `src/mobile/mod.rs`, `src/mobile_session.rs`, `src/features/mobile_api.rs`
- Test: `tests/parity_mobile.rs`, `src/mobile/registration_tests.rs`

**Interfaces:**
- Produces: registration request/response, primary session binding, mobile token/session.
- Preserves: `MobileAppStatePolicy`, `MobileEphemeralKeyProvider`, `MobileLoginAttempt`, `MobileLoginOptionsProvider`, `RegisteredMobileSession`, `RegistrationLoginPhoneIdMapping`, `PrimarySelfIdentityProfile`, `PrimaryDeviceIdentityPolicy`, `MobileMediaHttpProfile`, `validate_registration_login_profile`, `PendingCompanion`, `AcknowledgedCompanion`, `VerifiedCompanion`.

- [ ] **Step 1: Write failing tests** for registration wire, primary session binding, token/session transitions and registration tests.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --features mobile-core --test parity_mobile
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/wacore/src/mobile/` and `src/mobile/`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --features mobile-core --test parity_mobile
cargo check -p wa-rs --features mobile-core
cargo test -p wa-rs-bridge
```

- [ ] **Step 5: Commit** with `feat(mobile): port registration and primary session`.

### Task G8: Social group types and IQ

**Files:**
- Create: `wacore/src/iq/{join_shapes,community_create,newsletter,bot,business,profile,abprops,contextual,devices}.rs`
- Create: `wacore/src/iq/usync/`
- Create: `wacore/src/store/ab_props.rs` (AbPropsCache, owned by G8; G2a defers `ab_prop_enabled`)
- Modify: `wacore/src/iq/{mod,groups,props,usync}.rs`
- Test: `tests/group_lookup_contract.rs`, `tests/iq_parse_errors.rs`, `tests/newsletter_public_api.rs`, `tests/ab_prop_watch_coverage.rs`

**Interfaces:**
- Produces: `GroupError`, `GroupLookupResult`, `GroupOverview`, `GroupOverviewResult`, `GroupEphemeralSettings`, `GroupJoinError`, `GroupParticipantDetails`, `JoinGroupResult`, `MembershipRequest`, `GroupPictureEntry`, `GroupProfilePicture`, `ReportedGroupMessage`, `ReportedGroupMessages`, `StarredMessages`, `CappingStatus`, `AbPropsCache`, `ab_prop_enabled`.
- Preserves: `GroupMetadata`, `CreateGroupResult`, `GroupParticipant`, `MemberAddMode`, `MembershipApprovalMode`.

- [ ] **Step 1: Write failing tests** for group lookup, IQ parse errors, newsletter public API and AB-prop watch coverage.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --test group_lookup_contract --test iq_parse_errors --test newsletter_public_api --test ab_prop_watch_coverage
cargo test -p wa-rs-core --test parity_iq_social
```

- [ ] **Step 3: Port the IQ modules** from `reference/whatsapp-rust/wacore/src/iq/`. `abprops.rs` may re-export the existing `props.rs`; `pilots.rs` may re-export `usync.rs`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --test group_lookup_contract --test iq_parse_errors --test newsletter_public_api --test ab_prop_watch_coverage
cargo test -p wa-rs-core --test parity_iq_social
cargo check -p wa-rs
```

- [ ] **Step 5: Commit** with `feat(social): port group types and IQ domains`.

### Task G9: Social app state, polls and MEX operations

**Files:**
- Create: `wacore/src/appstate_sync.rs`, `wacore/src/appstate_sync/`, `wacore/src/poll.rs`
- Create: `wacore/src/iq/{mex_operations,mex_operations_unlisted,pilots,targets}.rs`
- Create: `src/appstate_known_verbs.rs`
- Modify: `src/appstate_sync.rs`, `src/features/app_state_settings.rs`, `src/features/app_state_resync.rs`, `src/features/polls.rs`, `src/features/events.rs`
- Test: `tests/mex_public_api.rs`, `tests/reachout_timelock_public_api.rs`, `tests/parity_appstate.rs`

**Interfaces:**
- Produces: `MexDoc`, `MexOperation`, `TcTokenError`, `AppStateSettings`, `AppStateResyncMode`, `AppStateResyncReport`, poll creation/vote/decrypt surfaces.
- Preserves: existing MEX/TCToken/status/polls/events types.

- [ ] **Step 1: Write failing tests** for MEX operations, reachout timelock, app-state known verbs and poll round-trips.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --test mex_public_api --test reachout_timelock_public_api --test parity_appstate
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/wacore/src/appstate_sync*`, `wacore/src/poll.rs`, `wacore/src/iq/{mex_operations*,pilots,targets}.rs`, and `src/appstate_known_verbs.rs`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --test mex_public_api --test reachout_timelock_public_api --test parity_appstate
cargo check -p wa-rs
```

- [ ] **Step 5: Commit** with `feat(social): port app state polls and mex operations`.

### Task G10: Handlers and notification coverage

**Files:**
- Create: `src/handlers/notification/{device,groups,privacy_business,profile,reachout_tests}.rs`
- Create: `src/handlers/presence.rs`
- Modify: `src/handlers/notification/mod.rs`, `src/handlers/mod.rs`
- Test: `src/handlers/notification/reachout_tests.rs`, `tests/parity_handlers.rs`

**Interfaces:**
- Produces: device/group/privacy-business/profile notification handlers and presence handling.
- Preserves: current handler router and event dispatch.

- [ ] **Step 1: Write failing tests** for device/group/privacy-business/profile notifications and reachout timelock.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --test parity_handlers
```

- [ ] **Step 3: Port the handlers** from `reference/whatsapp-rust/src/handlers/notification/` and `src/handlers/presence.rs`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --test parity_handlers
cargo check -p wa-rs
cargo test -p wa-rs-bridge
```

- [ ] **Step 5: Commit** with `feat(handlers): port notification and presence coverage`.

### Task G11: VoIP control boundary split

**Files:**
- Create: `wacore/src/voip_control/{app_data,audio_format,control,engine_bridge,events,fake_backend,group,kdf,media_stats,ports,registry,relay_parse,resident_session,signaling,signaling_plan,spec_build,ssrc,transport}.rs`
- Create: `src/voip_control/wacore_backend.rs`
- Modify: `wacore/src/voip_control/mod.rs`, `src/voip_control/mod.rs`
- Test: `wacore/tests/parity_voip_control.rs`

**Interfaces:**
- Preserves: `GroupControl`, `NormalizedGroupControl`, `GroupRawEpoch`, `VideoControl`, `VideoControlSender`, `VideoControlReceiver`, `PeerAnswer`, `CallEvent`, `MediaGroupEpoch`, `MediaSessionKey`, `MediaAudioSpec`, `MediaKeyframeUrgency`, `VideoFecConfig`, `MediaVideoUpgradeToken`, `MediaGroupTransition`, `MediaCodecDecisionSource`, `MediaSilenceReason`, `MediaEncodedFrame`, `MediaRtcpReportBlock`, `MediaRtcpFeedback`, `MediaSessionSpec`, `MediaDirectPeer`, `MediaGroupSpec`, `MediaCommand`, `MediaGroupControlKind`, `MediaCloseReason`, `MediaSetupError`.
- Produces: the upstream file split and backend seam.

- [ ] **Step 1: Write failing tests** for the split modules, relay parsing, signaling plans, spec building and backend seam.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs-core --features voip-control --test parity_voip_control
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/wacore/src/voip_control/` and `src/voip_control/`. Keep `mod.rs` re-exports so existing imports stay valid.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs-core --features voip-control --test parity_voip_control
cargo check -p wa-rs-core --features voip-control
```

- [ ] **Step 5: Commit** with `feat(voip): split the control boundary`.

### Task G12: VoIP crypto and codecs

**Files:**
- Create: `wacore/src/voip/{e2e_srtp,hbh_srtp,audio,codec_probe,demux,h264,opus_packet}.rs`
- Create: `wacore/src/voip/sframe/`, `wacore/src/voip/mlow/quality_tests.rs`
- Modify: `wacore/src/voip/{mod,srtp,sframe}.rs`, `wacore/src/voip/mlow/mod.rs`
- Test: `wacore/tests/parity_voip_crypto.rs`, `wacore/tests/parity_voip_mlow.rs`, `wacore/tests/mlow_logging.rs`

**Interfaces:**
- Preserves: RTP/RTCP/STUN/SFrame/SRTP/FEC and MLOW/Opus framing.
- Produces: E2E/HBH SRTP, SFrame split, audio codec probing, demux, H.264 and Opus packetization.

- [ ] **Step 1: Write failing vector tests** for E2E/HBH SRTP, SFrame, H.264/Opus packetization, demux and MLOW quality.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs-core --features voip --test parity_voip_crypto
cargo test -p wa-rs-core --features voip-mlow --test parity_voip_mlow --test mlow_logging
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/wacore/src/voip/`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs-core --features voip --test parity_voip_crypto
cargo test -p wa-rs-core --features voip-mlow --test parity_voip_mlow --test mlow_logging
cargo check -p wa-rs-core --features voip-mlow
```

- [ ] **Step 5: Commit** with `feat(voip): port crypto and codec modules`.

### Task G13: VoIP engine, session and group media

**Files:**
- Create: `wacore/src/voip/{driver,session,group_audio,group_media,tap,warp}.rs`
- Modify: `wacore/src/voip/{mod,engine}.rs`
- Test: `wacore/tests/parity_voip_engine.rs`

**Interfaces:**
- Preserves: `CallEngine`, `Driver`, `EngineCommand`, `EngineEvent`, `EngineState`, `Session`, `GroupAudio`, `GroupMedia`.
- Produces: session/driver/group-media/tap/warp behavior.

- [ ] **Step 1: Write failing tests** for driver/session transitions, group audio/media and tap/warp.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs-core --features voip --test parity_voip_engine
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/wacore/src/voip/`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs-core --features voip --test parity_voip_engine
cargo check -p wa-rs-core --features voip
```

- [ ] **Step 5: Commit** with `feat(voip): port engine session and group media`.

### Task G14: VoIP runtime, facade, native relay and calls

**Files:**
- Create: `src/voip/{audio,direct,direct_client,direct_control,direct_ingress,driver,events,facade,registry,session,state,video}.rs`, `src/voip/facade/`, `src/voip/transport/`
- Create: `src/handlers/call/`, `wacore/src/types/group_call.rs`
- Modify: `src/voip/mod.rs`, `src/features/call_log.rs`, `src/handlers/call.rs`, `Cargo.toml`
- Test: `tests/parity_voip_facade.rs`, `tests/parity_calls.rs`, upstream `tests/{voip_call_events,voip_call_fixture,voip_media_seam,voip_neutral_consumer}.rs`

**Interfaces:**
- Preserves: `CallEvents`, `AcceptCall`, `CallState`, `CallHandle`, `OutgoingCall`, `OutgoingGroupCall`, `GroupBoundCall`, `CallLinkCall`, `CallTermination`, `CallError`, `Voip`.
- Produces: direct/group call runtime, facade, native relay seam, call handler and group-call types.

- [ ] **Step 1: Write failing tests** by porting the four upstream VoIP integration tests and extending `parity_calls`.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --features voip-runtime --test parity_voip_facade
cargo test -p wa-rs --features voip --test parity_calls
```

- [ ] **Step 3: Port the runtime/facade/relay** from `reference/whatsapp-rust/src/voip/`, `src/handlers/call/`, and `wacore/src/types/group_call.rs`. Keep the current `src/voip/mod.rs` exports as compatibility wrappers.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --features voip-runtime --test parity_voip_facade
cargo test -p wa-rs --features voip --test parity_calls
cargo test -p wa-rs --features voip
cargo check -p wa-rs --features parity-voip
```

- [ ] **Step 5: Commit** with `feat(voip): port runtime facade relay and call handling`.

### Task G15: Phase 5 deep wiring

**Files:**
- Modify: `src/plugins/{mod,events,iq_reply}.rs`, `src/client/extension_lifecycle.rs`, `src/client/{builder,lifecycle}.rs`, `src/client.rs`
- Modify: `src/client/device_memo_stats.rs`, `src/client/accessors.rs`, `src/client/iq_ops.rs`
- Modify: `wacore/src/store/commands.rs`, `src/handshake.rs` (client profile wiring deferred from G2a)
- Modify: `storages/sqlite-storage/{Cargo.toml,src/}`, `transports/tokio-transport/{src/lib.rs,src/tcp.rs}`
- Create: `tests/handshake_span_scope.rs`
- Test: `tests/parity_plugins.rs`, `tests/parity_observability.rs`, storage wasm check

**Interfaces:**
- Produces: plugin install/callback queue integration, `ClientLifecycle` installation, real `MemoryReport`/`DeviceMemoStats` values, storage wasm shim, transport `StreamDialer`, and `set_client_profile` wiring (deferred from G2a).
- Preserves: existing plugin/lifecycle/observability public types.

- [ ] **Step 1: Write failing tests** for plugin install callbacks, lifecycle installation, real memory report values, dialer and handshake span scope.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --features plugins --test parity_plugins
cargo test -p wa-rs --features tracing,metrics --test parity_observability
cargo test -p wa-rs-tokio-transport --features tcp --test parity_tcp
```

- [ ] **Step 3: Implement the wiring** from `reference/whatsapp-rust/src/plugins/`, `src/client/extension_lifecycle.rs`, `src/client/device_memo_stats.rs`, `storages/sqlite-storage/src/pool.rs`, and `transports/tokio-transport/src/tcp.rs`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --features plugins --test parity_plugins
cargo test -p wa-rs --features tracing,metrics --test parity_observability
cargo test -p wa-rs-tokio-transport --features tcp --test parity_tcp
cargo check -p wa-rs-sqlite-storage --target wasm32-unknown-unknown --no-default-features --features js
```

- [ ] **Step 5: Commit** with `feat(platform): wire plugins lifecycle reports wasm and dialer`.

### Task G16: Final parity gate and report reconciliation

**Files:**
- Modify: `docs/superpowers/plans/2026-11-09-wa-rs-parity/test-parity-report.md`
- Modify: `reference/design/plans/parity-report.md`
- Modify: `reference/design/plans/test-parity-report.md`
- Modify: `.superpowers/sdd/02-phase-1-auth-session/progress.md`

**Interfaces:**
- Produces: a final row-by-row status (`ported`, `non-goal`, `blocked`) and gate evidence.

- [ ] **Step 1: Reconcile every row** in `07-test-parity-matrix.md` with a task above or an explicit non-goal.
- [ ] **Step 2: Run the full gate**

```bash
cargo fmt --all --check
cargo clippy --workspace --all-targets
cargo test --workspace -- --test-threads=4
cargo check --workspace
cargo build -p whatslite-desktop
cargo test -p whatslite-desktop
cargo check -p wa-rs-bridge
cargo test -p wa-rs-bridge
```

- [ ] **Step 3: Run the feature matrix**

```bash
cargo test -p wa-rs --features parity-core
cargo test -p wa-rs --features parity-messaging
cargo test -p wa-rs --features parity-social
cargo test -p wa-rs --features parity-voip
cargo test -p wa-rs --features parity-mobile
cargo test -p wa-rs --features parity-plugins
cargo test -p wa-rs --features parity-observability
cargo check -p wa-rs-sqlite-storage --target wasm32-unknown-unknown --no-default-features --features js
```

- [ ] **Step 4: Update the reports** with exact counts, deferred rows and the live-smoke blocker. Do not mark a row done unless its test passes.
- [ ] **Step 5: Commit** with `docs: close the parity gap report`.

---

## Execution Progress (2026-11-10)

Completed sub-tasks (reviewed, committed):

| Sub-task | Commit(s) | Result |
|---|---|---|
| G1 serde/message refs/context info | `5322027` | Approved; 2+4+7 tests |
| G2a client IQ ops | `ca3a05a` | Approved; 10+10+2 tests |
| G2b config accessors | `557ef39`, `ea2f80a` | Approved after 1 fix; 11 tests |
| G2e-1 subsystem/interceptor/notification scope | `255c69e` | Approved; 74+12 tests |
| G2e-2a member index | `14d8df9` | Approved; 12 tests |
| G2e-3 device topology | `bf0d3d7`, `7c346fc` | Approved after 1 fix; 9 tests |
| G2d-1 offline batch coordinator | `f4bb493` | Approved; 21 tests |
| G8a join_shapes/community_create/profile/devices | `32295b2` | Approved; 9 tests |
| G8b targets/pilots | `91ad36b` | Approved; 6 tests |
| G8c poll primitives | `3578471` | Approved; 19 tests |
| G8d newsletter IQ | `5599459` | Approved; 9 tests |
| G8e bot directory IQ | `2d04338` | Approved; 16 tests |
| G8f business profile IQ | `17b6afe` | Approved; 23 tests |
| G8g contextual IQ catalog | `4f98807` | Approved; 8 tests |
| G9a appstate known verbs | `6ecd2b8` | Approved; 4 tests |
| G9b shared appstate sync engine | `3879359`, `7880016` | Approved after 1 fix; 43 tests |
| G4a pending device sync set | `b9877ec` | Approved; 7 tests |
| G5a download byte budget | `688b987` | Approved; 9 tests |
| G5b download memory writer | `c96aefc`, `ac98715` | Approved after 1 fix; 9 tests |
| G5c media reupload | `f3f1d19`, `a20595e` | Approved after 1 fix; 15 tests |
| G10a presence handler | `09bba55` | Approved; 3 tests |
| G10a-wire presence registration | `1599b95` | Approved; 4 tests |
| G10b privacy/business notifications | `97f5109` | Approved; 11 tests |
| G10c stanza wire tags | `5c0fb4e` | Approved; 4 tests |
| G10d profile notifications + macro | `c9eeb90` | Approved; 24 tests |
| G10e device notifications | `3fad85d`, `6ff845a` | Approved after 1 fix; 18 tests |
| G8h abprops registry | `ad9abdc` | Approved; byte-identical generated file |
| G8i ab props cache | `a1ac57d` | Approved; 11+12 tests |
| G8j client ab props wiring | `8035be7` | Approved; 5 tests |
| G8k usync typed query codec | `8994406` | Approved; 32 tests |
| G8lm mex operation registries | `def2a97` | Approved; byte-identical |
| G8n-1 ab prop watch coverage | `9aca279` | Approved; 2 tests |
| G8n-2 newsletter poll vote alias | `13c5b4b` | Approved; 1 test |
| G10g1 normalized group parser core | `383d351` | Approved; 82 tests |
| G2e-2b group notification facts provider | `f347d85` | Approved; 6 tests |


Blocked sub-tasks (need a prerequisite subsystem before they can be independent):

| Sub-task | Blocker |
|---|---|
| G2c adapters | `signal_cache`, `message_semaphore_generation`, `session_lock_for`, `flush_signal_cache` absent |
| G2e-2b group_notifications | done (`f347d85`); inbound `w:gp2` wiring follows G10 |
| G2d-2 session_state | needs `Client` session-state field/wiring |
| G2f app_state/client tests | ~20k lines, depends on G2/G3/G4 |

Remaining workstreams: G3–G7, G8n-3+ social feature APIs/MEX execution, G10g2 conformance + groups handler, G11–G16. This is a multi-session effort; the plan is still authoritative.

## Execution Order

G1 → G2 → G3 → G4 → G5 → G6 → G7 → G8 → G9 → G10 → G11 → G12 → G13 → G14 → G15 → G16.

G1/G2 are foundational. G3–G6 depend on the client lifecycle. G7–G10 can run after G3/G4. G11–G14 depend on the core VoIP types. G15 closes Phase 5 wiring after the feature work. G16 is the final gate.

Each task above is a **workstream**, not a single commit-sized change. Before executing a workstream, split it into sub-tasks using the file lists and step checkboxes in the owning phase plan; each sub-task must end with a green targeted test and its own commit. The workstream is done only when every file/interface in its Files/Interfaces block exists and its listed tests pass.

## Non-Goals

- Upstream CI, Dockerfiles, oracle/whatspec/unwasm tooling, benchmark harnesses, and generated test-vector tooling are not ported.
- `tests/sqlite-sdk-browser/` remains external; only the wasm32 storage check is in scope.
- Live bridge pairing/connect/send/receive/media/call smoke requires a real WhatsApp account/phone and is not claimed by this plan.
- Excluded features: `bench-harness`, `test-support`, `session-simulator`, `dhat-heap`, `bench-internals`.

## Verification Notes

- Every task ends with a green targeted test and a commit.
- No task may delete an existing test; deferred upstream tests must be recorded in `test-parity-report.md`.
- `bridge/` and `desktop/` are checked at every task that touches shared types or the client.
- Live bridge pairing/send/receive/media/call smoke remains blocked on a real WhatsApp account/phone; do not claim it.
