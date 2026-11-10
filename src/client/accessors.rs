//! Configuration accessors: pure local state, no I/O.
//!
//! Every method here reads or mutates a field the builder already populated, so
//! they are safe to call from any context (sync or async).

use std::sync::Arc;
use std::sync::atomic::Ordering;

use super::Client;
use super::node_io::{NodeFilter, NodeWaiter};
use wacore_binary::Node;

impl Client {
    /// Whether a finished connection will be retried by [`Self::run`].
    pub fn auto_reconnect_enabled(&self) -> bool {
        self.enable_auto_reconnect.load(Ordering::Relaxed)
    }

    /// Change the reconnect preference without closing the current connection.
    ///
    /// An in-flight reconnect attempt may still complete; the preference is
    /// consulted before the next backoff cycle.
    ///
    /// # Terminal states are not cleared
    ///
    /// This fork stores the preference and the terminal stop verdict in one
    /// flag, and the client sets it to `false` on logout, stream conflict
    /// (`replaced`) and device removal (`516`). Nothing sets it back to `true`
    /// internally, so calling this with `true` after one of those events only
    /// flips the preference — it cannot revive the session, and the next
    /// connect attempt against a logged-out or replaced session will fail.
    /// Construct a new [`Client`] for a new session.
    pub fn set_auto_reconnect(&self, enabled: bool) {
        self.enable_auto_reconnect.store(enabled, Ordering::Relaxed);
    }

    /// Reuse the injected HTTP service instead of building a second client for
    /// auxiliary requests. Cloning this Arc retains the original host
    /// allocation; it cannot replace the client's service.
    pub fn http_client(&self) -> &Arc<dyn crate::http::HttpClient> {
        &self.http_client
    }

    /// Whether a custom handler is installed for this payload type.
    ///
    /// Handlers are registered at build time via
    /// [`crate::bot::BotBuilder::with_enc_handler`] and are not mutated
    /// afterwards.
    pub fn has_enc_handler(&self, payload_type: &str) -> bool {
        self.custom_enc_handlers.contains_key(payload_type)
    }

    /// Enable or disable skipping of history sync notifications at runtime.
    ///
    /// When enabled, the client will acknowledge incoming history sync
    /// notifications but will not download or process the data.
    pub fn set_skip_history_sync(&self, enabled: bool) {
        self.skip_history_sync.store(enabled, Ordering::Relaxed);
    }

    /// Returns `true` if history sync notifications are currently being skipped.
    pub fn skip_history_sync_enabled(&self) -> bool {
        self.skip_history_sync.load(Ordering::Relaxed)
    }

    /// Register a waiter for an incoming node matching the given filter.
    ///
    /// Register it **before** the action that triggers the expected stanza:
    /// the notification dispatcher only resolves waiters that already exist.
    /// Every matching waiter receives the same `Arc<Node>` in one resolve pass.
    pub fn wait_for_node(&self, filter: NodeFilter) -> NodeWaiter<Node> {
        self.node_waiters.wait(filter)
    }
}
