//! Public connection lifecycle vocabulary. The existing `Client::connect` and
//! `run` remain the compatibility entry points; these types describe stages and
//! failures for hosts and tests without changing their signatures.

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[non_exhaustive]
pub enum ConnectStage {
    Idle,
    Connecting,
    Handshaking,
    Authenticating,
    Ready,
    Disconnecting,
    Closed,
}

#[derive(Debug, Clone, PartialEq, Eq, thiserror::Error)]
#[non_exhaustive]
pub enum ConnectError {
    #[error("client is shut down")]
    Shutdown,
    #[error("client is paused")]
    Paused,
    #[error("client is not activated")]
    NotActivated,
    #[error("client is already connected")]
    AlreadyConnected,
    #[error("transport error: {0}")]
    Transport(String),
    #[error("handshake error: {0}")]
    Handshake(String),
    #[error("connect timed out")]
    Timeout,
}

impl ConnectError {
    /// A step of the connect flow ran out of time.
    pub fn is_timeout(&self) -> bool {
        matches!(self, Self::Timeout)
    }

    /// Whether a retry can plausibly succeed without host intervention.
    pub fn is_transient(&self) -> bool {
        matches!(
            self,
            Self::Transport(_) | Self::Handshake(_) | Self::Timeout
        )
    }
}
