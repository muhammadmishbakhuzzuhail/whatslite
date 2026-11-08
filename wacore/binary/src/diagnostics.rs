//! Borrowed, bounded protocol diagnostics.
//!
//! Phase 0 keeps the safe display wrapper; the full token/atom table is
//! ported with the binary diagnostics task.

use std::fmt;

/// Displays a known protocol atom; arbitrary server text is never a log payload.
pub struct ProtocolText<'a>(&'a str);

impl<'a> ProtocolText<'a> {
    pub const fn new(value: &'a str) -> Self {
        Self(value)
    }
}

impl fmt::Display for ProtocolText<'_> {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        if self.0.is_empty() {
            f.write_str("empty")
        } else {
            write!(f, "other(len={})", self.0.len())
        }
    }
}

impl fmt::Debug for ProtocolText<'_> {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        fmt::Display::fmt(self, f)
    }
}
