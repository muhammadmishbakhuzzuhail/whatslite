//! Device-topology change tracking for the per-group device-list memo.
//!
//! "Topology" here means anything that can change a device-list answer:
//! registry record writes/invalidations and LID-PN mapping changes. Instead of
//! trusting every write path to remember a manual generation bump, the bump
//! lives INSIDE the write chokepoints (`DeviceRegistryCache` and
//! `LidPnCache::add`), so a writer cannot forget it by construction.
//!
//! Each change also logs WHICH canonical users it touched (both namespaces),
//! so a memo whose generation went stale can prove "none of the changed users
//! are in my group" and re-stamp itself instead of recomputing. Every doubtful
//! case (log overflow, global events) degrades to a recompute, never to
//! serving stale data.

// Nothing reads the tracker until the device-registry memo and the LID-PN
// cache record through it; both are separate ports.
#![allow(
    dead_code,
    reason = "the device-registry memo and LID-PN cache that record here are wired in a follow-up"
)]

use std::collections::VecDeque;
use std::sync::Arc;
use std::sync::atomic::Ordering;

use portable_atomic::AtomicU64;

use super::member_index::MemberIndex;

/// Bound on the fingerprints the log retains, summed over every change still
/// in it. A change costs 8 bytes per user it touched: a single mapping or
/// registry write is one to three entries, a device-list refresh for a
/// 256-member group (three identifiers per member, recorded as ONE change)
/// is ~6 KiB, so the log holds several such refreshes before it overflows.
/// Overflow just disables the scoped-revalidation fast path for memos older
/// than the evicted change, which then recompute once.
///
/// Fingerprints rather than the identifiers themselves for the same reason
/// `MemberIndex` uses them: the only question ever asked is "did a change
/// touch this memo's members", and a collision can only force a recompute.
///
/// A bound, not a preallocation: the deque grows into it, because a session
/// that never touches a group never writes an entry.
pub(crate) const TOPOLOGY_LOG_CAPACITY: usize = 4096;

struct TopologyLog {
    /// (generation, the users that change touched), oldest first.
    entries: VecDeque<(u64, MemberIndex)>,
    /// Fingerprints held across `entries`, the quantity the bound is on.
    fingerprints: usize,
    /// Highest generation evicted from `entries` (0 = nothing evicted).
    /// A memo older than this cannot be proven clean and must recompute.
    floor: u64,
}

/// Shared tracker: a monotonic generation plus the bounded changed-users log.
pub(crate) struct DeviceTopology {
    generation: AtomicU64,
    log: std::sync::Mutex<TopologyLog>,
    registry_mutation: tokio::sync::Mutex<()>,
}

/// Proof that a device-registry mutation is serialized with authoritative
/// refresh publication. The cache write API requires this token so new write
/// paths cannot accidentally bypass the ordering invariant.
pub(crate) struct DeviceRegistryMutationGuard<'a> {
    _guard: tokio::sync::MutexGuard<'a, ()>,
}

impl DeviceTopology {
    pub(crate) fn new() -> Arc<Self> {
        Arc::new(Self {
            generation: AtomicU64::new(0),
            log: std::sync::Mutex::new(TopologyLog {
                entries: VecDeque::new(),
                fingerprints: 0,
                floor: 0,
            }),
            registry_mutation: tokio::sync::Mutex::new(()),
        })
    }

    pub(crate) fn current(&self) -> u64 {
        self.generation.load(Ordering::Acquire)
    }

    pub(crate) async fn lock_registry(&self) -> DeviceRegistryMutationGuard<'_> {
        DeviceRegistryMutationGuard {
            _guard: self.registry_mutation.lock().await,
        }
    }

    /// Record one topology change touching the given users (pass BOTH
    /// namespaces of an identity when known: a mapping change alters which
    /// canonical record either key resolves to).
    pub(crate) fn record<'a>(&self, users: impl IntoIterator<Item = &'a str>) {
        self.record_change(users);
    }

    fn record_change<'a>(&self, users: impl IntoIterator<Item = &'a str>) {
        // Fingerprinted before taking the lock: a refresh for a large group
        // records hundreds of identifiers in one change, and hashing them is
        // the bulk of the work.
        let touched = MemberIndex::from_users(users);
        let mut log = self.log.lock().unwrap_or_else(|p| p.into_inner());
        let generation = self.generation.load(Ordering::Acquire) + 1;
        // A change that touched nobody cannot invalidate a memo, so it needs
        // no entry; the generation still moves so its writer's snapshot
        // ordering is unchanged.
        if !touched.is_empty() {
            log.fingerprints += touched.len();
            log.entries.push_back((generation, touched));
            while log.fingerprints > TOPOLOGY_LOG_CAPACITY
                && let Some((evicted_gen, evicted)) = log.entries.pop_front()
            {
                log.fingerprints -= evicted.len();
                log.floor = evicted_gen;
            }
        }
        // Publish the generation only after the log holds the users, so a
        // reader that observes the new generation can always find (or rule
        // out) the corresponding entries.
        self.generation.store(generation, Ordering::Release);
    }

    /// Record a registry mutation while holding its serialization guard, so a
    /// refresh can compare-and-publish without a check/write race.
    pub(crate) fn record_registry<'a>(
        &self,
        _guard: &DeviceRegistryMutationGuard<'_>,
        users: impl IntoIterator<Item = &'a str>,
    ) {
        self.record_change(users);
    }

    /// Record a change whose blast radius is unknown (bulk warm-up, cache
    /// clear): bumps and poisons the scoped fast path so every memo
    /// recomputes once.
    pub(crate) fn record_global(&self) {
        let mut log = self.log.lock().unwrap_or_else(|p| p.into_inner());
        let generation = self.generation.load(Ordering::Acquire) + 1;
        log.entries.clear();
        log.fingerprints = 0;
        log.floor = generation;
        self.generation.store(generation, Ordering::Release);
    }

    /// Whether every change after `since` only touched users outside
    /// `members`. `false` on any doubt (log overflow past `since`, a
    /// fingerprint collision), so callers recompute.
    pub(crate) fn unchanged_for(&self, since: u64, members: &MemberIndex) -> bool {
        let log = self.log.lock().unwrap_or_else(|p| p.into_inner());
        if log.floor > since {
            return false;
        }
        // Entries are in generation order, so the changes after `since` are
        // a suffix.
        log.entries
            .iter()
            .rev()
            .take_while(|(generation, _)| *generation > since)
            .all(|(_, touched)| !members.intersects(touched))
    }
}

#[cfg(test)]
mod tests {
    use super::{DeviceTopology, MemberIndex, TOPOLOGY_LOG_CAPACITY};

    fn members(users: &[&str]) -> MemberIndex {
        MemberIndex::from_users(users.iter().copied())
    }

    /// The whole point of the log: a memo whose generation went stale proves
    /// none of the changed users are in its group and re-stamps itself.
    #[test]
    fn an_unrelated_change_leaves_a_memo_provably_clean() {
        let topology = DeviceTopology::new();
        let stamped = topology.current();
        topology.record(["someone-else"]);

        assert!(topology.current() > stamped, "the generation must advance");
        assert!(topology.unchanged_for(stamped, &members(&["member"])));
        assert!(
            !topology.unchanged_for(stamped, &members(&["someone-else"])),
            "a change touching a member cannot be proven clean"
        );
    }

    /// Growing into the bound must not raise it: past the cap the log evicts,
    /// and an evicted generation lifts the floor so anything older recomputes.
    #[test]
    fn overflowing_the_bound_evicts_and_poisons_older_generations() {
        let topology = DeviceTopology::new();
        let stamped = topology.current();
        for i in 0..TOPOLOGY_LOG_CAPACITY + 10 {
            topology.record([format!("user-{i}").as_str()]);
        }

        let held = topology.log.lock().unwrap().fingerprints;
        assert!(
            held <= TOPOLOGY_LOG_CAPACITY,
            "the log grew past its bound: {held} fingerprints"
        );
        assert!(
            !topology.unchanged_for(stamped, &members(&[])),
            "a generation behind the floor cannot be proven clean"
        );
        // Everything still in the log stays answerable, which is what the
        // eviction is trading the old entries for.
        let floor = topology.log.lock().unwrap().floor;
        assert!(topology.unchanged_for(floor, &members(&["absent"])));
    }

    /// One change over many users is one log entry costing one fingerprint
    /// per user: a refresh for a large group must fit without evicting the
    /// changes before it, or every memo older than the refresh recomputes.
    #[test]
    fn a_large_batch_is_one_change_and_fits() {
        let topology = DeviceTopology::new();
        topology.record(["earlier"]);
        let stamped = topology.current();
        let refreshed: Vec<String> = (0..768).map(|i| format!("member-{i}")).collect();
        topology.record(refreshed.iter().map(String::as_str));

        assert_eq!(topology.current(), stamped + 1, "one bump for the batch");
        assert_eq!(topology.log.lock().unwrap().entries.len(), 2);
        assert!(topology.unchanged_for(stamped, &members(&["unrelated"])));
        assert!(!topology.unchanged_for(stamped, &members(&["member-500"])));
        // The change before the batch is still in the log, so a memo stamped
        // before it is still provably clean of both.
        assert!(topology.unchanged_for(stamped - 1, &members(&["unrelated"])));
    }

    /// A single change larger than the whole bound cannot be kept: it lifts
    /// the floor to its own generation, so every memo older than it recomputes
    /// and one stamped after it is unaffected.
    #[test]
    fn an_oversized_change_poisons_only_what_precedes_it() {
        let topology = DeviceTopology::new();
        let stamped = topology.current();
        let huge: Vec<String> = (0..TOPOLOGY_LOG_CAPACITY + 1)
            .map(|i| format!("member-{i}"))
            .collect();
        topology.record(huge.iter().map(String::as_str));

        assert!(!topology.unchanged_for(stamped, &members(&[])));
        assert!(topology.unchanged_for(topology.current(), &members(&["member-1"])));
        assert_eq!(topology.log.lock().unwrap().fingerprints, 0);
    }

    /// A global change clears the log, so the generations it published must
    /// still be readable as "recompute" rather than as an empty (clean) log.
    #[test]
    fn a_global_change_poisons_every_earlier_generation() {
        let topology = DeviceTopology::new();
        topology.record(["a"]);
        let stamped = topology.current();
        topology.record_global();

        assert!(topology.log.lock().unwrap().entries.is_empty());
        assert!(
            !topology.unchanged_for(stamped, &members(&[])),
            "an emptied log must not read as nothing having changed"
        );
        assert!(topology.unchanged_for(topology.current(), &members(&["a"])));
    }

    /// Every write path stamps a memo with `current()` and compares later, so
    /// the generation must advance by exactly one per change — including a
    /// change that touched nobody, whose ordering still has to move.
    #[test]
    fn generation_advances_once_per_change() {
        let topology = DeviceTopology::new();
        assert_eq!(topology.current(), 0, "a fresh tracker starts at zero");

        topology.record(["a"]);
        assert_eq!(topology.current(), 1);
        topology.record(["b", "c"]);
        assert_eq!(topology.current(), 2);
        topology.record(std::iter::empty::<&str>());
        assert_eq!(topology.current(), 3, "a no-op change still bumps");
        topology.record_global();
        assert_eq!(topology.current(), 4);
        assert!(
            topology.current() > 0,
            "the counter only ever moves forward"
        );
    }

    /// A change touching nobody cannot invalidate a memo, so it writes no log
    /// entry even though the generation moves.
    #[test]
    fn an_empty_change_logs_nothing() {
        let topology = DeviceTopology::new();
        topology.record(std::iter::empty::<&str>());

        assert!(topology.log.lock().unwrap().entries.is_empty());
        assert!(topology.unchanged_for(0, &members(&["a"])));
    }

    /// The registry guard is the serialization proof the cache write API
    /// demands: a second mutation cannot proceed until the first is released.
    #[tokio::test]
    async fn registry_mutations_are_serialized() {
        let topology = DeviceTopology::new();
        let first = topology.lock_registry().await;

        let (acquired_tx, mut acquired_rx) = tokio::sync::oneshot::channel();
        let contender = {
            let topology = topology.clone();
            tokio::spawn(async move {
                let _guard = topology.lock_registry().await;
                let _ = acquired_tx.send(());
            })
        };

        // While the first guard is held the second acquisition must not
        // complete: a check/write race is exactly what the token prevents.
        assert!(
            tokio::time::timeout(std::time::Duration::from_millis(50), &mut acquired_rx)
                .await
                .is_err(),
            "a registry mutation ran while another held the guard"
        );

        drop(first);
        tokio::time::timeout(std::time::Duration::from_millis(500), &mut acquired_rx)
            .await
            .expect("the waiting mutation must proceed once the guard is released")
            .expect("the contender reports acquisition");
        contender.await.unwrap();
    }

    /// A registry write records through the guard it was handed, so the
    /// generation and the changed users move together with the mutation.
    #[tokio::test]
    async fn a_registry_write_records_its_users() {
        let topology = DeviceTopology::new();
        let stamped = topology.current();
        {
            let guard = topology.lock_registry().await;
            topology.record_registry(&guard, ["100000000000001"]);
        }

        assert!(topology.current() > stamped);
        assert!(!topology.unchanged_for(stamped, &members(&["100000000000001"])));
        assert!(topology.unchanged_for(stamped, &members(&["100000000000002"])));
    }
}
