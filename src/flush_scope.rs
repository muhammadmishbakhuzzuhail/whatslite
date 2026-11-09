//! Tracks spawned tasks so a shutdown point can wait for them to complete.
//!
//! Used for outbound stanzas (delivery receipts, etc.) that must land before
//! the transport is torn down (issue #571). The decrement runs via a Drop
//! guard so the counter can't leak if the task future is cancelled mid-await.

use std::future::Future;
use std::sync::Arc;
use std::sync::atomic::{AtomicUsize, Ordering};
use std::time::Duration;

use wacore::runtime::{Runtime, Spawnable};

/// What a bounded drain observed at its decision point.
/// Completion of tracked work includes cancellation, not successful delivery.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[non_exhaustive]
pub enum DrainOutcome {
    Completed,
    /// The producer reported non-durable state; work may remain for redelivery.
    Failed,
    /// The deadline won, even if work finished just after that decision.
    TimedOut,
    /// No result was received from the worker performing the drain.
    Unobserved,
}

/// Bit 0 of [`FlushScope::state`]: the scope is closed to new work.
const CLOSED: usize = 1;
/// One tracked unit of work, i.e. one live [`FlushGuard`]; the count lives in
/// bits 1.. of the same word. Packing the flag next to the counter is what
/// makes "refuse if closed, otherwise increment" a single read-modify-write,
/// and that in turn is what upholds the promise `close` makes: once it
/// returns, nothing further is tracked. Split across two atomics there would
/// be a window between the two operations, and that window is the whole bug.
const ONE_TRACKED: usize = 2;

pub struct FlushScope {
    state: AtomicUsize,
    idle: event_listener::Event,
    #[cfg(test)]
    track_rejections: AtomicUsize,
}

impl Default for FlushScope {
    fn default() -> Self {
        Self::new()
    }
}

impl FlushScope {
    pub fn new() -> Self {
        Self {
            state: AtomicUsize::new(0),
            idle: event_listener::Event::new(),
            #[cfg(test)]
            track_rejections: AtomicUsize::new(0),
        }
    }

    // Relaxed on both flag writes: they publish no data, and the exclusion
    // `try_track` relies on comes from every writer being a read-modify-write
    // on one location, which the per-location modification order already
    // totally orders regardless of the ordering argument.
    pub fn close(&self) {
        self.state.fetch_or(CLOSED, Ordering::Relaxed);
    }

    pub fn is_closed(&self) -> bool {
        self.state.load(Ordering::Relaxed) & CLOSED != 0
    }

    pub fn reopen(&self) {
        self.state.fetch_and(!CLOSED, Ordering::Relaxed);
    }

    /// Track a unit of outbound work without spawning a task (e.g. an item
    /// handed to a persistent worker). Returns `None` when the scope is
    /// closed — the work must be dropped, mirroring `spawn`. Dropping the
    /// guard marks the work as finished for `flush`.
    pub fn try_track(self: &Arc<Self>) -> Option<FlushGuard> {
        // `checked_add` rather than a bare add: a wrap would clobber the closed
        // flag in bit 0. It cannot be reached — the count only rises while a
        // guard is alive, and usize::MAX/2 live guards is more Arcs than the
        // address space holds — so refusing to track is a formality, handled
        // the same way as a closed scope.
        if self
            .state
            .try_update(Ordering::Relaxed, Ordering::Relaxed, |state| {
                if state & CLOSED != 0 {
                    return None;
                }
                state.checked_add(ONE_TRACKED)
            })
            .is_err()
        {
            #[cfg(test)]
            self.track_rejections.fetch_add(1, Ordering::Relaxed);
            return None;
        }
        Some(FlushGuard {
            scope: Arc::clone(self),
        })
    }

    /// Spawn a tracked task. The counter decrements on completion OR if the
    /// future is dropped (e.g. aborted, or dropped before its first poll), so
    /// `flush` can never deadlock waiting on a cancelled task.
    // `Spawnable` (not a hardcoded `Send`) so this compiles on wasm, where the
    // single-threaded runtime accepts !Send futures (see `wacore::runtime`).
    pub fn spawn<F>(self: &Arc<Self>, rt: &dyn Runtime, fut: F)
    where
        F: Future<Output = ()> + Spawnable,
    {
        // Construct the guard outside the async block so it's captured as an
        // upvalue. This puts it in the future's state machine BEFORE the first
        // poll, so even if the runtime drops the future without polling it,
        // the guard's Drop runs and decrements the counter. Constructing the
        // guard inside the async body would delay it until the first poll,
        // which leaks the count if the future is dropped earlier.
        let Some(guard) = self.try_track() else {
            return;
        };
        rt.spawn(Box::pin(async move {
            let _guard = guard;
            fut.await;
        }))
        .detach();
    }

    /// Wait until every tracked task has finished or the timeout elapses.
    /// Emits a warn log on timeout with the number of leaked tasks.
    pub async fn flush(&self, rt: &dyn Runtime, timeout: Duration) -> DrainOutcome {
        let deadline = rt.monotonic_time() + timeout;
        loop {
            let listener = self.idle.listen();
            if self.tracked() == 0 {
                return DrainOutcome::Completed;
            }

            let remaining = deadline.saturating_duration_since(rt.monotonic_time());
            if remaining.is_zero()
                || wacore::runtime::timeout(rt, remaining, listener)
                    .await
                    .is_err()
            {
                log::warn!(
                    "FlushScope timed out with {} pending task(s)",
                    self.tracked()
                );
                return DrainOutcome::TimedOut;
            }
        }
    }

    /// Live guards. Acquire pairs with the guards' release decrements, so a
    /// caller that tears the transport down once this reads zero has observed
    /// everything the tracked work did.
    fn tracked(&self) -> usize {
        self.state.load(Ordering::Acquire) >> 1
    }

    pub fn pending(&self) -> usize {
        self.tracked()
    }

    /// Number of tasks parked inside [`Self::flush`]. `flush` registers its
    /// listener before checking the counter, so a listener here means a flusher
    /// really is waiting — what tests poll instead of sleeping.
    #[cfg(test)]
    pub fn flush_waiters(&self) -> usize {
        self.idle.total_listeners()
    }

    #[cfg(test)]
    pub fn track_rejections(&self) -> usize {
        self.track_rejections.load(Ordering::Relaxed)
    }
}

/// RAII guard for one tracked unit of outbound work. Obtained via
/// [`FlushScope::try_track`]; `spawn` embeds one in the task it creates.
pub struct FlushGuard {
    scope: Arc<FlushScope>,
}

impl Drop for FlushGuard {
    fn drop(&mut self) {
        // Release so the finished work is visible to whoever reads the counter
        // down to zero; the flusher's acquire load closes the pair.
        if self.scope.state.fetch_sub(ONE_TRACKED, Ordering::Release) >> 1 == 1 {
            self.scope.idle.notify(usize::MAX);
        }
    }
}
