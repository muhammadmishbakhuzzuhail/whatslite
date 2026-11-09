//! Signal session inspection/migration surface.

#[derive(Debug, thiserror::Error, PartialEq, Eq)]
#[non_exhaustive]
pub enum SignalError {
    #[error("session not found")]
    SessionNotFound,
    #[error("migration failed: {0}")]
    Migration(String),
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SignalSessionInfo {
    pub jid: String,
    pub has_session: bool,
    pub device_count: usize,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum SignalSessionMigration {
    AlreadyMigrated,
    Migrated { sessions: usize },
}

#[derive(Debug, Clone, Copy, Default)]
pub struct Signal;
