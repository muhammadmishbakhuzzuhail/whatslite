//! Logging helpers that keep protocol payloads out of log records.
//!
//! [`StanzaSummary`] renders only the tag, attribute count and content shape;
//! attribute values, JIDs and payload bytes never reach the log. Enable the
//! `tracing-pii` feature when raw identifiers are intentionally required.

use std::fmt;

use wacore_binary::node::{Node, NodeContent};

/// Initialize `env_logger` once. Safe to call repeatedly.
pub fn init_logging() {
    let _ = env_logger::Builder::from_env(env_logger::Env::default().default_filter_or("info"))
        .try_init();
}

/// Borrowed, bounded protocol diagnostic for one stanza.
pub struct StanzaSummary<'a>(&'a Node);

impl<'a> From<&'a Node> for StanzaSummary<'a> {
    fn from(node: &'a Node) -> Self {
        Self(node)
    }
}

impl fmt::Display for StanzaSummary<'_> {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        let (kind, size) = match &self.0.content {
            Some(NodeContent::Nodes(children)) => ("children", children.len()),
            Some(NodeContent::Bytes(bytes)) => ("bytes", bytes.len()),
            Some(NodeContent::String(text)) => ("string", text.len()),
            None => ("empty", 0),
        };
        write!(
            f,
            "<{} attrs={} {}={}>",
            self.0.tag,
            self.0.attrs.0.len(),
            kind,
            size
        )
    }
}

impl fmt::Debug for StanzaSummary<'_> {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        fmt::Display::fmt(self, f)
    }
}

/// Create a tracing span for one client operation.
#[cfg(feature = "tracing")]
pub fn span(operation: &'static str) -> tracing::Span {
    tracing::info_span!("wa_operation", operation = operation)
}
