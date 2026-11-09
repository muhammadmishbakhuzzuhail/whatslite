//! Unique ids for the durability probes the client writes into its own store.
//!
//! A probe is identified by randomness (so two clients, or two processes, can
//! never collide) plus a process-local sequence (so a single process cannot
//! collide with itself even if the RNG repeats).

// The builder that stamps a probe id on a new client is wired in a follow-up.
#![allow(
    dead_code,
    reason = "the client builder that stamps probe ids is wired in a follow-up"
)]

use portable_atomic::{AtomicU64, Ordering};

pub(super) fn next() -> String {
    static PROBE_SEQ: AtomicU64 = AtomicU64::new(0);
    format!(
        "__wa_durability_probe_{}_{}__",
        rand::random::<u128>(),
        PROBE_SEQ.fetch_add(1, Ordering::Relaxed)
    )
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::collections::HashSet;

    #[test]
    fn probe_ids_are_unique() {
        let ids: HashSet<String> = (0..1_000).map(|_| next()).collect();
        assert_eq!(ids.len(), 1_000, "every probe id must be distinct");
    }

    #[test]
    fn probe_id_is_marked_and_delimited() {
        let id = next();
        assert!(
            id.starts_with("__wa_durability_probe_") && id.ends_with("__"),
            "probe id must be recognisable: {id}"
        );
    }

    #[test]
    fn probe_ids_share_the_marker_but_not_the_suffix() {
        let first = next();
        let second = next();
        assert_ne!(first, second, "consecutive ids must differ");
        assert_ne!(
            first.trim_end_matches("__").rsplit('_').next(),
            second.trim_end_matches("__").rsplit('_').next(),
            "the sequence must advance between calls"
        );
    }
}
