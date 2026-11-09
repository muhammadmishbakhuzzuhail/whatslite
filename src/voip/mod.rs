//! High-level VoIP facade.

use std::time::Duration;

pub use wacore::voip_control::{
    AudioSink, AudioSource, EncodedAudioSink, EncodedAudioSource, TimedVideoFrame, VideoFrame,
    VideoInput, VideoSink, VideoSource,
};

pub const VIDEO_UPGRADE_TIMEOUT: Duration = Duration::from_secs(30);

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct CallEvents {
    pub call_id: String,
    pub events: Vec<String>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct AcceptCall {
    pub call_id: String,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum CallState {
    Idle,
    Ringing,
    Connected,
    Ended,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct CallHandle {
    pub call_id: String,
    pub state: CallState,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct OutgoingCall {
    pub call_id: String,
    pub to: String,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct OutgoingGroupCall {
    pub call_id: String,
    pub group: String,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct GroupBoundCall {
    pub call_id: String,
    pub group: String,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct CallLinkCall {
    pub token: String,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum CallTermination {
    LocalHangup,
    RemoteHangup,
    Timeout,
    Error(String),
}

#[derive(Debug, thiserror::Error, PartialEq, Eq)]
#[non_exhaustive]
pub enum CallError {
    #[error("client is not connected")]
    NotConnected,
    #[error("invalid call state: {0}")]
    InvalidState(String),
    #[error("call timed out")]
    Timeout,
}

#[derive(Debug, Clone, Copy, Default)]
pub struct Voip;
