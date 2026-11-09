//! Community `create` request projection.
//!
//! Wire format:
//! ```xml
//! <iq xmlns="w:g2" type="set" to="g.us" id="...">
//!   <create subject="...">
//!     <parent default_membership_approval_mode="request_required"/>
//!     <allow_non_admin_sub_group_creation/>
//!     <create_general_chat/>
//!   </create>
//! </iq>
//! ```
//! `parent`, `allow_non_admin_sub_group_creation` and `create_general_chat`
//! are each emitted only when the caller asks for them.

use crate::request::InfoQuery;
use wacore_binary::builder::NodeBuilder;
use wacore_binary::jid::{GROUP_SERVER, Jid};
use wacore_binary::node::NodeContent;

/// Parameters of a community `create` request.
pub struct CommunityCreateRequest<'a> {
    pub any_subject: &'a str,
    /// `Some(())` requests membership approval for the parent group.
    pub has_parent_group_default_membership_approval_mode: Option<()>,
    pub has_allow_non_admin_sub_group_creation: bool,
    pub has_create_general_chat: bool,
}

/// Build the `<iq xmlns="w:g2"><create .../></iq>` projection.
pub fn build_community_create(request: CommunityCreateRequest<'_>) -> InfoQuery<'static> {
    let mut children = Vec::new();
    let mut parent = NodeBuilder::new("parent");
    if request
        .has_parent_group_default_membership_approval_mode
        .is_some()
    {
        parent = parent.attr("default_membership_approval_mode", "request_required");
    }
    children.push(parent.build());
    if request.has_allow_non_admin_sub_group_creation {
        children.push(NodeBuilder::new("allow_non_admin_sub_group_creation").build());
    }
    if request.has_create_general_chat {
        children.push(NodeBuilder::new("create_general_chat").build());
    }
    InfoQuery::set(
        "w:g2",
        Jid::new("", GROUP_SERVER),
        Some(NodeContent::Nodes(vec![
            NodeBuilder::new("create")
                .attr("subject", request.any_subject)
                .children(children)
                .build(),
        ])),
    )
}

#[cfg(test)]
mod tests {
    use super::*;
    use wacore_binary::node::Node;

    fn create_node<'a>(iq: &'a InfoQuery<'static>) -> &'a Node {
        match &iq.content {
            Some(NodeContent::Nodes(nodes)) => &nodes[0],
            other => panic!("expected Nodes content, got {other:?}"),
        }
    }

    #[test]
    fn build_defaults_to_bare_parent() {
        let iq = build_community_create(CommunityCreateRequest {
            any_subject: "My community",
            has_parent_group_default_membership_approval_mode: None,
            has_allow_non_admin_sub_group_creation: false,
            has_create_general_chat: false,
        });

        assert_eq!(iq.namespace, "w:g2");
        assert_eq!(iq.query_type, crate::request::InfoQueryType::Set);
        assert_eq!(iq.to.server, "g.us");
        assert!(iq.to.user.is_empty());

        let create = create_node(&iq);
        assert_eq!(create.tag, "create");
        assert_eq!(
            create.attrs().optional_string("subject"),
            Some("My community")
        );

        let children = match &create.content {
            Some(NodeContent::Nodes(children)) => children,
            other => panic!("expected create children, got {other:?}"),
        };
        assert_eq!(children.len(), 1);
        assert_eq!(children[0].tag, "parent");
        assert_eq!(
            children[0]
                .attrs()
                .optional_string("default_membership_approval_mode"),
            None
        );
    }

    #[test]
    fn build_with_all_flags() {
        let iq = build_community_create(CommunityCreateRequest {
            any_subject: "Closed",
            has_parent_group_default_membership_approval_mode: Some(()),
            has_allow_non_admin_sub_group_creation: true,
            has_create_general_chat: true,
        });

        let create = create_node(&iq);
        let children = match &create.content {
            Some(NodeContent::Nodes(children)) => children,
            other => panic!("expected create children, got {other:?}"),
        };
        let tags: Vec<&str> = children.iter().map(|n| n.tag.as_str()).collect();
        assert_eq!(
            tags,
            [
                "parent",
                "allow_non_admin_sub_group_creation",
                "create_general_chat"
            ]
        );
        assert_eq!(
            children[0]
                .attrs()
                .optional_string("default_membership_approval_mode"),
            Some("request_required")
        );
        assert!(children[1].content.is_none());
        assert!(children[2].content.is_none());
    }
}
