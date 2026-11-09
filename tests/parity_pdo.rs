//! Phase 2 PDO/history surface contract.

use wa_rs::pdo::{PendingPdoRequest, new_pdo_cache};
use wa_rs::types::events::{Event, EventKind};
use wa_rs::wacore::types::events::{Connected, LazyHistorySync};

#[test]
fn pdo_cache_and_pending_type_are_reachable() {
    let _cache = new_pdo_cache();
    let _: fn() = || {
        let _ = std::any::type_name::<PendingPdoRequest>();
    };
}

#[test]
fn history_event_kind_and_lazy_payload_are_reachable() {
    let event = Event::Connected(Connected::builder().build());
    assert_eq!(event.kind(), EventKind::Connected);
    let _: fn() = || {
        let _ = std::any::type_name::<LazyHistorySync>();
    };
}
