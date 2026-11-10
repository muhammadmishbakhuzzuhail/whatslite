//! Device and encrypt notification handlers.
//!
//! Ported from upstream `src/handlers/notification/device.rs`, adapted to this
//! fork's `&Node` handler style. Upstream uses `NodeRef<'_>`; this fork passes
//! owned `Node`s by reference. Log fields that upstream renders through
//! `ObservedId`/`ObservedJid` are reduced to non-identifying summaries here
//! (this fork has no `ObservedId` type and no log capture test helper).

use crate::client::Client;
use crate::lid_pn_cache::LearningSource;
use crate::types::events::Event;
use log::{debug, info, warn};
use std::sync::Arc;
use wacore::stanza::devices::{DeviceNotification, DeviceNotificationType};
use wacore::store::traits::DeviceListRecord;
use wacore::types::events::{DeviceListUpdate, DeviceNotificationInfo, EventKind, IdentityChange};
use wacore::types::jid::JidExt;
use wacore_binary::jid::{Jid, SERVER_JID};
use wacore_binary::node::Node;

/// What an `<notification type="encrypt">` stanza asks this client to do.
#[derive(Debug, PartialEq, Eq)]
pub(crate) enum EncryptTarget<'a> {
    /// `<identity/>` child: a peer reinstalled and its identity changed.
    Identity,
    /// `<count/>` from the server: the one-time prekey pool is low.
    PrekeyLow,
    /// `<pq_count/>` only: this client uploads no Kyber prekeys.
    PqCountOnly,
    /// `<digest/>` child: validate the server key bundle.
    Digest,
    /// A server child this client does not handle.
    Unhandled(&'a str),
    /// Not from the server or without a known child.
    NotServer,
}

/// Route on child tags, not position.
///
/// WA Web's stanza router sends both `count` and `pq_count` to
/// `WAWebHandlePreKeyLow`, which then decides by tag: `count` drives the
/// classic one-time prekey upload and `pq_count` a separate Kyber upload.
/// Matching only the first child dropped the whole notification when the
/// server put `<pq_count>` first, so `<count>` never refilled the pool.
pub(crate) fn classify_encrypt_notification(node: &Node) -> EncryptTarget<'_> {
    if node.get_optional_child("identity").is_some() {
        return EncryptTarget::Identity;
    }
    if node.attrs().optional_string("from") != Some(SERVER_JID) {
        return EncryptTarget::NotServer;
    }
    match node
        .children()
        .and_then(|children| children.first().map(|child| child.tag.as_str()))
    {
        Some("count" | "pq_count") if node.get_optional_child("count").is_some() => {
            EncryptTarget::PrekeyLow
        }
        Some("count" | "pq_count") => EncryptTarget::PqCountOnly,
        Some("digest") => EncryptTarget::Digest,
        Some(other) => EncryptTarget::Unhandled(other),
        None => EncryptTarget::Unhandled(""),
    }
}

pub(crate) async fn handle_encrypt_notification(client: &Arc<Client>, node: &Node) {
    match classify_encrypt_notification(node) {
        EncryptTarget::Identity => handle_identity_change(client, node).await,
        EncryptTarget::PrekeyLow => handle_prekey_low(client).await,
        EncryptTarget::PqCountOnly => {
            // PQ-only: this client uploads no Kyber prekeys, so there is
            // nothing to refill. The router still acks the stanza, matching WA
            // Web when its own PQ gate is off.
            debug!("encrypt notification carried only <pq_count>; no PQ prekeys to upload");
        }
        EncryptTarget::Digest => handle_digest_key(client),
        EncryptTarget::Unhandled(tag) => warn!("Unhandled encrypt notification child: {tag:?}"),
        EncryptTarget::NotServer => {}
    }
}

/// Handle encrypt/count notification (PreKey Low).
///
/// Matches WA Web's `WAWebHandlePreKeyLow`: wait for offline delivery to
/// complete, then top the one-time prekey pool back up.
///
/// Upstream additionally clears a persisted `server_has_prekeys` flag and
/// serializes uploads through `prekey_upload_lock`; neither exists in this
/// fork, so there is no dedup gate here and the upload path is shared with
/// login.
pub(crate) async fn handle_prekey_low(client: &Arc<Client>) {
    let client_clone = client.clone();
    tokio::spawn(async move {
        // Wait for offline delivery first (WA Web's waitForOfflineDeliveryEnd).
        client_clone.wait_for_offline_delivery_end().await;

        if !client_clone.is_logged_in() {
            debug!("Pre-key upload skipped: disconnected during offline delivery wait");
            return;
        }

        if let Err(e) = client_clone.upload_pre_keys().await {
            warn!("Failed to upload pre-keys after prekey_low notification: {e:?}");
        }
    });
}

/// Handle encrypt/digest notification (Digest Key validation).
///
/// Upstream `validate_digest_key` compares the server's digest response
/// against locally stored key material and re-uploads on 404. This fork's
/// `DigestKeyBundleResponse` only carries the digest bytes (no registration
/// id or prekey ids), so validation cannot be implemented yet; send the
/// digest query without interpreting the answer.
pub(crate) fn handle_digest_key(client: &Arc<Client>) {
    let client_clone = client.clone();
    tokio::spawn(async move {
        if let Err(e) = client_clone.send_digest_key_bundle().await {
            warn!("Digest key validation failed: {e:?}");
        }
    });
}

/// Load a stored device-list record for `user`, trying both LID and PN keys.
async fn load_device_record(client: &Client, user: &str) -> Option<DeviceListRecord> {
    let backend = client.persistence_manager.backend();
    for key in client.get_lookup_keys(user).await {
        match backend.get_devices(&key).await {
            Ok(Some(record)) => return Some(record),
            Ok(None) => {}
            Err(e) => warn!("Failed reading device record for a known user: {e}"),
        }
    }
    None
}

/// Handle identity change notification (user reinstalled WhatsApp).
///
/// Matches WA Web's `WAWebHandleIdentityChange`:
/// ```xml
/// <notification type="encrypt" from="user@s.whatsapp.net">
///   <identity/>
/// </notification>
/// ```
///
/// WA Web defers this when offline. We process immediately because all cleanup
/// is local-only, and `ensure_e2e_sessions` self-defers via
/// `wait_for_offline_delivery_end`.
pub(crate) async fn handle_identity_change(client: &Arc<Client>, node: &Node) {
    let Some(from_jid) = node.attrs().optional_jid("from") else {
        warn!("Identity change notification missing 'from' attribute");
        return;
    };

    // Only primary device identity changes matter.
    if from_jid.device != 0 {
        debug!("Ignoring identity change from a companion device");
        return;
    }

    // Self-identity changes use a different flow; clearing our own record would
    // break sessions.
    let device_snapshot = client.persistence_manager.get_device_snapshot().await;
    let is_me = device_snapshot
        .pn
        .as_ref()
        .is_some_and(|pn| pn.user == from_jid.user)
        || device_snapshot
            .lid
            .as_ref()
            .is_some_and(|lid| lid.user == from_jid.user);
    if is_me {
        debug!("Ignoring self-primary identity change");
        return;
    }

    // Always run the device-list cleanup, matching WA Web's
    // clearDeviceRecordForIdentityChange (which runs before the had-prior
    // identity gate): drop companion device sessions and force a fresh usync.
    if let Some(record) = load_device_record(client, &from_jid.user).await {
        let backend = client.persistence_manager.backend();
        for device in record.devices.iter().filter(|d| d.device_id() != 0) {
            let Ok(device_id) = u16::try_from(device.device_id()) else {
                continue;
            };
            let device_jid = if from_jid.is_lid() {
                Jid::lid_device(from_jid.user.clone(), device_id)
            } else {
                Jid::pn_device(from_jid.user.clone(), device_id)
            };
            let addr = device_jid.to_protocol_address().to_string();
            if let Err(e) = backend.delete_session(&addr).await {
                warn!("Identity change: failed deleting companion session: {e}");
            }
        }
    }
    client.invalidate_device_cache(&from_jid.user).await;

    // Read the stored identity non-destructively before deleting it. WA Web
    // gates the heavy reset behind `loadIdentityKey(addr) != null`; with no
    // prior identity the session delete/rebuild would eagerly fetch prekeys
    // and build a session that may never be used.
    //
    // Check every address the identity could live under, because PN/LID
    // resolution can diverge from where the state actually is:
    //   - the resolved (preferred LID-or-PN) address;
    //   - the original PN address (migration may not have run yet);
    //   - the LID carried by the stanza itself.
    let resolved = client.resolve_encryption_jid(&from_jid).await;
    let stanza_lid = node.attrs().optional_jid("lid");
    let backend = client.persistence_manager.backend();

    let mut reset_addrs = vec![resolved.to_protocol_address().to_string()];
    for candidate in [Some(from_jid.clone()), stanza_lid.clone()]
        .into_iter()
        .flatten()
    {
        let candidate_addr = candidate.to_protocol_address().to_string();
        if !reset_addrs.contains(&candidate_addr) {
            reset_addrs.push(candidate_addr);
        }
    }

    // Treat a backend read error as had-prior (fail-safe): run the reset rather
    // than silently skip it. Collapsing an Err into "no prior identity" would
    // be fail-open on a session-deletion path.
    let mut previous_identity_key = None;
    let mut had_prior_identity = false;
    for candidate in &reset_addrs {
        match backend.load_identity(candidate).await {
            Ok(Some(key)) => {
                previous_identity_key = <[u8; 32]>::try_from(key.as_slice()).ok();
                had_prior_identity = true;
                break;
            }
            Ok(None) => {}
            Err(e) => {
                warn!(
                    "Identity change: failed reading stored identity: {e}; proceeding with reset"
                );
                had_prior_identity = true;
                break;
            }
        }
    }

    if !had_prior_identity {
        info!("Identity change (had_prior_identity=false): device record cleared, skipping reset");
        return;
    }

    // Counted past the companion/self/no-prior gates so it reflects actual
    // session resets rather than every identity-change push received.
    wacore::telemetry::identity_change();
    info!("Identity change (had_prior_identity=true): resetting session");

    // Delete the session + identity at every candidate address so a fresh
    // session can be established, and rotate the status sender key for forward
    // secrecy.
    for candidate in &reset_addrs {
        // Hold the per-address session lock while deleting so concurrent
        // encrypt/decrypt cannot recreate the stale session. One lock at a
        // time, so no lock-ordering risk.
        let lock = client
            .session_locks
            .get_with(candidate.clone(), async {
                Arc::new(tokio::sync::Mutex::new(()))
            })
            .await;
        let _guard = lock.lock().await;
        if let Err(e) = backend.delete_session(candidate).await {
            warn!("Identity change: failed deleting session: {e}");
        }
        if let Err(e) = backend.delete_identity(candidate).await {
            warn!("Identity change: failed deleting identity: {e}");
        }
    }

    let status_group = "status@broadcast";
    for own_jid in device_snapshot.pn.iter().chain(device_snapshot.lid.iter()) {
        let sender = own_jid.to_protocol_address().to_string();
        let unique_key = format!("{status_group}:{sender}");
        if let Err(e) = backend.delete_sender_key(&unique_key).await {
            warn!("Identity change: failed rotating status sender key: {e}");
        }
    }

    // = addSecurityCodeChangedNotifications, which WA Web fires inside the gate.
    client
        .core
        .event_bus
        .dispatch_with(EventKind::IdentityChange, || {
            Event::IdentityChange(
                IdentityChange::builder()
                    .user(from_jid.clone())
                    .maybe_lid_user(stanza_lid)
                    .implicit(false)
                    .maybe_previous_identity_key(previous_identity_key)
                    .build(),
            )
        });

    // Re-establish the session eagerly so the next send is fast. Skip only
    // while the offline backlog is still draining, matching WA Web:
    // deferring every offline-tagged push would pile up a prekey-fetch burst
    // when the resume completes. Deferral is safe because every send path
    // re-establishes before encrypting.
    let arrived_during_resume =
        node.attrs().optional_string("offline").is_some() && !*client.offline_sync_state.borrow();
    if arrived_during_resume {
        debug!("Identity change arrived during offline resume; deferring session re-establishment");
    } else {
        let client_clone = client.clone();
        tokio::spawn(async move {
            if let Err(e) = client_clone.ensure_e2e_sessions(vec![from_jid]).await {
                warn!("Identity change: failed to re-establish session: {e}");
            }
        });
    }
}

/// Refresh the device list of the contact a hash-only `<update>` names.
///
/// An unresolvable hash is a contact we never learned a mapping for, so there
/// is nothing to refresh. WA Web resolves the hash with
/// `getContactRecordByHash`; this fork resolves it against the persisted LID
/// mappings and then invalidates that contact's caches directly (it has no
/// pending-device-sync queue).
async fn sync_hashed_contact(client: &Arc<Client>, wire_hash: Option<&str>) {
    let Some(hash) = wire_hash.and_then(wacore::crypto::parse_contact_notification_hash) else {
        debug!(
            "Device update with unusable contact hash (bytes={:?})",
            wire_hash.map(str::len)
        );
        return;
    };

    let backend = client.persistence_manager.backend();
    let mappings = match backend.get_all_lid_mappings().await {
        Ok(mappings) => mappings,
        Err(e) => {
            warn!("Device update: failed reading LID mappings: {e}");
            return;
        }
    };
    let Some(contact_lid) = mappings
        .into_iter()
        .map(|entry| entry.lid)
        .find(|lid| wacore::crypto::contact_notification_hash(lid) == hash)
    else {
        debug!("Device update for an unknown contact hash");
        return;
    };

    client.invalidate_device_cache(&contact_lid).await;
}

/// Handle device list change notifications.
/// Matches WhatsApp Web's WAWebHandleDeviceNotification.handleDevicesNotification().
///
/// Device notifications have the structure:
/// ```xml
/// <notification type="devices" from="user@s.whatsapp.net">
///   <add device_hash="..."> or <remove device_hash="..."> or <update hash="...">
///     <device jid="user:device@server"/>
///     <key-index-list ts="..."/>
///   </add/remove/update>
/// </notification>
/// ```
pub(crate) async fn handle_devices_notification(client: &Arc<Client>, node: &Node) {
    let notification = match DeviceNotification::try_parse(node) {
        Ok(notification) => notification,
        Err(e) => {
            warn!("Failed to parse device notification: {e}");
            return;
        }
    };

    // Learn LID-PN mapping if present.
    if let Some((lid, pn)) = notification.lid_pn_mapping()
        && let Err(e) = client
            .add_lid_pn_mapping(lid, pn, LearningSource::DeviceNotification)
            .await
    {
        warn!("Failed to add LID-PN mapping from device notification: {e}");
    }

    // Per WhatsApp Web: one operation per notification, priority
    // remove > add > update.
    let op = &notification.operation;
    debug!(
        "Device notification: user={}, type={:?}, devices={:?}",
        notification.user(),
        op.operation_type,
        op.device_ids()
    );

    match op.operation_type {
        // Upstream patches the registry with `patch_device_add`/`_remove` and
        // verifies the signed key index list. This fork has no ADV key-index
        // verification or incremental patch path, so invalidate instead: the
        // next send re-runs usync and learns the new list.
        DeviceNotificationType::Add | DeviceNotificationType::Remove => {
            client.invalidate_device_cache(notification.user()).await;
        }
        DeviceNotificationType::Update => {
            if op.devices.is_empty() {
                // The `hash` names the *contact* whose device list changed,
                // not the notification's `from` (our own account). Invalidating
                // `from` would drop our own companion list.
                sync_hashed_contact(client, op.contact_hash.as_deref()).await;
            } else {
                for _device in &op.devices {
                    client.invalidate_device_cache(notification.user()).await;
                }
            }
        }
    }

    // Dispatch event to notify the application layer.
    client
        .core
        .event_bus
        .dispatch_with(EventKind::DeviceListUpdate, || {
            Event::DeviceListUpdate(
                DeviceListUpdate::builder()
                    .user(notification.from.clone())
                    .maybe_lid_user(notification.lid_user.clone())
                    .update_type(op.operation_type.into())
                    .devices(
                        op.devices
                            .iter()
                            .map(|d| {
                                DeviceNotificationInfo::builder()
                                    .device_id(d.device_id())
                                    .maybe_key_index(d.key_index)
                                    .build()
                            })
                            .collect(),
                    )
                    .maybe_key_index(op.key_index.clone())
                    .maybe_contact_hash(op.contact_hash.clone())
                    .build(),
            )
        });
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_utils::create_test_client_with_name;
    use wacore::stanza::devices::DeviceNotificationType;
    use wacore::types::events::{ChannelEventHandler, DeviceListUpdateType, Event};
    use wacore_binary::builder::NodeBuilder;

    fn notification_from(
        user: &str,
        children: Vec<wacore_binary::node::Node>,
    ) -> wacore_binary::node::Node {
        NodeBuilder::new("notification")
            .attr("type", "devices")
            .attr("from", user)
            .children(children)
            .build()
    }

    #[tokio::test]
    async fn device_add_dispatches_event_and_invalidates_registry() {
        let client = create_test_client_with_name("device_add").await;
        let (handler, events) = ChannelEventHandler::with_capacity(4);
        let _subscription = client.subscribe_handler(handler);

        let user = "15551234567";
        client
            .update_device_list(wacore::store::traits::DeviceListRecord {
                user: user.to_string(),
                devices: vec![wacore::store::traits::DeviceInfo::new(0, None)],
                timestamp: 1,
                phash: None,
            })
            .await
            .expect("seed device list");

        let node = notification_from(
            "15551234567@s.whatsapp.net",
            vec![
                NodeBuilder::new("add")
                    .children([
                        NodeBuilder::new("device")
                            .attr("jid", "15551234567:5@s.whatsapp.net")
                            .build(),
                        NodeBuilder::new("key-index-list")
                            .attr("ts", "1000")
                            .bytes(vec![0x01, 0x02, 0x03])
                            .build(),
                    ])
                    .build(),
            ],
        );

        handle_devices_notification(&client, &node).await;

        let event = events.try_recv().expect("device event");
        match &*event {
            Event::DeviceListUpdate(update) => {
                assert_eq!(update.update_type, DeviceListUpdateType::Add);
                assert_eq!(update.devices.len(), 1);
                assert_eq!(update.devices[0].device_id, 5);
            }
            other => panic!("unexpected event: {other:?}"),
        }
        assert!(
            client.device_registry_cache.get(user).await.is_none(),
            "add must force a fresh usync by invalidating the registry"
        );
    }

    #[tokio::test]
    async fn device_remove_dispatches_event() {
        let client = create_test_client_with_name("device_remove").await;
        let (handler, events) = ChannelEventHandler::with_capacity(4);
        let _subscription = client.subscribe_handler(handler);

        let node = notification_from(
            "15551234567@s.whatsapp.net",
            vec![
                NodeBuilder::new("remove")
                    .children([
                        NodeBuilder::new("device")
                            .attr("jid", "15551234567:3@s.whatsapp.net")
                            .build(),
                        NodeBuilder::new("key-index-list")
                            .attr("ts", "2000")
                            .build(),
                    ])
                    .build(),
            ],
        );

        handle_devices_notification(&client, &node).await;

        let event = events.try_recv().expect("device event");
        match &*event {
            Event::DeviceListUpdate(update) => {
                assert_eq!(update.update_type, DeviceListUpdateType::Remove);
                assert_eq!(update.devices.len(), 1);
                assert_eq!(update.devices[0].device_id, 3);
            }
            other => panic!("unexpected event: {other:?}"),
        }
    }

    #[tokio::test]
    async fn malformed_device_notification_is_ignored() {
        let client = create_test_client_with_name("device_malformed").await;
        let (handler, events) = ChannelEventHandler::with_capacity(4);
        let _subscription = client.subscribe_handler(handler);

        let node = notification_from("15551234567@s.whatsapp.net", Vec::new());
        handle_devices_notification(&client, &node).await;

        assert!(
            events.try_recv().is_err(),
            "a notification without an operation must not dispatch"
        );
    }

    #[tokio::test]
    async fn hash_only_update_with_unusable_hash_keeps_the_notified_users_registry() {
        let client = create_test_client_with_name("device_hash_unusable").await;
        let user = "15551234567";
        client
            .update_device_list(wacore::store::traits::DeviceListRecord {
                user: user.to_string(),
                devices: vec![
                    wacore::store::traits::DeviceInfo::new(0, None),
                    wacore::store::traits::DeviceInfo::new(65, Some(2)),
                ],
                timestamp: 1,
                phash: None,
            })
            .await
            .expect("seed device list");

        for hash in ["synthetic-key-in-hash", "AQID"] {
            let node = notification_from(
                "15551234567@s.whatsapp.net",
                vec![NodeBuilder::new("update").attr("hash", hash).build()],
            );
            handle_devices_notification(&client, &node).await;
        }

        let record = client
            .device_registry_cache
            .get(user)
            .await
            .expect("an unusable hash must not drop the notified user's registry");
        assert!(record.devices.iter().any(|d| d.device_id() == 65));
    }

    #[tokio::test]
    async fn hash_only_update_invalidates_the_hashed_contact_not_the_sender() {
        let client = create_test_client_with_name("device_hash_contact").await;
        let contact_lid = "100000000000001"; // hashes to "s7oK"
        let contact_pn = "15551234567";
        client
            .add_lid_pn_mapping(
                contact_lid,
                contact_pn,
                crate::lid_pn_cache::LearningSource::Usync,
            )
            .await
            .expect("seed mapping");
        client
            .update_device_list(wacore::store::traits::DeviceListRecord {
                user: contact_lid.to_string(),
                devices: vec![wacore::store::traits::DeviceInfo::new(0, None)],
                timestamp: 1,
                phash: None,
            })
            .await
            .expect("seed contact registry");

        let node = notification_from(
            "100000000000042@lid",
            vec![NodeBuilder::new("update").attr("hash", "s7oK").build()],
        );
        handle_devices_notification(&client, &node).await;

        assert!(
            client
                .device_registry_cache
                .get(contact_lid)
                .await
                .is_none(),
            "the hashed contact must be the one refreshed"
        );
    }

    #[tokio::test]
    async fn identity_change_dispatches_and_deletes_prior_session_and_identity() {
        let client = create_test_client_with_name("identity_change").await;
        let (handler, events) = ChannelEventHandler::with_capacity(4);
        let _subscription = client.subscribe_handler(handler);

        let from: Jid = "15551234567@s.whatsapp.net".parse().expect("jid");
        let addr = from.to_protocol_address().to_string();
        let backend = client.persistence_manager.backend();
        backend
            .put_identity(&addr, [7u8; 32])
            .await
            .expect("identity");
        backend
            .put_session(&addr, &[1, 2, 3])
            .await
            .expect("session");

        let node = NodeBuilder::new("notification")
            .attr("type", "encrypt")
            .attr("from", "15551234567@s.whatsapp.net")
            .children([NodeBuilder::new("identity").build()])
            .build();
        handle_encrypt_notification(&client, &node).await;

        let event = events.try_recv().expect("identity event");
        match &*event {
            Event::IdentityChange(change) => {
                assert_eq!(change.user, from);
                assert!(!change.implicit);
                assert_eq!(change.previous_identity_key, Some([7u8; 32]));
            }
            other => panic!("unexpected event: {other:?}"),
        }
        assert!(
            backend
                .load_identity(&addr)
                .await
                .expect("read identity")
                .is_none(),
            "prior identity must be deleted"
        );
        assert!(
            backend
                .get_session(&addr)
                .await
                .expect("read session")
                .is_none(),
            "prior session must be deleted"
        );
    }

    #[tokio::test]
    async fn identity_change_without_a_prior_identity_is_a_no_op() {
        let client = create_test_client_with_name("identity_no_prior").await;
        let (handler, events) = ChannelEventHandler::with_capacity(4);
        let _subscription = client.subscribe_handler(handler);

        let node = NodeBuilder::new("notification")
            .attr("type", "encrypt")
            .attr("from", "15559998888@s.whatsapp.net")
            .children([NodeBuilder::new("identity").build()])
            .build();
        handle_encrypt_notification(&client, &node).await;

        assert!(
            events.try_recv().is_err(),
            "no prior identity means no session reset event"
        );
    }

    #[tokio::test]
    async fn identity_change_from_a_companion_device_is_ignored() {
        let client = create_test_client_with_name("identity_companion").await;
        let (handler, events) = ChannelEventHandler::with_capacity(4);
        let _subscription = client.subscribe_handler(handler);

        let node = NodeBuilder::new("notification")
            .attr("type", "encrypt")
            .attr("from", "15551234567:5@s.whatsapp.net")
            .children([NodeBuilder::new("identity").build()])
            .build();
        handle_encrypt_notification(&client, &node).await;

        assert!(events.try_recv().is_err());
    }

    #[tokio::test]
    async fn identity_change_for_self_is_ignored() {
        let client = create_test_client_with_name("identity_self").await;
        client
            .persistence_manager
            .modify_device(|d| d.pn = Some(Jid::pn("15551234567")))
            .await;
        let (handler, events) = ChannelEventHandler::with_capacity(4);
        let _subscription = client.subscribe_handler(handler);

        let node = NodeBuilder::new("notification")
            .attr("type", "encrypt")
            .attr("from", "15551234567@s.whatsapp.net")
            .children([NodeBuilder::new("identity").build()])
            .build();
        handle_encrypt_notification(&client, &node).await;

        assert!(events.try_recv().is_err());
    }

    fn encrypt_node(from: &str, tags: &[&'static str]) -> wacore_binary::node::Node {
        NodeBuilder::new("notification")
            .attr("type", "encrypt")
            .attr("from", from)
            .children(tags.iter().map(|t| NodeBuilder::new(*t).build()))
            .build()
    }

    #[test]
    fn identity_child_wins_over_the_server_check() {
        let node = encrypt_node("15551234567@s.whatsapp.net", &["identity"]);
        assert_eq!(
            classify_encrypt_notification(&node),
            EncryptTarget::Identity
        );
    }

    #[test]
    fn count_reaches_prekey_low_wherever_it_sits() {
        for tags in [
            vec!["count"],
            vec!["count", "pq_count"],
            vec!["pq_count", "count"],
        ] {
            let node = encrypt_node("s.whatsapp.net", &tags);
            assert_eq!(
                classify_encrypt_notification(&node),
                EncryptTarget::PrekeyLow,
                "children {tags:?}"
            );
        }
    }

    #[test]
    fn pq_count_alone_uploads_nothing() {
        let node = encrypt_node("s.whatsapp.net", &["pq_count"]);
        assert_eq!(
            classify_encrypt_notification(&node),
            EncryptTarget::PqCountOnly
        );
    }

    #[test]
    fn digest_routes_to_digest_validation() {
        let node = encrypt_node("s.whatsapp.net", &["digest"]);
        assert_eq!(classify_encrypt_notification(&node), EncryptTarget::Digest);
    }

    #[test]
    fn unknown_server_child_is_unhandled() {
        let node = encrypt_node("s.whatsapp.net", &["mystery"]);
        assert_eq!(
            classify_encrypt_notification(&node),
            EncryptTarget::Unhandled("mystery")
        );
    }

    #[test]
    fn peer_encrypt_notification_is_not_server_routed() {
        let node = encrypt_node("15551234567@s.whatsapp.net", &["count"]);
        assert_eq!(
            classify_encrypt_notification(&node),
            EncryptTarget::NotServer
        );
    }

    #[test]
    fn device_notification_types_convert_for_events() {
        assert_eq!(
            DeviceListUpdateType::from(DeviceNotificationType::Add),
            DeviceListUpdateType::Add
        );
    }
}
