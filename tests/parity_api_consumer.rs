//! Phase 0 public-surface consumer. Later phases extend this file with the
//! APIs they add; Phase 0 pins the crate aliases, client and event surface.

use std::sync::Arc;

use wa_rs::bot::{CallbackEventHandler, EventDelivery};
use wa_rs::types::events::{Connected, Event, EventHandler, EventInterest, EventKind, PairingCode};

#[test]
fn phase_zero_event_consumer_compiles_and_runs() {
    let interest = EventInterest::of(&[EventKind::Connected, EventKind::Messages]);
    assert!(interest.wants(EventKind::Connected));
    assert!(!interest.wants(EventKind::PairingCode));

    let event = Event::Connected(Connected::builder().build());
    assert_eq!(event.kind(), EventKind::Connected);

    let delivery = EventDelivery::default();
    let _ = delivery;

    let _: fn(Arc<dyn EventHandler>) = |_handler| {};
}

#[test]
fn phase_zero_pairing_payload_is_constructible() {
    let event = Event::PairingCode(
        PairingCode::builder()
            .code("ABCD1234".to_string())
            .timeout(std::time::Duration::from_secs(180))
            .build(),
    );
    assert_eq!(event.kind(), EventKind::PairingCode);
}

#[test]
fn phase_zero_callback_handler_type_is_reachable() {
    let _: fn() = || {
        let _ = std::any::type_name::<CallbackEventHandler>();
    };
}
