use std::sync::Arc;

use wa_rs_core::types::events::{
    ChannelEventHandler, CoreEventBus, Event, EventInterest, EventKind, PairingCode,
};

#[test]
fn event_kind_surface_has_eighty_variants() {
    assert_eq!(EventKind::ALL.len(), 80);
    assert_eq!(EventKind::CAPACITY, 128);
    assert_eq!(EventKind::Connected as usize, 0);
    assert_eq!(
        EventKind::ALL.last().copied(),
        Some(EventKind::LabelSublistUpdate)
    );
}

#[test]
fn event_interest_uses_one_bit_per_kind() {
    let interest = EventInterest::of(&[EventKind::Connected, EventKind::Messages]);
    assert!(interest.wants(EventKind::Connected));
    assert!(interest.wants(EventKind::Messages));
    assert!(!interest.wants(EventKind::Disconnected));

    let union = interest.union(EventInterest::of(&[EventKind::Disconnected]));
    assert!(union.wants(EventKind::Disconnected));
    assert!(!EventInterest::none().wants(EventKind::Connected));
    assert!(EventInterest::ALL.wants(EventKind::LabelSublistUpdate));
}

#[test]
fn event_payloads_serialize_with_their_kind() {
    let event = Event::PairingCode(
        PairingCode::builder()
            .code("ABCD1234".to_string())
            .timeout(std::time::Duration::from_secs(180))
            .build(),
    );
    assert_eq!(event.kind(), EventKind::PairingCode);

    let json = serde_json::to_value(&event).expect("event serializes");
    assert_eq!(json["PairingCode"]["code"], "ABCD1234");
    assert_eq!(json["PairingCode"]["timeout"]["secs"], 180);
}

#[test]
fn core_event_bus_filters_by_interest() {
    let bus = CoreEventBus::new();
    let (handler, rx) = ChannelEventHandler::with_capacity(8);
    let subscription = bus.subscribe(EventInterest::of(&[EventKind::PairingCode]), handler);

    bus.dispatch(Event::PairingCode(
        PairingCode::builder()
            .code("CODE0001".to_string())
            .timeout(std::time::Duration::ZERO)
            .build(),
    ));
    bus.dispatch(Event::Connected(
        wa_rs_core::types::events::Connected::builder().build(),
    ));

    let event = rx.try_recv().expect("interested event delivered");
    assert_eq!(event.kind(), EventKind::PairingCode);
    assert!(rx.try_recv().is_err(), "uninterested event must not arrive");

    drop(subscription);
    assert!(!bus.has_handler_for(EventKind::PairingCode));
}

#[test]
fn channel_handler_reports_overflow_without_blocking() {
    let bus = CoreEventBus::new();
    let (handler, _rx) = ChannelEventHandler::with_capacity(1);
    let _subscription = bus.subscribe_handler(handler.clone());

    bus.dispatch(Event::Connected(
        wa_rs_core::types::events::Connected::builder().build(),
    ));
    bus.dispatch(Event::Connected(
        wa_rs_core::types::events::Connected::builder().build(),
    ));

    let stats = handler.stats();
    assert_eq!(stats.enqueued, 1);
    assert_eq!(stats.dropped_full, 1);
    assert_eq!(stats.closed, 0);
}

#[test]
fn event_arc_is_shared_between_handlers() {
    let bus = CoreEventBus::new();
    let (first, first_rx) = ChannelEventHandler::with_capacity(4);
    let (second, second_rx) = ChannelEventHandler::with_capacity(4);
    let _first_sub = bus.subscribe_handler(first);
    let _second_sub = bus.subscribe_handler(second);

    bus.dispatch(Event::Connected(
        wa_rs_core::types::events::Connected::builder().build(),
    ));

    let a: Arc<Event> = first_rx.try_recv().expect("first handler");
    let b: Arc<Event> = second_rx.try_recv().expect("second handler");
    assert!(Arc::ptr_eq(&a, &b));
}
