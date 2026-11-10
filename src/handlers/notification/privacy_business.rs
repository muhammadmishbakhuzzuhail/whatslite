//! Privacy-token and business notifications.
//!
//! Wire formats mirror WhatsApp Web's `WAWebHandlePrivacyTokenNotification`
//! and `WAWebHandleBusinessNotification`.

use std::sync::Arc;

use log::{debug, info, warn};
use wacore::iq::tctoken::parse_privacy_token_notification;
use wacore::stanza::business::{BusinessNotification, BusinessNotificationType};
use wacore::store::traits::TcTokenEntry;
use wacore::types::events::{BusinessStatusUpdate, BusinessUpdateType, EventKind};
use wacore_binary::node::Node;

use crate::client::Client;
use crate::types::events::Event;

/// Handle `<notification type="privacy_token">`.
///
/// Stores trusted-contact tokens under the sender's LID (falling back to the
/// phone number when no LID mapping exists).
///
/// ```xml
/// <notification type="privacy_token" from="user@s.whatsapp.net" sender_lid="user@lid">
///   <tokens>
///     <token type="trusted_contact" t="1707000000"><!-- bytes --></token>
///   </tokens>
/// </notification>
/// ```
pub(crate) async fn handle_privacy_token_notification(client: &Arc<Client>, node: &Node) {
    let from_jid = node.attrs().optional_jid("from");

    // WA Web prefers the explicit `sender_lid` attribute.
    let sender_lid_jid = node
        .attrs()
        .optional_jid("sender_lid")
        .filter(|j| !j.user.is_empty());

    let resolved_lid: Option<String>;
    let sender_lid: &str = if let Some(ref lid_jid) = sender_lid_jid {
        lid_jid.user.as_str()
    } else {
        let Some(from) = from_jid.as_ref() else {
            warn!(target: "Client/TcToken", "privacy_token notification missing 'from' attribute");
            return;
        };

        if from.is_lid() {
            from.user.as_str()
        } else {
            resolved_lid = client.lid_pn_cache.get_current_lid(&from.user).await;
            match &resolved_lid {
                Some(lid) => lid.as_str(),
                None => {
                    debug!(
                        target: "Client/TcToken",
                        "Cannot resolve LID for privacy_token sender {}, storing under PN",
                        from
                    );
                    from.user.as_str()
                }
            }
        }
    };

    let received_tokens = match parse_privacy_token_notification(node) {
        Ok(tokens) => tokens,
        Err(e) => {
            warn!(target: "Client/TcToken", "Failed to parse privacy_token notification: {e}");
            return;
        }
    };

    if received_tokens.is_empty() {
        debug!(target: "Client/TcToken", "privacy_token notification had no trusted_contact tokens");
        return;
    }

    let backend = client.persistence_manager.backend();

    for received in &received_tokens {
        let existing = match backend.get_tc_token(sender_lid).await {
            Ok(entry) => entry,
            Err(e) => {
                warn!(target: "Client/TcToken", "Failed to read tc_token for {}: {e}, skipping", sender_lid);
                continue;
            }
        };

        if let Some(existing) = &existing {
            // Same bytes: only advance the timestamp forward.
            if existing.token == received.token {
                if received.timestamp > existing.token_timestamp {
                    let entry = TcTokenEntry {
                        token: existing.token.clone(),
                        token_timestamp: received.timestamp,
                        sender_timestamp: existing.sender_timestamp,
                    };
                    if let Err(e) = backend.put_tc_token(sender_lid, &entry).await {
                        warn!(target: "Client/TcToken", "Failed to refresh tc_token timestamp for {}: {e}", sender_lid);
                    }
                }
                continue;
            }

            // Never replace a real token with an older one. A byte-less
            // placeholder written by sender-side issuance has no real token
            // yet, so the contact's first real token always wins.
            if !existing.token.is_empty() && received.timestamp < existing.token_timestamp {
                debug!(
                    target: "Client/TcToken",
                    "Skipping older token for {} (incoming={}, existing={})",
                    sender_lid, received.timestamp, existing.token_timestamp
                );
                continue;
            }
        }

        let entry = TcTokenEntry {
            token: received.token.clone(),
            token_timestamp: received.timestamp,
            sender_timestamp: existing.and_then(|e| e.sender_timestamp),
        };
        if let Err(e) = backend.put_tc_token(sender_lid, &entry).await {
            warn!(target: "Client/TcToken", "Failed to store tc_token for {}: {e}", sender_lid);
        } else {
            debug!(target: "Client/TcToken", "Stored tc_token for {} (t={})", sender_lid, received.timestamp);
        }
    }
}

/// Handle `<notification type="business">`.
pub(crate) async fn handle_business_notification(client: &Arc<Client>, node: &Node) {
    let notification = match BusinessNotification::try_parse(node) {
        Ok(n) => n,
        Err(e) => {
            warn!(target: "Client/Business", "Failed to parse business notification: {e}");
            return;
        }
    };

    debug!(
        target: "Client/Business",
        "Business notification: from={}, type={}, jid={:?}",
        notification.from,
        notification.notification_type,
        notification.jid
    );

    match notification.notification_type {
        BusinessNotificationType::RemoveJid | BusinessNotificationType::RemoveHash => {
            info!(
                target: "Client/Business",
                "Contact {} is no longer a business account",
                notification.from
            );
        }
        BusinessNotificationType::VerifiedNameJid | BusinessNotificationType::VerifiedNameHash => {
            if let Some(name) = notification
                .verified_name
                .as_ref()
                .and_then(|vn| vn.name.as_ref())
            {
                info!(
                    target: "Client/Business",
                    "Contact {} verified business name: {}",
                    notification.from,
                    name
                );
            }
        }
        BusinessNotificationType::Profile | BusinessNotificationType::ProfileHash => {
            debug!(
                target: "Client/Business",
                "Contact {} business profile updated (hash: {:?})",
                notification.from,
                notification.hash
            );
        }
        _ => {}
    }

    let verified_name = notification
        .verified_name
        .as_ref()
        .and_then(|vn| vn.name.clone());

    client
        .core
        .event_bus
        .dispatch_with(EventKind::BusinessStatusUpdate, || {
            Event::BusinessStatusUpdate(Box::new(
                BusinessStatusUpdate::builder()
                    .jid(notification.from.clone())
                    .update_type(BusinessUpdateType::from(
                        notification.notification_type.clone(),
                    ))
                    .timestamp(
                        wacore::time::from_secs(notification.timestamp)
                            .unwrap_or_else(chrono::Utc::now),
                    )
                    .maybe_target_jid(notification.jid.clone())
                    .maybe_hash(notification.hash.clone())
                    .maybe_verified_name(verified_name)
                    .product_ids(notification.product_ids.clone())
                    .collection_ids(notification.collection_ids.clone())
                    .subscriptions(notification.subscriptions.clone())
                    .build(),
            ))
        });
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_utils::create_test_client;
    use std::sync::Arc;
    use wacore::store::traits::TcTokenEntry;
    use wacore::types::events::{ChannelEventHandler, EventKind};
    use wacore_binary::builder::NodeBuilder;

    fn privacy_token_notification(from: &str, t: i64, bytes: Vec<u8>) -> Node {
        NodeBuilder::new("notification")
            .attr("type", "privacy_token")
            .attr("from", from)
            .children([NodeBuilder::new("tokens")
                .children([NodeBuilder::new("token")
                    .attr("type", "trusted_contact")
                    .attr("t", t.to_string())
                    .bytes(bytes)
                    .build()])
                .build()])
            .build()
    }

    fn business_notification(from: &str, child: Node) -> Node {
        NodeBuilder::new("notification")
            .attr("type", "business")
            .attr("from", from)
            .attr("id", "123456")
            .attr("t", "1700000000")
            .children([child])
            .build()
    }

    fn subscribed(
        client: &Arc<Client>,
    ) -> (
        wacore::types::events::Subscription,
        async_channel::Receiver<Arc<Event>>,
    ) {
        let (handler, events) = ChannelEventHandler::with_capacity(8);
        (client.subscribe_handler(handler), events)
    }

    #[tokio::test]
    async fn stores_new_token_with_no_sender_timestamp() {
        let client = create_test_client().await;
        let node = privacy_token_notification("880000001@lid", 1_700_000_000, vec![0xAA, 0xBB]);

        handle_privacy_token_notification(&client, &node).await;

        let stored = client
            .persistence_manager
            .backend()
            .get_tc_token("880000001")
            .await
            .unwrap()
            .expect("token should be stored");
        assert_eq!(stored.token, vec![0xAA, 0xBB]);
        assert_eq!(stored.token_timestamp, 1_700_000_000);
        assert_eq!(stored.sender_timestamp, None);
    }

    #[tokio::test]
    async fn real_token_replaces_byteless_placeholder_and_keeps_sender_timestamp() {
        let client = create_test_client().await;
        let backend = client.persistence_manager.backend();

        // Placeholder written by sender-side issuance: no bytes yet, but a
        // sender_timestamp that is newer than the contact's incoming token.
        backend
            .put_tc_token(
                "880000002",
                &TcTokenEntry {
                    token: Vec::new(),
                    token_timestamp: 1_700_000_500,
                    sender_timestamp: Some(1_700_000_500),
                },
            )
            .await
            .unwrap();

        // Incoming real token has an OLDER timestamp than the placeholder — the
        // monotonicity guard must not drop it.
        let node =
            privacy_token_notification("880000002@lid", 1_700_000_000, vec![0x01, 0x02, 0x03]);
        handle_privacy_token_notification(&client, &node).await;

        let stored = backend.get_tc_token("880000002").await.unwrap().unwrap();
        assert_eq!(
            stored.token,
            vec![0x01, 0x02, 0x03],
            "real token must replace the placeholder even with an older timestamp"
        );
        assert_eq!(stored.token_timestamp, 1_700_000_000);
        assert_eq!(
            stored.sender_timestamp,
            Some(1_700_000_500),
            "sender_timestamp must survive the first real token"
        );
    }

    #[tokio::test]
    async fn identical_token_only_advances_timestamp() {
        let client = create_test_client().await;
        let backend = client.persistence_manager.backend();
        backend
            .put_tc_token(
                "880000003",
                &TcTokenEntry {
                    token: vec![0x0A, 0x0B],
                    token_timestamp: 1_700_000_000,
                    sender_timestamp: Some(1_700_000_100),
                },
            )
            .await
            .unwrap();

        let node = privacy_token_notification("880000003@lid", 1_700_000_900, vec![0x0A, 0x0B]);
        handle_privacy_token_notification(&client, &node).await;

        let stored = backend.get_tc_token("880000003").await.unwrap().unwrap();
        assert_eq!(stored.token, vec![0x0A, 0x0B]);
        assert_eq!(stored.token_timestamp, 1_700_000_900);
        assert_eq!(stored.sender_timestamp, Some(1_700_000_100));
    }

    #[tokio::test]
    async fn older_real_token_does_not_replace_newer_real_token() {
        let client = create_test_client().await;
        let backend = client.persistence_manager.backend();
        backend
            .put_tc_token(
                "880000004",
                &TcTokenEntry {
                    token: vec![0x11, 0x12],
                    token_timestamp: 1_700_000_900,
                    sender_timestamp: None,
                },
            )
            .await
            .unwrap();

        let node = privacy_token_notification("880000004@lid", 1_700_000_000, vec![0x21, 0x22]);
        handle_privacy_token_notification(&client, &node).await;

        let stored = backend.get_tc_token("880000004").await.unwrap().unwrap();
        assert_eq!(stored.token, vec![0x11, 0x12]);
        assert_eq!(stored.token_timestamp, 1_700_000_900);
    }

    #[tokio::test]
    async fn sender_lid_attribute_takes_precedence_over_from() {
        let client = create_test_client().await;
        let node = NodeBuilder::new("notification")
            .attr("type", "privacy_token")
            .attr("from", "880000005@s.whatsapp.net")
            .attr("sender_lid", "990000005@lid")
            .children([NodeBuilder::new("tokens")
                .children([NodeBuilder::new("token")
                    .attr("type", "trusted_contact")
                    .attr("t", "1700000000")
                    .bytes(vec![0x33])
                    .build()])
                .build()])
            .build();

        handle_privacy_token_notification(&client, &node).await;

        let backend = client.persistence_manager.backend();
        let stored = backend.get_tc_token("990000005").await.unwrap();
        assert!(stored.is_some(), "token should be stored under sender_lid");
        assert!(backend.get_tc_token("880000005").await.unwrap().is_none());
    }

    #[tokio::test]
    async fn pn_sender_without_mapping_is_stored_under_pn() {
        let client = create_test_client().await;
        let node =
            privacy_token_notification("880000006@s.whatsapp.net", 1_700_000_000, vec![0x44]);

        handle_privacy_token_notification(&client, &node).await;

        let stored = client
            .persistence_manager
            .backend()
            .get_tc_token("880000006")
            .await
            .unwrap();
        assert!(stored.is_some());
    }

    #[tokio::test]
    async fn notification_without_from_is_ignored() {
        let client = create_test_client().await;
        let node = NodeBuilder::new("notification")
            .attr("type", "privacy_token")
            .children([NodeBuilder::new("tokens")
                .children([NodeBuilder::new("token")
                    .attr("type", "trusted_contact")
                    .attr("t", "1700000000")
                    .bytes(vec![0x55])
                    .build()])
                .build()])
            .build();

        handle_privacy_token_notification(&client, &node).await;
    }

    #[tokio::test]
    async fn malformed_notification_is_ignored() {
        let client = create_test_client().await;
        let node = NodeBuilder::new("notification")
            .attr("type", "privacy_token")
            .attr("from", "880000007@lid")
            .build();

        handle_privacy_token_notification(&client, &node).await;
    }

    #[tokio::test]
    async fn business_notification_dispatches_status_update() {
        let client = create_test_client().await;
        let (_subscription, events) = subscribed(&client);

        let node = business_notification(
            "880000010@s.whatsapp.net",
            NodeBuilder::new("remove")
                .attr("jid", "880000010@s.whatsapp.net")
                .build(),
        );

        handle_business_notification(&client, &node).await;

        let event = tokio::time::timeout(std::time::Duration::from_secs(1), events.recv())
            .await
            .expect("business event should be dispatched promptly")
            .expect("business event");
        assert_eq!(event.kind(), EventKind::BusinessStatusUpdate);
        let Event::BusinessStatusUpdate(update) = &*event else {
            panic!("expected business status update, got {event:?}");
        };
        assert_eq!(update.jid.user, "880000010");
        assert_eq!(
            update.update_type,
            wacore::types::events::BusinessUpdateType::RemovedAsBusiness
        );
        assert_eq!(
            update.target_jid.as_ref().map(|j| j.user.as_str()),
            Some("880000010")
        );
        assert_eq!(update.timestamp.timestamp(), 1_700_000_000);
    }

    #[tokio::test]
    async fn verified_name_notification_carries_name() {
        let client = create_test_client().await;
        let (_subscription, events) = subscribed(&client);

        let node = business_notification(
            "880000011@s.whatsapp.net",
            NodeBuilder::new("verified_name")
                .attr("jid", "880000011@s.whatsapp.net")
                .attr("name", "Test Business")
                .build(),
        );

        handle_business_notification(&client, &node).await;

        let event = events.recv().await.expect("business event");
        let Event::BusinessStatusUpdate(update) = &*event else {
            panic!("expected business status update, got {event:?}");
        };
        assert_eq!(
            update.update_type,
            wacore::types::events::BusinessUpdateType::VerifiedNameChanged
        );
        assert_eq!(update.verified_name.as_deref(), Some("Test Business"));
    }

    #[tokio::test]
    async fn business_notification_without_from_does_not_dispatch() {
        let client = create_test_client().await;
        let (_subscription, events) = subscribed(&client);

        let node = NodeBuilder::new("notification")
            .attr("type", "business")
            .attr("t", "1700000000")
            .children([NodeBuilder::new("remove")
                .attr("jid", "880000012@s.whatsapp.net")
                .build()])
            .build();

        handle_business_notification(&client, &node).await;

        assert!(matches!(
            events.try_recv(),
            Err(async_channel::TryRecvError::Empty)
        ));
    }
}
