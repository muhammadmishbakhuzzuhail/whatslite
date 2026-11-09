//! Polls surface.

pub use crate::features::creation::{CreatedPoll, PollRef};

#[derive(Debug, thiserror::Error)]
#[non_exhaustive]
pub enum PollError {
    #[error("poll operation failed: {0}")]
    Failed(String),
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct PollOptionResult {
    pub name: String,
    pub votes: u32,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct PollVoteCiphertext {
    pub ciphertext: Vec<u8>,
}

#[derive(Debug, Clone, Copy, Default)]
pub struct Polls;
