use wa_rs_core::iq::groups::{GROUP_IQ_NAMESPACE, GROUP_SIZE_LIMIT, MemberAddMode};
use wa_rs_core::iq::props::{AbProp, PROPS_NAMESPACE};
use wa_rs_core::iq::usync::{UsyncContext, UsyncMode};
use wa_rs_core::protocol::ProtocolNode;
#[test]
fn social_iq_domains_are_stable() {
    assert_eq!(GROUP_IQ_NAMESPACE, "w:g2");
    assert_eq!(GROUP_SIZE_LIMIT, 257);
    assert_eq!(MemberAddMode::AdminAdd.as_str(), "admin_add");
    assert_eq!(PROPS_NAMESPACE, "abt");
    assert_eq!(UsyncMode::Query.as_str(), "query");
    assert_eq!(UsyncContext::Background.as_str(), "background");
}
#[test]
fn abprop_roundtrips_through_node() {
    let prop = AbProp {
        config_code: 1,
        config_value: "v".into(),
        config_expo_key: None,
    };
    let node = prop.clone().into_node();
    let back = AbProp::try_from_node(&node).expect("parse");
    assert_eq!(back.config_code, 1);
    assert_eq!(back.config_value, "v");
}
