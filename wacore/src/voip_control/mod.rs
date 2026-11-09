//! Neutral VoIP control-plane/media boundary. Names only primitives, not the
//! engine types, so a `voip-control` build can exist without `voip`.

use std::sync::Arc;

use wacore_binary::Jid;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct MediaGroupEpoch(pub Vec<u8>);

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct MediaSessionKey {
    pub key: [u8; 32],
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct MediaAudioSpec {
    pub sample_rate: u32,
    pub channels: u8,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum MediaKeyframeUrgency {
    Low,
    High,
    Immediate,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct VideoFecConfig {
    pub overhead_percent: u8,
}

impl Default for VideoFecConfig {
    fn default() -> Self {
        Self {
            overhead_percent: 20,
        }
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct MediaVideoUpgradeToken(pub String);

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct MediaGroupTransition {
    pub from: MediaGroupEpoch,
    pub to: MediaGroupEpoch,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum MediaCodecDecisionSource {
    Remote,
    Local,
    Negotiated,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum MediaSilenceReason {
    UserMuted,
    Network,
    Engine,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct MediaEncodedFrame {
    pub payload: Vec<u8>,
    pub timestamp: u32,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct MediaRtcpReportBlock {
    pub fraction_lost: u8,
    pub packets_lost: i32,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct MediaRtcpFeedback {
    pub kind: String,
    pub parameter: Option<String>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct MediaDirectPeer {
    pub jid: Jid,
    pub endpoint: String,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct MediaGroupSpec {
    pub participants: Vec<Jid>,
    pub epoch: MediaGroupEpoch,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct MediaSessionSpec {
    pub call_id: String,
    pub audio: MediaAudioSpec,
    pub direct_peer: Option<MediaDirectPeer>,
    pub group: Option<MediaGroupSpec>,
    pub video: bool,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum MediaCommand {
    Start,
    Stop,
    UpdateAudio(MediaAudioSpec),
    RequestKeyframe(MediaKeyframeUrgency),
    SetVideoFec(VideoFecConfig),
    UpgradeVideo(MediaVideoUpgradeToken),
    GroupTransition(MediaGroupTransition),
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum MediaGroupControlKind {
    Join,
    Leave,
    Update,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum MediaCloseReason {
    Normal,
    RemoteHangup,
    TransportFailure,
    Timeout,
}

#[derive(Debug, thiserror::Error, PartialEq, Eq)]
#[non_exhaustive]
pub enum MediaSetupError {
    #[error("media backend is not available")]
    BackendUnavailable,
    #[error("invalid media spec: {0}")]
    InvalidSpec(String),
    #[error("media session already active")]
    AlreadyActive,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct PeerAnswer {
    pub jid: Jid,
    pub sdp: String,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct GroupRawEpoch(pub Vec<u8>);

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct NormalizedGroupControl {
    pub kind: MediaGroupControlKind,
    pub epoch: GroupRawEpoch,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct GroupControl {
    pub call_id: String,
    pub normalized: NormalizedGroupControl,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum VideoControl {
    Start,
    Stop,
    Upgrade,
}

#[derive(Debug, Clone)]
pub struct VideoControlSender(pub async_channel::Sender<VideoControl>);

#[derive(Debug)]
pub struct VideoControlReceiver(pub async_channel::Receiver<VideoControl>);

/// Ordered call events exposed to the control plane.
#[derive(Debug, Clone, PartialEq, Eq)]
#[non_exhaustive]
pub enum CallEvent {
    Incoming {
        call_id: String,
        from: Jid,
    },
    Accepted {
        call_id: String,
    },
    Ended {
        call_id: String,
        reason: MediaCloseReason,
    },
    VideoUpgraded {
        call_id: String,
    },
    GroupUpdated {
        call_id: String,
    },
}

/// Active-call registry with monotonically increasing ids.
#[derive(Debug, Default)]
pub struct CallRegistry {
    active: std::collections::HashMap<String, MediaSessionSpec>,
    next: u64,
}

impl CallRegistry {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn allocate_id(&mut self) -> String {
        self.next += 1;
        format!("call-{}", self.next)
    }

    pub fn insert(&mut self, spec: MediaSessionSpec) -> Option<MediaSessionSpec> {
        self.active.insert(spec.call_id.clone(), spec)
    }

    pub fn remove(&mut self, call_id: &str) -> Option<MediaSessionSpec> {
        self.active.remove(call_id)
    }

    pub fn get(&self, call_id: &str) -> Option<&MediaSessionSpec> {
        self.active.get(call_id)
    }

    pub fn len(&self) -> usize {
        self.active.len()
    }

    pub fn is_empty(&self) -> bool {
        self.active.is_empty()
    }
}

/// One parsed relay endpoint.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct RelayEndpoint {
    pub host: String,
    pub port: u16,
    pub token: Option<String>,
}

/// Parse `<relay>` children into endpoints; malformed entries are skipped.
pub fn parse_relay_endpoints(node: &wacore_binary::NodeRef<'_>) -> Vec<RelayEndpoint> {
    let Some(children) = node.children() else {
        return Vec::new();
    };
    children
        .iter()
        .filter_map(|child| {
            let host = child.get_attr("host").and_then(|v| v.as_str())?.to_string();
            let port = child
                .get_attr("port")
                .and_then(|v| v.as_str())?
                .parse()
                .ok()?;
            Some(RelayEndpoint {
                host,
                port,
                token: child
                    .get_attr("token")
                    .and_then(|v| v.as_str())
                    .map(str::to_string),
            })
        })
        .collect()
}

pub trait AudioSource: Send + Sync {}
pub trait AudioSink: Send + Sync {}
pub trait EncodedAudioSource: Send + Sync {}
pub trait EncodedAudioSink: Send + Sync {}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct VideoFrame {
    pub payload: Vec<u8>,
    pub width: u32,
    pub height: u32,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct TimedVideoFrame {
    pub frame: VideoFrame,
    pub timestamp_ms: u64,
}

pub trait VideoInput: Send + Sync {}
pub trait VideoSource: Send + Sync {}
pub trait VideoSink: Send + Sync {}

pub type SharedMediaSession = Arc<dyn Send + Sync>;
