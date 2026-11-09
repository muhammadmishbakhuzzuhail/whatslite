//! Chat action helpers.

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ClearChatOptions {
    pub delete_media: bool,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct SyncActionMessageRange {
    pub start: Option<i64>,
    pub end: Option<i64>,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct MediaDeletion {
    pub delete_media: bool,
}

#[derive(Debug, Clone, Copy, Default)]
pub struct ChatActions;
