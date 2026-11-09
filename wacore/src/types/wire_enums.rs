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
