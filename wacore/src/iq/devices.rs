//! Device management IQ specs.

use std::time::Duration;

use crate::iq::spec::IqSpec;
use crate::request::InfoQuery;
use wacore_binary::builder::NodeBuilder;
use wacore_binary::jid::{Jid, SERVER_JID};
use wacore_binary::node::{Node, NodeContent};

/// WA Web uses a 3s timeout for the logout IQ (Socket/Model.js).
const LOGOUT_TIMEOUT: Duration = Duration::from_secs(3);

/// Deregister this companion device from the WhatsApp account.
pub struct RemoveCompanionDeviceSpec {
    jid: Jid,
}

impl RemoveCompanionDeviceSpec {
    pub fn new(jid: &Jid) -> Self {
        Self { jid: jid.clone() }
    }
}

impl IqSpec for RemoveCompanionDeviceSpec {
    type Response = ();

    fn build_iq(&self) -> InfoQuery<'static> {
        let child = NodeBuilder::new("remove-companion-device")
            .attr("jid", self.jid.to_string())
            .attr("reason", "user_initiated")
            .build();

        InfoQuery::set(
            "md",
            Jid::new("", SERVER_JID),
            Some(NodeContent::Nodes(vec![child])),
        )
        .with_timeout(LOGOUT_TIMEOUT)
    }

    fn parse_response(&self, _response: &Node) -> Result<Self::Response, anyhow::Error> {
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn iq_structure() {
        let jid: Jid = "551199887766:87@s.whatsapp.net".parse().unwrap();
        let spec = RemoveCompanionDeviceSpec::new(&jid);
        let iq = spec.build_iq();

        assert_eq!(iq.namespace, "md");
        assert_eq!(iq.query_type, crate::request::InfoQueryType::Set);
        assert_eq!(iq.to.server, "s.whatsapp.net");
        assert!(iq.to.user.is_empty());
        assert_eq!(iq.timeout, Some(Duration::from_secs(3)));

        let children = match &iq.content {
            Some(NodeContent::Nodes(c)) => c,
            other => panic!("expected Nodes content, got {other:?}"),
        };
        assert_eq!(children.len(), 1);
        assert_eq!(children[0].tag, "remove-companion-device");
        assert_eq!(
            children[0].attrs().optional_string("jid"),
            Some("551199887766:87@s.whatsapp.net")
        );
        assert_eq!(
            children[0].attrs().optional_string("reason"),
            Some("user_initiated")
        );
    }

    #[test]
    fn parse_response_is_ok() {
        let jid: Jid = "551199887766:87@s.whatsapp.net".parse().unwrap();
        let spec = RemoveCompanionDeviceSpec::new(&jid);
        let response = wacore_binary::builder::NodeBuilder::new("iq")
            .attr("type", "result")
            .build();

        assert!(spec.parse_response(&response).is_ok());
    }
}
