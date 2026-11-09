//! Group history sharing tokens and outcomes.

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum GroupHistorySkipReason {
    NoHistory,
    NotMember,
    Disabled,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct GroupHistoryRetryToken(pub String);

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum GroupHistoryShareOutcome {
    Shared,
    Skipped(GroupHistorySkipReason),
    Retry(GroupHistoryRetryToken),
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct GroupHistoryAddResult {
    pub added: usize,
    pub outcome: GroupHistoryShareOutcome,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct HistorySharePreparation {
    pub messages: usize,
}
