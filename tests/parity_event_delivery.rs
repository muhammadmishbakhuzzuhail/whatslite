use std::sync::Arc;
use std::time::Duration;

use wa_rs::bot::{CallbackEventHandler, EventDelivery};
use wa_rs::test_utils::create_test_client;
use wa_rs::types::events::{Connected, Event, EventHandler, EventInterest, EventKind, PairingCode};

fn event(code: &str) -> Arc<Event> {
    Arc::new(Event::PairingCode(
        PairingCode::builder()
            .code(code.to_string())
            .timeout(Duration::ZERO)
            .build(),
    ))
}

#[tokio::test]
async fn ordered_delivery_processes_callbacks_in_enqueue_order() {
    let client = create_test_client().await;
    let (tx, rx) = async_channel::unbounded();
    let handler = CallbackEventHandler::from_callback(
        &client,
        EventInterest::of(&[EventKind::PairingCode]),
        EventDelivery::Ordered { capacity: 8 },
        move |event, _client| {
            let tx = tx.clone();
            async move {
                if let Event::PairingCode(code) = &*event {
                    tx.send(code.code.clone()).await.expect("send order");
                }
            }
        },
    );
    let _subscription = client.subscribe_handler(handler.clone());

    handler.handle_event(event("ONE"));
    handler.handle_event(event("TWO"));
    handler.handle_event(event("THREE"));

    assert_eq!(rx.recv().await.unwrap(), "ONE");
    assert_eq!(rx.recv().await.unwrap(), "TWO");
    assert_eq!(rx.recv().await.unwrap(), "THREE");
    assert_eq!(handler.stats().accepted, 3);
    assert_eq!(handler.stats().callbacks_completed, 3);
    handler.cancel();
}

#[tokio::test]
async fn bounded_delivery_drops_newest_when_mailbox_is_full() {
    let client = create_test_client().await;
    let (_release_tx, release_rx) = async_channel::bounded::<()>(1);
    let (started_tx, started_rx) = async_channel::bounded::<()>(1);
    let handler = CallbackEventHandler::from_callback(
        &client,
        EventInterest::ALL,
        EventDelivery::BoundedConcurrent {
            capacity: 1,
            max_concurrency: 1,
        },
        move |_event, _client| {
            let release = release_rx.clone();
            let started = started_tx.clone();
            async move {
                let _ = started.try_send(());
                let _ = release.recv().await;
            }
        },
    );
    let _subscription = client.subscribe_handler(handler.clone());

    handler.handle_event(event("running"));
    started_rx.recv().await.expect("callback started");
    handler.handle_event(event("queued"));
    handler.handle_event(event("overflow"));

    let stats = handler.stats();
    assert_eq!(stats.accepted, 2);
    assert_eq!(stats.dropped_full, 1);
    handler.cancel();
}

#[tokio::test]
async fn callback_interest_filters_before_enqueue() {
    let client = create_test_client().await;
    let (tx, rx) = async_channel::unbounded();
    let handler = CallbackEventHandler::from_callback(
        &client,
        EventInterest::of(&[EventKind::PairingCode]),
        EventDelivery::default(),
        move |_event, _client| {
            let tx = tx.clone();
            async move {
                tx.send(()).await.expect("callback");
            }
        },
    );
    let _subscription = client.subscribe_handler(handler.clone());

    handler.handle_event(Arc::new(Event::Connected(Connected::builder().build())));
    handler.handle_event(event("CODE0001"));

    rx.recv().await.expect("pairing code callback");
    assert!(rx.try_recv().is_err());
    assert_eq!(handler.stats().accepted, 1);
    handler.cancel();
}
