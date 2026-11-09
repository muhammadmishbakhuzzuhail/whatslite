#![cfg(feature = "voip-mlow")]
use wa_rs_core::voip::mlow::OpusMlowPacketError;
#[test]
fn mlow_errors_have_stable_messages() {
    assert!(OpusMlowPacketError::Empty.to_string().contains("empty"));
    assert!(
        OpusMlowPacketError::Truncated
            .to_string()
            .contains("truncated")
    );
}
