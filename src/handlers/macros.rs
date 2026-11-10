//! Shared handler macros.

/// Extract the required `from` JID attribute from a node, or log a warning
/// and return from the enclosing function.
///
/// Handler functions across the crate share the exact same pattern: pull
/// `from`, bail out with a warning if it is missing. This macro collapses that
/// five-line `match` into one line while preserving the (optional) log target.
///
/// # Variants
///
/// * `require_from_jid!(node, "Context")` — warns without a target, then `return;`.
/// * `require_from_jid!(node, target: "Client/Foo", "Context")` — warns with target.
#[macro_export]
macro_rules! require_from_jid {
    ($node:expr, $context:literal) => {
        match $node.attrs().optional_jid("from") {
            Some(jid) => jid,
            None => {
                ::log::warn!(concat!($context, " missing required 'from' attribute"));
                return;
            }
        }
    };
    ($node:expr, target: $target:literal, $context:literal) => {
        match $node.attrs().optional_jid("from") {
            Some(jid) => jid,
            None => {
                ::log::warn!(
                    target: $target,
                    concat!($context, " missing required 'from' attribute"),
                );
                return;
            }
        }
    };
}

#[cfg(test)]
mod tests {
    use wacore_binary::builder::NodeBuilder;
    use wacore_binary::jid::Jid;
    use wacore_binary::node::Node;

    fn extract_from(node: &Node, out: &mut Option<Jid>) {
        let jid = require_from_jid!(node, target: "Test/Macro", "test notification");
        *out = Some(jid);
    }

    #[test]
    fn require_from_jid_extracts_the_jid() {
        let node = NodeBuilder::new("notification")
            .attr("from", "15551234567@s.whatsapp.net")
            .build();
        let mut out = None;

        extract_from(&node, &mut out);

        assert_eq!(
            out.expect("jid should be extracted"),
            Jid::pn("15551234567")
        );
    }

    #[test]
    fn require_from_jid_returns_early_without_the_attribute() {
        let node = NodeBuilder::new("notification").build();
        let mut out = None;

        extract_from(&node, &mut out);

        assert!(out.is_none(), "macro must return before using the value");
    }
}
