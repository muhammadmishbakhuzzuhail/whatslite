//! Phase 2 messaging/media helper contract: WebP detection, ZIP packaging,
//! sticker-pack URLs and media-retry receipt shape.

use wa_rs_core::media_retry::{
    MediaRetryResult, build_media_retry_receipt, parse_media_retry_notification,
};
use wa_rs_core::sticker_pack::sticker_pack_data_url;
use wa_rs_core::webp::is_animated;
use wa_rs_core::zip::ZipWriter;
use wacore_binary::builder::NodeBuilder;
use wacore_binary::marshal::{marshal, unmarshal_ref};

#[test]
fn webp_detection_rejects_short_and_non_webp_payloads() {
    assert!(!is_animated(b"short"));
    let mut not_webp = b"RIFF0000NOPE".to_vec();
    not_webp.extend_from_slice(&[0u8; 16]);
    assert!(!is_animated(&not_webp));
}

#[test]
fn zip_writer_emits_a_stored_zip_with_named_files() {
    let mut zip = ZipWriter::new();
    zip.add_file("a.txt", b"alpha");
    zip.add_file("b.bin", &[1, 2, 3]);
    let bytes = zip.finish();
    assert!(bytes.starts_with(b"PK\x03\x04"));
    assert!(bytes.windows(5).any(|w| w == b"a.txt"));
    assert!(bytes.windows(5).any(|w| w == b"b.bin"));
    assert!(bytes.windows(4).any(|w| w == b"PK\x05\x06"));
}

#[test]
fn sticker_pack_data_url_is_deterministic() {
    let first = sticker_pack_data_url("pack-id", "en");
    let second = sticker_pack_data_url("pack-id", "en");
    assert_eq!(first, second);
    assert!(first.contains("pack-id"));
    assert!(first.contains("en"));
}

#[test]
fn media_retry_receipt_has_expected_shape_and_parses_success() {
    let own = wacore_binary::Jid::pn("111");
    let chat = wacore_binary::Jid::pn("222");
    let receipt = build_media_retry_receipt(
        &own,
        "MSG1",
        &chat,
        false,
        Some(&wacore_binary::Jid::pn("333")),
        &[1, 2, 3],
        &[4u8; 12],
    );
    assert_eq!(receipt.tag.as_str(), "receipt");
    assert_eq!(
        receipt.attrs().optional_string("type"),
        Some("server-error")
    );
    assert!(receipt.get_optional_child_by_tag(&["encrypt"]).is_some());
    assert!(receipt.get_optional_child_by_tag(&["rmr"]).is_some());

    let error = NodeBuilder::new("notification")
        .attr("id", "MSG1")
        .children([NodeBuilder::new("error").attr("code", "2").build()])
        .build();
    let bytes = marshal(&error).expect("marshal");
    let nr = unmarshal_ref(&bytes[1..]).expect("unmarshal");
    let parsed = parse_media_retry_notification(&nr, &[0x42; 32]).expect("parse");
    assert!(matches!(parsed, MediaRetryResult::NotFound));
}
