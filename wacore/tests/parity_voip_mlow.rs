#![cfg(feature = "voip-mlow")]
//! Phase 4 MLOW/Opus framing contract.
use wa_rs_core::voip::mlow::{
    AudioConfig, OpusMlowPacketError, depacketize_opus_from_mlow, packetize_opus_for_mlow,
};
#[test]
fn audio_config_defaults() {
    let config = AudioConfig::default();
    assert_eq!(config.sample_rate, 48_000);
    assert_eq!(config.channels, 1);
    assert_eq!(config.frame_ms, 20);
}
#[test]
fn opus_mlow_roundtrips_and_rejects_bad_packets() {
    let packet = packetize_opus_for_mlow(&[1, 2, 3]).unwrap();
    assert_eq!(depacketize_opus_from_mlow(&packet).unwrap(), vec![1, 2, 3]);
    assert_eq!(
        packetize_opus_for_mlow(&[]).unwrap_err(),
        OpusMlowPacketError::Empty
    );
    assert_eq!(
        depacketize_opus_from_mlow(&[0]).unwrap_err(),
        OpusMlowPacketError::Truncated
    );
}
