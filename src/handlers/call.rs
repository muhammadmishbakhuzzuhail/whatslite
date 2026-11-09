//! Call stanza handler seam.
use wacore::types::call::CallEvent;

#[derive(Debug, Default)]
pub struct CallHandler;

impl CallHandler {
    pub fn new() -> Self {
        Self
    }
    /// Classify an already-decoded call event. Stanza parsing lands with the
    /// full call handler; this keeps the handler entry point stable.
    pub fn classify(&self, event: CallEvent) -> CallEvent {
        event
    }
}
