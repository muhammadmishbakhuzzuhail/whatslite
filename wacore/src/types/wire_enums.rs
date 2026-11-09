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
