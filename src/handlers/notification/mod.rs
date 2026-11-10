use super::traits::StanzaHandler;
use crate::client::Client;

mod companion_reg;
mod device;
mod groups;
mod mex;
mod newsletter;
mod privacy_business;
pub mod profile;
use crate::types::events::Event;
use async_trait::async_trait;
use log::{debug, info, warn};
use std::sync::Arc;
use wacore::store::traits::{DeviceInfo, DeviceListRecord};
use wacore_binary::jid::{Jid, JidExt};
use wacore_binary::node::Node;

/// Handler for `<notification>` stanzas.
///
/// Processes various notification types including:
/// - Encrypt notifications (key upload requests)
/// - Server sync notifications
/// - Account sync notifications (push name updates)
/// - Device notifications (device add/remove/update)
#[derive(Default)]
pub struct NotificationHandler;

#[async_trait]
impl StanzaHandler for NotificationHandler {
    fn tag(&self) -> &'static str {
        "notification"
    }

    async fn handle(&self, client: Arc<Client>, node: Arc<Node>, _cancelled: &mut bool) -> bool {
        handle_notification_impl(&client, &node).await;
        true
    }
}

async fn handle_notification_impl(client: &Arc<Client>, node: &Node) {
    let notification_type = node.attrs().optional_string("type").unwrap_or_default();

    match notification_type {
        "encrypt" => {
            device::handle_encrypt_notification(client, node).await;
        }
        "server_sync" => {
            // Server sync notifications inform us of app state changes from other devices.
            // For bot use case, we don't need to sync these (pins, mutes, archives, etc.).
            // Just acknowledge without syncing.
            if let Some(children) = node.children() {
                for collection_node in children.iter().filter(|c| c.tag == "collection") {
                    let name = collection_node
                        .attrs()
                        .optional_string("name")
                        .unwrap_or("<unknown>");
                    let version = collection_node.attrs().optional_u64("version").unwrap_or(0);
                    debug!(
                        target: "Client/AppState",
                        "Received server_sync for collection '{}' version {} (not syncing)",
                        name, version
                    );
                }
            }
        }
        "account_sync" => {
            // Handle push name updates
            if let Some(new_push_name) = node.attrs().optional_string("pushname") {
                client
                    .clone()
                    .update_push_name_and_notify(new_push_name.to_string())
                    .await;
            }

            // Handle device list updates (when a new device is paired)
            // Matches WhatsApp Web's handleAccountSyncNotification for DEVICES type
            if let Some(devices_node) = node.get_optional_child_by_tag(&["devices"]) {
                handle_account_sync_devices(client, node, devices_node).await;
            }

            // WA Web's account_sync parser reads a `<disappearing_mode>` child
            // alongside `<devices>`, and both arms end at the same place the
            // standalone `type="disappearing_mode"` notification does. The two
            // arms are mutually exclusive: with `action` present the stanza
            // carries no duration/`t`, and WA Web answers `action="modify"` by
            // re-querying the server, which this client cannot do (it has only
            // the set IQ). Log that arm instead of announcing it as a malformed
            // stanza.
            if let Some(dm) = node.get_optional_child("disappearing_mode") {
                if dm.attrs().optional_string("action").is_some() {
                    debug!(
                        target: "Client/Notification",
                        "account_sync disappearing_mode carried an action and no duration; \
                         re-querying the account default is not implemented"
                    );
                } else {
                    groups::handle_disappearing_mode_notification(client, &node.as_node_ref());
                }
            }
        }
        "devices" => {
            // Handle device list change notifications (WhatsApp Web: handleDevicesNotification)
            // These are sent when a user adds, removes, or updates a device
            device::handle_devices_notification(client, node).await;
        }
        "link_code_companion_reg" => {
            // Handle pair code notification (stage 2 of pair code authentication)
            // This is sent when the user enters the code on their phone
            companion_reg::handle(client, node).await;
        }
        "business" => {
            // Handle business notification (WhatsApp Web: handleBusinessNotification)
            // Notifies about business account status changes: verified name, profile, removal
            privacy_business::handle_business_notification(client, node).await;
        }
        "privacy_token" => {
            // Handle incoming trusted contact privacy token notifications.
            // Matches WhatsApp Web's WAWebHandlePrivacyTokenNotification.
            privacy_business::handle_privacy_token_notification(client, node).await;
        }
        "w:gp2" => {
            // Group metadata / participant / settings updates. Emits one typed
            // GroupUpdate per action plus the legacy raw Notification.
            groups::handle_group_notification(client, node).await;
        }
        "disappearing_mode" => {
            // A contact changed their default disappearing messages setting.
            // Matches WhatsApp Web's WAWebHandleDisappearingModeNotification.
            groups::handle_disappearing_mode_notification(client, &node.as_node_ref());
        }
        "picture" => {
            // Matches WhatsApp Web's WAWebHandleProfilePicNotification.
            profile::handle_picture_notification(client, node);
        }
        "status" => {
            // Matches WhatsApp Web's WAWebHandleAboutNotification.
            profile::handle_status_notification(client, node);
        }
        "contacts" => {
            // Matches WhatsApp Web's WAWebHandleContactNotification.
            profile::handle_contacts_notification(client, node).await;
        }
        "mex" => {
            // MEX notifications are routed by `op_name`, not by GraphQL doc ID,
            // matching WA Web's WAWebHandleMexNotification. The typed
            // `MexNotification` replaces the raw fallback for this stanza type.
            mex::handle_mex_notification(client, node);
        }
        "newsletter" => {
            // Newsletter live updates yield the typed counter event plus the
            // raw twin, which this arm therefore dispatches itself.
            newsletter::handle_newsletter_notification(client, node);
        }
        _ => {
            warn!("TODO: Implement handler for <notification type='{notification_type}'>");
            client
                .core
                .event_bus
                .dispatch(Event::Notification(std::sync::Arc::new(node.clone())));
        }
    }
}

/// Parsed device info from account_sync notification
struct AccountSyncDevice {
    jid: Jid,
    key_index: Option<u32>,
}

/// Parse devices from account_sync notification's <devices> child.
///
/// Example structure:
/// ```xml
/// <devices dhash="2:FnEWjS13">
///   <device jid="15551234567@s.whatsapp.net"/>
///   <device jid="15551234567:64@s.whatsapp.net" key-index="2"/>
///   <key-index-list ts="1766612162"><!-- bytes --></key-index-list>
/// </devices>
/// ```
fn parse_account_sync_device_list(devices_node: &Node) -> Vec<AccountSyncDevice> {
    let Some(children) = devices_node.children() else {
        return Vec::new();
    };

    children
        .iter()
        .filter(|n| n.tag == "device")
        .filter_map(|n| {
            let jid = n.attrs().optional_jid("jid")?;
            let key_index = n.attrs().optional_u64("key-index").map(|v| v as u32);
            Some(AccountSyncDevice { jid, key_index })
        })
        .collect()
}

/// Handle account_sync notification with <devices> child.
///
/// This is sent when devices are added/removed from OUR account (e.g., pairing a new WhatsApp Web).
/// Matches WhatsApp Web's `handleAccountSyncNotification` for `AccountSyncType.DEVICES`.
///
/// Key behaviors:
/// 1. Check if notification is for our own account (isSameAccountAndAddressingMode)
/// 2. Parse device list from notification
/// 3. Update device registry with new device list
/// 4. Does NOT trigger app state sync (that's handled by server_sync)
async fn handle_account_sync_devices(client: &Arc<Client>, node: &Node, devices_node: &Node) {
    // Extract the "from" JID - this is the account the notification is about
    let from_jid = match node.attrs().optional_jid("from") {
        Some(jid) => jid,
        None => {
            warn!(target: "Client/AccountSync", "account_sync devices missing 'from' attribute");
            return;
        }
    };

    // Get our own JIDs (PN and LID) to verify this is about our account
    let device_snapshot = client.persistence_manager.get_device_snapshot().await;
    let own_pn = device_snapshot.pn.as_ref();
    let own_lid = device_snapshot.lid.as_ref();

    // Check if notification is about our own account
    // Matches WhatsApp Web's isSameAccountAndAddressingMode check
    let is_own_account = own_pn.is_some_and(|pn| pn.is_same_user_as(&from_jid))
        || own_lid.is_some_and(|lid| lid.is_same_user_as(&from_jid));

    if !is_own_account {
        // WhatsApp Web logs "wid-is-not-self" error in this case
        warn!(
            target: "Client/AccountSync",
            "Received account_sync devices for non-self user: {} (our PN: {:?}, LID: {:?})",
            from_jid,
            own_pn.map(|j| j.user.as_str()),
            own_lid.map(|j| j.user.as_str())
        );
        return;
    }

    // Parse device list from notification
    let devices = parse_account_sync_device_list(devices_node);
    if devices.is_empty() {
        debug!(target: "Client/AccountSync", "account_sync devices list is empty");
        return;
    }

    // Extract dhash (device hash) for cache validation
    let dhash = devices_node
        .attrs()
        .optional_string("dhash")
        .map(String::from);

    // Get timestamp from notification
    let timestamp = node.attrs().optional_u64("t").unwrap_or_else(|| {
        std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap_or_default()
            .as_secs()
    }) as i64;

    // Build DeviceListRecord for storage
    // Note: update_device_list() will automatically store under LID if mapping is known
    let device_list = DeviceListRecord {
        user: from_jid.user.clone(),
        devices: devices
            .iter()
            .map(|d| DeviceInfo {
                device_id: d.jid.device as u32,
                key_index: d.key_index,
            })
            .collect(),
        timestamp,
        phash: dhash,
    };

    if let Err(e) = client.update_device_list(device_list).await {
        warn!(
            target: "Client/AccountSync",
            "Failed to update device list from account_sync: {}",
            e
        );
        return;
    }

    info!(
        target: "Client/AccountSync",
        "Updated own device list from account_sync: {} devices (user: {})",
        devices.len(),
        from_jid.user
    );

    // Log individual devices at debug level
    for device in &devices {
        debug!(
            target: "Client/AccountSync",
            "  Device: {} (key-index: {:?})",
            device.jid,
            device.key_index
        );
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_utils::create_test_client;
    use wacore::stanza::devices::{DeviceNotification, DeviceNotificationType};
    use wacore::types::events::{ChannelEventHandler, DeviceListUpdateType};
    use wacore_binary::builder::NodeBuilder;

    #[tokio::test]
    async fn routes_picture_notifications_to_the_profile_handler() {
        let client = create_test_client().await;
        let (handler, events) = ChannelEventHandler::with_capacity(4);
        let _subscription = client.subscribe_handler(handler);

        let node = NodeBuilder::new("notification")
            .attr("type", "picture")
            .attr("from", "1234567890@s.whatsapp.net")
            .attr("t", "1700000000")
            .children([NodeBuilder::new("delete").build()])
            .build();
        let mut cancelled = false;
        NotificationHandler
            .handle(client.clone(), Arc::new(node), &mut cancelled)
            .await;

        let event = events.try_recv().expect("picture event");
        assert!(matches!(&*event, Event::PictureUpdate(_)));
    }

    #[tokio::test]
    async fn routes_status_notifications_to_the_profile_handler() {
        let client = create_test_client().await;
        let (handler, events) = ChannelEventHandler::with_capacity(4);
        let _subscription = client.subscribe_handler(handler);

        let node = NodeBuilder::new("notification")
            .attr("type", "status")
            .attr("from", "1234567890@s.whatsapp.net")
            .children([NodeBuilder::new("set").string_content("Hello").build()])
            .build();
        let mut cancelled = false;
        NotificationHandler
            .handle(client.clone(), Arc::new(node), &mut cancelled)
            .await;

        let event = events.try_recv().expect("about event");
        assert!(matches!(&*event, Event::UserAboutUpdate(_)));
    }

    #[tokio::test]
    async fn routes_contacts_notifications_to_the_profile_handler() {
        let client = create_test_client().await;
        let (handler, events) = ChannelEventHandler::with_capacity(4);
        let _subscription = client.subscribe_handler(handler);

        let node = NodeBuilder::new("notification")
            .attr("type", "contacts")
            .attr("from", "s.whatsapp.net")
            .children([NodeBuilder::new("update")
                .attr("jid", "1234567890@s.whatsapp.net")
                .build()])
            .build();
        let mut cancelled = false;
        NotificationHandler
            .handle(client.clone(), Arc::new(node), &mut cancelled)
            .await;

        let event = events.try_recv().expect("contacts event");
        assert!(matches!(&*event, Event::ContactUpdated(_)));
    }

    #[tokio::test]
    async fn routes_group_notifications_to_the_group_handler() {
        let client = create_test_client().await;
        let (handler, events) = ChannelEventHandler::with_capacity(8);
        let _subscription = client.subscribe_handler(handler);

        let node = NodeBuilder::new("notification")
            .attr("type", "w:gp2")
            .attr("from", "120363000000000001@g.us")
            .attr("id", "notif-1")
            .attr("t", "1700000000")
            .children([NodeBuilder::new("subject")
                .attr("subject", "renamed")
                .build()])
            .build();
        let mut cancelled = false;
        NotificationHandler
            .handle(client.clone(), Arc::new(node), &mut cancelled)
            .await;

        let event = events.try_recv().expect("group update event");
        assert!(matches!(&*event, Event::GroupUpdate(_)));
    }

    #[tokio::test]
    async fn routes_disappearing_mode_notifications_to_the_handler() {
        let client = create_test_client().await;
        let (handler, events) = ChannelEventHandler::with_capacity(4);
        let _subscription = client.subscribe_handler(handler);

        let node = NodeBuilder::new("notification")
            .attr("type", "disappearing_mode")
            .attr("from", "15550000001:75@s.whatsapp.net")
            .attr("t", "1700000000")
            .children([NodeBuilder::new("disappearing_mode")
                .attr("duration", "86400")
                .attr("t", "1700000000")
                .build()])
            .build();
        let mut cancelled = false;
        NotificationHandler
            .handle(client.clone(), Arc::new(node), &mut cancelled)
            .await;

        let event = events.try_recv().expect("disappearing mode event");
        let Event::DisappearingModeChanged(changed) = &*event else {
            panic!("unexpected event: {event:?}");
        };
        assert_eq!(changed.from.user, "15550000001");
        assert_eq!(changed.from.device, 0);
        assert_eq!(changed.duration, 86_400);
    }

    #[tokio::test]
    async fn account_sync_with_disappearing_mode_dispatches_the_event() {
        let client = create_test_client().await;
        let (handler, events) = ChannelEventHandler::with_capacity(4);
        let _subscription = client.subscribe_handler(handler);

        let node = NodeBuilder::new("notification")
            .attr("type", "account_sync")
            .attr("from", "15550000001@s.whatsapp.net")
            .attr("t", "1700000000")
            .children([NodeBuilder::new("disappearing_mode")
                .attr("duration", "604800")
                .attr("t", "1700000000")
                .build()])
            .build();
        let mut cancelled = false;
        NotificationHandler
            .handle(client.clone(), Arc::new(node), &mut cancelled)
            .await;

        let event = events.try_recv().expect("disappearing mode event");
        let Event::DisappearingModeChanged(changed) = &*event else {
            panic!("unexpected event: {event:?}");
        };
        assert_eq!(changed.duration, 604_800);
        assert_eq!(changed.setting_timestamp.timestamp(), 1_700_000_000);
    }

    #[tokio::test]
    async fn account_sync_with_disappearing_mode_action_dispatches_nothing() {
        let client = create_test_client().await;
        let (handler, events) = ChannelEventHandler::with_capacity(4);
        let _subscription = client.subscribe_handler(handler);

        // With `action` present the stanza carries no duration/t: WA Web
        // re-queries, which this client does not implement, so it must not
        // dispatch a made-up update.
        let node = NodeBuilder::new("notification")
            .attr("type", "account_sync")
            .attr("from", "15550000001@s.whatsapp.net")
            .children([NodeBuilder::new("disappearing_mode")
                .attr("action", "modify")
                .build()])
            .build();
        let mut cancelled = false;
        NotificationHandler
            .handle(client.clone(), Arc::new(node), &mut cancelled)
            .await;

        assert!(events.try_recv().is_err(), "action arm dispatches nothing");
    }

    #[tokio::test]
    async fn routes_mex_notifications_to_the_mex_handler() {
        let client = create_test_client().await;
        let (handler, events) = ChannelEventHandler::with_capacity(4);
        let _subscription = client.subscribe_handler(handler);

        let node = NodeBuilder::new("notification")
            .attr("type", "mex")
            .attr("from", "s.whatsapp.net")
            .attr("id", "mex-notif-1")
            .children([NodeBuilder::new("update")
                .attr("op_name", "SomeFutureOp")
                .string_content(r#"{"data":{"xwa2_something":true}}"#)
                .build()])
            .build();
        let mut cancelled = false;
        NotificationHandler
            .handle(client.clone(), Arc::new(node), &mut cancelled)
            .await;

        let event = events.try_recv().expect("raw mex event");
        let Event::MexNotification(raw) = &*event else {
            panic!("unexpected event: {event:?}");
        };
        assert_eq!(raw.op_name, "SomeFutureOp");
        // The mex arm must not fall through to the raw stanza fallback.
        assert!(events.try_recv().is_err(), "exactly one event");
    }

    #[tokio::test]
    async fn routes_newsletter_notifications_to_the_newsletter_handler() {
        let client = create_test_client().await;
        let (handler, events) = ChannelEventHandler::with_capacity(4);
        let _subscription = client.subscribe_handler(handler);

        let node = NodeBuilder::new("notification")
            .attr("type", "newsletter")
            .attr("from", "120363000000000001@newsletter")
            .children([NodeBuilder::new("live_updates")
                .children([NodeBuilder::new("messages")
                    .children([NodeBuilder::new("message").attr("server_id", "42").build()])
                    .build()])
                .build()])
            .build();
        let mut cancelled = false;
        NotificationHandler
            .handle(client.clone(), Arc::new(node), &mut cancelled)
            .await;

        let event = events.try_recv().expect("newsletter live update");
        assert!(matches!(&*event, Event::NewsletterLiveUpdate(_)));
        let raw = events.try_recv().expect("raw newsletter notification");
        assert!(matches!(&*raw, Event::Notification(_)));
        assert!(events.try_recv().is_err(), "exactly two events");
    }

    #[test]
    fn test_parse_device_add_notification() {
        // Per WhatsApp Web: add operation has single device + key-index-list
        let node = NodeBuilder::new("notification")
            .attr("type", "devices")
            .attr("from", "1234567890@s.whatsapp.net")
            .children([NodeBuilder::new("add")
                .children([
                    NodeBuilder::new("device")
                        .attr("jid", "1234567890:1@s.whatsapp.net")
                        .build(),
                    NodeBuilder::new("key-index-list")
                        .attr("ts", "1000")
                        .bytes(vec![0x01, 0x02, 0x03])
                        .build(),
                ])
                .build()])
            .build();

        let parsed = DeviceNotification::try_parse(&node).unwrap();
        assert_eq!(parsed.operation.operation_type, DeviceNotificationType::Add);
        assert_eq!(parsed.operation.device_ids(), vec![1]);
        // Verify key index info
        assert!(parsed.operation.key_index.is_some());
        assert_eq!(parsed.operation.key_index.as_ref().unwrap().timestamp, 1000);
    }

    #[test]
    fn test_parse_device_remove_notification() {
        let node = NodeBuilder::new("notification")
            .attr("type", "devices")
            .attr("from", "1234567890@s.whatsapp.net")
            .children([NodeBuilder::new("remove")
                .children([
                    NodeBuilder::new("device")
                        .attr("jid", "1234567890:3@s.whatsapp.net")
                        .build(),
                    NodeBuilder::new("key-index-list")
                        .attr("ts", "2000")
                        .build(),
                ])
                .build()])
            .build();

        let parsed = DeviceNotification::try_parse(&node).unwrap();
        assert_eq!(
            parsed.operation.operation_type,
            DeviceNotificationType::Remove
        );
        assert_eq!(parsed.operation.device_ids(), vec![3]);
    }

    #[test]
    fn test_parse_device_update_notification_with_hash() {
        let node = NodeBuilder::new("notification")
            .attr("type", "devices")
            .attr("from", "1234567890@s.whatsapp.net")
            .children([NodeBuilder::new("update")
                .attr("hash", "2:abcdef123456")
                .build()])
            .build();

        let parsed = DeviceNotification::try_parse(&node).unwrap();
        assert_eq!(
            parsed.operation.operation_type,
            DeviceNotificationType::Update
        );
        assert_eq!(
            parsed.operation.contact_hash,
            Some("2:abcdef123456".to_string())
        );
        // Update operations don't have devices (just hash for lookup)
        assert!(parsed.operation.devices.is_empty());
    }

    #[test]
    fn test_parse_empty_device_notification_fails() {
        // Per WhatsApp Web: at least one operation (add/remove/update) is required
        let node = NodeBuilder::new("notification")
            .attr("type", "devices")
            .attr("from", "1234567890@s.whatsapp.net")
            .build();

        let result = DeviceNotification::try_parse(&node);
        assert!(result.is_err());
        assert!(
            result
                .unwrap_err()
                .to_string()
                .contains("missing required operation")
        );
    }

    #[test]
    fn test_parse_multiple_operations_uses_priority() {
        // Per WhatsApp Web: only ONE operation is processed with priority remove > add > update
        // If both remove and add are present, remove should be processed
        let node = NodeBuilder::new("notification")
            .attr("type", "devices")
            .attr("from", "1234567890@s.whatsapp.net")
            .children([
                NodeBuilder::new("add")
                    .children([
                        NodeBuilder::new("device")
                            .attr("jid", "1234567890:5@s.whatsapp.net")
                            .build(),
                        NodeBuilder::new("key-index-list")
                            .attr("ts", "3000")
                            .build(),
                    ])
                    .build(),
                NodeBuilder::new("remove")
                    .children([
                        NodeBuilder::new("device")
                            .attr("jid", "1234567890:2@s.whatsapp.net")
                            .build(),
                        NodeBuilder::new("key-index-list")
                            .attr("ts", "3001")
                            .build(),
                    ])
                    .build(),
            ])
            .build();

        let parsed = DeviceNotification::try_parse(&node).unwrap();
        // Should process remove, not add (priority: remove > add > update)
        assert_eq!(
            parsed.operation.operation_type,
            DeviceNotificationType::Remove
        );
        assert_eq!(parsed.operation.device_ids(), vec![2]);
    }

    #[test]
    fn test_device_list_update_type_from_notification_type() {
        assert_eq!(
            DeviceListUpdateType::from(DeviceNotificationType::Add),
            DeviceListUpdateType::Add
        );
        assert_eq!(
            DeviceListUpdateType::from(DeviceNotificationType::Remove),
            DeviceListUpdateType::Remove
        );
        assert_eq!(
            DeviceListUpdateType::from(DeviceNotificationType::Update),
            DeviceListUpdateType::Update
        );
    }

    // Tests for account_sync device parsing

    #[test]
    fn test_parse_account_sync_device_list_basic() {
        let devices_node = NodeBuilder::new("devices")
            .attr("dhash", "2:FnEWjS13")
            .children([
                NodeBuilder::new("device")
                    .attr("jid", "15551234567@s.whatsapp.net")
                    .build(),
                NodeBuilder::new("device")
                    .attr("jid", "15551234567:64@s.whatsapp.net")
                    .attr("key-index", "2")
                    .build(),
            ])
            .build();

        let devices = parse_account_sync_device_list(&devices_node);
        assert_eq!(devices.len(), 2);

        // Primary device (device 0)
        assert_eq!(devices[0].jid.user, "15551234567");
        assert_eq!(devices[0].jid.device, 0);
        assert_eq!(devices[0].key_index, None);

        // Companion device (device 64)
        assert_eq!(devices[1].jid.user, "15551234567");
        assert_eq!(devices[1].jid.device, 64);
        assert_eq!(devices[1].key_index, Some(2));
    }

    #[test]
    fn test_parse_account_sync_device_list_with_key_index_list() {
        // Real-world structure includes <key-index-list> which should be ignored
        let devices_node = NodeBuilder::new("devices")
            .attr("dhash", "2:FnEWjS13")
            .children([
                NodeBuilder::new("device")
                    .attr("jid", "15551234567@s.whatsapp.net")
                    .build(),
                NodeBuilder::new("device")
                    .attr("jid", "15551234567:77@s.whatsapp.net")
                    .attr("key-index", "15")
                    .build(),
                NodeBuilder::new("key-index-list")
                    .attr("ts", "1766612162")
                    .bytes(vec![0x01, 0x02, 0x03]) // Simulated signed bytes
                    .build(),
            ])
            .build();

        let devices = parse_account_sync_device_list(&devices_node);
        // Should only parse <device> tags, not <key-index-list>
        assert_eq!(devices.len(), 2);
        assert_eq!(devices[0].jid.device, 0);
        assert_eq!(devices[1].jid.device, 77);
        assert_eq!(devices[1].key_index, Some(15));
    }

    #[test]
    fn test_parse_account_sync_device_list_empty() {
        let devices_node = NodeBuilder::new("devices")
            .attr("dhash", "2:FnEWjS13")
            .build();

        let devices = parse_account_sync_device_list(&devices_node);
        assert!(devices.is_empty());
    }

    #[test]
    fn test_parse_account_sync_device_list_multiple_devices() {
        let devices_node = NodeBuilder::new("devices")
            .attr("dhash", "2:XYZ123")
            .children([
                NodeBuilder::new("device")
                    .attr("jid", "1234567890@s.whatsapp.net")
                    .build(),
                NodeBuilder::new("device")
                    .attr("jid", "1234567890:1@s.whatsapp.net")
                    .attr("key-index", "1")
                    .build(),
                NodeBuilder::new("device")
                    .attr("jid", "1234567890:2@s.whatsapp.net")
                    .attr("key-index", "5")
                    .build(),
                NodeBuilder::new("device")
                    .attr("jid", "1234567890:3@s.whatsapp.net")
                    .attr("key-index", "10")
                    .build(),
            ])
            .build();

        let devices = parse_account_sync_device_list(&devices_node);
        assert_eq!(devices.len(), 4);

        // Verify device IDs are correctly parsed
        assert_eq!(devices[0].jid.device, 0);
        assert_eq!(devices[1].jid.device, 1);
        assert_eq!(devices[2].jid.device, 2);
        assert_eq!(devices[3].jid.device, 3);

        // Verify key indexes
        assert_eq!(devices[0].key_index, None);
        assert_eq!(devices[1].key_index, Some(1));
        assert_eq!(devices[2].key_index, Some(5));
        assert_eq!(devices[3].key_index, Some(10));
    }
}
