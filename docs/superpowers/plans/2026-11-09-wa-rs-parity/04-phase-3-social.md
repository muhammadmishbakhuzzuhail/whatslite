# Phase 3 — Social features

> Part of the [wa-rs Parity Plan](README.md). Read the overview first for
> global constraints, the feature contract, crate aliases, the gate protocol,
> and the test parity contract.
>
> Test parity obligations: [Test Parity Matrix](07-test-parity-matrix.md).

**Phase goal:** all 34 upstream `src/features` modules, their IQ domains, notification handlers and event surface.

### Task 3.1: Port social foundations and upgrade existing feature modules

**Files:**
- Modify: `src/features/mod.rs`, `src/lib.rs`, `wacore/src/iq/mex.rs`
- Create: `src/features/creation.rs`
- Create: `src/features/group_history.rs`
- Modify: `src/features/mex.rs`
- Modify: `src/features/tctoken.rs`
- Create: `src/features/stanza.rs`
- Create: `src/features/signal.rs`
- Create: `src/features/rotate_key.rs`
- Modify: `src/features/blocking.rs`
- Modify: `src/features/chatstate.rs`
- Modify: `src/features/contacts.rs`
- Modify: `src/features/presence.rs`
- Test: `tests/parity_social_foundations.rs`, `wacore/tests/parity_iq_social.rs`

**Interfaces:**
- Produces: `creation_types!` and the `CreatedPoll`, `CreatedEvent`, `PollRef`, `EventRef` types.
- Produces: `GroupHistorySkipReason`, `GroupHistoryRetryToken`, `GroupHistoryShareOutcome`.
- Produces: `Mex`, `MexError`, `MexOperation`, `MexRequest`, `MexDoc`, `CappingStatus`.
- Produces: `TcToken`, `TcTokenError`, `TcTokenEntry`, `ReceivedTcToken`.
- Produces: `StanzaRejection`, `StanzaResponseError`, `RetryRequestOptions`, `RetryRequestOutcome`, `MessageRetransmission`.
- Produces: `Signal`, `SignalError`, `SignalSessionInfo`, `SignalSessionMigration`.
- Preserves: existing `Blocking`, `Chatstate`, `Contacts`, `Presence` method names.

- [ ] **Step 1: Write failing tests** for creation helpers, group-history tokens, MEX requests, TCToken lifecycle, stanza rejection mapping, signal sessions and rotate-key.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --test parity_social_foundations
cargo test -p wa-rs-core --test parity_iq_social
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/src/features/{creation,group_history,mex,tctoken,stanza,signal,rotate_key,blocking,chatstate,contacts,presence}.rs` and the matching `wacore/src/iq/` domains.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --test parity_social_foundations
cargo test -p wa-rs-core --test parity_iq_social
cargo check -p wa-rs
```

- [ ] **Step 5: Commit** with `feat(social): port social foundations and upgrade existing features`.

### Task 3.2a: Port group metadata, lookup and creation

**Files:**
- Modify: `src/features/mod.rs`, `src/features/groups.rs` (metadata/lookup/create parts)
- Modify: `wacore/src/iq/groups.rs` (query/create parts)
- Create: `wacore/src/iq/join_shapes.rs`
- Test: `tests/parity_groups.rs`, `tests/group_lookup_contract.rs`

**Interfaces:**
- Produces: `Groups`, `GroupError`, `GroupLookupResult`, `GroupOverview`, `GroupOverviewResult`, `GroupMetadata`, `GroupParticipant`, `GroupCreateOptions`, `CreateGroupResult`, `GroupSubject`, `GroupDescription`.
- Preserves: existing basic `Groups` methods.

- [ ] **Step 1: Port the upstream tests** `tests/group_lookup_contract.rs` and add `tests/parity_groups.rs` for lookup/create/metadata.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --test parity_groups --test group_lookup_contract
```

- [ ] **Step 3: Port the metadata/lookup/create parts** from `reference/whatsapp-rust/src/features/groups.rs` and `wacore/src/iq/groups.rs`; add `join_shapes.rs`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --test parity_groups --test group_lookup_contract
cargo check -p wa-rs
```

- [ ] **Step 5: Commit** with `feat(groups): port group metadata lookup and creation`.

### Task 3.2b: Port group membership, approval and settings

**Files:**
- Modify: `src/features/groups.rs` (membership/settings parts)
- Modify: `wacore/src/iq/groups.rs` (membership/settings parts)
- Test: `tests/parity_groups_membership.rs`

**Interfaces:**
- Produces: `MembershipRequest`, `MembershipApprovalMode`, `MemberAddMode`, `GroupJoinError`, `JoinGroupResult`, `ParticipantChangeResponse`, `GroupEphemeralSettings`, `GroupParticipantOptions`, `GroupParticipantDetails`.
- Preserves: existing join/leave/participant methods.

- [ ] **Step 1: Port the upstream membership tests** from `wacore/src/iq/groups.rs` and add `tests/parity_groups_membership.rs`.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --test parity_groups_membership
```

- [ ] **Step 3: Port the membership/settings parts** from `reference/whatsapp-rust/src/features/groups.rs` and `wacore/src/iq/groups.rs`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --test parity_groups_membership
cargo check -p wa-rs
```

- [ ] **Step 5: Commit** with `feat(groups): port membership approval and settings`.

### Task 3.2c: Port group history, profile pictures and reports

**Files:**
- Modify: `src/features/groups.rs` (history/picture/report parts)
- Create: `src/features/group_history.rs`
- Modify: `wacore/src/iq/groups.rs` (history/picture/report parts)
- Test: `tests/parity_group_history.rs`

**Interfaces:**
- Produces: `GroupHistoryAddResult`, `HistorySharePreparation`, `GroupHistoryRetryToken`, `GroupHistoryShareOutcome`, `GroupProfilePicture`, `GroupPictureEntry`, `ReportedGroupMessage`, `ReportedGroupMessages`.

- [ ] **Step 1: Port the upstream history/picture/report tests** from `src/features/groups.rs` and add `tests/parity_group_history.rs`.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --test parity_group_history
```

- [ ] **Step 3: Port the history/picture/report parts** from `reference/whatsapp-rust/src/features/groups.rs`, `group_history.rs`, and `wacore/src/iq/groups.rs`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --test parity_group_history
cargo check -p wa-rs
```

- [ ] **Step 5: Commit** with `feat(groups): port history profile pictures and reports`.

### Task 3.2d: Port chat actions, labels, favorites and quick replies

**Files:**
- Create: `src/features/chat_actions.rs`
- Create: `src/features/labels.rs`
- Create: `src/features/favorites.rs`
- Create: `src/features/quick_replies.rs`
- Modify: `src/features/mod.rs`
- Test: `tests/parity_chat_actions.rs`, `tests/label_sublist_public_api.rs`, `tests/label_timestamp_public_api.rs`

**Interfaces:**
- Produces: `ChatActions`, `ClearChatOptions`, `StarredMessages`, `MediaDeletion`, `SyncActionMessageRange`, `Labels`, `Favorites`, `QuickReplies`.

- [ ] **Step 1: Port the upstream tests** `tests/label_sublist_public_api.rs`, `tests/label_timestamp_public_api.rs`, and add `tests/parity_chat_actions.rs`.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --test parity_chat_actions --test label_sublist_public_api --test label_timestamp_public_api
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/src/features/chat_actions.rs`, `labels.rs`, `favorites.rs`, `quick_replies.rs`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --test parity_chat_actions --test label_sublist_public_api --test label_timestamp_public_api
cargo check -p wa-rs
```

- [ ] **Step 5: Commit** with `feat(chat): port chat actions labels favorites and quick replies`.

### Task 3.3: Port communities and newsletters

**Files:**
- Modify: `src/features/mod.rs`, `wacore/src/types/newsletter.rs`
- Create: `src/features/community.rs`
- Create: `src/features/newsletter.rs`
- Create: `wacore/src/iq/community_create.rs`
- Create: `wacore/src/iq/newsletter.rs`
- Test: `tests/parity_community.rs`, `tests/parity_newsletter.rs`

**Interfaces:**
- Produces: `Community<'a>`, `CommunityError`, `CreateCommunityOptions`, `CreateCommunityResult`, `CreateSubgroupOptions`, `LinkSubgroupOptions`, `CommunitySubgroup`, `SubgroupFailure`, `SubgroupVisibility`.
- Produces: `Newsletter<'a>`, `NewsletterError`, `NewsletterMetadata`, `NewsletterMessage`, `NewsletterAdminInfo`, `NewsletterRole`, `NewsletterState`.

- [ ] **Step 1: Write failing tests** for community create/deactivate, subgroup create/link/unlink, newsletter list/get/create/join/leave/update/admin/messages.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --test parity_community --test parity_newsletter
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/src/features/community.rs`, `newsletter.rs` and the matching IQ domains.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --test parity_community --test parity_newsletter
cargo check -p wa-rs
```

- [ ] **Step 5: Commit** with `feat(social): port communities and newsletters`.

### Task 3.4: Port status, polls and events

**Files:**
- Modify: `src/features/mod.rs`
- Create: `src/features/status.rs`
- Create: `src/features/polls.rs`
- Create: `src/features/events.rs`
- Create: `wacore/src/poll.rs`
- Test: `tests/parity_status_polls_events.rs`

**Interfaces:**
- Produces: `Status`, `StatusSendOptions`, `StatusPrivacySetting`.
- Produces: `Polls`, `PollError`, `CreatedPoll`, `PollRef`, `PollOptionResult`, `PollVoteCiphertext`.
- Produces: `Events`, `EventCreationParams`, `CreatedEvent`, `EventRef`, `EventResponseType`.

- [ ] **Step 1: Write failing tests** for status send/revoke/privacy, poll create/vote/decrypt/aggregate, and event create/respond/cancel.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --test parity_status_polls_events
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/src/features/status.rs`, `polls.rs`, `events.rs`, `creation.rs` and `wacore/src/poll.rs`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --test parity_status_polls_events
cargo check -p wa-rs
```

- [ ] **Step 5: Commit** with `feat(social): port status polls and events`.

### Task 3.5: Port stickers, business, bots, comments, pictures, profile and app state

**Files:**
- Modify: `src/features/mod.rs`
- Create: `src/features/{picture_mutation_tests.rs,pictures_metrics_tests.rs,pictures_tests.rs}`
- Create: `src/features/stickers.rs`
- Create: `src/features/business.rs`
- Create: `src/features/bots.rs`
- Create: `src/features/comments.rs`
- Create: `src/features/pictures.rs`
- Create: `src/features/profile.rs`
- Create: `src/features/app_state_settings.rs`
- Create: `src/features/app_state_resync.rs`
- Modify: `src/appstate_sync.rs`
- Create: `src/appstate_known_verbs.rs`
- Create: `wacore/src/appstate_sync.rs`, `wacore/src/appstate_sync/`
- Create: `wacore/src/iq/{bot,business,profile}.rs`
- Test: `tests/parity_social_misc.rs`, `tests/parity_appstate.rs`

**Interfaces:**
- Produces: `Stickers`, `Business`, `Catalog`, `Collection`, `Order`, `Bots`, `Comments`, `Pictures`, `Profile`.
- Produces: `AppStateSettings`, `AppStateResyncMode`, `AppStateResyncReport`.
- Preserves: existing contact/profile picture calls.

- [ ] **Step 1: Write failing tests** for catalog/order, bot list, comments, profile picture, text status, app-state settings/resync and app-state sync.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --test parity_social_misc --test parity_appstate
```

- [ ] **Step 3: Port the modules** from `reference/whatsapp-rust/src/features/{stickers,business,bots,comments,pictures,profile,app_state_settings,app_state_resync}.rs`, `src/appstate_sync.rs`, `wacore/src/appstate_sync.rs`, `wacore/src/appstate_sync/`, and the matching IQ domains. Declare `picture_mutation_tests`, `pictures_metrics_tests`, and `pictures_tests` under `#[cfg(test)]` in `src/features/mod.rs`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --test parity_social_misc --test parity_appstate
cargo check -p wa-rs
```

- [ ] **Step 5: Commit** with `feat(social): port stickers business bots comments pictures profile and app state`.

### Task 3.6a: Port the existing and core IQ domains

**Files:**
- Modify: `wacore/src/iq/mod.rs`
- Modify: `wacore/src/iq/{blocklist,chatstate,contacts,dirty,keepalive,mediaconn,node,passive,prekeys,privacy,props,spam_report,spec,tctoken}.rs`
- Create: `wacore/src/iq/{abprops,contextual,devices}.rs`
- Test: extend `wacore/tests/parity_iq_social.rs`; port `wacore/tests/abprops_retained.rs`, `wacore/tests/props_stream_memory.rs`, `wacore/tests/iq_pilots.rs`, `tests/ab_prop_watch_coverage.rs`, `tests/iq_parse_errors.rs`

**Interfaces:**
- Produces: the core IQ specs and parsers (`abprops`, `devices`, `privacy`, `props`, `spec`, `blocklist`, `chatstate`, `contacts`, `dirty`, `keepalive`, `mediaconn`, `node`, `passive`, `prekeys`, `spam_report`, `tctoken`).

- [ ] **Step 1: Port the upstream tests** listed above and extend `wacore/tests/parity_iq_social.rs`.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs-core --test parity_iq_social --test abprops_retained --test props_stream_memory --test iq_pilots
cargo test -p wa-rs --test ab_prop_watch_coverage --test iq_parse_errors
```

- [ ] **Step 3: Port the core IQ domains** from `reference/whatsapp-rust/wacore/src/iq/`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs-core --test parity_iq_social --test abprops_retained --test props_stream_memory --test iq_pilots
cargo test -p wa-rs --test ab_prop_watch_coverage --test iq_parse_errors
```

- [ ] **Step 5: Commit** with `feat(iq): port core and existing IQ domains`.

### Task 3.6b: Port bot, business, profile, newsletter, community and USync IQ

**Files:**
- Create: `wacore/src/iq/{bot,business,profile,newsletter,community_create,join_shapes}.rs`
- Modify: `wacore/src/iq/usync.rs`
- Create: `wacore/src/iq/usync/`
- Modify: `wacore/src/iq/mod.rs`
- Test: extend `wacore/tests/parity_iq_social.rs`; port `tests/newsletter_public_api.rs`, `tests/iq_parse_errors.rs`

**Interfaces:**
- Produces: bot, business, profile, newsletter, community and USync IQ specs and parsers.

- [ ] **Step 1: Port the upstream tests** listed above and extend `wacore/tests/parity_iq_social.rs`.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs-core --test parity_iq_social
cargo test -p wa-rs --test newsletter_public_api --test iq_parse_errors
```

- [ ] **Step 3: Port the IQ domains** from `reference/whatsapp-rust/wacore/src/iq/`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs-core --test parity_iq_social
cargo test -p wa-rs --test newsletter_public_api --test iq_parse_errors
```

- [ ] **Step 5: Commit** with `feat(iq): port bot business profile newsletter community and usync`.

### Task 3.6c: Port MEX, pilots and target IQ

**Files:**
- Modify: `wacore/src/iq/mex.rs`
- Create: `wacore/src/iq/{mex_operations,mex_operations_unlisted,pilots,targets}.rs`
- Modify: `wacore/src/iq/mod.rs`
- Test: extend `wacore/tests/parity_iq_social.rs`; port `wacore/tests/mex_retained.rs`, `tests/mex_public_api.rs`, `tests/reachout_timelock_public_api.rs`, `wacore/tests/iq_pilots.rs`

**Interfaces:**
- Produces: MEX operations, pilots and target IQ specs and parsers.

- [ ] **Step 1: Port the upstream tests** listed above and extend `wacore/tests/parity_iq_social.rs`.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs-core --test parity_iq_social --test mex_retained --test iq_pilots
cargo test -p wa-rs --test mex_public_api --test reachout_timelock_public_api
```

- [ ] **Step 3: Port the IQ domains** from `reference/whatsapp-rust/wacore/src/iq/`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs-core --test parity_iq_social --test mex_retained --test iq_pilots
cargo test -p wa-rs --test mex_public_api --test reachout_timelock_public_api
```

- [ ] **Step 5: Commit** with `feat(iq): port mex pilots and target IQ`.

### Task 3.6d: Port handlers and notification coverage

**Files:**
- Modify: `src/handlers/{basic.rs,chatstate.rs,ib.rs,iq.rs,message.rs,receipt.rs,router.rs,traits.rs}`
- Create: `src/handlers/presence.rs`
- Modify: `src/handlers/mod.rs`
- Modify: `src/handlers/notification/{mod.rs,companion_reg.rs}`
- Create: `src/handlers/notification/{device.rs,groups.rs,privacy_business.rs,profile.rs,reachout_tests.rs}`
- Test: create `tests/parity_handlers.rs`

**Interfaces:**
- Produces: every upstream handler and notification type.
- Produces: 80-variant event surface exercised through handlers.
- Preserves: existing handler entry points used by `bridge`.

- [ ] **Step 1: Port the upstream handler tests and add `tests/parity_handlers.rs`.
- [ ] **Step 2: Run and verify failure**

```bash
cargo test -p wa-rs --test parity_handlers
```

- [ ] **Step 3: Port the handler and notification modules.** Declare `reachout_tests` under `#[cfg(test)]` in `src/handlers/notification/mod.rs`.
- [ ] **Step 4: Verify**

```bash
cargo test -p wa-rs --test parity_handlers
cargo check -p wa-rs-bridge
```

- [ ] **Step 5: Commit** with `feat(iq): port handlers and notification coverage`.

### Phase 3 Gate

- [x] `cargo test -p wa-rs --test parity_social_foundations --test parity_groups --test parity_chat_actions`
- [x] `cargo test -p wa-rs --test parity_groups_membership --test parity_group_history --test label_sublist_public_api --test label_timestamp_public_api`
- [x] `cargo test -p wa-rs --test parity_community --test parity_newsletter --test parity_status_polls_events`
- [x] `cargo test -p wa-rs --test parity_social_misc --test parity_appstate --test parity_handlers`
- [x] `cargo test -p wa-rs-core --test parity_iq_social --test abprops_retained --test props_stream_memory --test iq_pilots --test mex_retained`
- [x] `cargo test -p wa-rs`
- [x] `cargo test -p wa-rs-core`
- [x] `cargo check -p wa-rs --features parity-social`
- [x] `cargo test -p wa-rs-bridge`
- [x] `cargo test -p whatslite-desktop`
- [x] `cargo build -p whatslite-desktop`
- [x] Event surface test: 80 `EventKind` and 80 `Event` variants (covered by `parity_events` and `parity_handlers`).

---

Previous: [Phase 2](03-phase-2-messaging-media.md) | Next: [Phase 4](05-phase-4-voip.md)
