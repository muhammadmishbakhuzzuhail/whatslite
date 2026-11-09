#![cfg(feature = "voip")]
//! Phase 4 call types/handler contract.
use chrono::Utc;
use wa_rs::features::call_log::{CallEvent, CallLog, GroupCallUpdate};
use wa_rs::handlers::call::CallHandler;
use wacore_binary::Jid;
#[test]
fn call_log_and_group_update_are_stable() {
    let log = CallLog::new("c", Jid::pn("15551234567"), Utc::now(), 10, false, false);
    assert_eq!(log.call_id, "c");
    let update = GroupCallUpdate::new(Jid::pn("123@g.us"), "c", vec![Jid::pn("15551234567")]);
    assert_eq!(update.participants.len(), 1);
    let event = CallEvent::Log(log);
    assert!(matches!(
        CallHandler::new().classify(event),
        CallEvent::Log(_)
    ));
}
