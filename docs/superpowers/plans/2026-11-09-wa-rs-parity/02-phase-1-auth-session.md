# Phase 1 — Auth, session and recovery

> Part of the [wa-rs Parity Plan](README.md). Read the overview first for
> global constraints, the feature contract, crate aliases, the gate protocol,
> and the test parity contract.
>
> Test parity obligations: [Test Parity Matrix](07-test-parity-matrix.md).

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
- Create: `src/mobile/{primary.rs,registration.rs,registration_tests.rs}`
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
- [ ] `cargo test -p wa-rs`
- [ ] `cargo test -p wa-rs --features passkey`
- [ ] `cargo test -p wa-rs --features mobile-core`
- [ ] `cargo check -p wa-rs --features parity-core`
- [ ] `cargo check -p wa-rs --features parity-mobile`
- [ ] `cargo test -p wa-rs-bridge`
- [ ] `cargo test -p whatslite-desktop`
- [ ] `cargo build -p whatslite-desktop`
- [ ] Bridge pairing and reconnect smoke test

---

Previous: [Phase 0](01-phase-0-foundation.md) | Next: [Phase 2](03-phase-2-messaging-media.md)
