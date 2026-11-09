use wa_rs::types::events::{EventInterest, EventKind};
#[test]
fn handler_event_surface_is_available() {
    let interest = EventInterest::of(&[EventKind::Messages, EventKind::Receipt]);
    assert!(interest.wants(EventKind::Messages));
    assert!(interest.wants(EventKind::Receipt));
    assert_eq!(EventKind::ALL.len(), 80);
}
