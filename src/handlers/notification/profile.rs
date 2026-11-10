//! Profile-related notifications: picture, status/about and contacts.

use log::{debug, warn};
use std::sync::Arc;
use wacore::types::events::{
    ContactNumberChanged, ContactSyncRequested, ContactUpdated, PictureUpdate, UserAboutUpdate,
};
use wacore_binary::jid::Jid;
use wacore_binary::node::{Node, NodeContent};

use crate::client::Client;
use crate::lid_pn_cache::LearningSource;
use crate::types::events::{Event, EventKind};

/// Handle profile picture change notifications.
///
/// Matches WhatsApp Web's `WAWebHandleProfilePicNotification`.
///
/// Structure:
/// ```xml
/// <notification type="picture" from="user@s.whatsapp.net" t="1234567890" id="...">
///   <set jid="user@s.whatsapp.net" id="pic_id" author="author@s.whatsapp.net"/>
/// </notification>
/// ```
///
/// Or for removal (no child or `<delete>` child):
/// ```xml
/// <notification type="picture" from="user@s.whatsapp.net" t="1234567890" id="...">
///   <delete jid="user@s.whatsapp.net"/>
/// </notification>
/// ```
pub(crate) fn handle_picture_notification(client: &Arc<Client>, node: &Node) {
    let from = crate::require_from_jid!(
        node,
        target: "Client/Picture",
        "picture notification"
    );

    let timestamp = notification_timestamp(node, chrono::Utc::now);

    // Look for <set>, <delete>, or no child to determine the action.
    // WhatsApp Web has two formats:
    // - With `jid` attr: direct update for that JID
    // - With `hash` attr (no `jid`): side contact, resolved via contact hash lookup
    let (jid, author, removed, picture_id) = if let Some(set_node) = node.get_optional_child("set")
    {
        let jid = set_node.attrs().optional_jid("jid").unwrap_or_else(|| {
            if set_node.attrs().optional_string("hash").is_some() {
                debug!(
                    target: "Client/Picture",
                    "Hash-based picture notification (no jid), using from={from}"
                );
            }
            from.clone()
        });
        let author = set_node.attrs().optional_jid("author");
        let pic_id = set_node
            .attrs()
            .optional_string("id")
            .map(|s| s.to_string());
        (jid, author, false, pic_id)
    } else if let Some(delete_node) = node.get_optional_child("delete") {
        let jid = delete_node
            .attrs()
            .optional_jid("jid")
            .unwrap_or_else(|| from.clone());
        let author = delete_node.attrs().optional_jid("author");
        (jid, author, true, None)
    } else {
        // No <set> or <delete> child. A bare notification is WhatsApp's
        // deletion signal.
        let children = node.children().map(|c| c.len()).unwrap_or(0);
        if children == 0 {
            let jid = node
                .attrs()
                .optional_jid("jid")
                .unwrap_or_else(|| from.clone());
            let author = node.attrs().optional_jid("author");
            (jid, author, true, None)
        } else {
            // Unknown child type (e.g., "request", "set_avatar") — log and skip
            let child_tag = node
                .children()
                .and_then(|c| c.first().map(|n| n.tag.as_str()));
            debug!(
                target: "Client/Picture",
                "Ignoring picture notification with child {:?} from {from}", child_tag
            );
            return;
        }
    };

    debug!(
        target: "Client/Picture",
        "Picture {}: jid={}, author={:?}, pic_id={:?}",
        if removed { "removed" } else { "updated" },
        jid, author, picture_id
    );

    let event = Event::PictureUpdate(
        PictureUpdate::builder()
            .jid(jid)
            .maybe_author(author)
            .timestamp(timestamp)
            .removed(removed)
            .maybe_picture_id(picture_id)
            .build(),
    );
    client.core.event_bus.dispatch(event);
}

/// Handle status/about text change notifications.
///
/// Matches WhatsApp Web's `WAWebHandleAboutNotification`.
///
/// Structure:
/// ```xml
/// <notification type="status" from="user@s.whatsapp.net" t="1234567890" notify="PushName">
///   <set>new status text</set>
/// </notification>
/// ```
pub(crate) fn handle_status_notification(client: &Arc<Client>, node: &Node) {
    let from = crate::require_from_jid!(
        node,
        target: "Client/Status",
        "status notification"
    );

    let timestamp = notification_timestamp(node, chrono::Utc::now);

    if let Some(set_node) = node.get_optional_child("set") {
        let status_text = match set_node.content.as_ref() {
            Some(NodeContent::String(s)) => s.clone(),
            Some(NodeContent::Bytes(b)) => String::from_utf8_lossy(b).into_owned(),
            _ => String::new(),
        };

        debug!(
            target: "Client/Status",
            "Status update from {} (length={})", from, status_text.len()
        );

        let event = Event::UserAboutUpdate(
            UserAboutUpdate::builder()
                .jid(from)
                .status(status_text)
                .timestamp(timestamp)
                .build(),
        );
        client.core.event_bus.dispatch(event);
    } else {
        debug!(
            target: "Client/Status",
            "Status notification from {from} without <set> child, ignoring"
        );
    }
}

/// Resolve the notification's `t` attribute, falling back to `now` when the
/// wire value is absent or out of range.
pub(crate) fn notification_timestamp(
    node: &Node,
    now: impl FnOnce() -> chrono::DateTime<chrono::Utc>,
) -> chrono::DateTime<chrono::Utc> {
    node.attrs()
        .optional_u64("t")
        .and_then(|t| i64::try_from(t).ok())
        .and_then(wacore::time::from_secs)
        .unwrap_or_else(now)
}

/// Learn LID-PN mappings from a contacts modify notification.
///
/// WA Web (`WAWebHandleContactNotification` → `WAWebDBCreateLidPnMappings`):
/// The `<modify>` child carries four attributes:
/// - `old` / `new` — old and new PN (phone number) JIDs
/// - `old_lid` / `new_lid` — old and new LID JIDs (optional)
///
/// When both `old_lid` and `new_lid` are present, WA Web creates two mappings:
/// `{ lid: old_lid, pn: old }` and `{ lid: new_lid, pn: new }`.
pub(crate) async fn learn_contact_modify_mappings(
    client: &Arc<Client>,
    old_pn: &Jid,
    new_pn: &Jid,
    old_lid: Option<&Jid>,
    new_lid: Option<&Jid>,
) {
    // WA Web: createLidPnMappings({mappings:[{lid:oldLid,pn:oldJid},{lid:newLid,pn:newJid}]})
    if let (Some(old_lid), Some(new_lid)) = (old_lid, new_lid) {
        for (lid, pn) in [(old_lid, old_pn), (new_lid, new_pn)] {
            if let Err(e) = client
                .add_lid_pn_mapping(&lid.user, &pn.user, LearningSource::DeviceNotification)
                .await
            {
                warn!(
                    target: "Client/Contacts",
                    "Failed to add LID-PN mapping lid={} pn={}: {e}", lid, pn
                );
            }
        }
    } else {
        debug!(
            target: "Client/Contacts",
            "Contacts modify without old_lid/new_lid, skipping LID-PN mapping (old={}, new={})",
            old_pn, new_pn
        );
    }
}

/// Handle contact change notifications.
///
/// WA Web: `WAWebHandleContactNotification`
///
/// These stanzas are sent as `<notification type="contacts">` with a single child action:
/// - `<update jid="..."/>` — contact profile changed. Consumers should
///   invalidate cached presence/profile picture (WA Web resets PresenceCollection
///   and refreshes profile pic thumb).
/// - `<modify old="..." new="..." old_lid="..." new_lid="..."/>` — contact
///   changed phone number. Creates LID-PN mappings when LID attrs present.
/// - `<sync after="..."/>` — server requests full contact re-sync.
/// - `<add .../>` or `<remove .../>` — lightweight roster changes (ACK only).
pub(crate) async fn handle_contacts_notification(client: &Arc<Client>, node: &Node) {
    let timestamp = notification_timestamp(node, chrono::Utc::now);

    let Some(child) = node.children().and_then(|children| children.first()) else {
        debug!(
            target: "Client/Contacts",
            "Ignoring contacts notification without child action"
        );
        return;
    };

    match child.tag.as_str() {
        "update" => {
            let Some(jid) = child.attrs().optional_jid("jid") else {
                // WA Web: when no jid, tries hash-based lookup against local contacts
                // (first 4 chars of contact userhash). If no match, it's a no-op.
                // We don't maintain a userhash index, so just ack and move on.
                debug!(target: "Client/Contacts", "contacts update with hash but no jid, ignoring (hash={:?})",
                    child.attrs().optional_string("hash"));
                return;
            };

            debug!(target: "Client/Contacts", "Contact updated for {jid}");
            client.core.event_bus.dispatch(Event::ContactUpdated(
                ContactUpdated::builder()
                    .jid(jid)
                    .timestamp(timestamp)
                    .build(),
            ));
        }
        "modify" => {
            // WA Web: old/new are PN JIDs, old_lid/new_lid are optional LID JIDs.
            let Some(old_jid) = child.attrs().optional_jid("old") else {
                warn!(target: "Client/Contacts", "contacts modify missing 'old' attribute");
                return;
            };
            let Some(new_jid) = child.attrs().optional_jid("new") else {
                warn!(target: "Client/Contacts", "contacts modify missing 'new' attribute");
                return;
            };
            let old_lid = child.attrs().optional_jid("old_lid");
            let new_lid = child.attrs().optional_jid("new_lid");

            learn_contact_modify_mappings(
                client,
                &old_jid,
                &new_jid,
                old_lid.as_ref(),
                new_lid.as_ref(),
            )
            .await;

            debug!(
                target: "Client/Contacts",
                "Contact number changed: {} -> {} (old_lid={:?}, new_lid={:?})",
                old_jid, new_jid, old_lid, new_lid
            );
            client
                .core
                .event_bus
                .dispatch_with(EventKind::ContactNumberChanged, || {
                    Event::ContactNumberChanged(
                        ContactNumberChanged::builder()
                            .old_jid(old_jid)
                            .new_jid(new_jid)
                            .maybe_old_lid(old_lid)
                            .maybe_new_lid(new_lid)
                            .timestamp(timestamp)
                            .build(),
                    )
                });
        }
        "sync" => {
            let after = child
                .attrs()
                .optional_u64("after")
                .and_then(|after| i64::try_from(after).ok())
                .and_then(wacore::time::from_secs);

            debug!(
                target: "Client/Contacts",
                "Contact sync requested after {:?}",
                after
            );
            client.core.event_bus.dispatch(Event::ContactSyncRequested(
                ContactSyncRequested::builder()
                    .maybe_after(after)
                    .timestamp(timestamp)
                    .build(),
            ));
        }
        "add" | "remove" => {
            debug!(
                target: "Client/Contacts",
                "Contact {} notification handled without extra work",
                child.tag
            );
        }
        other => {
            debug!(
                target: "Client/Contacts",
                "Ignoring unknown contacts notification child {:?}",
                other
            );
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_utils::create_test_client;
    use wacore::types::events::{ChannelEventHandler, Event, Subscription};
    use wacore_binary::builder::NodeBuilder;
    use wacore_binary::jid::Jid;
    use wacore_binary::node::Node;

    const FROM: &str = "15551234567@s.whatsapp.net";
    const OTHER: &str = "15557654321@s.whatsapp.net";
    const AUTHOR: &str = "15550000001@s.whatsapp.net";

    fn subscribed(client: &Arc<Client>) -> (Subscription, async_channel::Receiver<Arc<Event>>) {
        let (handler, events) = ChannelEventHandler::with_capacity(8);
        (client.subscribe_handler(handler), events)
    }

    fn notification(notification_type: &str, from: Option<&str>) -> NodeBuilder {
        let mut builder = NodeBuilder::new("notification").attr("type", notification_type);
        if let Some(from) = from {
            builder = builder.attr("from", from);
        }
        builder
    }

    fn picture_set(from: &str, child: Node) -> Node {
        NodeBuilder::new("notification")
            .attr("type", "picture")
            .attr("from", from)
            .attr("t", "1700000000")
            .attr("id", "notif-1")
            .children([child])
            .build()
    }

    #[tokio::test]
    async fn picture_set_dispatches_picture_update() {
        let client = create_test_client().await;
        let (_subscription, events) = subscribed(&client);

        let node = picture_set(
            FROM,
            NodeBuilder::new("set")
                .attr("jid", OTHER)
                .attr("author", AUTHOR)
                .attr("id", "pic-1")
                .build(),
        );

        handle_picture_notification(&client, &node);

        let event = events.try_recv().expect("picture event");
        let Event::PictureUpdate(update) = &*event else {
            panic!("expected PictureUpdate, got {:?}", event);
        };
        assert_eq!(update.jid, Jid::pn("15557654321"));
        assert_eq!(update.author, Some(Jid::pn("15550000001")));
        assert!(!update.removed);
        assert_eq!(update.picture_id.as_deref(), Some("pic-1"));
        assert_eq!(update.timestamp.timestamp(), 1_700_000_000);
        assert!(events.try_recv().is_err(), "exactly one event");
    }

    #[tokio::test]
    async fn picture_set_without_jid_falls_back_to_from() {
        let client = create_test_client().await;
        let (_subscription, events) = subscribed(&client);

        let node = picture_set(
            FROM,
            NodeBuilder::new("set")
                .attr("hash", "2:abcd")
                .attr("id", "pic-2")
                .build(),
        );

        handle_picture_notification(&client, &node);

        let event = events.try_recv().expect("picture event");
        let Event::PictureUpdate(update) = &*event else {
            panic!("expected PictureUpdate, got {:?}", event);
        };
        assert_eq!(update.jid, Jid::pn("15551234567"));
        assert_eq!(update.author, None);
        assert!(!update.removed);
    }

    #[tokio::test]
    async fn picture_delete_dispatches_removal() {
        let client = create_test_client().await;
        let (_subscription, events) = subscribed(&client);

        let node = picture_set(
            FROM,
            NodeBuilder::new("delete")
                .attr("jid", OTHER)
                .attr("author", AUTHOR)
                .build(),
        );

        handle_picture_notification(&client, &node);

        let event = events.try_recv().expect("picture event");
        let Event::PictureUpdate(update) = &*event else {
            panic!("expected PictureUpdate, got {:?}", event);
        };
        assert_eq!(update.jid, Jid::pn("15557654321"));
        assert!(update.removed);
        assert_eq!(update.picture_id, None);
    }

    #[tokio::test]
    async fn bare_picture_notification_counts_as_removal() {
        let client = create_test_client().await;
        let (_subscription, events) = subscribed(&client);

        let node = NodeBuilder::new("notification")
            .attr("type", "picture")
            .attr("from", FROM)
            .attr("t", "1700000000")
            .build();

        handle_picture_notification(&client, &node);

        let event = events.try_recv().expect("picture event");
        let Event::PictureUpdate(update) = &*event else {
            panic!("expected PictureUpdate, got {:?}", event);
        };
        assert_eq!(update.jid, Jid::pn("15551234567"));
        assert!(update.removed);
    }

    #[tokio::test]
    async fn unknown_picture_child_is_ignored() {
        let client = create_test_client().await;
        let (_subscription, events) = subscribed(&client);

        let node = picture_set(FROM, NodeBuilder::new("request").build());

        handle_picture_notification(&client, &node);

        assert!(events.try_recv().is_err(), "no event for unknown child");
    }

    #[tokio::test]
    async fn picture_without_from_is_ignored() {
        let client = create_test_client().await;
        let (_subscription, events) = subscribed(&client);

        let node = NodeBuilder::new("notification")
            .attr("type", "picture")
            .attr("t", "1700000000")
            .children([NodeBuilder::new("delete").attr("jid", OTHER).build()])
            .build();

        handle_picture_notification(&client, &node);

        assert!(events.try_recv().is_err());
    }

    #[tokio::test]
    async fn status_set_dispatches_about_update() {
        let client = create_test_client().await;
        let (_subscription, events) = subscribed(&client);

        let node = NodeBuilder::new("notification")
            .attr("type", "status")
            .attr("from", FROM)
            .attr("t", "1700000000")
            .children([NodeBuilder::new("set")
                .string_content("On vacation")
                .build()])
            .build();

        handle_status_notification(&client, &node);

        let event = events.try_recv().expect("about event");
        let Event::UserAboutUpdate(update) = &*event else {
            panic!("expected UserAboutUpdate, got {:?}", event);
        };
        assert_eq!(update.jid, Jid::pn("15551234567"));
        assert_eq!(update.status, "On vacation");
        assert_eq!(update.timestamp.timestamp(), 1_700_000_000);
    }

    #[tokio::test]
    async fn status_set_accepts_bytes_content() {
        let client = create_test_client().await;
        let (_subscription, events) = subscribed(&client);

        let node = NodeBuilder::new("notification")
            .attr("type", "status")
            .attr("from", FROM)
            .children([NodeBuilder::new("set")
                .bytes(b"Byte status".to_vec())
                .build()])
            .build();

        handle_status_notification(&client, &node);

        let event = events.try_recv().expect("about event");
        let Event::UserAboutUpdate(update) = &*event else {
            panic!("expected UserAboutUpdate, got {:?}", event);
        };
        assert_eq!(update.status, "Byte status");
    }

    #[tokio::test]
    async fn status_without_set_is_ignored() {
        let client = create_test_client().await;
        let (_subscription, events) = subscribed(&client);

        let node = notification("status", Some(FROM)).build();

        handle_status_notification(&client, &node);

        assert!(events.try_recv().is_err());
    }

    #[tokio::test]
    async fn status_without_from_is_ignored() {
        let client = create_test_client().await;
        let (_subscription, events) = subscribed(&client);

        let node = NodeBuilder::new("notification")
            .attr("type", "status")
            .children([NodeBuilder::new("set").string_content("Hi").build()])
            .build();

        handle_status_notification(&client, &node);

        assert!(events.try_recv().is_err());
    }

    #[tokio::test]
    async fn contacts_update_dispatches_contact_updated() {
        let client = create_test_client().await;
        let (_subscription, events) = subscribed(&client);

        let node = NodeBuilder::new("notification")
            .attr("type", "contacts")
            .attr("from", "s.whatsapp.net")
            .attr("t", "1700000000")
            .children([NodeBuilder::new("update").attr("jid", FROM).build()])
            .build();

        handle_contacts_notification(&client, &node).await;

        let event = events.try_recv().expect("contact event");
        let Event::ContactUpdated(update) = &*event else {
            panic!("expected ContactUpdated, got {:?}", event);
        };
        assert_eq!(update.jid, Jid::pn("15551234567"));
        assert!(events.try_recv().is_err());
    }

    #[tokio::test]
    async fn contacts_modify_with_lid_creates_mappings_and_event() {
        let client = create_test_client().await;
        let (_subscription, events) = subscribed(&client);

        let node = NodeBuilder::new("notification")
            .attr("type", "contacts")
            .attr("from", "s.whatsapp.net")
            .children([NodeBuilder::new("modify")
                .attr("old", FROM)
                .attr("new", OTHER)
                .attr("old_lid", "100000011111111@lid")
                .attr("new_lid", "100000022222222@lid")
                .build()])
            .build();

        handle_contacts_notification(&client, &node).await;

        assert_eq!(
            client
                .lid_pn_cache
                .get_phone_number("100000011111111")
                .await,
            Some("15551234567".to_string())
        );
        assert_eq!(
            client
                .lid_pn_cache
                .get_phone_number("100000022222222")
                .await,
            Some("15557654321".to_string())
        );

        let event = events.try_recv().expect("number change event");
        let Event::ContactNumberChanged(update) = &*event else {
            panic!("expected ContactNumberChanged, got {:?}", event);
        };
        assert_eq!(update.old_jid, Jid::pn("15551234567"));
        assert_eq!(update.new_jid, Jid::pn("15557654321"));
        assert_eq!(update.old_lid, Some(Jid::lid("100000011111111")));
        assert_eq!(update.new_lid, Some(Jid::lid("100000022222222")));
    }

    #[tokio::test]
    async fn contacts_modify_without_lid_skips_mapping() {
        let client = create_test_client().await;
        let (_subscription, events) = subscribed(&client);

        let node = NodeBuilder::new("notification")
            .attr("type", "contacts")
            .attr("from", "s.whatsapp.net")
            .children([NodeBuilder::new("modify")
                .attr("old", FROM)
                .attr("new", OTHER)
                .attr("new_lid", "100000022222222@lid")
                .build()])
            .build();

        handle_contacts_notification(&client, &node).await;

        assert!(
            client
                .lid_pn_cache
                .get_phone_number("100000022222222")
                .await
                .is_none()
        );
        let event = events.try_recv().expect("number change event");
        assert!(matches!(&*event, Event::ContactNumberChanged(_)));
    }

    #[tokio::test]
    async fn contacts_modify_missing_new_is_dropped() {
        let client = create_test_client().await;
        let (_subscription, events) = subscribed(&client);

        let node = NodeBuilder::new("notification")
            .attr("type", "contacts")
            .attr("from", "s.whatsapp.net")
            .children([NodeBuilder::new("modify").attr("old", FROM).build()])
            .build();

        handle_contacts_notification(&client, &node).await;

        assert!(events.try_recv().is_err());
    }

    #[tokio::test]
    async fn contacts_sync_dispatches_with_after() {
        let client = create_test_client().await;
        let (_subscription, events) = subscribed(&client);

        let node = NodeBuilder::new("notification")
            .attr("type", "contacts")
            .attr("from", "s.whatsapp.net")
            .children([NodeBuilder::new("sync").attr("after", "1700000000").build()])
            .build();

        handle_contacts_notification(&client, &node).await;

        let event = events.try_recv().expect("sync event");
        let Event::ContactSyncRequested(update) = &*event else {
            panic!("expected ContactSyncRequested, got {:?}", event);
        };
        assert_eq!(
            update.after.map(|after| after.timestamp()),
            Some(1_700_000_000)
        );
    }

    #[tokio::test]
    async fn contacts_add_remove_are_acknowledged_without_events() {
        let client = create_test_client().await;
        let (_subscription, events) = subscribed(&client);

        for tag in ["add", "remove"] {
            let node = NodeBuilder::new("notification")
                .attr("type", "contacts")
                .attr("from", "s.whatsapp.net")
                .children([NodeBuilder::new(tag).build()])
                .build();
            handle_contacts_notification(&client, &node).await;
        }

        assert!(events.try_recv().is_err());
    }

    #[tokio::test]
    async fn empty_contacts_notification_is_ignored() {
        let client = create_test_client().await;
        let (_subscription, events) = subscribed(&client);

        let node = notification("contacts", Some("s.whatsapp.net")).build();

        handle_contacts_notification(&client, &node).await;

        assert!(events.try_recv().is_err());
    }

    #[test]
    fn notification_timestamp_prefers_the_wire_value() {
        let node = NodeBuilder::new("notification")
            .attr("t", "1700000000")
            .build();

        let timestamp = notification_timestamp(&node, || {
            chrono::DateTime::from_timestamp(42, 0).expect("valid timestamp")
        });

        assert_eq!(timestamp.timestamp(), 1_700_000_000);
    }

    #[test]
    fn notification_timestamp_falls_back_to_now() {
        let node = NodeBuilder::new("notification").build();

        let timestamp = notification_timestamp(&node, || {
            chrono::DateTime::from_timestamp(42, 0).expect("valid timestamp")
        });

        assert_eq!(timestamp.timestamp(), 42);
    }
}
