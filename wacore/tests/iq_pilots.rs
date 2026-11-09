use wa_rs_core::iq::usync::{UsyncContext, UsyncMode};
#[test]
fn usync_pilots_modes_are_stable() {
    assert_eq!(UsyncMode::Full.as_str(), "full");
    assert_eq!(UsyncContext::Interactive.as_str(), "interactive");
    assert_eq!(UsyncContext::Message.as_str(), "message");
}
