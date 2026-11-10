//! `<notification type="newsletter">` live updates.
//!
//! Each `<message server_id>` in `<live_updates><messages>` carries the same
//! `<forwards_count>`, `<votes>` and `<reactions>` shapes as the history IQ, so
//! both paths read the helpers in [`crate::features::newsletter`]. The first
//! notification after subscribing is a snapshot of recent messages and later
//! ones carry only the messages that changed, so a message missing from an
//! update has not changed, not dropped to zero. The raw notification is always
//! forwarded too.

use crate::client::Client;
use crate::features::newsletter::{parse_count_child, parse_poll_votes, parse_reaction_counts};
use std::sync::Arc;
use wacore::types::events::{
    Event, NewsletterLiveUpdate, NewsletterLiveUpdateMessage, NewsletterLiveUpdatePollVote,
    NewsletterLiveUpdateReaction,
};
use wacore_binary::node::Node;

/// Handle a `<notification type="newsletter">` stanza.
pub(crate) fn handle_newsletter_notification(client: &Arc<Client>, node: &Node) {
    let Some(newsletter_jid) = node.attrs().optional_jid("from") else {
        return;
    };

    if let Some(live_updates) = node.get_optional_child("live_updates")
        && let Some(messages_node) = live_updates.get_optional_child("messages")
        && let Some(children) = messages_node.children()
    {
        let messages: Vec<_> = children
            .iter()
            .filter(|n| n.tag == "message")
            .filter_map(|msg_node| {
                let msg = msg_node.as_node_ref();
                let server_id = msg
                    .get_attr("server_id")
                    .and_then(|v| v.as_str())
                    .and_then(|s| s.parse::<u64>().ok())?;

                let reactions = parse_reaction_counts(&msg)
                    .into_iter()
                    .map(|r| {
                        NewsletterLiveUpdateReaction::builder()
                            .code(r.code)
                            .count(r.count)
                            .build()
                    })
                    .collect();

                let votes = parse_poll_votes(&msg)
                    .into_iter()
                    .map(|v| {
                        NewsletterLiveUpdatePollVote::builder()
                            .option_hash(v.option_hash)
                            .count(v.count)
                            .build()
                    })
                    .collect();

                Some(
                    NewsletterLiveUpdateMessage::builder()
                        .server_id(server_id)
                        .reactions(reactions)
                        .votes(votes)
                        .maybe_forwards_count(parse_count_child(&msg, "forwards_count"))
                        .build(),
                )
            })
            .collect();

        if !messages.is_empty() {
            client.core.event_bus.dispatch(Event::NewsletterLiveUpdate(
                NewsletterLiveUpdate::builder()
                    .newsletter_jid(newsletter_jid)
                    .messages(messages)
                    .build(),
            ));
        }
    }

    // Also dispatch raw notification for backward compatibility
    client
        .core
        .event_bus
        .dispatch(Event::Notification(Arc::new(node.clone())));
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::handlers::notification::NotificationHandler;
    use crate::handlers::traits::StanzaHandler;
    use crate::test_utils::create_test_client;
    use wacore::types::events::{ChannelEventHandler, Event};
    use wacore_binary::builder::NodeBuilder;

    const NEWSLETTER: &str = "120363000000000001@newsletter";

    fn message(server_id: Option<&str>) -> Node {
        let message = NodeBuilder::new("message");
        let message = match server_id {
            Some(id) => message.attr("server_id", id),
            None => message,
        };
        message
            .children([
                NodeBuilder::new("reactions")
                    .children([
                        NodeBuilder::new("reaction")
                            .attr("code", "👍")
                            .attr("count", "3")
                            .build(),
                        NodeBuilder::new("reaction")
                            .attr("code", "❤️")
                            .attr("count", "1")
                            .build(),
                    ])
                    .build(),
                NodeBuilder::new("votes")
                    .children([NodeBuilder::new("vote")
                        .attr("count", "7")
                        .bytes(vec![7u8; 32])
                        .build()])
                    .build(),
                NodeBuilder::new("forwards_count")
                    .attr("count", "2")
                    .build(),
            ])
            .build()
    }

    fn notification(from: Option<&str>, messages: Option<Vec<Node>>) -> Node {
        let builder = NodeBuilder::new("notification").attr("type", "newsletter");
        let builder = match from {
            Some(from) => builder.attr("from", from),
            None => builder,
        };
        let builder = match messages {
            Some(messages) => builder.children([NodeBuilder::new("live_updates")
                .children([NodeBuilder::new("messages").children(messages).build()])
                .build()]),
            None => builder,
        };
        builder.build()
    }

    async fn client_with_collector() -> (
        Arc<Client>,
        async_channel::Receiver<Arc<Event>>,
        wacore::types::events::Subscription,
    ) {
        let client = create_test_client().await;
        let (handler, events) = ChannelEventHandler::with_capacity(8);
        let subscription = client.subscribe_handler(handler);
        (client, events, subscription)
    }

    async fn deliver(client: &Arc<Client>, node: Node) {
        let mut cancelled = false;
        assert!(
            NotificationHandler
                .handle(client.clone(), Arc::new(node), &mut cancelled)
                .await
        );
        assert!(!cancelled);
    }

    #[tokio::test]
    async fn live_update_dispatches_typed_counters_then_raw_twin() {
        let (client, events, _subscription) = client_with_collector().await;
        deliver(
            &client,
            notification(Some(NEWSLETTER), Some(vec![message(Some("9001"))])),
        )
        .await;

        let event = events.try_recv().expect("live update event");
        let Event::NewsletterLiveUpdate(update) = &*event else {
            panic!("unexpected event: {event:?}");
        };
        assert_eq!(update.newsletter_jid.user, "120363000000000001");
        assert_eq!(update.messages.len(), 1);
        let message = &update.messages[0];
        assert_eq!(message.server_id, 9001);
        assert_eq!(message.reactions.len(), 2);
        assert_eq!(message.reactions[0].code, "👍");
        assert_eq!(message.reactions[0].count, 3);
        assert_eq!(message.reactions[1].code, "❤️");
        assert_eq!(message.reactions[1].count, 1);
        assert_eq!(message.votes.len(), 1);
        assert_eq!(message.votes[0].option_hash, [7u8; 32]);
        assert_eq!(message.votes[0].count, 7);
        assert_eq!(message.forwards_count, Some(2));

        let raw = events.try_recv().expect("raw notification twin");
        assert!(matches!(&*raw, Event::Notification(_)));
        assert!(events.try_recv().is_err(), "exactly two events");
    }

    #[tokio::test]
    async fn message_without_server_id_is_skipped_but_raw_is_kept() {
        let (client, events, _subscription) = client_with_collector().await;
        deliver(
            &client,
            notification(Some(NEWSLETTER), Some(vec![message(None)])),
        )
        .await;

        let raw = events.try_recv().expect("raw notification twin");
        assert!(matches!(&*raw, Event::Notification(_)));
        assert!(
            events.try_recv().is_err(),
            "no typed event for a skipped message"
        );
    }

    #[tokio::test]
    async fn missing_from_dispatches_nothing() {
        let (client, events, _subscription) = client_with_collector().await;
        deliver(
            &client,
            notification(None, Some(vec![message(Some("9001"))])),
        )
        .await;

        assert!(events.try_recv().is_err(), "no raw twin without a source");
    }

    #[tokio::test]
    async fn newsletter_without_live_updates_dispatches_raw_only() {
        let (client, events, _subscription) = client_with_collector().await;
        deliver(&client, notification(Some(NEWSLETTER), None)).await;

        let raw = events.try_recv().expect("raw notification twin");
        assert!(matches!(&*raw, Event::Notification(_)));
        assert!(
            events.try_recv().is_err(),
            "no typed event without messages"
        );
    }
}
