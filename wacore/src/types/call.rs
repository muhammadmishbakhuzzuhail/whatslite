use chrono::{DateTime, Utc};
use serde::Serialize;
use wacore_binary::jid::Jid;

#[derive(Debug, Clone)]
pub struct BasicCallMeta {
    pub from: Jid,
    pub timestamp: DateTime<Utc>,
    pub call_creator: Jid,
    pub call_id: String,
}

#[derive(Debug, Clone)]
pub struct CallRemoteMeta {
    pub remote_platform: String,
    pub remote_version: String,
}

/// An incoming call offer surfaced as an event. Minimal Phase 0 payload;
/// richer call signaling lands with the VoIP phase.
#[derive(Debug, Clone, Serialize)]
#[non_exhaustive]
pub struct IncomingCall {
    pub from: Jid,
    pub stanza_id: String,
    pub call_id: String,
    pub timestamp: Option<DateTime<Utc>>,
    pub offline: bool,
}

/// A call that must not ring (replayed offer / missed call).
#[derive(Debug, Clone, Serialize)]
#[non_exhaustive]
pub struct MissedCall {
    pub from: Jid,
    pub call_id: String,
    pub timestamp: DateTime<Utc>,
    pub reason: String,
}

/// A ringing call answered or declined on another device.
#[derive(Debug, Clone, Serialize)]
#[non_exhaustive]
pub struct CallEndedElsewhere {
    pub from: Jid,
    pub call_id: String,
    pub timestamp: DateTime<Utc>,
    pub outcome: String,
}

/// One persisted call-log record.
#[derive(Debug, Clone, Serialize)]
#[non_exhaustive]
pub struct CallLog {
    pub call_id: String,
    pub peer: Jid,
    pub timestamp: DateTime<Utc>,
    pub duration_secs: u32,
    pub video: bool,
    pub missed: bool,
}

impl CallLog {
    pub fn new(
        call_id: impl Into<String>,
        peer: Jid,
        timestamp: DateTime<Utc>,
        duration_secs: u32,
        video: bool,
        missed: bool,
    ) -> Self {
        Self {
            call_id: call_id.into(),
            peer,
            timestamp,
            duration_secs,
            video,
            missed,
        }
    }
}

/// Group-call state update.
#[derive(Debug, Clone, Serialize)]
#[non_exhaustive]
pub struct GroupCallUpdate {
    pub group: Jid,
    pub call_id: String,
    pub participants: Vec<Jid>,
}

impl GroupCallUpdate {
    pub fn new(group: Jid, call_id: impl Into<String>, participants: Vec<Jid>) -> Self {
        Self {
            group,
            call_id: call_id.into(),
            participants,
        }
    }
}

/// Typed call event union.
#[derive(Debug, Clone, Serialize)]
#[non_exhaustive]
pub enum CallEvent {
    Incoming(IncomingCall),
    Missed(MissedCall),
    EndedElsewhere(CallEndedElsewhere),
    Log(CallLog),
    Group(GroupCallUpdate),
}
