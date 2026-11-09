//! IQ pilot builders and result payloads (WhatsApp 2.3000.1052559453).
//!
//! Payload-only: envelope correlation and error conversion stay in the request
//! layer, so these are builders rather than `IqSpec` implementations.

use crate::iq::groups::GROUP_IQ_NAMESPACE;
use crate::request::InfoQuery;
use wacore_binary::builder::NodeBuilder;
use wacore_binary::jid::Jid;
use wacore_binary::node::{NodeContent, NodeRef};

/// Inputs of `SetGroupSubjectIq`'s `<subject>` stanza.
pub struct SetSubjectRequest<'a> {
    pub iq_to: &'a Jid,
    pub subject_element_value: &'a str,
}

pub fn build_set_subject(request: SetSubjectRequest<'_>) -> InfoQuery<'static> {
    let SetSubjectRequest {
        iq_to,
        subject_element_value,
    } = request;
    InfoQuery::set_ref(
        GROUP_IQ_NAMESPACE,
        iq_to,
        Some(NodeContent::Nodes(vec![
            NodeBuilder::new("subject")
                .string_content(subject_element_value)
                .build(),
        ])),
    )
}

#[derive(Debug, PartialEq, Eq)]
pub enum SetSubjectSuccess {
    Success,
}

pub fn parse_set_subject_payload(_response: &NodeRef<'_>) -> SetSubjectSuccess {
    SetSubjectSuccess::Success
}

/// Inputs of `AcceptGroupAddIq`'s `<accept>` stanza.
pub struct AcceptGroupAddRequest<'a> {
    pub iq_to: &'a Jid,
    pub accept_code: &'a str,
    pub accept_expiration: i64,
    pub accept_admin: &'a Jid,
}

pub fn build_accept_group_add(request: AcceptGroupAddRequest<'_>) -> InfoQuery<'static> {
    let AcceptGroupAddRequest {
        iq_to,
        accept_code,
        accept_expiration,
        accept_admin,
    } = request;
    InfoQuery::set_ref(
        GROUP_IQ_NAMESPACE,
        iq_to,
        Some(NodeContent::Nodes(vec![
            NodeBuilder::new("accept")
                .attr("code", accept_code)
                .attr("expiration", accept_expiration.to_string())
                .jid_attr("admin", accept_admin.clone())
                .build(),
        ])),
    )
}

#[derive(Debug, PartialEq, Eq)]
pub enum AcceptGroupAddSuccess {
    GroupJoinRequestSuccess,
    Success,
}

pub fn parse_accept_group_add_payload(response: &NodeRef<'_>) -> AcceptGroupAddSuccess {
    if response
        .get_children_by_tag("membership_approval_request")
        .take(2)
        .count()
        == 1
    {
        return AcceptGroupAddSuccess::GroupJoinRequestSuccess;
    }
    AcceptGroupAddSuccess::Success
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::request::InfoQueryType;
    use wacore_binary::jid::Jid;
    use wacore_binary::node::{Node, NodeContent};

    fn group() -> Jid {
        "120363000000000001@g.us".parse().unwrap()
    }

    fn single_child<'a>(query: &'a InfoQuery<'_>) -> &'a Node {
        match query.content.as_ref() {
            Some(NodeContent::Nodes(nodes)) if nodes.len() == 1 => &nodes[0],
            other => panic!("expected exactly one child node, got {other:?}"),
        }
    }

    fn text_content(node: &Node) -> Option<&str> {
        match node.content.as_ref() {
            Some(NodeContent::String(s)) => Some(s),
            _ => None,
        }
    }

    #[test]
    fn set_subject_builds_the_reviewed_stanza() {
        let jid = group();
        let query = build_set_subject(SetSubjectRequest {
            iq_to: &jid,
            subject_element_value: "New subject",
        });

        assert_eq!(query.namespace, "w:g2");
        assert_eq!(query.query_type, InfoQueryType::Set);
        assert_eq!(query.to, jid);

        let subject = single_child(&query);
        assert_eq!(subject.tag, "subject");
        assert_eq!(text_content(subject), Some("New subject"));
    }

    #[test]
    fn accept_group_add_builds_the_reviewed_stanza() {
        let jid = group();
        let admin: Jid = "15551234567@s.whatsapp.net".parse().unwrap();
        let query = build_accept_group_add(AcceptGroupAddRequest {
            iq_to: &jid,
            accept_code: "ABC123",
            accept_expiration: 1_730_000_000,
            accept_admin: &admin,
        });

        assert_eq!(query.namespace, "w:g2");
        assert_eq!(query.query_type, InfoQueryType::Set);
        assert_eq!(query.to, jid);

        let accept = single_child(&query);
        assert_eq!(accept.tag, "accept");
        assert_eq!(accept.attrs.get("code"), Some(&"ABC123".into()));
        // Numbers are carried as their decimal string; the encoder packs it as
        // a nibble string, matching the official client's numeric attribute.
        assert_eq!(accept.attrs.get("expiration"), Some(&"1730000000".into()));
        assert_eq!(
            accept.attrs.get("admin").map(|v| v.to_string_value()),
            Some(admin.to_string())
        );
        assert!(accept.content.is_none());
    }

    #[test]
    fn set_subject_payload_is_always_success() {
        let response = NodeBuilder::new("iq").attr("type", "result").build();
        assert_eq!(
            parse_set_subject_payload(&response.as_node_ref()),
            SetSubjectSuccess::Success
        );
    }

    /// Ported from the whatspec bundle conformance cases: a lone
    /// `<membership_approval_request/>` means the add turned into a join
    /// request, and anything else (absent, duplicated, unknown siblings,
    /// scalar content) means the member was added outright.
    #[test]
    fn accept_group_add_matches_reviewed_bundle_cases() {
        let response = NodeBuilder::new("iq")
            .attr("type", "result")
            .apply_content(None)
            .build();
        assert_eq!(
            parse_accept_group_add_payload(&response.as_node_ref()),
            AcceptGroupAddSuccess::Success,
            "absent"
        );
        let response = NodeBuilder::new("iq")
            .attr("type", "result")
            .apply_content(Some(NodeContent::Nodes(vec![
                NodeBuilder::new("membership_approval_request").build(),
            ])))
            .build();
        assert_eq!(
            parse_accept_group_add_payload(&response.as_node_ref()),
            AcceptGroupAddSuccess::GroupJoinRequestSuccess,
            "unique"
        );
        let response = NodeBuilder::new("iq")
            .attr("type", "result")
            .apply_content(Some(NodeContent::Nodes(vec![
                NodeBuilder::new("membership_approval_request").build(),
                NodeBuilder::new("membership_approval_request").build(),
            ])))
            .build();
        assert_eq!(
            parse_accept_group_add_payload(&response.as_node_ref()),
            AcceptGroupAddSuccess::Success,
            "duplicate"
        );
        let response = NodeBuilder::new("iq")
            .attr("type", "result")
            .apply_content(Some(NodeContent::Nodes(vec![
                NodeBuilder::new("future").build(),
            ])))
            .build();
        assert_eq!(
            parse_accept_group_add_payload(&response.as_node_ref()),
            AcceptGroupAddSuccess::Success,
            "unknown"
        );
        let response = NodeBuilder::new("iq")
            .attr("type", "result")
            .apply_content(Some(NodeContent::Nodes(vec![
                NodeBuilder::new("future").build(),
                NodeBuilder::new("membership_approval_request").build(),
            ])))
            .build();
        assert_eq!(
            parse_accept_group_add_payload(&response.as_node_ref()),
            AcceptGroupAddSuccess::GroupJoinRequestSuccess,
            "unique_with_unknown"
        );
        let response = NodeBuilder::new("iq")
            .attr("type", "result")
            .apply_content(Some(NodeContent::Nodes(vec![
                NodeBuilder::new("membership_approval_request").build(),
                NodeBuilder::new("future").build(),
                NodeBuilder::new("membership_approval_request").build(),
            ])))
            .build();
        assert_eq!(
            parse_accept_group_add_payload(&response.as_node_ref()),
            AcceptGroupAddSuccess::Success,
            "unknown_between_duplicates"
        );
        let response = NodeBuilder::new("iq")
            .attr("type", "result")
            .apply_content(Some(NodeContent::Bytes(vec![
                0x66, 0x75, 0x74, 0x75, 0x72, 0x65,
            ])))
            .build();
        assert_eq!(
            parse_accept_group_add_payload(&response.as_node_ref()),
            AcceptGroupAddSuccess::Success,
            "scalar"
        );
    }
}
