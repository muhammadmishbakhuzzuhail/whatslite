use wa_rs_core::iq::props::{PROPS_NAMESPACE, PROPS_PROTOCOL_VERSION};
#[test]
fn props_namespace_and_version_are_retained() {
    assert_eq!(PROPS_NAMESPACE, "abt");
    assert_eq!(PROPS_PROTOCOL_VERSION, "1");
}
