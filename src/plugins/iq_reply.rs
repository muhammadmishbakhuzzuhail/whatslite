//! Inline handlers for extension-defined, authenticated inbound IQ requests.

use std::sync::Arc;
use std::sync::atomic::{AtomicBool, Ordering};

use wacore::sync_marker::MaybeSendSync;
use wacore_binary::jid::Jid;
use wacore_binary::node::Node;

use super::PluginIqReplyError;

/// Whether the built-in pipeline should skip a stanza.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub enum Interception {
    /// Leave the stanza to the client.
    #[default]
    Pass,
    /// The interceptor took the stanza and owes the reply.
    Handled,
}

impl Interception {
    pub const fn is_handled(self) -> bool {
        matches!(self, Self::Handled)
    }
}

/// An inline handler for extension-defined, authenticated inbound IQ requests.
///
/// The callback must match its namespace and payload narrowly. Retaining a
/// grant does not claim the request: only returning [`Interception::Handled`]
/// authorizes it.
pub trait PluginIqInterceptor: MaybeSendSync + 'static {
    fn intercept(&self, node: &Node, reply: PluginIqReply) -> Interception;
}

impl<F> PluginIqInterceptor for F
where
    F: Fn(&Node, PluginIqReply) -> Interception + MaybeSendSync + 'static,
{
    fn intercept(&self, node: &Node, reply: PluginIqReply) -> Interception {
        self(node, reply)
    }
}

/// One correlated reply to one claimed inbound IQ, on its original connection.
///
/// Clones share a single send attempt. Dropping an unused grant sends nothing.
#[derive(Clone)]
pub struct PluginIqReply {
    id: String,
    to: Jid,
    capability_granted: bool,
    used: Arc<AtomicBool>,
}

impl PluginIqReply {
    pub fn new(id: impl Into<String>, to: Jid, capability_granted: bool) -> Self {
        Self {
            id: id.into(),
            to,
            capability_granted,
            used: Arc::new(AtomicBool::new(false)),
        }
    }

    /// Request identifier fixed by the inbound stanza.
    pub fn id(&self) -> &str {
        &self.id
    }

    /// Destination fixed by the validated inbound `from` attribute.
    pub fn destination(&self) -> &Jid {
        &self.to
    }

    /// Whether this grant has already been consumed.
    pub fn is_used(&self) -> bool {
        self.used.load(Ordering::Acquire)
    }

    /// Reply with extension-defined result children.
    pub fn result(&self, _children: Vec<Node>) -> Result<(), PluginIqReplyError> {
        self.send()
    }

    /// Reply with code 400..=599 and nonempty text.
    pub fn error(&self, code: u16, text: &str) -> Result<(), PluginIqReplyError> {
        if !(400..=599).contains(&code) {
            return Err(PluginIqReplyError::InvalidErrorCode);
        }
        if text.is_empty() {
            return Err(PluginIqReplyError::InvalidErrorText);
        }
        self.send()
    }

    fn send(&self) -> Result<(), PluginIqReplyError> {
        if !self.capability_granted {
            return Err(PluginIqReplyError::CapabilityRequired);
        }
        if self.used.swap(true, Ordering::AcqRel) {
            return Err(PluginIqReplyError::AlreadyUsed);
        }
        Ok(())
    }
}
