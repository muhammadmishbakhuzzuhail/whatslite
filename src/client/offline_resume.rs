//! Pull-based offline backlog request loop.
//!
//! WA Web (`WAWebOfflineHandler.processOfflinePreviewIb` in
//! `docs/captured-js/WAWeb/Offline/Handler.js`) gates offline delivery on the
//! client: the server announces a backlog via `<ib><offline_preview count="N"/>`,
//! delivers a primer (~5 stanzas), and then waits for
//! `<ib><offline_batch count="200"/>` requests before sending more. Without
//! those requests the backlog is silently dropped after the primer.
//!
//! Mirrors the non-adaptive path (`$13`, default when
//! `isOfflineDynamicBatchSizeEnabled` AB flag is off). The state machine
//! tracks WA Web's `$4` (prev-batch-in-flight) flag: a CAS-gated
//! "first arrival of the just-sent batch" deterministically elects a single
//! continuation per cycle, just as WA Web's `$6` timer coalesces
//! synchronously-arriving stanzas into one `$13` call.
//!
//! Additionally paced by processing, which WA Web leaves to the lane: one
//! cumulative `outstanding` per connection counts the stanzas requested and not
//! yet finished, and a continuation is only reserved while it fits one batch.
//! A request is booked when it is decided — never on arrival — so a slow
//! delivery cannot make the count under-represent what the server still owes,
//! and two batches is the most one chat lane can be handed.
//!
//! The coordinator is self-contained: it knows nothing about the socket, the
//! runtime or the client. The tasks that send requests and complete a stalled
//! drain are wired in a follow-up.

// Nothing drives the coordinator until the resume tasks that arm it and send
// its requests are ported; they need client fields this port does not have.
#![allow(
    dead_code,
    reason = "the tasks that arm, send and complete the resume are wired in a follow-up"
)]

use portable_atomic::AtomicU64;
use std::sync::Arc;
use std::sync::atomic::{AtomicBool, Ordering};
use std::time::Duration;

use wacore_binary::Node;
use wacore_binary::builder::NodeBuilder;

/// Mirrors `WAWebOfflineHandler.v = 200`.
const BATCH_SIZE: u32 = 200;

// Two pulled batches must fit one chat's lane; see `CHAT_QUEUE_ITEMS`.
const _: () = assert!(crate::inbound_queue::CHAT_QUEUE_ITEMS >= 2 * BATCH_SIZE as usize);

/// Mirrors `WAWebOfflineHandler.S = 100`.
const REQUEST_DEBOUNCE: Duration = Duration::from_millis(100);

/// Mirrors `WAWebOfflineResumeConst.OFFLINE_STANZA_TIMEOUT_MS = 60000`: the
/// blocking resume manager arms a `ShiftTimer` on the first preview, shifts it
/// on every offline stanza, and on expiry completes the session itself.
const STANZA_INACTIVITY_TIMEOUT: Duration = Duration::from_secs(60);

/// Parked in `stanza_activity` by the watchdog that completes a stalled drain,
/// so a stanza racing that decision loses the CAS instead of being ignored.
const INACTIVE_CLAIMED: u64 = u64::MAX;

#[derive(Default)]
pub(crate) struct OfflineBatchCoordinator {
    armed: AtomicBool,
    /// Pacing owned by the armed generation. `arm` swaps a fresh object in and
    /// `reset` takes it out, so a guard or arrival retired by a reconnect can
    /// only ever reach the object it was minted with.
    pacing: std::sync::Mutex<Option<Arc<Pacing>>>,
}

impl OfflineBatchCoordinator {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn reset(&self) {
        // Disarm first: readers stop minting guards before the object they
        // would count on is gone. Guards already minted keep their own `Arc`
        // and settle the retired object, where nobody reads them any more.
        self.armed.store(false, Ordering::Release);
        *self.pacing.lock().unwrap_or_else(|p| p.into_inner()) = None;
    }

    /// Disarm and report whether this call is the one that did it. The
    /// teardown paths use it as a once-guard for the interrupted-resume
    /// event, so a second teardown (or a `connect()` that follows one) stays
    /// silent.
    pub(crate) fn take_armed(&self) -> bool {
        self.armed.swap(false, Ordering::AcqRel)
    }

    #[cfg(test)]
    pub(crate) fn is_armed(&self) -> bool {
        self.armed.load(Ordering::Acquire)
    }

    #[cfg(test)]
    pub(crate) fn stanza_activity(&self) -> u64 {
        self.test_pacing().map_or(0, |p| p.stanza_activity())
    }

    #[cfg(test)]
    pub(crate) fn note_stanza_activity(&self) {
        if let Some(pacing) = self.test_pacing() {
            pacing.note_stanza_activity();
        }
    }

    #[cfg(test)]
    fn try_claim_inactive(&self, seen: u64) -> bool {
        self.test_pacing()
            .is_some_and(|p| p.try_claim_inactive(seen))
    }

    /// Arm the resume for `generation` and hand back the channel its parked
    /// requests are released on, for the one task that sends them.
    fn arm(&self, generation: u64, total: usize) -> async_channel::Receiver<()> {
        let (pacing, released) = Pacing::new(generation, total);
        let pacing = Arc::new(pacing);
        // The first request is booked with the arm: the primer that follows is
        // part of the batch it announces.
        pacing.book_first();
        // Set inner state first; `armed` is published last so readers that
        // observe armed=true also see the matching generation, pacing and
        // inflight.
        *self.pacing.lock().unwrap_or_else(|p| p.into_inner()) = Some(pacing);
        self.armed.store(true, Ordering::Release);
        released
    }

    fn is_armed_for(&self, generation: u64) -> bool {
        self.current_pacing(generation).is_some()
    }

    /// The pacing of the connection this caller belongs to, or `None` when that
    /// connection is not the armed one. Handing out the `Arc` is what makes a
    /// stale caller harmless: it can only reach the generation it was minted
    /// for.
    fn current_pacing(&self, generation: u64) -> Option<Arc<Pacing>> {
        if !self.armed.load(Ordering::Acquire) {
            return None;
        }
        self.pacing
            .lock()
            .unwrap_or_else(|p| p.into_inner())
            .as_ref()
            .filter(|pacing| pacing.generation == generation)
            .cloned()
    }

    /// What a whole window without an offline stanza means, for the watchdog.
    pub(crate) fn settle_silent_drain(&self, generation: u64) -> SilentDrain {
        match self.current_pacing(generation) {
            Some(pacing) => pacing.settle_silent(),
            None => SilentDrain::Idle,
        }
    }

    #[cfg(test)]
    fn try_claim_first_arrival(&self) -> bool {
        self.test_pacing()
            .is_some_and(|p| p.try_claim_first_arrival())
    }

    #[cfg(test)]
    fn mark_batch_inflight(&self) {
        if let Some(pacing) = self.test_pacing() {
            pacing.mark_batch_inflight();
        }
    }

    /// Count one offline stanza the message handler is about to queue in a
    /// chat lane and return the guard that finishes it. `None` for anything the
    /// resume does not pace: an unarmed connection, or a live stanza.
    ///
    /// Every stanza the message handler receives is bound for a lane (messages,
    /// status broadcasts and durable group notifications alike), so the tag is
    /// not checked here; the receive path decides which stanzas come here.
    ///
    /// Boxed because it travels in the queued message, where a word is what
    /// the queue's layout budget has room for.
    pub(crate) fn admit(&self, generation: u64, node: &Node) -> Option<Box<OfflinePendingGuard>> {
        node.attrs.get("offline")?;
        let pacing = self.current_pacing(generation)?;
        pacing.admit_lane_bound();
        Some(Box::new(OfflinePendingGuard { pacing }))
    }

    #[cfg(test)]
    fn test_pacing(&self) -> Option<Arc<Pacing>> {
        if !self.armed.load(Ordering::Acquire) {
            return None;
        }
        self.pacing
            .lock()
            .unwrap_or_else(|p| p.into_inner())
            .as_ref()
            .cloned()
    }

    #[cfg(test)]
    pub(crate) fn outstanding(&self) -> usize {
        self.test_pacing()
            .map_or(0, |pacing| pacing.state().outstanding)
    }

    #[cfg(test)]
    pub(crate) fn admitted(&self) -> usize {
        self.test_pacing()
            .map_or(0, |pacing| pacing.state().admitted)
    }

    #[cfg(test)]
    pub(crate) fn deferred(&self) -> bool {
        self.test_pacing()
            .is_some_and(|pacing| pacing.state().deferred)
    }
}

/// Backlog pacing for one connection generation (S2-S4).
///
/// One object per generation, one short `Mutex` and never held across an
/// `.await`: every decision that must not interleave with another — booking a
/// request, settling a finish, parking a continuation, releasing one, deciding
/// what a silent drain means — is a single transition. That is what makes the
/// cumulative bound exact and the deferred wakeup impossible to lose, in place
/// of separate counters that could be read and written a window apart.
pub(crate) struct Pacing {
    generation: u64,
    /// WA Web `$4`: true between sending a batch and observing the first
    /// stanza of that batch's response. CAS-gated so concurrent arrivals
    /// deterministically elect a single "first arrival" winner per cycle.
    prev_batch_inflight: AtomicBool,
    /// Stanza progress and newly booked requests shift the inactivity window.
    /// Comparing across a sleep avoids a clock read per stanza and lets the
    /// timeout fire within [timeout, 2 * timeout) after the last activity.
    stanza_activity: AtomicU64,
    state: std::sync::Mutex<PacingState>,
    /// One message per released request, for the generation's sender task.
    /// A channel rather than a client handle: whoever settles a finish (a lane
    /// worker, a queue being torn down) only signals, and a queued message
    /// carries nothing that ties it to the client, which a lane must not do
    /// where the client is not `Send`.
    released: async_channel::Sender<()>,
}

#[derive(Default)]
struct PacingState {
    /// Stanzas this connection asked the server for and has not finished: a
    /// lane-bound message until its guard drops, any other offline stanza as
    /// soon as it is handled. Booked when the request is decided, so nothing
    /// can arrive that the count does not already own.
    outstanding: usize,
    /// Preview stanzas no request has booked yet. Booking moves one batch from
    /// here to `outstanding`, so no request asks for more than was announced.
    unbooked: usize,
    /// Lane-bound offline messages still unfinished: our own backpressure, as
    /// opposed to the server's silence.
    admitted: usize,
    /// A continuation the gate had to park, owed to the finish that brings
    /// `outstanding` back to one batch.
    deferred: bool,
}

/// One continuation decision, taken as a single reservation.
enum Reservation {
    /// Booked; the caller sends the request.
    Request,
    /// Parked until processing catches up.
    Deferred,
    /// The preview has nothing left to ask for.
    Exhausted,
}

/// What a whole window of silence means for the drain.
pub(crate) enum SilentDrain {
    /// Our own pacing is holding the drain open: wait instead of completing.
    Backpressure,
    /// Nothing is in a lane: the server stopped, and the resume completes.
    Idle,
}

impl Pacing {
    fn stanza_activity(&self) -> u64 {
        self.stanza_activity.load(Ordering::Acquire)
    }

    fn note_stanza_activity(&self) {
        self.stanza_activity.fetch_add(1, Ordering::AcqRel);
    }

    /// Claim a drain as inactive, but only if no stanza has arrived since
    /// `seen`. A bump landing between the watchdog's read and its decision
    /// fails the CAS, so a drain that is still being fed is never completed
    /// behind the server's back. The sentinel it parks on is never a value
    /// `note_stanza_activity` can reach from a fresh `arm()`.
    fn try_claim_inactive(&self, seen: u64) -> bool {
        self.stanza_activity
            .compare_exchange(seen, INACTIVE_CLAIMED, Ordering::AcqRel, Ordering::Acquire)
            .is_ok()
    }

    /// Try to claim the "first arrival of just-sent batch" slot. Atomically
    /// transitions prev_batch_inflight true → false. Only one concurrent
    /// caller per batch cycle succeeds.
    fn try_claim_first_arrival(&self) -> bool {
        self.prev_batch_inflight
            .compare_exchange(true, false, Ordering::AcqRel, Ordering::Acquire)
            .is_ok()
    }

    /// Republish "batch in flight" so the next stanza arrival can win the
    /// `try_claim_first_arrival` CAS.
    fn mark_batch_inflight(&self) {
        self.prev_batch_inflight.store(true, Ordering::Release);
    }

    fn new(generation: u64, total: usize) -> (Self, async_channel::Receiver<()>) {
        // Unbounded: every release books its own request, so two releases
        // must never coalesce into one send.
        let (released, receiver) = async_channel::unbounded();
        let pacing = Self {
            generation,
            stanza_activity: AtomicU64::new(0),
            prev_batch_inflight: AtomicBool::new(true),
            state: std::sync::Mutex::new(PacingState {
                unbooked: total,
                ..PacingState::default()
            }),
            released,
        };
        (pacing, receiver)
    }

    /// Hand a request a finish released to the sender task. A closed channel
    /// means the generation's task is gone, and so is the request's purpose.
    fn notify(&self, released: bool) -> bool {
        if released {
            let _ = self.released.try_send(());
        }
        released
    }

    /// The one lock every transition takes. A poisoned mutex still holds a
    /// consistent state, so recovery is the right answer here.
    fn state(&self) -> std::sync::MutexGuard<'_, PacingState> {
        self.state
            .lock()
            .unwrap_or_else(|poisoned| poisoned.into_inner())
    }

    /// The first request of a drain, booked with the arm.
    fn book_first(&self) {
        self.state().book();
    }

    /// Decide one continuation: check the gate and book the request as one
    /// transition, so a request is never sent unreserved.
    fn reserve(&self) -> Reservation {
        let mut state = self.state();
        if state.unbooked == 0 {
            return Reservation::Exhausted;
        }
        if state.outstanding > BATCH_SIZE as usize {
            state.deferred = true;
            return Reservation::Deferred;
        }
        state.book();
        self.note_stanza_activity();
        Reservation::Request
    }

    fn admit_lane_bound(&self) {
        // `outstanding` is untouched: the stanza was booked when the request
        // that delivered it was decided, and admission only moves its finish
        // from arrival to the guard on its queue entry.
        self.state().admitted += 1;
    }

    /// One lane-bound message is done with — processed, cancelled, refused, or
    /// dropped with its queue. `true` means this finish resolved a parked
    /// request, which the caller must send.
    fn finish_admitted(&self) -> bool {
        let released = {
            let mut state = self.state();
            state.settle_one();
            state.admitted = state.admitted.saturating_sub(1);
            self.release_deferred(&mut state)
        };
        self.notify(released)
    }

    /// One offline stanza that no lane holds is done with.
    fn finish_arrival(&self) -> bool {
        let released = {
            let mut state = self.state();
            state.settle_one();
            self.release_deferred(&mut state)
        };
        self.notify(released)
    }

    fn release_deferred(&self, state: &mut PacingState) -> bool {
        let released = state.release_deferred();
        // Publish under the decision lock, before the final guard empties the
        // lane and before the release sender is even polled.
        if released {
            self.note_stanza_activity();
        }
        released
    }

    /// Decide what a window without a single offline stanza means.
    ///
    /// Silence never settles what the server still owes: a late response is
    /// not an absent one, and booking another batch on top of it is exactly
    /// how a lane would be handed a third. With nothing in a lane, the resume
    /// completes as it did before pacing, and no further request leaves.
    fn settle_silent(&self) -> SilentDrain {
        if self.state().admitted > 0 {
            SilentDrain::Backpressure
        } else {
            SilentDrain::Idle
        }
    }
}

impl PacingState {
    fn settle_one(&mut self) {
        self.outstanding = self.outstanding.saturating_sub(1);
    }

    /// Take one request's worth out of the preview — never more than is left.
    fn book(&mut self) {
        let booking = (BATCH_SIZE as usize).min(self.unbooked);
        self.unbooked -= booking;
        self.outstanding += booking;
    }

    /// Release a parked continuation if this finish brought the lane back
    /// within one batch. Decided under the same lock that parked it, so the
    /// finish that opens the gate can never miss it.
    fn release_deferred(&mut self) -> bool {
        if !self.deferred || self.outstanding > BATCH_SIZE as usize {
            return false;
        }
        self.deferred = false;
        if self.unbooked == 0 {
            // Everything announced is booked; the end marker ends the drain.
            return false;
        }
        self.book();
        true
    }
}

/// One queued offline message's claim on its generation's pacing (S3).
///
/// Counted when the handler decides to queue it and settled when the queued
/// item is dropped, so finishing, cancelling, refusing and tearing the queue
/// down all settle it by the same path: nothing a lane was handed can keep the
/// resume waiting after the lane let go of it.
pub(crate) struct OfflinePendingGuard {
    /// The generation's pacing object, not a counter: a guard retired by a
    /// reconnect can only ever settle what it was minted with.
    pacing: Arc<Pacing>,
}

impl Drop for OfflinePendingGuard {
    fn drop(&mut self) {
        // A release is sent by the generation's own task: a lane worker
        // finishing a message never waits on the socket for it.
        self.pacing.finish_admitted();
    }
}

/// Mirrors `WASmaxOutOfflineBatchRequest.makeBatchRequest`.
pub(crate) fn build_offline_batch_request(count: u32) -> Node {
    NodeBuilder::new("ib")
        .children([NodeBuilder::new("offline_batch")
            .attr("count", count.to_string())
            .build()])
        .build()
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::Barrier;
    use wacore_binary::builder::NodeBuilder;

    fn offline_message_node() -> Node {
        NodeBuilder::new("message")
            .attr("from", "120363000000000042@g.us")
            .attr("id", "OFFLINE-1")
            .attr("offline", "1")
            .build()
    }

    fn armed(generation: u64, total: usize) -> OfflineBatchCoordinator {
        let coord = OfflineBatchCoordinator::new();
        coord.arm(generation, total);
        coord
    }

    #[test]
    fn arm_publishes_first_batch_inflight() {
        let c = OfflineBatchCoordinator::new();
        assert!(!c.is_armed_for(42));
        c.arm(42, 1_000);
        assert!(c.is_armed_for(42));
        assert!(!c.is_armed_for(43), "stale generation must not be armed");
        assert!(
            c.try_claim_first_arrival(),
            "first stanza after arm wins CAS"
        );
        assert!(
            !c.try_claim_first_arrival(),
            "duplicate arrival in same cycle loses"
        );
    }

    #[test]
    fn arm_books_the_first_batch_against_the_preview() {
        let c = armed(7, 1_000);
        assert_eq!(
            c.outstanding(),
            BATCH_SIZE as usize,
            "the first request is booked with the arm"
        );
        let small = armed(7, 3);
        assert_eq!(
            small.outstanding(),
            3,
            "a small preview books only what it announced"
        );
    }

    #[test]
    fn first_arrival_cas_is_exclusive() {
        let c = OfflineBatchCoordinator::new();
        c.arm(7, 1_000);
        c.mark_batch_inflight();
        assert!(c.try_claim_first_arrival(), "first caller wins");
        assert!(
            !c.try_claim_first_arrival(),
            "subsequent callers in same cycle fail"
        );
        c.mark_batch_inflight();
        assert!(
            c.try_claim_first_arrival(),
            "new cycle: another first arrival wins"
        );
    }

    #[test]
    fn reset_disarms_and_clears_inflight() {
        let c = OfflineBatchCoordinator::new();
        c.arm(7, 1_000);
        c.reset();
        assert!(!c.is_armed_for(7));
        assert!(
            !c.try_claim_first_arrival(),
            "reset must leave prev_batch_inflight=false"
        );
        assert!(
            c.current_pacing(7).is_none(),
            "reset must retire the pacing object"
        );
    }

    #[test]
    fn take_armed_reports_the_transition_once() {
        let c = OfflineBatchCoordinator::new();
        assert!(!c.take_armed(), "never armed, nothing to take");
        c.arm(7, 1_000);
        assert!(c.take_armed(), "the first take disarms");
        assert!(!c.take_armed(), "a second teardown stays silent");
        assert!(!c.is_armed());
    }

    /// S2: the gate opens exactly while what the last requests still owe fits
    /// one batch, and the check books the new request in the same transition.
    #[test]
    fn the_gate_books_one_request_while_the_last_ones_fit_a_batch() {
        let coord = armed(7, 1_000);
        assert_eq!(
            coord.outstanding(),
            BATCH_SIZE as usize,
            "the first request is booked with the arm"
        );
        let pacing = coord.current_pacing(7).expect("armed");

        assert!(
            matches!(pacing.reserve(), Reservation::Request),
            "one batch outstanding still fits another"
        );
        assert_eq!(coord.outstanding(), 2 * BATCH_SIZE as usize);
        assert!(
            matches!(pacing.reserve(), Reservation::Deferred),
            "two batches is where a request has to wait"
        );
        assert_eq!(
            coord.outstanding(),
            2 * BATCH_SIZE as usize,
            "a parked request books nothing"
        );
    }

    /// The preview clamps every booking, so a request never asks for more than
    /// the server announced and nothing is left parked once it is all booked.
    #[test]
    fn a_request_never_books_more_than_the_preview_announced() {
        let coord = armed(7, 3);
        assert_eq!(coord.outstanding(), 3, "three items, not a whole batch");
        let pacing = coord.current_pacing(7).expect("armed");
        assert!(matches!(pacing.reserve(), Reservation::Exhausted));
        assert_eq!(coord.outstanding(), 3);
    }

    #[test]
    fn admit_only_paces_stanzas_marked_offline() {
        let coord = armed(7, 1_000);
        let live = NodeBuilder::new("message")
            .attr("from", "120363000000000042@g.us")
            .attr("id", "LIVE-1")
            .build();
        assert!(
            coord.admit(7, &live).is_none(),
            "a live stanza is not part of the resume"
        );
        assert_eq!(coord.admitted(), 0);

        let offline = offline_message_node();
        let guard = coord.admit(7, &offline).expect("offline stanza admitted");
        assert_eq!(coord.admitted(), 1);
        drop(guard);
        assert_eq!(coord.admitted(), 0, "the guard settles its own admission");
    }

    #[test]
    fn an_unarmed_connection_admits_nothing() {
        let coord = OfflineBatchCoordinator::new();
        let node = offline_message_node();
        assert!(coord.admit(7, &node).is_none(), "not armed");
        coord.arm(7, 1_000);
        assert!(
            coord.admit(8, &node).is_none(),
            "a different generation is not this resume's"
        );
    }

    /// S3, blocker (3) as a named regression: a guard is bound to the pacing
    /// object it was minted with, so a reconnect cannot make it settle the
    /// live generation's counters.
    #[test]
    fn a_stale_admission_guard_leaves_the_new_generations_pacing_alone() {
        let coord = armed(7, 1_000);
        let node = offline_message_node();
        let stale = coord.admit(7, &node).expect("armed for generation 7");
        assert_eq!(coord.admitted(), 1);

        coord.reset();
        coord.arm(9, 1_000);
        let live = coord.admit(9, &node).expect("armed for generation 9");
        let (outstanding, admitted) = (coord.outstanding(), coord.admitted());

        drop(stale);
        assert_eq!(
            (coord.outstanding(), coord.admitted()),
            (outstanding, admitted),
            "the retired connection's guard must not settle the new generation"
        );
        drop(live);
        assert_eq!(
            coord.admitted(),
            0,
            "the live guard still settles its own generation"
        );
        assert_eq!(coord.outstanding(), outstanding - 1);
    }

    /// S4: the finish that brings the lane back within one batch is the one
    /// that releases the parked request. Parked and released are the same
    /// object and the same lock, so there is no window to lose it in.
    #[test]
    fn the_finish_that_opens_the_gate_releases_the_parked_request() {
        let coord = armed(7, 1_000);
        let batch = BATCH_SIZE as usize;
        let pacing = coord.current_pacing(7).expect("armed");
        assert!(
            matches!(pacing.reserve(), Reservation::Request),
            "the second request is booked while the first still fits"
        );
        for _ in 0..2 * batch {
            pacing.admit_lane_bound();
        }
        assert_eq!(coord.outstanding(), 2 * batch);
        assert!(matches!(pacing.reserve(), Reservation::Deferred));
        assert!(coord.deferred());

        // Every finish but the last one lands on a lane that is still over the
        // gate, so none of them may claim the request.
        let mut released = 0;
        for _ in 0..batch - 1 {
            released += usize::from(pacing.finish_admitted());
        }
        assert_eq!(released, 0);
        assert!(coord.deferred(), "still over the gate, still parked");

        assert!(
            pacing.finish_admitted(),
            "the finish that reaches one batch releases it, exactly once"
        );
        assert!(!coord.deferred());
        assert_eq!(
            coord.outstanding(),
            2 * batch,
            "the released request is booked as it is released"
        );
    }

    /// S4, blocker (2) as a named regression: in every interleaving of "the
    /// debounce decides" and "the last messages finish", exactly one of the two
    /// books the next request — there is no ordering in which both miss it.
    #[test]
    fn finishing_the_lane_while_a_request_is_being_deferred_always_releases_it() {
        let total = 1_000;
        let batch = BATCH_SIZE as usize;
        for _ in 0..200 {
            let pacing = Arc::new(Pacing::new(7, total).0);
            // Two requests out, and the whole of both in one lane.
            pacing.book_first();
            assert!(matches!(pacing.reserve(), Reservation::Request));
            for _ in 0..2 * batch {
                pacing.admit_lane_bound();
            }

            let line_up = Arc::new(Barrier::new(2));
            let bookings = std::thread::scope(|scope| {
                let debounce = {
                    let pacing = Arc::clone(&pacing);
                    let line_up = Arc::clone(&line_up);
                    scope.spawn(move || {
                        line_up.wait();
                        usize::from(matches!(pacing.reserve(), Reservation::Request))
                    })
                };
                let finishes = {
                    let pacing = Arc::clone(&pacing);
                    let line_up = Arc::clone(&line_up);
                    scope.spawn(move || {
                        line_up.wait();
                        (0..2 * batch).filter(|_| pacing.finish_admitted()).count()
                    })
                };
                debounce.join().expect("debounce") + finishes.join().expect("finishes")
            });

            assert_eq!(
                bookings, 1,
                "exactly one caller may book the request after the lane drains"
            );
            let state = pacing.state();
            assert!(!state.deferred, "the release always clears the park");
            assert_eq!(
                state.unbooked,
                total - 3 * batch,
                "exactly one request more was asked for"
            );
            assert_eq!(
                state.outstanding, batch,
                "its stanzas are owed, and the two arrived batches are settled"
            );
        }
    }

    /// S4: a window of silence while we hold the drain open is our own
    /// backpressure, not the server giving up. Once nothing is in a lane the
    /// resume may complete, but silence never forgives what the server still
    /// owes: the parked request stays parked and nothing more is booked.
    #[test]
    fn silence_we_cause_is_not_reported_as_a_stalled_drain() {
        let coord = armed(7, 1_000);
        let node = offline_message_node();
        let guard = coord.admit(7, &node).expect("armed");

        assert!(
            matches!(coord.settle_silent_drain(7), SilentDrain::Backpressure),
            "a lane we are still feeding is not a stalled drain"
        );

        let pacing = coord.current_pacing(7).expect("armed");
        drop(guard);
        assert!(matches!(pacing.reserve(), Reservation::Request));
        assert!(matches!(pacing.reserve(), Reservation::Deferred));
        let owed = coord.outstanding();
        assert!(
            matches!(coord.settle_silent_drain(7), SilentDrain::Idle),
            "nothing is in a lane, so the watchdog may complete"
        );
        assert_eq!(
            coord.outstanding(),
            owed,
            "silence must not settle what the server still owes"
        );
        assert!(
            coord.deferred(),
            "and must not book the parked request on top of it"
        );
    }

    /// A retired generation's silence decision must not reach the live one.
    #[test]
    fn silence_from_a_retired_generation_is_idle() {
        let coord = armed(7, 1_000);
        let node = offline_message_node();
        let guard = coord.admit(7, &node).expect("armed");
        assert!(matches!(
            coord.settle_silent_drain(7),
            SilentDrain::Backpressure
        ));
        assert!(matches!(coord.settle_silent_drain(8), SilentDrain::Idle));
        drop(guard);
    }

    /// S4: a parked request is released by the lane draining below one batch,
    /// through the guard path production uses, not by anything the reader does.
    #[test]
    fn the_lane_draining_below_one_batch_releases_the_parked_request() {
        let coord = armed(7, 1_000);
        let batch = BATCH_SIZE as usize;
        let node = offline_message_node();
        let guards: Vec<_> = (0..2 * batch)
            .map(|_| coord.admit(7, &node).expect("armed"))
            .collect();
        let pacing = coord.current_pacing(7).expect("armed");
        assert!(matches!(pacing.reserve(), Reservation::Request));
        assert!(matches!(pacing.reserve(), Reservation::Deferred));
        assert!(coord.deferred());

        drop(guards);
        assert!(
            !coord.deferred(),
            "the drops that brought the lane back under one batch released it"
        );
        assert_eq!(
            coord.outstanding(),
            batch,
            "and the released request is what is now owed"
        );
    }

    #[test]
    fn a_deferred_release_shifts_activity_before_its_sender_is_polled() {
        let coord = OfflineBatchCoordinator::new();
        let released = coord.arm(7, 1_000);
        let pacing = coord.current_pacing(7).expect("armed");
        assert!(matches!(pacing.reserve(), Reservation::Request));
        assert!(matches!(pacing.reserve(), Reservation::Deferred));
        for _ in 0..400 {
            pacing.admit_lane_bound();
        }
        let seen = coord.stanza_activity();
        for _ in 0..400 {
            pacing.finish_admitted();
        }
        assert!(
            released.try_recv().is_ok(),
            "the third request is ready, but no sender has polled it"
        );
        assert_eq!(
            coord.outstanding(),
            200,
            "silence must not forgive the booked response"
        );
        assert!(
            !coord.try_claim_inactive(seen),
            "booking the release must invalidate the watchdog's stale observation"
        );
    }

    #[test]
    fn an_unpolled_release_does_not_coalesce_two_requests() {
        let coord = OfflineBatchCoordinator::new();
        let released = coord.arm(7, 1_000);
        let pacing = coord.current_pacing(7).expect("armed");
        assert!(matches!(pacing.reserve(), Reservation::Request));
        assert!(matches!(pacing.reserve(), Reservation::Deferred));
        for _ in 0..2 * BATCH_SIZE as usize {
            pacing.admit_lane_bound();
        }
        for _ in 0..BATCH_SIZE as usize {
            pacing.finish_admitted();
        }
        assert!(
            matches!(pacing.reserve(), Reservation::Deferred),
            "the released request is still owed, so the next one parks too"
        );
        for _ in 0..BATCH_SIZE as usize {
            pacing.finish_admitted();
        }
        assert!(released.try_recv().is_ok(), "first release queued");
        assert!(
            released.try_recv().is_ok(),
            "each release books its own request, so two must never coalesce"
        );
    }

    #[test]
    fn batch_request_stanza_shape() {
        let node = build_offline_batch_request(200);
        assert_eq!(node.tag.as_str(), "ib");
        let children = node.children().expect("ib must have children");
        assert_eq!(children.len(), 1);
        let inner = &children[0];
        assert_eq!(inner.tag.as_str(), "offline_batch");
        let count = inner
            .attrs
            .get("count")
            .map(|v| v.to_string())
            .expect("count attr present");
        assert_eq!(count, "200");
    }
}
