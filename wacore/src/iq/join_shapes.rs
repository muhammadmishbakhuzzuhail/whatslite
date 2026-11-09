//! Response presence-gate shapes for social RPCs.
//!
//! The success shapes of RPCs whose answers carry presence gates, from the
//! whatspec IQ index. Each entry is `(variant tag, required child tags)`; a
//! variant with no required children accepts a bare `<iq type="result">`.

/// Success variants of `WASmaxGroupsAcceptGroupAddRPC`, in RPC cascade order.
pub const ACCEPT_GROUP_ADD_SUCCESS: &[(&str, &[&str])] = &[
    (
        "AcceptGroupAddResponseGroupJoinRequestSuccess",
        &["membership_approval_request"],
    ),
    ("AcceptGroupAddResponseSuccess", &[]),
];

/// Success variants of `WASmaxGroupsJoinLinkedGroupRPC`, in RPC cascade order.
pub const JOIN_LINKED_GROUP_SUCCESS: &[(&str, &[&str])] = &[
    (
        "JoinLinkedGroupResponseGroupJoinRequestSuccess",
        &["membership_approval_request"],
    ),
    ("JoinLinkedGroupResponseSuccess", &[]),
];

/// Success variants of `WASmaxPassiveModeActiveIQRPC`, in RPC cascade order.
pub const PASSIVE_ACTIVE_SUCCESS: &[(&str, &[&str])] = &[("ActiveIQResponseSuccess", &["active"])];

/// Success variants of `WASmaxPassiveModePassiveIQRPC`, in RPC cascade order.
pub const PASSIVE_PASSIVE_SUCCESS: &[(&str, &[&str])] =
    &[("PassiveIQResponseSuccess", &["passive"])];

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn accept_group_add_success_shape() {
        assert_eq!(
            ACCEPT_GROUP_ADD_SUCCESS,
            &[
                (
                    "AcceptGroupAddResponseGroupJoinRequestSuccess",
                    &["membership_approval_request"][..],
                ),
                ("AcceptGroupAddResponseSuccess", &[][..]),
            ]
        );
    }

    #[test]
    fn join_linked_group_success_shape() {
        assert_eq!(
            JOIN_LINKED_GROUP_SUCCESS,
            &[
                (
                    "JoinLinkedGroupResponseGroupJoinRequestSuccess",
                    &["membership_approval_request"][..],
                ),
                ("JoinLinkedGroupResponseSuccess", &[][..]),
            ]
        );
    }

    #[test]
    fn passive_success_shapes() {
        assert_eq!(
            PASSIVE_ACTIVE_SUCCESS,
            &[("ActiveIQResponseSuccess", &["active"][..])]
        );
        assert_eq!(
            PASSIVE_PASSIVE_SUCCESS,
            &[("PassiveIQResponseSuccess", &["passive"][..])]
        );
    }
}
