//! Public consumer checks for decoded IQ structure errors.
//!
//! Ports the passive/newsletter/empty-attribute cases of upstream's
//! `tests/iq_parse_errors.rs`. The group projections there need `GroupError`,
//! which lands separately.

use anyhow::Context;
use wa_rs_core::iq::{
    groups::GetGroupInviteLinkIq, newsletter::MyAddOnsSpec, node::IqParseError,
    passive::PassiveModeSpec, spec::IqSpec,
};
use wacore_binary::builder::NodeBuilder;

#[test]
fn passive_response_requires_the_requested_child_and_keeps_its_name() {
    for (passive, expected, other) in [(true, "passive", "active"), (false, "active", "passive")] {
        let response = NodeBuilder::new("iq")
            .attr("type", "result")
            .children([NodeBuilder::new(other).build()])
            .build();
        let error = PassiveModeSpec::new(passive)
            .parse_response(&response)
            .unwrap_err();
        assert!(matches!(error.downcast_ref::<IqParseError>(),
                Some(IqParseError::MissingChild { tag }) if tag == expected));
        assert_eq!(error.to_string(), format!("<{expected}> child not found"));

        let response = NodeBuilder::new("iq")
            .attr("type", "result")
            .children([NodeBuilder::new(expected).build()])
            .build();
        assert!(
            PassiveModeSpec::new(passive)
                .parse_response(&response)
                .is_ok()
        );
    }
}

#[test]
fn newsletter_missing_child_and_attribute_remain_distinct() {
    let jid = "120363000000000001@newsletter".parse().unwrap();
    let spec = MyAddOnsSpec::new(&jid, 20);
    let missing_child = NodeBuilder::new("iq").attr("type", "result").build();
    let error = spec.parse_response(&missing_child).unwrap_err();
    assert!(matches!(error.downcast_ref::<IqParseError>(),
            Some(IqParseError::MissingChild { tag }) if tag == "my_addons"));

    let missing_attribute = NodeBuilder::new("iq")
        .children([NodeBuilder::new("my_addons")
            .children([NodeBuilder::new("messages")
                .attr("unrelated", "fixture-secret-must-not-appear")
                .build()])
            .build()])
        .build();
    let error = spec
        .parse_response(&missing_attribute)
        .context("reading newsletter add-ons")
        .unwrap_err();
    assert!(matches!(error.downcast_ref::<IqParseError>(),
            Some(IqParseError::MissingAttribute { name }) if name == "jid"));
    assert_eq!(
        error.root_cause().to_string(),
        "missing required attribute jid"
    );
    assert!(!format!("{error:?}").contains("fixture-secret-must-not-appear"));
}

#[test]
fn empty_present_attribute_is_still_accepted_by_the_helper() {
    let jid = "120363000000000001@g.us".parse().unwrap();
    let response = NodeBuilder::new("iq")
        .children([NodeBuilder::new("invite").attr("code", "").build()])
        .build();
    assert_eq!(
        GetGroupInviteLinkIq::new(&jid, false)
            .parse_response(&response)
            .unwrap(),
        "https://chat.whatsapp.com/"
    );
}
