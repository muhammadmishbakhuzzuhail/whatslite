//! Client lifecycle extension seam.
//!
//! A [`ClientLifecycle`] is installed while the client is inert and receives
//! one [`ConnectionScope`] per authenticated connection generation. The full
//! callback queue lives in the client lifecycle; this module owns the stable
//! scope/state contract and the default no-op trait methods.

use std::sync::atomic::{AtomicU8, Ordering};
use std::sync::{Arc, Weak};

use wacore::runtime::{BoxFuture, ShutdownNotifier, ShutdownSignal};
use wacore::sync_marker::MaybeSendSync;

const SCOPE_OPEN: u8 = 0;
const SCOPE_READY: u8 = 1;
const SCOPE_CANCELLED: u8 = 2;
const SCOPE_CLOSED: u8 = 3;

/// Observable state of one authenticated connection generation.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[non_exhaustive]
pub enum ConnectionScopeState {
    Open,
    Ready,
    Cancelled,
    Closed,
}

struct ConnectionScopeInner {
    generation: u64,
    state: AtomicU8,
    cancellation: ShutdownNotifier,
}

/// Stable handle for work owned by one authenticated connection generation.
///
/// A scope is cancelled synchronously when its generation is retired and is
/// marked closed only after the client's authoritative connection cleanup.
#[derive(Clone)]
pub struct ConnectionScope {
    inner: Arc<ConnectionScopeInner>,
}

impl std::fmt::Debug for ConnectionScope {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        formatter
            .debug_struct("ConnectionScope")
            .field("generation", &self.generation())
            .field("state", &self.state())
            .finish()
    }
}

impl ConnectionScope {
    pub fn new(generation: u64) -> Self {
        Self {
            inner: Arc::new(ConnectionScopeInner {
                generation,
                state: AtomicU8::new(SCOPE_OPEN),
                cancellation: ShutdownNotifier::new(),
            }),
        }
    }

    pub fn generation(&self) -> u64 {
        self.inner.generation
    }

    pub fn state(&self) -> ConnectionScopeState {
        match self.inner.state.load(Ordering::Acquire) {
            SCOPE_OPEN => ConnectionScopeState::Open,
            SCOPE_READY => ConnectionScopeState::Ready,
            SCOPE_CANCELLED => ConnectionScopeState::Cancelled,
            _ => ConnectionScopeState::Closed,
        }
    }

    /// Fires when this scope stops owning connection work, whether it is
    /// cancelled during retirement or reaches final closure after cleanup.
    pub fn cancellation_signal(&self) -> ShutdownSignal {
        self.inner.cancellation.subscribe()
    }

    /// Returns `true` after either cancellation or final closure.
    pub fn is_cancelled(&self) -> bool {
        self.inner.state.load(Ordering::Acquire) >= SCOPE_CANCELLED
    }

    /// Transition `Open` to `Ready`. Returns `false` if the scope already left
    /// the open state.
    pub fn mark_ready(&self) -> bool {
        self.inner
            .state
            .compare_exchange(SCOPE_OPEN, SCOPE_READY, Ordering::AcqRel, Ordering::Acquire)
            .is_ok()
    }

    /// Cancel work owned by this generation. Idempotent.
    pub fn cancel(&self) {
        let mut state = self.inner.state.load(Ordering::Acquire);
        while state < SCOPE_CANCELLED {
            match self.inner.state.compare_exchange_weak(
                state,
                SCOPE_CANCELLED,
                Ordering::AcqRel,
                Ordering::Acquire,
            ) {
                Ok(_) => {
                    self.inner.cancellation.notify();
                    return;
                }
                Err(actual) => state = actual,
            }
        }
    }

    /// Mark the scope closed after authoritative connection cleanup.
    /// Idempotent; also fires the cancellation signal if it was still open.
    pub fn close(&self) {
        self.cancel();
        self.inner.state.store(SCOPE_CLOSED, Ordering::Release);
    }
}

/// Aggregate lifecycle seam installed during client construction.
///
/// Implementations must make `install` transactional. Connection callbacks are
/// serialized and `on_closed` only schedules cleanup so a stalled extension
/// cannot block reconnect. `signal_shutdown` is the non-blocking boundary for
/// resources that must stop even when the host cannot await `shutdown`.
pub trait ClientLifecycle: MaybeSendSync {
    fn install<'a>(&'a self, _client: Weak<crate::Client>) -> BoxFuture<'a, anyhow::Result<()>> {
        Box::pin(async { Ok(()) })
    }

    fn on_ready<'a>(&'a self, _scope: ConnectionScope) -> BoxFuture<'a, anyhow::Result<()>> {
        Box::pin(async { Ok(()) })
    }

    fn on_closed<'a>(&'a self, _scope: ConnectionScope) -> BoxFuture<'a, anyhow::Result<()>> {
        Box::pin(async { Ok(()) })
    }

    /// Stop synchronously owned resources before asynchronous shutdown begins.
    /// Implementations must return promptly and make repeated calls harmless.
    fn signal_shutdown(&self) {}

    fn shutdown(&self) -> BoxFuture<'_, anyhow::Result<()>> {
        Box::pin(async { Ok(()) })
    }
}
