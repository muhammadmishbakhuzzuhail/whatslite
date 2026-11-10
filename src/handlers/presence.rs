//! Handler for incoming `<presence>` stanzas.

use super::traits::StanzaHandler;
use crate::client::Client;
use async_trait::async_trait;
use log::debug;
use std::sync::Arc;
use wacore::types::events::{Event, PresenceUpdate};
use wacore::types::presence::PresenceStatus;
use wacore_binary::node::Node;

/// Handler for `<presence>` stanzas.
///
/// Parses global availability updates and dispatches `Event::Presence` on the
/// core event bus. Chat-scoped activity (`<chatstate>`) has its own handler.
#[derive(Default)]
pub struct PresenceHandler;

#[async_trait]
impl StanzaHandler for PresenceHandler {
    fn tag(&self) -> &'static str {
        "presence"
    }

    async fn handle(&self, client: Arc<Client>, node: Arc<Node>, _cancelled: &mut bool) -> bool {
        let mut attrs = node.attrs();

        let Some(from) = attrs.optional_jid("from") else {
            debug!(
                target: "PresenceHandler",
                "Presence stanza missing or invalid 'from' attribute"
            );
            return true;
        };

        // Only an explicit "unavailable" means offline; unknown type values keep
        // WhatsApp Web's optimistic default of available.
        let unavailable = attrs.optional_string("type") == Some("unavailable");

        let last_seen = attrs
            .optional_unix_time("last")
            .and_then(wacore::time::from_secs);

        debug!(
            target: "PresenceHandler",
            "Received presence from {from}: unavailable={unavailable}"
        );

        client.core.event_bus.dispatch(Event::Presence(
            PresenceUpdate::builder()
                .from(from)
                .status(if unavailable {
                    PresenceStatus::Unavailable
                } else {
                    PresenceStatus::Available
                })
                .maybe_last_seen(last_seen)
                .build(),
        ));

        true
    }
}

#[cfg(test)]
mod tests {
    use std::sync::Arc;

    use super::*;
    use crate::test_utils::create_test_client;
    use wacore::types::events::{ChannelEventHandler, Event, Subscription};
    use wacore_binary::builder::NodeBuilder;

    const FROM: &str = "12025550100@s.whatsapp.net";

    async fn dispatch(client: &Arc<Client>, node: Node) -> bool {
        PresenceHandler
            .handle(client.clone(), Arc::new(node), &mut false)
            .await
    }

    fn subscribed(client: &Arc<Client>) -> (Subscription, async_channel::Receiver<Arc<Event>>) {
        let (handler, events) = ChannelEventHandler::with_capacity(8);
        (client.subscribe_handler(handler), events)
    }

    #[tokio::test]
    async fn availability_event_retains_identity_and_last_seen() {
        let client = create_test_client().await;
        let (_subscription, events) = subscribed(&client);

        for (wire_type, status) in [
            (None, PresenceStatus::Available),
            (Some("available"), PresenceStatus::Available),
            (Some("unavailable"), PresenceStatus::Unavailable),
            (Some("future-value"), PresenceStatus::Available),
        ] {
            let mut builder = NodeBuilder::new("presence")
                .attr("from", FROM)
                .attr("last", "1700000000");
            if let Some(value) = wire_type {
                builder = builder.attr("type", value);
            }

            assert!(dispatch(&client, builder.build()).await);

            let event = events.recv().await.expect("presence event");
            let Event::Presence(update) = &*event else {
                panic!("expected presence event, got {:?}", event);
            };
            assert_eq!(update.from.to_string(), FROM);
            assert_eq!(update.status, status);
            assert_eq!(
                update.last_seen.expect("last seen").timestamp(),
                1_700_000_000
            );
        }
    }

    #[tokio::test]
    async fn presence_without_last_attribute_has_no_last_seen() {
        let client = create_test_client().await;
        let (_subscription, events) = subscribed(&client);

        assert!(
            dispatch(
                &client,
                NodeBuilder::new("presence")
                    .attr("from", FROM)
                    .attr("type", "unavailable")
                    .build(),
            )
            .await
        );

        let event = events.recv().await.expect("presence event");
        let Event::Presence(update) = &*event else {
            panic!("expected presence event, got {:?}", event);
        };
        assert_eq!(update.status, PresenceStatus::Unavailable);
        assert!(update.last_seen.is_none());
    }

    #[tokio::test]
    async fn missing_or_invalid_from_is_ignored() {
        let client = create_test_client().await;
        let (_subscription, events) = subscribed(&client);

        let nodes = [
            NodeBuilder::new("presence").build(),
            NodeBuilder::new("presence")
                .attr("from", "not-a-jid")
                .attr("type", "available")
                .build(),
        ];

        for node in nodes {
            assert!(dispatch(&client, node).await);
        }

        assert!(matches!(
            events.try_recv(),
            Err(async_channel::TryRecvError::Empty)
        ));
    }
}
