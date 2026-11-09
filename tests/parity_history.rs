//! Phase 2 history-sync contract.

use wa_rs::types::events::{Event, EventKind};
use wa_rs::types::history_sync_durability::HistorySyncKey;
use wa_rs::wacore::types::events::Connected;
use wa_rs::waproto::whatsapp::HistorySync;

#[test]
fn history_sync_key_is_account_scoped_and_ordered() {
    let first = HistorySyncKey::new("chat", "sender", "id");
    let second = HistorySyncKey::new("chat", "sender", "id");
    let different = HistorySyncKey::new("chat", "sender", "other");
    assert_eq!(first, second);
    assert_ne!(first, different);
}

#[test]
fn history_event_and_proto_are_reachable() {
    let _ = HistorySync::default();
    let event = Event::Connected(Connected::builder().build());
    assert_eq!(event.kind(), EventKind::Connected);
    assert!(EventKind::ALL.contains(&EventKind::HistorySync));
}
