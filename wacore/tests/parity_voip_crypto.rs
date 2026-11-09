#![cfg(feature = "voip")]
//! Phase 4 VoIP crypto/transport vector contract.

use wa_rs_core::voip::fec::{recover_missing, xor_parity};
use wa_rs_core::voip::rtcp::RtcpPacket;
use wa_rs_core::voip::rtp::RtpPacket;
use wa_rs_core::voip::sframe::SFrame;
use wa_rs_core::voip::srtp::SrtpContext;
use wa_rs_core::voip::stun::StunMessage;

#[test]
fn rtp_roundtrips() {
    let packet = RtpPacket {
        payload_type: 111,
        sequence: 7,
        timestamp: 9000,
        ssrc: 0x1234,
        payload: vec![1, 2, 3],
    };
    assert_eq!(RtpPacket::decode(&packet.encode()), Some(packet));
}

#[test]
fn rtcp_and_stun_headers_parse() {
    let rtcp = RtcpPacket {
        packet_type: 200,
        payload: vec![0; 8],
    };
    assert_eq!(RtcpPacket::decode(&rtcp.encode()), Some(rtcp));

    let mut stun = vec![0x00, 0x01, 0x00, 0x04];
    stun.extend_from_slice(&[0u8; 4]);
    stun.extend_from_slice(&[9u8; 12]);
    stun.extend_from_slice(&[1, 2, 3, 4]);
    let parsed = StunMessage::decode(&stun).expect("stun");
    assert_eq!(parsed.message_type, 1);
    assert_eq!(parsed.transaction_id, [9u8; 12]);
    assert_eq!(parsed.body, vec![1, 2, 3, 4]);
}

#[test]
fn sframe_roundtrips_and_rejects_tampering() {
    let frame = SFrame::new(7, &[0x42; 32]);
    let ciphertext = frame.encrypt(1, b"payload").expect("encrypt");
    assert_eq!(frame.decrypt(1, &ciphertext).expect("decrypt"), b"payload");
    let mut tampered = ciphertext;
    tampered[0] ^= 1;
    assert!(frame.decrypt(1, &tampered).is_err());
}

#[test]
fn srtp_and_fec_roundtrip() {
    let context = SrtpContext::new(1, [0x11; 32]);
    let protected = context.protect(b"media");
    assert_eq!(context.unprotect(&protected), b"media");

    let a = b"abc";
    let b = b"def";
    let parity = xor_parity(&[a, b]);
    assert_eq!(recover_missing(&[a, b], &parity, 1), b);
}
