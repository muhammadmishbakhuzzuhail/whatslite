# Phase 4 — Calls and VoIP

> Part of the [wa-rs Parity Plan](README.md). Read the overview first for
> global constraints, the feature contract, crate aliases, the gate protocol,
> and the test parity contract.
>
> Test parity obligations: [Test Parity Matrix](07-test-parity-matrix.md).

**Phase goal:** upstream call control and media stack, including the runtime, engine, relay and codec features.

### Task 4.1: Port the VoIP control boundary

**Files:**
- Create: `wacore/src/voip_control/{mod.rs,app_data.rs,audio_format.rs,control.rs,engine_bridge.rs,events.rs,fake_backend.rs,group.rs,kdf.rs,media_stats.rs,ports.rs,registry.rs,relay_parse.rs,resident_session.rs,signaling.rs,signaling_plan.rs,spec_build.rs,ssrc.rs,transport.rs}`
- Create: `src/voip_control/{mod.rs,wacore_backend.rs}`
- Modify: `Cargo.toml` (`voip-control`)
- Modify: `wacore/Cargo.toml` (`voip-control`)
- Test: `wacore/tests/parity_voip_control.rs`

**Interfaces:**
- Produces: `GroupControl`, `NormalizedGroupControl`, `GroupRawEpoch`, `VideoControl`, `VideoControlSender`, `VideoControlReceiver`, `PeerAnswer`.
- Produces: `CallEvent`, `MediaGroupEpoch`, `MediaSessionKey`, `MediaAudioSpec`, `MediaKeyframeUrgency`, `VideoFecConfig`, `MediaVideoUpgradeToken`, `MediaGroupTransition`, `MediaCodecDecisionSource`, `MediaSilenceReason`, `MediaEncodedFrame`, `MediaRtcpReportBlock`, `MediaRtcpFeedback`, `MediaSessionSpec`, `MediaDirectPeer`, `MediaGroupSpec`, `MediaCommand`, `MediaGroupControlKind`, `MediaCloseReason`, `MediaSetupError`.
- Produces: `AudioSource`, `AudioSink`, `EncodedAudioSource`, `EncodedAudioSink`, `VideoFrame`, `TimedVideoFrame`, `VideoInput`.
- Gate: `voip-control`.

- [ ] **Step 1: Write failing tests** for call registry, control-plane boundary, group control normalization, media setup errors and relay parsing.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs-core --features voip-control --test parity_voip_control
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/wacore/src/voip_control/` and `src/voip_control/`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs-core --features voip-control --test parity_voip_control
cargo check -p wa-rs-core --features voip-control
cargo check -p wa-rs --features voip-control
```

- [ ] **Step 5: Commit** with `feat(voip): port control boundary`.

### Task 4.2a: Port RTP/RTCP, SRTP, SFrame, STUN and FEC

**Files:**
- Create: `wacore/src/voip/{mod.rs,rtp.rs,rtcp.rs,e2e_srtp.rs,hbh_srtp.rs,sframe.rs,stun.rs,fec.rs}`
- Create: `wacore/src/voip/sframe/`
- Modify: `Cargo.toml` (`voip`), `wacore/Cargo.toml` (`voip`)
- Test: `wacore/tests/parity_voip_crypto.rs`

**Interfaces:**
- Produces: SRTP, SFrame, WAHKDF, RTP/RTCP, STUN and FEC primitives.
- Gate: `voip`.

- [ ] **Step 1: Write failing vector tests** for SRTP/SFrame/WAHKDF and add `wacore/tests/parity_voip_crypto.rs`.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs-core --features voip --test parity_voip_crypto
```

- [ ] **Step 3: Port the crypto and transport primitives** from `reference/whatsapp-rust/wacore/src/voip/`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs-core --features voip --test parity_voip_crypto
cargo check -p wa-rs-core --features voip
```

- [ ] **Step 5: Commit** with `feat(voip): port rtp srtp sframe stun and fec`.

### Task 4.2b: Port MLOW, Opus, H.264 and audio/video framing

**Files:**
- Create: `wacore/src/voip/{audio.rs,codec_probe.rs,demux.rs,h264.rs,opus_packet.rs}`
- Create: `wacore/src/voip/mlow/`
- Modify: `Cargo.toml` (`voip-mlow`, `voip-libopus`, `voip-encoded`)
- Modify: `wacore/Cargo.toml` (`voip-mlow`)
- Test: `wacore/tests/parity_voip_mlow.rs`, `wacore/tests/mlow_logging.rs`, `wacore/src/voip/mlow/quality_tests.rs`

**Interfaces:**
- Produces: `AudioConfig`, `OpusMlowPacketError`, `depacketize_opus_from_mlow`, `packetize_opus_for_mlow`, H.264 frame boundaries.
- Gate: `voip-mlow`, `voip-libopus`, `voip-encoded`.

- [ ] **Step 1: Port the upstream tests** `wacore/tests/mlow_logging.rs`, `wacore/src/voip/mlow/quality_tests.rs`, and add `wacore/tests/parity_voip_mlow.rs`.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs-core --features voip-mlow --test parity_voip_mlow --test mlow_logging
```

- [ ] **Step 3: Port the codec and framing modules** from `reference/whatsapp-rust/wacore/src/voip/` and `mlow/`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs-core --features voip-mlow --test parity_voip_mlow --test mlow_logging
cargo check -p wa-rs-core --features voip-mlow
```

- [ ] **Step 5: Commit** with `feat(voip): port mlow opus h264 and audio video framing`.

### Task 4.2c: Port the engine, driver, session and group media

**Files:**
- Create: `wacore/src/voip/{engine.rs,driver.rs,session.rs,group_audio.rs,group_media.rs,tap.rs,warp.rs}`
- Modify: `Cargo.toml` (`voip-engine-wacore`)
- Test: `wacore/tests/parity_voip_engine.rs`

**Interfaces:**
- Produces: the sans-IO engine, driver, session, group audio/media and relay session traits.
- Gate: `voip-engine-wacore`.

- [ ] **Step 1: Port the upstream engine/session tests** from `wacore/src/voip/` and add `wacore/tests/parity_voip_engine.rs`.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs-core --features voip --test parity_voip_engine
```

- [ ] **Step 3: Port the engine modules** from `reference/whatsapp-rust/wacore/src/voip/`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs-core --features voip --test parity_voip_engine
cargo check -p wa-rs-core --features voip
```

- [ ] **Step 5: Commit** with `feat(voip): port engine driver session and group media`.

### Task 4.3: Port the VoIP runtime, facade and native relay

**Files:**
- Create: `src/voip/{mod.rs,audio.rs,direct.rs,direct_client.rs,direct_control.rs,direct_ingress.rs,driver.rs,events.rs,registry.rs,session.rs,state.rs,video.rs}`
- Create: `src/voip/facade/`, `src/voip/facade.rs`, `src/voip/transport/`
- Modify: `Cargo.toml` (`voip-runtime`, `voip-engine-wacore`, `voip-relay-native`)
- Test: `tests/parity_voip_facade.rs`

**Interfaces:**
- Produces: `CallEvents`, `AcceptCall`, `CallHandle`, `CallLinkCall`, `CallTermination`, `GroupBoundCall`, `OutgoingCall`, `OutgoingGroupCall`, `VIDEO_UPGRADE_TIMEOUT`.
- Produces: `TimedVideoFrame`, `VideoFrame`, `VideoSink`, `VideoSource`, `AudioSink`, `AudioSource`, `EncodedAudioSink`, `EncodedAudioSource`.
- Produces: `CallError`, `Voip`.
- Gate: `voip-runtime`, `voip-engine-wacore`, `voip-relay-native`.

- [ ] **Step 1: Write failing tests** for outgoing/incoming calls, call links, group calls, screen sharing, video upgrade and call termination.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --features voip-runtime --test parity_voip_facade
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/src/voip/`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --features voip-runtime --test parity_voip_facade
cargo check -p wa-rs --features voip
cargo check -p wa-rs --features voip-encoded
cargo check -p wa-rs --features voip-libopus
```

- [ ] **Step 5: Commit** with `feat(voip): port runtime facade and native relay`.

### Task 4.4: Port call handlers, call log and call types

**Files:**
- Create: `src/handlers/call.rs`, `src/handlers/call/`
- Create: `src/features/call_log.rs`
- Modify: `wacore/src/types/call.rs`
- Create: `wacore/src/types/group_call.rs`
- Modify: `wacore/src/types/mod.rs`
- Test: `tests/parity_calls.rs`

**Interfaces:**
- Produces: incoming/missed/ended call events, call log, call links and screen-sharing events.
- Produces: `CallLog`, `GroupCallUpdate`, `CallEvent`.

- [ ] **Step 1: Write failing tests** for incoming call, group call, call ended elsewhere, call log and call link.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --features voip --test parity_calls
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/src/handlers/call.rs`, `src/handlers/call/`, `src/features/call_log.rs`, and `wacore/src/types/call.rs`, `group_call.rs`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --features voip --test parity_calls
cargo check -p wa-rs --features voip
cargo check -p wa-rs-bridge
```

- [ ] **Step 5: Commit** with `feat(calls): port handlers call log and call types`.

### Phase 4 Gate

- [ ] `cargo test -p wa-rs-core --features voip-control --test parity_voip_control`
- [ ] `cargo test -p wa-rs-core --features voip --test parity_voip_crypto`
- [ ] `cargo test -p wa-rs-core --features voip --test parity_voip_engine`
- [ ] `cargo test -p wa-rs-core --features voip-mlow --test parity_voip_mlow --test mlow_logging`
- [ ] `cargo test -p wa-rs --features voip-runtime --test parity_voip_facade`
- [ ] `cargo test -p wa-rs --features voip --test parity_calls`
- [ ] `cargo test -p wa-rs --features voip`
- [ ] `cargo test -p wa-rs-core --features voip-control`
- [ ] `cargo test -p wa-rs-core --features voip`
- [ ] `cargo test -p wa-rs-core --features voip-mlow`
- [ ] `cargo check -p wa-rs --features parity-voip`
- [ ] `cargo check -p wa-rs --features voip-encoded`
- [ ] `cargo check -p wa-rs --features voip-libopus`
- [ ] `cargo test -p wa-rs-bridge`
- [ ] `cargo test -p whatslite-desktop`
- [ ] `cargo build -p whatslite-desktop`

---

Previous: [Phase 3](04-phase-3-social.md) | Next: [Phase 5](06-phase-5-platform.md)
