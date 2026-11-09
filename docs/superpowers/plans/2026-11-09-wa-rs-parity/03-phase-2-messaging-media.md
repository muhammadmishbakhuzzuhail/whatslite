# Phase 2 — Messaging, media and reliability

> Part of the [wa-rs Parity Plan](README.md). Read the overview first for
> global constraints, the feature contract, crate aliases, the gate protocol,
> and the test parity contract.
>
> Test parity obligations: [Test Parity Matrix](07-test-parity-matrix.md).

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
- Create: `src/handlers/notification/durability_tests.rs`
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

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/src/message/`, `src/inbound_queue.rs`, `src/flush_scope.rs`, `src/signal_flush.rs`, and `src/msg_secret_buffer.rs`. Declare `durability_tests` under `#[cfg(test)]` in `src/handlers/notification/mod.rs`.
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

### Task 2.6a: Port protocol, XML, version and serde helpers

**Files:**
- Modify: `wacore/src/net.rs`, `wacore/src/version.rs`, `wacore/src/xml.rs`, `wacore/src/proto_helpers.rs`, `wacore/src/stanza/`
- Create: `wacore/src/serde_helpers.rs`
- Replace: `wacore/src/protocol.rs` with `wacore/src/protocol/`
- Test: `wacore/tests/parity_protocol.rs`, `wacore/tests/binary_protocol_test.rs`, `wacore/binary/tests/*`

**Interfaces:**
- Produces: `Protocol`, `Xml`, `Version`, `Stanza`, `ProtoHelpers`.
- Preserves: existing `wacore::protocol`, `wacore::xml`, `wacore::version` APIs.

- [ ] **Step 1: Port the upstream tests** `wacore/tests/binary_protocol_test.rs`, `wacore/binary/tests/*`, and add `wacore/tests/parity_protocol.rs`.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs-core --test parity_protocol --test binary_protocol_test
cargo test -p wa-rs-binary
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/wacore/src/net.rs`, `version.rs`, `xml.rs`, `proto_helpers.rs`, `stanza/`, `protocol/`, and `serde_helpers.rs`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs-core --test parity_protocol --test binary_protocol_test
cargo test -p wa-rs-binary
cargo check -p wa-rs-core
```

- [ ] **Step 5: Commit** with `feat(core): port protocol xml version and serde helpers`.

### Task 2.6b: Port handshake, transport, socket and store

**Files:**
- Modify: `wacore/src/handshake/`, `src/handshake.rs`, `src/http.rs`, `src/mediaconn.rs`, `src/socket/`, `src/store/`, `src/version.rs`
- Test: `tests/parity_transport.rs`, `wacore/tests/noise_handshake_test.rs`, `tests/handshake_integration.rs`, `tests/noise_frame_size_contract.rs`, `src/store/release_tests.rs`

**Interfaces:**
- Produces: `NoiseHandshake`, `NoiseError`, `HandshakeError`, `Transport`, `TransportFactory`, `HttpClient`, `MediaConn`, `PersistenceManager`, `DeviceCommand`.
- Preserves: existing `wa_rs::handshake`, `wa_rs::http`, `wa_rs::mediaconn`, `wa_rs::store` APIs.

- [ ] **Step 1: Port the upstream tests** `wacore/tests/noise_handshake_test.rs`, `tests/handshake_integration.rs`, `tests/noise_frame_size_contract.rs`, `src/store/release_tests.rs`, and add `tests/parity_transport.rs`.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --test parity_transport --test handshake_integration --test noise_frame_size_contract
cargo test -p wa-rs-core --test noise_handshake_test
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/wacore/src/handshake/`, `src/handshake.rs`, `http.rs`, `mediaconn.rs`, `socket/`, `store/`, and `version.rs`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --test parity_transport --test handshake_integration --test noise_frame_size_contract
cargo test -p wa-rs-core --test noise_handshake_test
cargo check -p wa-rs
```

- [ ] **Step 5: Commit** with `feat(core): port handshake transport socket and store`.

### Task 2.6c: Port types, message refs, prekeys, request/usync/history sync and root utilities

**Files:**
- Modify: `wacore/src/ib.rs`, `wacore/src/pair.rs`, `wacore/src/pair_code.rs`, `wacore/src/prekeys.rs`, `wacore/src/request.rs`, `wacore/src/send.rs`, `wacore/src/usync.rs`, `wacore/src/history_sync.rs`
- Modify: `wacore/src/types/{jid.rs,lid_pn.rs,message.rs,mod.rs,presence.rs,spam_report.rs,user.rs}`
- Create: `wacore/src/types/{message_ref.rs,message_secret.rs,wire_enums.rs}`, `wacore/src/types/message_ref/`
- Modify: `src/jid_utils.rs`, `src/spam_report.rs`, `src/sync_task.rs`, `src/unified_session.rs`
- Create: `src/retry_conformance_tests.rs`
- Test: `wacore/tests/{context_info,jid_test,stored_message_secret,usync_response_fixture,usync_stream_response}.rs`, `src/usync/*_tests.rs`, `src/retry_conformance_tests.rs`, `src/request_context_tests.rs`, `wacore/src/types/message_ref/tests.rs`

**Interfaces:**
- Produces: `MessageRef`, `MessageSecret`, `UsyncQuery`, `UsyncProtocol`, `JidUtils`, `SpamFlow`, `SyncTask`, `UnifiedSession`.
- Preserves: existing `wa_rs::usync`, `wa_rs::prekeys`, `wa_rs::request`, `wa_rs::unified_session` APIs.

- [ ] **Step 1: Port the upstream tests** listed above and add `tests/parity_types.rs`.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs-core --test context_info --test jid_test --test stored_message_secret --test usync_response_fixture --test usync_stream_response
cargo test -p wa-rs --test parity_types
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/wacore/src/` and the matching root `src/` files.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs-core --test context_info --test jid_test --test stored_message_secret --test usync_response_fixture --test usync_stream_response
cargo test -p wa-rs --test parity_types
cargo check -p wa-rs
```

- [ ] **Step 5: Commit** with `feat(core): port types message refs prekeys usync and history sync`.

### Phase 2 Gate

- [x] `cargo test -p wa-rs-core --test parity_messaging --test parity_secrets`
- [x] `cargo test -p wa-rs --test parity_inbound --test parity_send --test parity_cache --test parity_media`
- [x] `cargo test -p wa-rs --test parity_reliability --test parity_pdo --test parity_history`
- [x] `cargo test -p wa-rs --test parity_protocol --test parity_transport --test parity_types`
- [x] `cargo test -p wa-rs-binary`
- [x] `cargo test -p wa-rs-core --test noise_handshake_test`
- [x] `cargo test -p wa-rs`
- [x] `cargo test -p wa-rs-core`
- [x] `cargo check -p wa-rs-core`
- [x] `cargo check -p wa-rs --features parity-messaging`
- [x] `cargo test -p wa-rs-bridge`
- [x] `cargo test -p whatslite-desktop`
- [x] `cargo build -p whatslite-desktop`
- [ ] Bridge send/receive/media smoke test — blocked on a live WhatsApp account/phone; HTTP router smoke is green.

---

Previous: [Phase 1](02-phase-1-auth-session.md) | Next: [Phase 3](04-phase-3-social.md)
