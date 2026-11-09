//! Phase 2 protocol contract: binary round-trip, JID identity and XML escaping.

use wacore_binary::builder::NodeBuilder;
use wacore_binary::jid::{Jid, JidExt};
use wacore_binary::marshal::{marshal, unmarshal_ref};
use wacore_binary::node::NodeContentRef;

#[test]
fn node_roundtrips_with_attributes_and_bytes() {
    let node = NodeBuilder::new("message")
        .attr("type", "text")
        .attr("to", "15551234567@s.whatsapp.net")
        .bytes(vec![1, 2, 3])
        .build();
    let bytes = marshal(&node).expect("marshal");
    let decoded = unmarshal_ref(&bytes[1..]).expect("unmarshal");
    assert_eq!(decoded.tag, "message");
    assert_eq!(
        decoded.get_attr("type").and_then(|v| v.as_str()),
        Some("text")
    );
    match decoded.content.as_deref() {
        Some(NodeContentRef::Bytes(bytes)) => assert_eq!(bytes.as_ref(), &[1, 2, 3]),
        _ => panic!("expected bytes content"),
    }
}

#[test]
fn jid_variants_and_device_forms_are_distinct() {
    let pn: Jid = "15551234567@s.whatsapp.net".parse().unwrap();
    let lid: Jid = "100000000000001@lid".parse().unwrap();
    let device = Jid::pn_device("15551234567", 7);
    assert!(pn.is_pn());
    assert!(lid.is_lid());
    assert!(device.is_ad());
    assert_eq!(device.device, 7);
    assert_eq!(pn.to_non_ad(), pn);
}
