//! `<notification type="w:gp2">` handling.
//!
//! Parses the notification into typed actions, emits one [`Event::GroupUpdate`]
//! per action, patches or invalidates the group metadata cache, and keeps
//! forwarding the raw stanza via [`Event::Notification`].
//!
//! Reference: WhatsApp Web `WAWebHandleGroupNotification`.
//!
//! Deferred: sender-key rotation, group-metadata lane guarding, and the
//! unrelated `server_sync` / `newsletter` / `mex` / `disappearing_mode`
//! handlers, whose subsystems are not ported yet.

use crate::client::Client;
use log::{debug, warn};
use std::sync::Arc;
use wacore::stanza::groups::{GroupNotificationAction, GroupParticipantInfo, parse_groups_dirty};
use wacore::types::events::{Event, EventKind, GroupUpdate};
use wacore_binary::jid::{Jid, JidExt as _};
use wacore_binary::node::Node;

/// Hand the repeated envelope fields to every update but the last, which takes
/// them, avoiding an owned clone per action.
#[inline]
fn clone_or_take_last<T: Clone>(value: &mut Option<T>, is_last: bool) -> Option<T> {
    if is_last { value.take() } else { value.clone() }
}

enum ParticipantChange {
    Add,
    Remove,
}

/// Handle a `<notification type="w:gp2">` stanza.
pub(crate) async fn handle_group_notification(client: &Arc<Client>, node: &Node) {
    let node_ref = node.as_node_ref();

    // `<groups_dirty>` is the one `w:gp2` stanza about the server itself: `from`
    // is `s.whatsapp.net`, not a group, so it must not reach the group parser.
    if let Some(groups) = parse_groups_dirty(&node_ref) {
        handle_groups_dirty(client, groups).await;
        client
            .core
            .event_bus
            .dispatch(Event::Notification(Arc::new(node.clone())));
        return;
    }

    let mut notification = match client.parse_group_notification(&node_ref).await {
        Ok(Some(notification)) => notification,
        Ok(None) => {
            warn!(target: "Client/Group", "w:gp2 notification missing 'from' attribute");
            return;
        }
        Err(error) => {
            warn!(target: "Client/Group", "w:gp2 notification rejected: {error}");
            return;
        }
    };

    let timestamp = i64::try_from(notification.timestamp)
        .ok()
        .and_then(wacore::time::from_secs)
        .unwrap_or_else(chrono::Utc::now);

    let actions = std::mem::take(&mut notification.actions);
    let action_count = actions.len();

    for (action_index, action) in actions.into_iter().enumerate() {
        // Patch the cache granularly for participant changes, matching WA Web's
        // addParticipantInfo/removeParticipantInfo, so no metadata IQ is needed.
        match &action {
            GroupNotificationAction::Add { participants, .. } => {
                patch_cached_participants(
                    client,
                    &notification.group_jid,
                    participants,
                    ParticipantChange::Add,
                )
                .await;
            }
            GroupNotificationAction::Remove { participants, .. } => {
                patch_cached_participants(
                    client,
                    &notification.group_jid,
                    participants,
                    ParticipantChange::Remove,
                )
                .await;
            }
            GroupNotificationAction::Modify { participants, .. } => {
                // The action carries only the new participants, not the old JID,
                // so the cached membership cannot be patched in place.
                debug!(
                    target: "Client/Group",
                    "Group modify ({} participants) for {}: invalidating cached metadata",
                    participants.len(), notification.group_jid
                );
                let cache = client.get_group_cache().await;
                cache.invalidate(&notification.group_jid).await;
            }
            _ => {}
        }

        debug!(
            target: "Client/Group",
            "Group notification: group={}, action={}",
            notification.group_jid, action.tag_name()
        );

        let is_last_action = action_index + 1 == action_count;
        client
            .core
            .event_bus
            .dispatch_with(EventKind::GroupUpdate, || {
                Event::GroupUpdate(Box::new(
                    GroupUpdate::builder()
                        .group_jid(notification.group_jid.clone())
                        .maybe_notification_id(clone_or_take_last(
                            &mut notification.notification_id,
                            is_last_action,
                        ))
                        .maybe_notify(clone_or_take_last(&mut notification.notify, is_last_action))
                        .maybe_offline(clone_or_take_last(
                            &mut notification.offline,
                            is_last_action,
                        ))
                        .action_index(u32::try_from(action_index).unwrap_or(u32::MAX))
                        .maybe_participant(clone_or_take_last(
                            &mut notification.participant,
                            is_last_action,
                        ))
                        .maybe_participant_pn(clone_or_take_last(
                            &mut notification.participant_pn,
                            is_last_action,
                        ))
                        .maybe_participant_username(clone_or_take_last(
                            &mut notification.participant_username,
                            is_last_action,
                        ))
                        .maybe_participant_country_code(clone_or_take_last(
                            &mut notification.participant_country_code,
                            is_last_action,
                        ))
                        .timestamp(timestamp)
                        .is_lid_addressing_mode(notification.is_lid_addressing_mode)
                        .has_incomplete_participant_information(
                            notification.has_incomplete_participant_information,
                        )
                        .action(Box::new(action))
                        .build(),
                ))
            });
    }

    // Keep forwarding the raw stanza for hosts that only consume the legacy event.
    client
        .core
        .event_bus
        .dispatch(Event::Notification(Arc::new(node.clone())));
}

/// Add or remove participants in the cached metadata, when there is one.
///
/// A cold cache is left cold: the next send or `group_info` query re-fetches,
/// which is equivalent to inventing a one-action membership list here.
async fn patch_cached_participants(
    client: &Arc<Client>,
    group_jid: &Jid,
    participants: &[GroupParticipantInfo],
    change: ParticipantChange,
) {
    let cache = client.get_group_cache().await;
    let Some(mut info) = cache.get(group_jid).await else {
        debug!(
            target: "Client/Group",
            "Group cache cold for {group_jid}: nothing to patch"
        );
        return;
    };

    match change {
        ParticipantChange::Add => {
            for participant in participants {
                if !info
                    .participants
                    .iter()
                    .any(|existing| existing.is_same_user_as(&participant.jid))
                {
                    info.participants.push(participant.jid.clone());
                }
            }
        }
        ParticipantChange::Remove => {
            info.participants.retain(|existing| {
                !participants
                    .iter()
                    .any(|removed| existing.is_same_user_as(&removed.jid))
            });
        }
    }

    cache.insert(group_jid.clone(), info).await;
    debug!(
        target: "Client/Group",
        "Patched group cache for {group_jid}: {} participants changed",
        participants.len()
    );
}

/// Drop the cached metadata of every group the server marked stale.
///
/// Invalidation is idempotent, so no per-entry lane or generation guard is
/// needed here: the next send or `group_info` query re-fetches.
async fn handle_groups_dirty(client: &Arc<Client>, groups: Vec<Jid>) {
    let cache = client.get_group_cache().await;
    for group_jid in groups {
        debug!(
            target: "Client/Group",
            "groups_dirty: invalidating cached metadata for {group_jid}"
        );
        cache.invalidate(&group_jid).await;
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_utils::create_test_client;
    use wacore::client::context::GroupInfo;
    use wacore::stanza::groups::GroupNotificationAction;
    use wacore::types::events::{ChannelEventHandler, Event};
    use wacore::types::message::AddressingMode;
    use wacore_binary::builder::NodeBuilder;
    use wacore_binary::jid::Jid;
    use wacore_binary::node::Node;

    const GROUP: &str = "120363000000000001@g.us";
    const ADMIN: &str = "15550000001@s.whatsapp.net";

    fn group_jid() -> Jid {
        GROUP.parse().expect("group jid")
    }

    fn pn(user: &str) -> Jid {
        Jid::pn(user)
    }

    fn notification(child: Node) -> Node {
        NodeBuilder::new("notification")
            .attr("type", "w:gp2")
            .attr("from", GROUP)
            .attr("id", "notif-1")
            .attr("t", "1700000000")
            .attr("participant", ADMIN)
            .children([child])
            .build()
    }

    fn participant(jid: &Jid) -> Node {
        NodeBuilder::new("participant")
            .attr("jid", jid.clone())
            .build()
    }

    async fn subscribe(
        client: &Arc<Client>,
    ) -> (
        wacore::types::events::Subscription,
        async_channel::Receiver<Arc<Event>>,
    ) {
        let (handler, events) = ChannelEventHandler::with_capacity(16);
        (client.subscribe_handler(handler), events)
    }

    async fn seed_group(client: &Arc<Client>, participants: Vec<Jid>) {
        client
            .get_group_cache()
            .await
            .insert(
                group_jid(),
                GroupInfo::new(participants, AddressingMode::Pn),
            )
            .await;
    }

    async fn cached_participants(client: &Arc<Client>) -> Option<Vec<Jid>> {
        client
            .get_group_cache()
            .await
            .get(&group_jid())
            .await
            .map(|info| info.participants)
    }

    fn group_updates(events: &async_channel::Receiver<Arc<Event>>) -> Vec<GroupUpdate> {
        let mut updates = Vec::new();
        while let Ok(event) = events.try_recv() {
            if let Event::GroupUpdate(update) = &*event {
                updates.push((**update).clone());
            }
        }
        updates
    }

    #[tokio::test]
    async fn add_action_patches_cached_participants_and_dispatches_update() {
        let client = create_test_client().await;
        let (_sub, events) = subscribe(&client).await;
        seed_group(&client, vec![pn("15550000002"), pn("15550000003")]).await;

        let added = pn("15550000004");
        let node = notification(
            NodeBuilder::new("add")
                .children([participant(&added)])
                .build(),
        );
        handle_group_notification(&client, &node).await;

        let participants = cached_participants(&client)
            .await
            .expect("group metadata stays cached after an add");
        assert_eq!(
            participants,
            vec![pn("15550000002"), pn("15550000003"), added],
        );

        let mut updates = Vec::new();
        let mut raw_notifications = 0;
        while let Ok(event) = events.try_recv() {
            match &*event {
                Event::GroupUpdate(update) => updates.push((**update).clone()),
                Event::Notification(node) => {
                    assert_eq!(node.tag, "notification");
                    raw_notifications += 1;
                }
                other => panic!("unexpected event: {other:?}"),
            }
        }
        assert_eq!(raw_notifications, 1, "raw stanza forwarded once");
        assert_eq!(updates.len(), 1);
        let update = updates.pop().expect("one group update");
        assert_eq!(update.group_jid, group_jid());
        assert_eq!(update.notification_id.as_deref(), Some("notif-1"));
        assert_eq!(update.action_index, 0);
        assert_eq!(update.timestamp.timestamp(), 1_700_000_000);
        assert!(matches!(
            *update.action,
            GroupNotificationAction::Add { .. }
        ));
    }

    #[tokio::test]
    async fn remove_action_patches_cached_participants() {
        let client = create_test_client().await;
        let (_sub, events) = subscribe(&client).await;
        seed_group(
            &client,
            vec![pn("15550000002"), pn("15550000003"), pn("15550000004")],
        )
        .await;

        let removed = pn("15550000003");
        let node = notification(
            NodeBuilder::new("remove")
                .children([participant(&removed)])
                .build(),
        );
        handle_group_notification(&client, &node).await;

        assert_eq!(
            cached_participants(&client).await,
            Some(vec![pn("15550000002"), pn("15550000004")]),
        );
        let updates = group_updates(&events);
        assert_eq!(updates.len(), 1);
        assert!(matches!(
            *updates[0].action,
            GroupNotificationAction::Remove { .. }
        ));
    }

    #[tokio::test]
    async fn modify_action_invalidates_cached_metadata() {
        let client = create_test_client().await;
        let (_sub, events) = subscribe(&client).await;
        seed_group(&client, vec![pn("15550000002")]).await;

        let migrated = pn("15550000009");
        let node = notification(
            NodeBuilder::new("modify")
                .children([participant(&migrated)])
                .build(),
        );
        handle_group_notification(&client, &node).await;

        assert_eq!(cached_participants(&client).await, None);
        let updates = group_updates(&events);
        assert_eq!(updates.len(), 1);
        assert!(matches!(
            *updates[0].action,
            GroupNotificationAction::Modify { .. }
        ));
    }

    #[tokio::test]
    async fn add_without_cached_metadata_leaves_the_cache_empty() {
        let client = create_test_client().await;
        let (_sub, _events) = subscribe(&client).await;

        let node = notification(
            NodeBuilder::new("add")
                .children([participant(&pn("15550000002"))])
                .build(),
        );
        handle_group_notification(&client, &node).await;

        assert_eq!(cached_participants(&client).await, None);
    }

    #[tokio::test]
    async fn groups_dirty_invalidates_each_named_group() {
        let client = create_test_client().await;
        let (_sub, events) = subscribe(&client).await;

        let first: Jid = "120363000000000002@g.us".parse().expect("group jid");
        let second: Jid = "120363000000000003@g.us".parse().expect("group jid");
        let cache = client.get_group_cache().await;
        for jid in [&first, &second] {
            cache
                .insert(
                    jid.clone(),
                    GroupInfo::new(vec![pn("15550000002")], AddressingMode::Pn),
                )
                .await;
        }

        let node = NodeBuilder::new("notification")
            .attr("type", "w:gp2")
            .attr("from", "s.whatsapp.net")
            .attr("id", "dirty-1")
            .attr("t", "1700000000")
            .children([NodeBuilder::new("groups_dirty")
                .children([
                    NodeBuilder::new("group").attr("jid", first.clone()).build(),
                    NodeBuilder::new("group")
                        .attr("jid", second.clone())
                        .build(),
                ])
                .build()])
            .build();
        handle_group_notification(&client, &node).await;

        assert!(cache.get(&first).await.is_none(), "first group invalidated");
        assert!(
            cache.get(&second).await.is_none(),
            "second group invalidated"
        );
        let raw = events.try_recv().expect("raw notification forwarded");
        assert!(matches!(&*raw, Event::Notification(node) if node.tag == "notification"));
        assert!(
            events.try_recv().is_err(),
            "groups_dirty emits no GroupUpdate"
        );
    }

    #[tokio::test]
    async fn missing_from_dispatches_nothing() {
        let client = create_test_client().await;
        let (_sub, events) = subscribe(&client).await;

        let node = NodeBuilder::new("notification")
            .attr("type", "w:gp2")
            .attr("id", "notif-1")
            .children([NodeBuilder::new("add")
                .children([participant(&pn("15550000002"))])
                .build()])
            .build();
        handle_group_notification(&client, &node).await;

        assert!(events.try_recv().is_err(), "no events for a malformed node");
    }

    #[tokio::test]
    async fn each_action_dispatches_its_own_update_with_increasing_index() {
        let client = create_test_client().await;
        let (_sub, events) = subscribe(&client).await;

        let node = NodeBuilder::new("notification")
            .attr("type", "w:gp2")
            .attr("from", GROUP)
            .attr("id", "notif-1")
            .attr("t", "1700000000")
            .attr("participant", ADMIN)
            .children([
                NodeBuilder::new("add")
                    .children([participant(&pn("15550000002"))])
                    .build(),
                NodeBuilder::new("subject")
                    .attr("subject", "renamed")
                    .build(),
            ])
            .build();
        handle_group_notification(&client, &node).await;

        let updates = group_updates(&events);
        assert_eq!(updates.len(), 2, "one update per action");
        assert_eq!(updates[0].action_index, 0);
        assert_eq!(updates[1].action_index, 1);
        assert!(matches!(
            *updates[0].action,
            GroupNotificationAction::Add { .. }
        ));
        assert!(matches!(
            *updates[1].action,
            GroupNotificationAction::Subject { .. }
        ));
        // Every update repeats the envelope metadata.
        assert_eq!(updates[0].notification_id, updates[1].notification_id);
        assert_eq!(updates[0].participant, updates[1].participant);
    }

    #[tokio::test]
    async fn unknown_action_parses_through_the_legacy_adapter() {
        let client = create_test_client().await;
        let (_sub, events) = subscribe(&client).await;

        let node = notification(NodeBuilder::new("future_action").build());
        handle_group_notification(&client, &node).await;

        let updates = group_updates(&events);
        assert_eq!(updates.len(), 1);
        assert!(matches!(
            &*updates[0].action,
            GroupNotificationAction::Unknown { tag, .. } if tag == "future_action"
        ));
    }
}
