//! Batches unknown-device users during offline sync for deferred usync.
//! WA Web: `OfflinePendingDeviceCache` + `doPendingDeviceSync()`.
//!
//! The send pipeline wiring lands separately; until then nothing constructs
//! this set, so the whole module is exempt from dead-code lints.
#![allow(dead_code)]

use std::collections::HashSet;
use wacore_binary::Jid;

pub(crate) struct PendingDeviceSync {
    /// Sync lock: no critical section here does anything but a set operation.
    pending: std::sync::Mutex<HashSet<Jid>>,
}

impl PendingDeviceSync {
    pub(crate) fn new() -> Self {
        Self {
            pending: std::sync::Mutex::new(HashSet::new()),
        }
    }

    fn lock(&self) -> std::sync::MutexGuard<'_, HashSet<Jid>> {
        self.pending.lock().unwrap_or_else(|p| p.into_inner())
    }

    /// Insert a user. Returns `true` if newly inserted, `false` if already present.
    ///
    /// Takes `&Jid` and clones only on a real insert, so the dedup path (the common
    /// case during a retry storm) does no allocation.
    pub(crate) fn add(&self, jid: &Jid) -> bool {
        let mut pending = self.lock();
        if pending.contains(jid) {
            false
        } else {
            pending.insert(jid.clone());
            true
        }
    }

    /// Takes the set's allocation with it: the queue fills during an offline
    /// drain and is empty the rest of the process lifetime, so `drain` would
    /// pin the high-water table for nothing.
    pub(crate) fn take_all(&self) -> Vec<Jid> {
        std::mem::take(&mut *self.lock()).into_iter().collect()
    }

    /// Release a user whose online refresh has finished, so the next unknown
    /// device from them triggers a refresh again instead of hitting the dedup
    /// for the rest of the connection. A no-op for an entry an offline drain
    /// already took.
    pub(crate) fn remove(&self, jid: &Jid) {
        let mut pending = self.lock();
        pending.remove(jid);
        crate::client::release_after_burst(&mut pending);
    }

    /// Users queued for the next `doPendingDeviceSync`, plus the users with an
    /// online refresh in flight — [`Self::add`] is the dedup for both. Offline
    /// entries leave with [`Self::take_all`]; an online entry leaves when its
    /// refresh finishes ([`Self::remove`]), so outside a drain this holds only
    /// what is in flight.
    ///
    /// The set has no capacity cap, and a cap would be the wrong fix: dropping a
    /// user silently skips a device refresh and leaves the next send to them
    /// addressed to a stale device list. The bound is the drain, not a number —
    /// which is why the count belongs in `memory_report()`.
    pub(crate) fn len(&self) -> usize {
        self.lock().len()
    }

    pub(crate) fn clear(&self) {
        self.lock().clear();
    }
}

#[cfg(test)]
mod tests {
    use super::PendingDeviceSync;
    use crate::Jid;

    fn jid(user: &str) -> Jid {
        format!("{user}@s.whatsapp.net")
            .parse()
            .expect("valid test jid")
    }

    #[test]
    fn add_dedups_against_inserted_user() {
        let pending = PendingDeviceSync::new();

        assert!(pending.add(&jid("111")));
        assert!(!pending.add(&jid("111")));
        assert!(pending.add(&jid("222")));
        assert_eq!(pending.len(), 2);
    }

    #[test]
    fn take_all_drains_and_clears() {
        let pending = PendingDeviceSync::new();
        assert!(pending.add(&jid("111")));
        assert!(pending.add(&jid("222")));

        let mut taken = pending.take_all();
        taken.sort_by(|a, b| a.user.cmp(&b.user));

        assert_eq!(
            taken.iter().map(|j| j.user.as_str()).collect::<Vec<_>>(),
            vec!["111", "222"]
        );
        assert_eq!(pending.len(), 0);
        assert!(pending.take_all().is_empty());
    }

    #[test]
    fn remove_releases_user_for_a_future_add() {
        let pending = PendingDeviceSync::new();
        assert!(pending.add(&jid("111")));

        pending.remove(&jid("111"));
        assert_eq!(pending.len(), 0);

        // The dedup entry is gone, so the next unknown device re-triggers a refresh.
        assert!(pending.add(&jid("111")));
    }

    #[test]
    fn remove_after_take_all_is_a_noop() {
        let pending = PendingDeviceSync::new();
        assert!(pending.add(&jid("111")));
        assert_eq!(pending.take_all().len(), 1);

        pending.remove(&jid("111"));
        assert_eq!(pending.len(), 0);
    }

    #[test]
    fn clear_drops_every_entry() {
        let pending = PendingDeviceSync::new();
        assert!(pending.add(&jid("111")));
        assert!(pending.add(&jid("222")));

        pending.clear();
        assert_eq!(pending.len(), 0);
    }

    #[test]
    fn remove_releases_the_burst_table() {
        let pending = PendingDeviceSync::new();
        let users: Vec<Jid> = (0..64).map(|i| jid(&format!("9{i:05}"))).collect();
        for user in &users {
            assert!(pending.add(user));
        }

        for user in &users {
            pending.remove(user);
        }

        let capacity = pending.pending.lock().expect("lock").capacity();
        assert_eq!(pending.len(), 0);
        assert!(
            capacity < users.len(),
            "table should shrink, got {capacity}"
        );
    }

    #[test]
    fn poisoned_lock_still_works() {
        let pending = PendingDeviceSync::new();
        assert!(pending.add(&jid("111")));

        let poisoned = std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| {
            let _guard = pending.pending.lock().expect("lock");
            panic!("poison the pending device set");
        }));
        assert!(poisoned.is_err());

        // Poison recovery keeps the set usable rather than propagating panics to send paths.
        assert_eq!(pending.len(), 1);
        assert!(!pending.add(&jid("111")));
        assert!(pending.add(&jid("222")));
        assert_eq!(pending.take_all().len(), 2);
    }
}
