//! The core's single attachment point for an optional subsystem.
//!
//! The [`subsystems!`] list below is the one place the core is allowed to name
//! one. A subsystem that passes the cut rule in the architecture notes
//! implements [`Subsystem`], parks its per-client state through the associated
//! type and lists the notification types it models, so turning it on adds no
//! `Client` field and no `cfg` anywhere else.
//!
//! Three items here have no user in a build with no subsystem attached: the two
//! traits and the lookup. That is the seam working rather than rot, so each
//! carries its own `allow` and the module carries none.
//!
//! # Notification claims
//!
//! A subsystem claims the wire tags of the `<notification type="…">` stanzas it
//! models, as strings: this fork dispatches notifications by tag, so the claim
//! is the same value the dispatcher already holds.
//!
//! # What the client does not carry yet
//!
//! `Client` has no [`Subsystems`] field, so there is no `Client::subsystem`
//! lookup: [`Attached::state`] is the lookup that method will delegate to once
//! the client owns the state, and a concrete subsystem that populates the list
//! is a separate piece of work.

use std::sync::Arc;

use wacore::stats::CollectionStats;
use wacore::sync_marker::MaybeSendSync;
use wacore_binary::node::Node;

use super::Client;
use super::device_memo_stats::SubsystemMemory;

/// One optional subsystem the core can carry.
///
/// The implementing type is the whole handle: its state, the notification types
/// it claims and the hooks it fills all hang off it, so they cannot drift apart
/// the way a record of function pointers lets them. Nothing here is `dyn`, so a
/// hook a subsystem does not fill costs no branch and no vtable.
#[allow(async_fn_in_trait)] // never used through `dyn`: every call names a concrete impl, so auto traits still flow
#[allow(
    dead_code,
    reason = "no implementor in a build with no subsystem attached"
)]
pub(crate) trait Subsystem: 'static {
    /// What one client retains for this subsystem. Stored inline under its real
    /// type, so reaching it back is a field access and not a downcast that has
    /// to answer for failing. `MaybeSendSync` is the bound the client itself
    /// carries: `Send + Sync` on native, dropped on wasm32.
    type State: Default + MaybeSendSync;

    /// How the memory report names this subsystem's collections.
    const NAME: &'static str;

    /// The notification tags this subsystem models. The core routes on the tag
    /// alone and never looks inside the stanza. Two subsystems claiming one tag
    /// fails the build; see the assertion below.
    const NOTIFICATIONS: &'static [&'static str] = &[];

    /// Handles one of [`Self::NOTIFICATIONS`]. Never called for a tag this
    /// subsystem did not claim, which is why claiming none needs no handler.
    async fn handle_notification(
        _client: &Arc<Client>,
        _notification_type: &str,
        _node: Arc<Node>,
    ) {
    }

    /// Runs while a connection's state is torn down, before the socket closes.
    async fn on_connection_cleanup(_client: &Client) {}

    /// Sees a response before the waiter it belongs to is woken. On the ack
    /// path and synchronous, so an implementation has to stay cheap; the point
    /// of running here rather than on the waiter's side is that the waking is
    /// what a subsystem would otherwise race.
    fn on_response(_client: &Client, _node: &Node) {}

    /// Retained collections, named as the memory report prints them under
    /// [`Self::NAME`]. Takes the state rather than the client, so reporting
    /// cannot quietly become a second way for a subsystem to read the core.
    fn memory(
        _state: &Self::State,
        _now: impl FnOnce() -> wacore::time::Instant,
    ) -> Vec<(&'static str, CollectionStats)> {
        Vec::new()
    }
}

/// Generates the core's whole side of the seam from one list of subsystems:
/// where their state lives, which of them a build can ask for, and the four
/// points the core calls out from.
///
/// The generated functions carry `unused_variables` because a build with no
/// subsystem attached expands every one of them to a body that reads none of
/// its arguments.
macro_rules! subsystems {
    ($($(#[$attr:meta])* $field:ident: $subsystem:ty),* $(,)?) => {
        /// The state of every subsystem attached to this build, each under its
        /// own type. No fields, and nothing to construct, when none is.
        #[derive(Default)]
        pub(crate) struct Subsystems {
            $($(#[$attr])* $field: <$subsystem as Subsystem>::State,)*
        }

        /// Implemented only for a subsystem this build attached. That is what
        /// makes the client-side lookup infallible: asking for a detached
        /// subsystem's state is a compile error, not a `None` or a panic every
        /// caller has to carry a story for.
        #[allow(dead_code, reason = "no implementor in a build with no subsystem attached")]
        pub(crate) trait Attached: Subsystem {
            fn state(subsystems: &Subsystems) -> &Self::State;
        }

        $(
            $(#[$attr])*
            impl Attached for $subsystem {
                fn state(subsystems: &Subsystems) -> &Self::State {
                    &subsystems.$field
                }
            }
        )*

        /// What each attached subsystem claims, in list order.
        pub(crate) const CLAIMS: &[&[&str]] = &[
            $($(#[$attr])* <$subsystem as Subsystem>::NOTIFICATIONS,)*
        ];

        /// Hands a notification to the subsystem that models its type. `false`
        /// means no attached subsystem claims it, leaving the core's own
        /// fallback in charge.
        #[allow(unused_variables, dead_code, reason = "no implementor in a build with no subsystem attached")]
        pub(crate) async fn dispatch_notification(
            client: &Arc<Client>,
            notification_type: &str,
            node: Arc<Node>,
        ) -> bool {
            $(
                $(#[$attr])*
                if <$subsystem as Subsystem>::NOTIFICATIONS.contains(&notification_type) {
                    <$subsystem as Subsystem>::handle_notification(
                        client,
                        notification_type,
                        node,
                    )
                    .await;
                    return true;
                }
            )*
            false
        }

        /// Lets every attached subsystem release what one connection owned.
        #[allow(unused_variables, dead_code, reason = "no implementor in a build with no subsystem attached")]
        pub(crate) async fn on_connection_cleanup(client: &Client) {
            $(
                $(#[$attr])*
                <$subsystem as Subsystem>::on_connection_cleanup(client).await;
            )*
        }

        /// Offers a response to every attached subsystem before its waiter is woken.
        #[allow(unused_variables, dead_code, reason = "no implementor in a build with no subsystem attached")]
        pub(crate) fn on_response(client: &Client, node: &Node) {
            $(
                $(#[$attr])*
                <$subsystem as Subsystem>::on_response(client, node);
            )*
        }

        /// Collects what the attached subsystems retain, for the memory report.
        #[allow(unused_variables, unused_mut, dead_code, reason = "no implementor in a build with no subsystem attached")]
        pub(crate) fn memory(subsystems: &Subsystems, now: impl Fn() -> wacore::time::Instant) -> Vec<SubsystemMemory> {
            let mut retained = Vec::new();
            $(
                $(#[$attr])*
                retained.extend(
                    <$subsystem as Subsystem>::memory(&subsystems.$field, &now)
                        .into_iter()
                        .map(|(collection, stats)| SubsystemMemory {
                            subsystem: <$subsystem as Subsystem>::NAME,
                            collection,
                            stats,
                        }),
                );
            )*
            retained
        }
    };
}

// No subsystem is attached to this build: upstream's list names passkey and
// VoIP, and neither is a subsystem in this fork yet. The seam stays because it
// is what a later attachment plugs into, and an empty list is the state it is
// designed to compile and run in.
subsystems! {}

/// Dispatch picks the first subsystem whose claim matches, so two of them
/// claiming one notification tag would make routing depend on list order and
/// silently starve the loser. Checked here rather than in a test because the
/// list is a `const`: there is no configuration where this is a runtime
/// question.
const _: () = assert!(
    !claims_overlap(CLAIMS),
    "two subsystems claim the same notification type"
);

/// Whether any two of `claims` name one notification tag. `const` and hence
/// written as index loops: iterators are not available here, and neither is
/// `str` comparison.
const fn claims_overlap(claims: &[&[&str]]) -> bool {
    let mut left = 0;
    while left < claims.len() {
        let mut right = left + 1;
        while right < claims.len() {
            let mut this = 0;
            while this < claims[left].len() {
                let mut other = 0;
                while other < claims[right].len() {
                    if str_eq(claims[left][this], claims[right][other]) {
                        return true;
                    }
                    other += 1;
                }
                this += 1;
            }
            right += 1;
        }
        left += 1;
    }
    false
}

const fn str_eq(left: &str, right: &str) -> bool {
    let left = left.as_bytes();
    let right = right.as_bytes();
    if left.len() != right.len() {
        return false;
    }
    let mut index = 0;
    while index < left.len() {
        if left[index] != right[index] {
            return false;
        }
        index += 1;
    }
    true
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::Arc;
    use std::sync::atomic::{AtomicUsize, Ordering};
    use wacore_binary::builder::NodeBuilder;
    use wacore_binary::node::Node;

    mod seam {
        use super::*;

        pub static HANDLED: AtomicUsize = AtomicUsize::new(0);

        pub struct Demo;

        impl Subsystem for Demo {
            type State = u32;

            const NAME: &'static str = "demo";
            const NOTIFICATIONS: &'static [&'static str] = &["demo"];

            async fn handle_notification(
                _client: &Arc<Client>,
                notification_type: &str,
                _node: Arc<Node>,
            ) {
                assert_eq!(notification_type, "demo");
                HANDLED.fetch_add(1, Ordering::Relaxed);
            }
        }

        subsystems! { demo: Demo }
    }

    #[test]
    fn overlapping_claims_are_detected() {
        const A: &[&str] = &["devices", "picture"];
        const B: &[&str] = &["picture"];
        const C: &[&str] = &["contacts"];

        assert!(
            claims_overlap(&[A, B]),
            "two lists sharing a tag must be reported as overlapping"
        );
        assert!(
            !claims_overlap(&[A, C]),
            "disjoint lists must not be reported as overlapping"
        );
        assert!(
            !claims_overlap(CLAIMS),
            "the shipping list must stay free of overlap"
        );
    }

    #[tokio::test]
    async fn a_claimed_notification_reaches_its_subsystem() {
        let client = crate::test_utils::create_test_client().await;
        let node = Arc::new(
            NodeBuilder::new("notification")
                .attr("type", "demo")
                .build(),
        );

        assert!(seam::dispatch_notification(&client, "demo", node).await);
        assert_eq!(seam::HANDLED.load(Ordering::Relaxed), 1);
    }

    #[tokio::test]
    async fn an_unclaimed_notification_falls_through_to_the_core() {
        let client = crate::test_utils::create_test_client().await;
        let node = Arc::new(
            NodeBuilder::new("notification")
                .attr("type", "devices")
                .build(),
        );

        assert!(!dispatch_notification(&client, "devices", node).await);
    }

    #[test]
    fn an_attached_subsystem_state_is_reachable_by_its_type() {
        let state = seam::Subsystems::default();

        assert_eq!(*<seam::Demo as seam::Attached>::state(&state), 0);
    }

    #[test]
    fn no_attached_subsystem_retains_anything() {
        let retained = memory(&Subsystems::default(), wacore::time::Instant::now);

        assert!(retained.is_empty());
    }
}
