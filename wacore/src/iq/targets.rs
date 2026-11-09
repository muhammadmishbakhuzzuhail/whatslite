//! IQ addressing (WhatsApp 2.3000.1052559453).
//!
//! How the official client addresses the requests this repository sends.
//!
//! Derived from the whatspec IR; upstream regenerates this file with
//! `cargo run -p whatspec-codegen`, so keep it a pure constants module.

/// Who a request is addressed to.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum IqTarget {
    /// The main server, `s.whatsapp.net`.
    MainServer,
    /// The group server, `g.us`: it answers about groups in general rather than
    /// about one group.
    GroupServer,
    /// The one group the request acts on.
    GroupJid,
}

/// Addressing of `LeaveGroupIq`, from `leaveGroup` in `WAWebGroupExitJob` (`set` in `w:g2`).
pub const LEAVE_GROUP: IqTarget = IqTarget::GroupServer;

/// Addressing of `GroupParticipatingIq`, from `makeGetParticipatingGroupsRequest` in `WASmaxGroupsGetParticipatingGroupsRPC` (`get` in `w:g2`).
pub const GROUP_PARTICIPATING: IqTarget = IqTarget::GroupServer;

/// Addressing of `BatchGetGroupInfoIq`, from `makeBatchGetGroupInfoRequest` in `WASmaxGroupsBatchGetGroupInfoRPC` (`get` in `w:g2`).
pub const BATCH_GET_GROUP_INFO: IqTarget = IqTarget::GroupServer;

/// Addressing of `GetGroupInviteInfoIq`, from `makeGetInviteGroupInfoRequest` in `WASmaxGroupsGetInviteGroupInfoRPC` (`get` in `w:g2`).
pub const GET_GROUP_INVITE_INFO: IqTarget = IqTarget::GroupServer;

/// Addressing of `GroupQueryIq`, from `makeGetGroupInfoRequest` in `WASmaxGroupsGetGroupInfoRPC` (`get` in `w:g2`).
pub const GROUP_QUERY: IqTarget = IqTarget::GroupJid;

/// Addressing of `SetGroupSubjectIq`, from `makeSetSubjectRequest` in `WASmaxGroupsSetSubjectRPC` (`set` in `w:g2`).
pub const SET_GROUP_SUBJECT: IqTarget = IqTarget::GroupJid;

/// Addressing of `SetGroupDescriptionIq`, from `makeSetDescriptionRequest` in `WASmaxGroupsSetDescriptionRPC` (`set` in `w:g2`).
pub const SET_GROUP_DESCRIPTION: IqTarget = IqTarget::GroupJid;

/// Addressing of `SetGroupLockedIq`, from `makeSetPropertyRequest` in `WASmaxGroupsSetPropertyRPC` (`set` in `w:g2`).
pub const SET_GROUP_LOCKED: IqTarget = IqTarget::GroupJid;

/// Addressing of `ReportGroupMessagesIq`, from `makeReportMessagesRequest` in `WASmaxGroupsReportMessagesRPC` (`set` in `w:g2`).
pub const REPORT_GROUP_MESSAGES: IqTarget = IqTarget::GroupJid;

/// Addressing of `GetReportedGroupMessagesIq`, from `makeGetReportedMessagesRequest` in `WASmaxGroupsGetReportedMessagesRPC` (`get` in `w:g2`).
pub const GET_REPORTED_GROUP_MESSAGES: IqTarget = IqTarget::GroupJid;

/// Addressing of `GetMembershipRequestsIq`, from `makeGetMembershipApprovalRequestsRequest` in `WASmaxGroupsGetMembershipApprovalRequestsRPC` (`get` in `w:g2`).
pub const GET_MEMBERSHIP_REQUESTS: IqTarget = IqTarget::GroupJid;

/// Addressing of `MembershipRequestActionIq`, from `makeMembershipRequestsActionRequest` in `WASmaxGroupsMembershipRequestsActionRPC` (`set` in `w:g2`).
pub const MEMBERSHIP_REQUEST_ACTION: IqTarget = IqTarget::GroupJid;

#[cfg(test)]
mod tests {
    use super::*;
    use crate::iq::groups::{
        GroupParticipatingIq, GroupQueryIq, GroupSubject, LeaveGroupIq, SetGroupDescriptionIq,
        SetGroupSubjectIq,
    };
    use crate::iq::spec::IqSpec;
    use wacore_binary::jid::{GROUP_SERVER, Jid};

    fn addressed(want: IqTarget, group: &Jid) -> Jid {
        match want {
            IqTarget::GroupServer => Jid::new("", GROUP_SERVER),
            IqTarget::GroupJid => group.clone(),
            IqTarget::MainServer => Jid::new("", "s.whatsapp.net"),
        }
    }

    /// Whether a `w:g2` request is addressed to the group server or to one
    /// group's own JID, checked against what the whatspec IR resolves for each
    /// upstream builder.
    ///
    /// The IR only started answering this in schema 4: before it, every `w:g2`
    /// stanza reported the namespace's base target of `s.whatsapp.net`, so a
    /// request sent to the wrong one of the two looked exactly like a request
    /// sent to the right one. It now splits them (26 `group_jid`, 6 `g.us`),
    /// and a mistake here is otherwise silent -- the server ignores the IQ and
    /// the caller waits out its timeout.
    ///
    /// This is a regression lock, not a derivation: the expectations below were
    /// read off the IR by hand, since the IR is not available at test time.
    #[test]
    fn group_requests_are_addressed_the_way_the_ir_resolves_them() {
        let group: Jid = "120363000000000001@g.us".parse().unwrap();

        // The expectation comes from the generated constants, so a request
        // upstream moves between the two targets flips the constant on the next
        // sync and fails here, rather than turning into a silent timeout.
        let expect = |actual: Jid, want: IqTarget, spec: &str| {
            assert_eq!(
                actual,
                addressed(want, &group),
                "{spec} is addressed to the wrong target"
            );
        };

        expect(
            LeaveGroupIq::new(&group).build_iq().to,
            LEAVE_GROUP,
            "LeaveGroupIq",
        );
        expect(
            GroupParticipatingIq::new().build_iq().to,
            GROUP_PARTICIPATING,
            "GroupParticipatingIq",
        );
        expect(
            GroupQueryIq::new(&group).build_iq().to,
            GROUP_QUERY,
            "GroupQueryIq",
        );
        expect(
            SetGroupSubjectIq::new(&group, GroupSubject::new("x").unwrap())
                .build_iq()
                .to,
            SET_GROUP_SUBJECT,
            "SetGroupSubjectIq",
        );
        expect(
            SetGroupDescriptionIq::new(&group, None, None).build_iq().to,
            SET_GROUP_DESCRIPTION,
            "SetGroupDescriptionIq",
        );

        // Still hand-asserted: `resetGroupInviteCode` has two entries in
        // `WAWebGroupInviteJob` resolving to different targets, so the emitter
        // refuses to pin either. Ours is the `group_jid` overload, the one with
        // an empty `<invite/>`; the `g.us` overload carries a code and is a call
        // this repository does not make.
        assert_eq!(
            crate::iq::groups::GetGroupInviteLinkIq::new(&group, false)
                .build_iq()
                .to,
            group
        );
    }

    /// The remaining constants pin requests that live outside `groups.rs` (or
    /// are not ported yet), so their addressing is asserted directly.
    #[test]
    fn remaining_targets_match_the_reviewed_ir() {
        assert_eq!(BATCH_GET_GROUP_INFO, IqTarget::GroupServer);
        assert_eq!(GET_GROUP_INVITE_INFO, IqTarget::GroupServer);
        assert_eq!(SET_GROUP_LOCKED, IqTarget::GroupJid);
        assert_eq!(REPORT_GROUP_MESSAGES, IqTarget::GroupJid);
        assert_eq!(GET_REPORTED_GROUP_MESSAGES, IqTarget::GroupJid);
        assert_eq!(GET_MEMBERSHIP_REQUESTS, IqTarget::GroupJid);
        assert_eq!(MEMBERSHIP_REQUEST_ACTION, IqTarget::GroupJid);
    }
}
