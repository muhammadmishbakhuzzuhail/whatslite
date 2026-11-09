//! Profile IQ types for setting own status text.
//!
//! ## Wire Format
//!
//! ### Set Status Text
//! ```xml
//! <iq xmlns="status" type="set" to="s.whatsapp.net" id="...">
//!   <status>Hello world!</status>
//! </iq>
//! ```
//! Response: `<iq type="result" .../>`

use crate::iq::spec::IqSpec;
use crate::request::InfoQuery;
use wacore_binary::builder::NodeBuilder;
use wacore_binary::jid::{Jid, SERVER_JID};
use wacore_binary::node::{Node, NodeContent};

/// IQ spec for setting the user's own status text (about).
pub struct SetStatusTextSpec {
    text: String,
}

impl SetStatusTextSpec {
    pub fn new(text: impl Into<String>) -> Self {
        Self { text: text.into() }
    }
}

impl IqSpec for SetStatusTextSpec {
    type Response = ();

    fn build_iq(&self) -> InfoQuery<'static> {
        InfoQuery::set(
            "status",
            Jid::new("", SERVER_JID),
            Some(NodeContent::Nodes(vec![
                NodeBuilder::new("status")
                    .string_content(self.text.as_str())
                    .build(),
            ])),
        )
    }

    fn parse_response(&self, _response: &Node) -> Result<Self::Response, anyhow::Error> {
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use wacore_binary::builder::NodeBuilder;
    use wacore_binary::node::NodeContent;

    #[test]
    fn test_set_status_text_iq() {
        let spec = SetStatusTextSpec::new("Hello world!");
        let iq = spec.build_iq();

        assert_eq!(iq.namespace, "status");
        assert_eq!(iq.query_type, crate::request::InfoQueryType::Set);
        assert_eq!(iq.to.server, "s.whatsapp.net");
        assert!(iq.to.user.is_empty());

        let children = match &iq.content {
            Some(NodeContent::Nodes(children)) => children,
            other => panic!("expected Nodes content, got {other:?}"),
        };
        assert_eq!(children.len(), 1);
        assert_eq!(children[0].tag, "status");
        assert_eq!(
            children[0].content,
            Some(NodeContent::String("Hello world!".to_string()))
        );
    }

    #[test]
    fn test_set_status_text_parse_response() {
        let spec = SetStatusTextSpec::new("Hello world!");
        let response = NodeBuilder::new("iq").attr("type", "result").build();

        assert!(spec.parse_response(&response).is_ok());
    }
}
