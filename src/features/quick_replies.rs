//! Quick replies surface.

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct QuickReply {
    pub id: String,
    pub shortcut: String,
    pub message: String,
}

#[derive(Debug, Clone, Copy, Default)]
pub struct QuickReplies;
