//! Wire-encoded enum helpers used by the event model.

use crate::WireEnum;

/// The `decrypt-fail` attribute of an `<enc>` node.
#[derive(Debug, Clone, Copy, PartialEq, Eq, WireEnum)]
pub enum DecryptFailMode {
    #[wire = "hide"]
    Hide,
    #[wire_default]
    #[wire = "show"]
    Show,
}

/// `HID_FAILED_DECRYPT` bit of a receipt's `<meta mode>` bitmask.
pub const RECEIPT_MODE_HID_FAILED_DECRYPT: u32 = 1 << 2;

/// Whether a privacy disallowed-list entry is being added or removed.
///
/// Generated from `PrivacyUserAction` in `WAWebSetPrivacyJob`.
#[derive(Debug, Clone, Copy, PartialEq, Eq, WireEnum)]
pub enum DisallowedListAction {
    #[wire_default]
    #[wire = "add"]
    Add,
    #[wire = "remove"]
    Remove,
}

/// A participant's role inside a group.
///
/// Wire format: the `type` attribute of `<participant>` children in group
/// notifications, mirroring WA Web's `maybeAttrEnum("type")`.
#[derive(Debug, Clone, Copy, PartialEq, Eq, WireEnum)]
pub enum GroupParticipantType {
    #[wire = "superadmin"]
    SuperAdmin,
    #[wire = "admin"]
    Admin,
    #[wire_default]
    #[wire = "participant"]
    Participant,
}

/// Who may share a group's history with a new participant.
///
/// Generated from `MemberShareGroupHistoryMode` in `WAWebGroupHistoryShareMode`.
#[derive(Debug, Clone, Copy, PartialEq, Eq, WireEnum)]
pub enum MemberShareHistoryMode {
    #[wire_default]
    #[wire = "admin_share"]
    AdminShare,
    #[wire = "all_member_share"]
    AllMemberShare,
}
