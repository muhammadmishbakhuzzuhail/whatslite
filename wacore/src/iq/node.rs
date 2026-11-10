//! Helper functions for parsing protocol nodes in IQ responses.
//!
//! These functions provide a consistent way to extract required and optional
//! children/attributes from protocol nodes with clear error messages.

use crate::protocol::ProtocolNode;
use wacore_binary::jid::Jid;
use wacore_binary::node::Node;

/// Missing structure in a decoded IQ response.
///
/// The IQ parsers keep their `anyhow::Error` return type; callers can recover
/// this error by downcasting, including through additional context. It records
/// only the expected field name, never attribute values or response contents.
/// This is not a server rejection or a failure to decode the binary stanza.
/// It covers the required-node helpers, not every IQ parser failure.
///
/// ```
/// use wa_rs_core::iq::{node::IqParseError, passive::PassiveModeSpec, spec::IqSpec};
/// use wacore_binary::builder::NodeBuilder;
///
/// let response = NodeBuilder::new("iq").attr("type", "result").build();
/// let error = PassiveModeSpec::new(true)
///     .parse_response(&response)
///     .unwrap_err();
/// assert!(matches!(error.downcast_ref::<IqParseError>(),
///     Some(IqParseError::MissingChild { tag }) if tag == "passive"));
/// ```
#[derive(Debug, Clone, PartialEq, Eq, thiserror::Error)]
#[non_exhaustive]
pub enum IqParseError {
    /// A required child with this tag was absent.
    #[error("<{tag}> child not found")]
    MissingChild { tag: String },
    /// A required attribute with this name was absent.
    #[error("missing required attribute {name}")]
    MissingAttribute { name: String },
}

/// Get a required child node by tag, returning an error if not found.
pub fn required_child<'a>(node: &'a Node, tag: &str) -> Result<&'a Node, anyhow::Error> {
    node.get_optional_child(tag).ok_or_else(|| {
        IqParseError::MissingChild {
            tag: tag.to_owned(),
        }
        .into()
    })
}

/// Get an optional child node by tag.
pub fn optional_child<'a>(node: &'a Node, tag: &str) -> Option<&'a Node> {
    node.get_optional_child(tag)
}

/// Get a required string attribute, returning an error if not found.
pub fn required_attr(node: &Node, key: &str) -> Result<String, anyhow::Error> {
    node.attrs()
        .optional_string(key)
        .map(str::to_string)
        .ok_or_else(|| {
            IqParseError::MissingAttribute {
                name: key.to_owned(),
            }
            .into()
        })
}

/// Get an optional string attribute.
pub fn optional_attr<'a>(node: &'a Node, key: &str) -> Option<&'a str> {
    node.attrs().optional_string(key)
}

/// Get an optional u64 attribute.
pub fn optional_u64(node: &Node, key: &str) -> Option<u64> {
    node.attrs().optional_u64(key)
}

/// Get a required JID attribute, returning an error if not found or invalid.
///
/// This properly handles JID attributes stored as either:
/// - Direct JID values (binary protocol stores JIDs as structured data)
/// - String values that need to be parsed
pub fn required_jid(node: &Node, key: &str) -> Result<Jid, anyhow::Error> {
    node.attrs()
        .optional_jid(key)
        .ok_or_else(|| anyhow::anyhow!("missing required attribute {key}"))
}

/// Get an optional JID attribute.
///
/// This properly handles JID attributes stored as either:
/// - Direct JID values (binary protocol stores JIDs as structured data)
/// - String values that need to be parsed
///
/// Returns `Ok(None)` if the attribute is missing.
/// Note: Parse errors are handled internally by the attrs parser.
pub fn optional_jid(node: &Node, key: &str) -> Result<Option<Jid>, anyhow::Error> {
    Ok(node.attrs().optional_jid(key))
}

/// Parse all children with a given tag into a Vec of ProtocolNodes.
///
/// Returns an error if any child fails to parse.
///
/// # Example
/// ```ignore
/// let participants = collect_children::<GroupParticipantResponse>(node, "participant")?;
/// ```
pub fn collect_children<T: ProtocolNode>(node: &Node, tag: &str) -> Result<Vec<T>, anyhow::Error> {
    node.get_children_by_tag(tag)
        .map(|child| T::try_from_node(child))
        .collect()
}
