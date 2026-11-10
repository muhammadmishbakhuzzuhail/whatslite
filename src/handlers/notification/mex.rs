//! `<notification type="mex"><update op_name="…">{json}</update></notification>`
//!
//! WA Web routes MEX notifications by `op_name` (`WAWebHandleMexNotification`)
//! so the dispatcher survives GraphQL bundle rebuilds. Every stanza yields the
//! raw [`Event::MexNotification`]; the reachout timed-lock op additionally
//! yields the typed [`Event::ReachoutTimelockUpdate`] when someone listens.

use crate::client::Client;
use log::{debug, warn};
use serde::Deserialize;
use std::sync::Arc;
use wacore::types::events::{
    Event, EventKind, MexNotification, ReachoutTimelock, ReachoutTimelockUpdate,
};
use wacore_binary::node::{Node, NodeContent};

/// Handle a `<notification type="mex">` stanza.
pub(crate) fn handle_mex_notification(client: &Arc<Client>, node: &Node) {
    let Some(update_node) = node.get_optional_child("update") else {
        warn!(
            target: "Client/Mex",
            "mex notification missing <update> child: {}",
            crate::logging::StanzaSummary::from(node)
        );
        return;
    };

    let Some(op_name) = update_node.attrs().optional_string("op_name") else {
        warn!(
            target: "Client/Mex",
            "mex notification <update> missing op_name attribute: {}",
            crate::logging::StanzaSummary::from(node)
        );
        return;
    };

    // `from_str` skips the redundant UTF-8 validation `from_slice` does on a
    // `&str`, while a binary body still gets parsed from its raw bytes.
    let parsed = match update_node.content.as_ref() {
        Some(NodeContent::String(s)) => serde_json::from_str(s),
        Some(NodeContent::Bytes(b)) => serde_json::from_slice(b),
        _ => {
            warn!(target: "Client/Mex", "mex notification op={op_name} has no JSON body");
            return;
        }
    };
    let payload: serde_json::Value = match parsed {
        Ok(payload) => payload,
        Err(error) => {
            warn!(
                target: "Client/Mex",
                "mex notification op={op_name} JSON parse failed: {error}"
            );
            return;
        }
    };

    let mut attrs = node.attrs();
    let from = attrs.optional_jid("from");
    let stanza_id = attrs.optional_string("id").map(str::to_owned);
    let offline = attrs.optional_string("offline").map(str::to_owned);
    let op_name = op_name.to_owned();

    debug!(
        target: "Client/Mex",
        "mex notification received: op_name={op_name} offline={}",
        offline.is_some()
    );

    // WAWebHandleMexNotification routes this exact op with mexResponse.data;
    // WAWebMexReachoutTimelockNotificationHandler reads the notify field. Reuse
    // the pull state's Deserialize impl on the borrowed subtree: the raw twin
    // retains the entire original Value, including fields we do not model.
    if op_name == "NotificationUserReachoutTimelockUpdate"
        && client
            .core
            .event_bus
            .has_handler_for(EventKind::ReachoutTimelockUpdate)
        && let Some(state) = payload
            .get("data")
            .and_then(|data| data.get("xwa2_notify_account_reachout_timelock"))
            .filter(|state| state.is_object())
    {
        match ReachoutTimelock::deserialize(state) {
            Ok(state) => client
                .core
                .event_bus
                .dispatch(Event::ReachoutTimelockUpdate(
                    ReachoutTimelockUpdate::builder()
                        .state(state)
                        .maybe_from(from.clone())
                        .maybe_stanza_id(stanza_id.clone())
                        .maybe_offline(offline.clone())
                        .build(),
                )),
            Err(error) => {
                warn!(
                    target: "Client/Mex",
                    "mex notification op={op_name} invalid reachout state: {error}"
                );
            }
        }
    }

    client
        .core
        .event_bus
        .dispatch_with(EventKind::MexNotification, || {
            Event::MexNotification(
                MexNotification::builder()
                    .op_name(op_name)
                    .maybe_from(from)
                    .maybe_stanza_id(stanza_id)
                    .maybe_offline(offline)
                    .payload(payload)
                    .build(),
            )
        });
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::handlers::notification::NotificationHandler;
    use crate::handlers::traits::StanzaHandler;
    use crate::test_utils::create_test_client;
    use serde_json::{Value, json};
    use std::sync::Mutex;
    use wacore::iq::mex_operations::fetch_reachout_timelock;
    use wacore::types::events::{
        ChannelEventHandler, Event, EventHandler, EventInterest, EventKind,
    };
    use wacore_binary::builder::NodeBuilder;

    const OP: &str = "NotificationUserReachoutTimelockUpdate";

    // Synthetic, sanitized fixtures. WA Web 2.3000.1045368834:
    // WAWebHandleMexNotification takes `mexResponse.data`;
    // WAWebMexReachoutTimelockNotificationHandler reads the notify field and
    // passes the same three fields as WAWebGetReachoutTimelockJob's pull update.
    fn notification(op: &str, body: &str, bytes: bool) -> Node {
        let update = NodeBuilder::new("update").attr("op_name", op);
        let update = if bytes {
            update.bytes(body.as_bytes().to_vec())
        } else {
            update.string_content(body)
        };
        NodeBuilder::new("notification")
            .attr("type", "mex")
            .attr("from", "s.whatsapp.net")
            .attr("id", "mex-notif-1")
            .attr("offline", "1704067200")
            .children([update.build()])
            .build()
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

    async fn client_with_collector(
        capacity: usize,
    ) -> (
        Arc<Client>,
        async_channel::Receiver<Arc<Event>>,
        wacore::types::events::Subscription,
    ) {
        let client = create_test_client().await;
        let (handler, events) = ChannelEventHandler::with_capacity(capacity);
        let subscription = client.subscribe_handler(handler);
        (client, events, subscription)
    }

    #[tokio::test]
    async fn unrelated_op_dispatches_envelope_and_payload() {
        let (client, events, _subscription) = client_with_collector(4).await;
        let payload = json!({"data": {"xwa2_something": {"keep": [1, 2, 3]}}});
        deliver(
            &client,
            notification("SomeFutureOp", &payload.to_string(), false),
        )
        .await;

        let event = events.try_recv().expect("raw mex event");
        let Event::MexNotification(raw) = &*event else {
            panic!("unexpected event: {event:?}");
        };
        assert_eq!(event.kind(), EventKind::MexNotification);
        assert_eq!(raw.op_name, "SomeFutureOp");
        assert_eq!(raw.payload, payload);
        assert_eq!(
            raw.from.as_ref().map(ToString::to_string).as_deref(),
            Some("s.whatsapp.net")
        );
        assert_eq!(raw.stanza_id.as_deref(), Some("mex-notif-1"));
        assert_eq!(raw.offline.as_deref(), Some("1704067200"));
        assert!(events.try_recv().is_err(), "exactly one event");
    }

    #[tokio::test]
    async fn byte_body_parses_like_string_body() {
        let (client, events, _subscription) = client_with_collector(4).await;
        let payload = json!({"data": {"xwa2_something": "value"}});
        deliver(
            &client,
            notification("SomeFutureOp", &payload.to_string(), true),
        )
        .await;

        let event = events.try_recv().expect("raw mex event");
        let Event::MexNotification(raw) = &*event else {
            panic!("unexpected event: {event:?}");
        };
        assert_eq!(raw.payload, payload);
    }

    #[tokio::test]
    async fn missing_update_child_dispatches_nothing() {
        let (client, events, _subscription) = client_with_collector(4).await;
        let node = NodeBuilder::new("notification").attr("type", "mex").build();
        deliver(&client, node).await;
        assert!(events.try_recv().is_err());
    }

    #[tokio::test]
    async fn missing_op_name_dispatches_nothing() {
        let (client, events, _subscription) = client_with_collector(4).await;
        let node = NodeBuilder::new("notification")
            .attr("type", "mex")
            .children([NodeBuilder::new("update").string_content("{}").build()])
            .build();
        deliver(&client, node).await;
        assert!(events.try_recv().is_err());
    }

    #[tokio::test]
    async fn invalid_json_dispatches_nothing() {
        for bytes in [false, true] {
            let (client, events, _subscription) = client_with_collector(4).await;
            deliver(&client, notification("SomeFutureOp", "{invalid", bytes)).await;
            assert!(events.try_recv().is_err());
        }
    }

    #[tokio::test]
    async fn reachout_timelock_dispatches_typed_before_raw() {
        // Active, lifted and unrecognized/nullable enforcement retain the pull's
        // state, rather than inventing a deadline or mapping strings to an enum.
        for (state, bytes) in [
            (
                json!({"enforcement_type": "BIZ_QUALITY", "is_active": true,
                    "time_enforcement_ends": "1704153600"}),
                false,
            ),
            (
                json!({"enforcement_type": "WEB_COMPANION_ONLY", "is_active": false,
                    "time_enforcement_ends": null}),
                true,
            ),
            (
                json!({"enforcement_type": "FUTURE_ENFORCEMENT", "is_active": true,
                    "time_enforcement_ends": "1704240000", "future_field": {"keep": true}}),
                true,
            ),
            (
                json!({"enforcement_type": null, "is_active": null,
                    "time_enforcement_ends": null}),
                false,
            ),
            (json!({}), false),
        ] {
            let (client, events, _subscription) = client_with_collector(4).await;
            let payload = json!({
                "data": {"xwa2_notify_account_reachout_timelock": state},
                "extensions": {"preserve": [1, 2, 3]},
            });
            deliver(&client, notification(OP, &payload.to_string(), bytes)).await;

            let typed = events.try_recv().expect("typed update first");
            assert_eq!(typed.kind(), EventKind::ReachoutTimelockUpdate);
            let Event::ReachoutTimelockUpdate(update) = &*typed else {
                panic!("expected typed update, got: {typed:?}");
            };
            let pull: fetch_reachout_timelock::Response =
                serde_json::from_value(json!({ "xwa2_fetch_account_reachout_timelock": state }))
                    .expect("pull response decodes");
            let pull = pull
                .xwa2_fetch_account_reachout_timelock
                .expect("pull state present");
            // Same concrete DTO/Deserialize implementation, including optionality.
            let typed_state: &crate::features::ReachoutTimelock = &update.state;
            assert_eq!(typed_state.enforcement_type, pull.enforcement_type);
            assert_eq!(typed_state.is_active, pull.is_active);
            assert_eq!(
                typed_state.time_enforcement_ends,
                pull.time_enforcement_ends
            );

            let raw_event = events.try_recv().expect("raw twin second");
            let Event::MexNotification(raw) = &*raw_event else {
                panic!("expected raw twin, got: {raw_event:?}");
            };
            assert_eq!(raw.op_name, OP);
            assert_eq!(raw.payload, payload);
            assert_eq!(
                raw.from.as_ref().map(ToString::to_string).as_deref(),
                Some("s.whatsapp.net")
            );
            assert_eq!(raw.stanza_id.as_deref(), Some("mex-notif-1"));
            assert_eq!(raw.offline.as_deref(), Some("1704067200"));
            assert_eq!(update.from, raw.from);
            assert_eq!(update.stanza_id, raw.stanza_id);
            assert_eq!(update.offline, raw.offline);
            assert!(events.try_recv().is_err(), "exactly two events");
        }
    }

    #[tokio::test]
    async fn reachout_invalid_state_or_unrelated_op_stays_raw() {
        let bodies = [
            json!({}),
            json!({"data": null}),
            json!({"data": {}}),
            json!({"data": {"xwa2_notify_account_reachout_timelock": null}}),
            json!({"data": {"xwa2_notify_account_reachout_timelock": []}}),
            json!({"data": {"xwa2_notify_account_reachout_timelock": [null, true, "1704153600"]}}),
            json!({"data": {"xwa2_notify_account_reachout_timelock": "invalid"}}),
            json!({"data": {"xwa2_notify_account_reachout_timelock": {"is_active": "false"}}}),
            json!({"data": {"xwa2_notify_account_reachout_timelock": {"enforcement_type": 42}}}),
            json!({"data": {"xwa2_notify_account_reachout_timelock": {"time_enforcement_ends": 42}}}),
            json!({"data": {"xwa2_fetch_account_reachout_timelock": {"is_active": true}}}),
            Value::Null,
        ];
        for payload in bodies {
            assert_raw_only(OP, payload).await;
        }
        let payload = json!({"data": {
            "xwa2_notify_account_reachout_timelock": {"is_active": true}
        }});
        assert_raw_only("UnrelatedNotification", payload.clone()).await;
        assert_raw_only("NotificationUserReachoutTimelockUpdateExtra", payload).await;
    }

    async fn assert_raw_only(op: &str, payload: Value) {
        let (client, events, _subscription) = client_with_collector(4).await;
        deliver(&client, notification(op, &payload.to_string(), false)).await;

        let event = events.try_recv().expect("raw mex event");
        let Event::MexNotification(raw) = &*event else {
            panic!("invalid state or unrelated op must not invent a typed event");
        };
        assert_eq!(raw.op_name, op);
        assert_eq!(raw.payload, payload);
        assert!(events.try_recv().is_err(), "exactly one event");
    }

    #[tokio::test]
    async fn typed_and_raw_interests_are_independent() {
        struct NarrowCollector {
            kinds: EventInterest,
            events: Mutex<Vec<Arc<Event>>>,
        }

        impl EventHandler for NarrowCollector {
            fn interest(&self) -> EventInterest {
                self.kinds
            }

            fn handle_event(&self, event: Arc<Event>) {
                self.events.lock().expect("collector lock").push(event);
            }
        }

        for kind in [
            EventKind::ReachoutTimelockUpdate,
            EventKind::MexNotification,
        ] {
            let client = create_test_client().await;
            let collector = Arc::new(NarrowCollector {
                kinds: EventInterest::of(&[kind]),
                events: Mutex::new(Vec::new()),
            });
            let _subscription = client.subscribe_handler(collector.clone());
            // Missing source attributes remain None, not empty-string sentinels.
            let node = NodeBuilder::new("notification")
                .attr("type", "mex")
                .children([NodeBuilder::new("update")
                    .attr("op_name", OP)
                    .string_content(
                        r#"{"data":{"xwa2_notify_account_reachout_timelock":{"is_active":false}}}"#,
                    )
                    .build()])
                .build();
            deliver(&client, node).await;

            let events = collector.events.lock().expect("collector lock");
            assert_eq!(events.len(), 1, "only the interested twin is delivered");
            assert_eq!(events[0].kind(), kind);
            match &*events[0] {
                Event::ReachoutTimelockUpdate(update) => {
                    assert_eq!(update.state.is_active, Some(false));
                    assert!(
                        update.from.is_none()
                            && update.stanza_id.is_none()
                            && update.offline.is_none()
                    );
                }
                Event::MexNotification(raw) => {
                    assert!(raw.from.is_none() && raw.stanza_id.is_none() && raw.offline.is_none());
                }
                other => panic!("unexpected event: {other:?}"),
            }
        }
    }
}
