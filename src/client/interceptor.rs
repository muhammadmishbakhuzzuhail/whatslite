//! Taking over a stanza before the built-in pipeline sees it.
//!
//! The client models the stanzas it knows about and nacks the rest, which is
//! the default for a tag it cannot handle: a `<nack>` reports that limitation.
//! The local response does not establish the server's redelivery policy.
//!
//! But it leaves no room for a consumer that *can* act on it. A stanza this
//! version does not model gets nacked whether or not the application would have
//! known what to do with it, and there is no way to say otherwise: the internal
//! router rejects duplicate tags, so even a handler for an existing tag cannot
//! be replaced.
//!
//! An interceptor is that room. It runs where dispatch would have, and either
//! steps aside or claims the stanza — in which case the built-in handler is
//! skipped, and whatever answer the client owed the server becomes the
//! claimant's to send.
//!
//! # What an interceptor sees
//!
//! Stanzas that would have reached dispatch. A response the client already
//! correlated to a pending request, a `<xmlstreamend>` that ends the stream,
//! and the connection-critical tags below never get there, so an interceptor
//! does not see them either. To observe *everything* decoded, including those,
//! use [`Event::RawNode`] — it is emitted before any of the early returns. The
//! two are different tools: one watches, this one takes over.
//!
//! [`Event::RawNode`]: wacore::types::events::Event::RawNode
//!
//! # What cannot be claimed
//!
//! `success`, `failure`, `stream:error` and `ack` settle connection state:
//! authentication, shutdown and reconnection, and the waiters a send blocks on.
//! An interceptor that took one would not extend the client — it would leave it
//! authenticated-but-unaware, or never reconnecting, or waiting forever on a
//! send that already completed. They are never offered.
//!
//! Housekeeping is likewise untouched: offline-sync tracking and
//! response-waiter resolution run before dispatch and keep running whether or
//! not a stanza is claimed.
//!
//! A server-initiated `<iq>` ping is not offered either, for the same reason as
//! the four above: a claimed ping is a pong never sent, and the server drops the
//! connection over it.
//!
//! Every other `<iq>` *is* offered, including ones the client answers on its
//! own — a pairing step, a query it models. Claiming one leaves the server
//! without the reply it expects, so match narrowly.
//!
//! # Acknowledgement
//!
//! Claims preserve the response class:
//!
//! - Direct/group messages and IQ requests receive no generic ACK. The claimant
//!   owns decryption/session state and the appropriate receipt or IQ response.
//! - Newsletter/status messages, receipts, notifications, calls and unknown tags
//!   receive a transport ACK when `id` and `from` are present. Message/status
//!   ACKs also require the local device PN. This fallback is not a
//!   Web-equivalence or server redelivery guarantee.
//!
//! # Cost
//!
//! Nothing runs while no interceptor is registered: the read loop checks one
//! relaxed atomic and carries on. Registering is what turns the check into a
//! walk.
//!
//! # Example
//!
//! Handling a stanza the client does not model, instead of nacking it:
//!
//! ```
//! use wa_rs::client::interceptor::{Interception, StanzaInterceptor};
//! use wacore_binary::node::Node;
//!
//! struct Vendor;
//!
//! impl StanzaInterceptor for Vendor {
//!     fn intercept(&self, node: &Node) -> Interception {
//!         if node.tag == "vendor:thing" {
//!             // … act on it …
//!             Interception::Handled
//!         } else {
//!             Interception::Pass
//!         }
//!     }
//! }
//! ```
//!
//! Registration is the client's job: a `Client` owns an interceptor registry and
//! hands out an [`InterceptorHandle`] per registration, which removes it again
//! on drop.

use std::sync::atomic::{AtomicU64, AtomicUsize, Ordering};
use std::sync::{Arc, RwLock, Weak};

use wacore::sync_marker::MaybeSendSync;
use wacore_binary::node::Node;

/// Connection identity passed by the dispatcher to scoped interceptors.
///
/// Its fields are private so a caller cannot manufacture an inbound reply grant.
#[doc(hidden)]
pub struct InterceptionScope {
    pub(crate) generation: u64,
    pub(crate) shutdown: wacore::runtime::ShutdownSignal,
}

impl InterceptionScope {
    /// Connection generation captured by the dispatcher.
    pub fn generation(&self) -> u64 {
        self.generation
    }

    /// Whether that connection has requested cancellation.
    pub fn is_cancelled(&self) -> bool {
        self.shutdown.is_fired()
    }
}

/// What an interceptor decided about a stanza.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Default)]
pub enum Interception {
    /// Leave the stanza to the client.
    ///
    /// The default, so an interceptor that only cares about one tag needs no
    /// branch for the rest.
    #[default]
    Pass,
    /// The interceptor took the stanza.
    ///
    /// The built-in pipeline is skipped. Where the client's answer was a
    /// transport ack it still sends one; where the answer was something else —
    /// a delivery `<receipt>` for a direct `<message>`, an `<iq type="result">`
    /// — nothing is sent, and the claimant owes that reply. See the
    /// [acknowledgement](index.html#acknowledgement) rules.
    Handled,
}

impl Interception {
    /// Whether the built-in pipeline should be skipped.
    #[must_use]
    pub const fn is_handled(self) -> bool {
        matches!(self, Self::Handled)
    }
}

/// Sees a stanza on its way to the built-in pipeline.
///
/// Not every decoded stanza: see the module documentation for what never
/// reaches this point.
///
/// Runs on the read loop, so it must return quickly: time spent here is time
/// the next stanza waits. Work that can take a while belongs on a task.
///
/// Must not panic. Like an [`EventHandler`], this is called directly by the read
/// loop, and an unwind there takes the connection with it.
///
/// [`EventHandler`]: wacore::types::events::EventHandler
pub trait StanzaInterceptor: MaybeSendSync + 'static {
    /// Decide what happens to `node`.
    fn intercept(&self, node: &Node) -> Interception;

    /// Dispatcher hook for connection-bound capabilities. Existing interceptors
    /// use `intercept`; the default preserves their behavior.
    #[doc(hidden)]
    fn intercept_scoped(&self, node: &Node, _scope: &InterceptionScope) -> Interception {
        self.intercept(node)
    }
}

impl<F> StanzaInterceptor for F
where
    F: Fn(&Node) -> Interception + MaybeSendSync + 'static,
{
    fn intercept(&self, node: &Node) -> Interception {
        self(node)
    }
}

/// Keeps an interceptor registered. Dropping it removes the interceptor.
///
/// Holds a weak registry reference, so a forgotten handle cannot keep a client
/// alive.
#[must_use = "dropping the handle immediately removes the interceptor"]
#[derive(Debug)]
pub struct InterceptorHandle {
    pub(crate) registry: Weak<InterceptorRegistry>,
    pub(crate) id: u64,
}

impl Drop for InterceptorHandle {
    fn drop(&mut self) {
        if let Some(registry) = self.registry.upgrade() {
            registry.remove(self.id);
        }
    }
}

/// One registered interceptor.
#[derive(Clone)]
pub(crate) struct Registration {
    pub(crate) id: u64,
    #[allow(
        dead_code,
        reason = "read by the dispatcher once the client owns the registry"
    )]
    pub(crate) interceptor: Arc<dyn StanzaInterceptor>,
}

/// The interceptors registered against one client.
///
/// This is the storage the client will own; keeping the copy-on-write list and
/// the id counter here rather than on `Client` is what lets registration, its
/// handle and the read loop's "anything registered?" probe be exercised before
/// the client is wired to them. The handle keeps the registry weakly, so a
/// client is never kept alive by a forgotten handle.
#[allow(
    dead_code,
    reason = "the client does not own the registry yet; registration is wired in a follow-up"
)]
#[derive(Default)]
pub(crate) struct InterceptorRegistry {
    next_id: AtomicU64,
    registered: RwLock<Arc<Vec<Registration>>>,
    count: AtomicUsize,
}

#[allow(
    dead_code,
    reason = "the client does not own the registry yet; registration is wired in a follow-up"
)]
impl InterceptorRegistry {
    /// Register `interceptor` and return the handle that keeps it registered.
    pub(crate) fn add(
        self: &Arc<Self>,
        interceptor: Arc<dyn StanzaInterceptor>,
    ) -> InterceptorHandle {
        let id = self.next_id.fetch_add(1, Ordering::Relaxed);
        self.update(|registered| registered.push(Registration { id, interceptor }));
        InterceptorHandle {
            registry: Arc::downgrade(self),
            id,
        }
    }

    pub(crate) fn remove(&self, id: u64) {
        self.update(|registered| registered.retain(|entry| entry.id != id));
    }

    /// Whether any interceptor is registered.
    ///
    /// One relaxed load, so the read loop pays nothing while none are. Relaxed
    /// is enough because the lock behind it does the synchronising: a reader
    /// racing a registration either sees the count in time or does not, and a
    /// stanza that arrived before the registration finished was never that
    /// interceptor's to see.
    pub(crate) fn has_any(&self) -> bool {
        self.count.load(Ordering::Relaxed) != 0
    }

    /// The current interceptors.
    ///
    /// A refcount bump, not a copy — and the snapshot is released before any
    /// interceptor runs, so one that registers another cannot deadlock.
    pub(crate) fn snapshot(&self) -> Arc<Vec<Registration>> {
        Arc::clone(
            &self
                .registered
                .read()
                .unwrap_or_else(std::sync::PoisonError::into_inner),
        )
    }

    /// Copy-on-write, as the event bus does it: a reader keeps whatever
    /// snapshot it took, so registering never blocks the read loop for longer
    /// than the swap.
    fn update(&self, edit: impl FnOnce(&mut Vec<Registration>)) {
        let mut guard = self
            .registered
            .write()
            .unwrap_or_else(std::sync::PoisonError::into_inner);
        let mut next = guard.as_ref().clone();
        edit(&mut next);
        // Stored while the write lock is held, so a reader never sees a count
        // promising more than the snapshot holds.
        self.count.store(next.len(), Ordering::Relaxed);
        *guard = Arc::new(next);
    }
}

impl std::fmt::Debug for InterceptorRegistry {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("InterceptorRegistry")
            .field("registered", &self.count.load(Ordering::Relaxed))
            .finish()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::Arc;
    use wacore::runtime::{ShutdownNotifier, ShutdownSignal};
    use wacore_binary::builder::NodeBuilder;
    use wacore_binary::node::Node;

    fn node(tag: &str) -> Node {
        NodeBuilder::new(tag).build()
    }

    struct Always;

    impl StanzaInterceptor for Always {
        fn intercept(&self, _node: &Node) -> Interception {
            Interception::Handled
        }
    }

    #[test]
    fn pass_is_the_default_so_narrow_interceptors_stay_short() {
        assert_eq!(Interception::default(), Interception::Pass);
        assert!(!Interception::Pass.is_handled());
        assert!(Interception::Handled.is_handled());
    }

    #[test]
    fn a_closure_is_an_interceptor() {
        fn takes(_: impl StanzaInterceptor) {}
        takes(|_node: &Node| Interception::Pass);
    }

    #[test]
    fn interception_is_comparable_and_debuggable() {
        assert_eq!(Interception::Pass, Interception::Pass);
        assert_ne!(Interception::Pass, Interception::Handled);
        assert!(!format!("{:?}", Interception::Handled).is_empty());
    }

    #[test]
    fn the_scoped_hook_defaults_to_the_plain_hook() {
        let scope = InterceptionScope {
            generation: 3,
            shutdown: ShutdownSignal::never(),
        };
        assert_eq!(
            Always.intercept_scoped(&node("vendor:thing"), &scope),
            Interception::Handled
        );
    }

    #[test]
    fn a_scope_reports_its_generation_and_cancellation() {
        let notifier = ShutdownNotifier::new();
        let scope = InterceptionScope {
            generation: 42,
            shutdown: notifier.subscribe(),
        };
        assert_eq!(scope.generation(), 42);
        assert!(!scope.is_cancelled());
        notifier.notify();
        assert!(scope.is_cancelled());
    }

    #[test]
    fn dropping_the_handle_removes_only_that_registration() {
        let registry = Arc::new(InterceptorRegistry::default());
        let first = registry.add(Arc::new(Always));
        let second = registry.add(Arc::new(Always));
        assert_eq!(registry.snapshot().len(), 2);

        drop(first);

        let remaining = registry.snapshot();
        assert_eq!(remaining.len(), 1);
        assert_eq!(remaining[0].id, second.id);
    }

    #[test]
    fn a_snapshot_taken_before_registration_does_not_see_it() {
        let registry = Arc::new(InterceptorRegistry::default());
        let before = registry.snapshot();
        let _handle = registry.add(Arc::new(Always));

        assert!(before.is_empty());
        assert_eq!(registry.snapshot().len(), 1);
    }

    #[test]
    fn the_registry_reports_whether_anything_is_registered() {
        let registry = Arc::new(InterceptorRegistry::default());
        assert!(!registry.has_any());

        let handle = registry.add(Arc::new(Always));
        assert!(registry.has_any());

        drop(handle);
        assert!(!registry.has_any());
    }

    #[test]
    fn a_handle_outliving_its_registry_drops_quietly() {
        let registry = Arc::new(InterceptorRegistry::default());
        let handle = registry.add(Arc::new(Always));
        drop(registry);
        drop(handle);
    }
}
