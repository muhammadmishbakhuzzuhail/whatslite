use async_channel::Receiver;
use thiserror::Error;
use std::fmt;

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct InvalidStableId;

impl fmt::Display for InvalidStableId {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result { f.write_str("stable IDs must be lowercase ASCII letters, digits, or hyphens") }
}
impl std::error::Error for InvalidStableId {}

macro_rules! stable_id {
    ($name:ident) => {
        #[derive(Clone, Debug, PartialEq, Eq, Hash, PartialOrd, Ord)]
        pub struct $name(String);
        impl $name {
            pub fn new(value: impl Into<String>) -> Result<Self, InvalidStableId> {
                let value = value.into();
                if !value.is_empty() && value.bytes().all(|b| b.is_ascii_lowercase() || b.is_ascii_digit() || b == b'-') { Ok(Self(value)) } else { Err(InvalidStableId) }
            }
            pub fn as_str(&self) -> &str { &self.0 }
        }
        impl fmt::Display for $name { fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result { self.0.fmt(f) } }
    };
}
stable_id!(AccountId);
stable_id!(ChatId);
stable_id!(ContactId);
stable_id!(MessageId);
stable_id!(MediaId);

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ChatSummary {
    pub id: ChatId,
    pub title: String,
    pub preview: String,
    pub timestamp: String,
    pub unread: bool,
    pub favorite: bool,
    pub is_group: bool,
    pub archived: bool,
    pub pinned: bool,
}
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ContactSummary { pub id: ContactId, pub name: String }
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct MessageDraft { pub chat_id: ChatId, pub text: String }
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct MediaItem { pub id: MediaId, pub name: String }
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct PairingStatus { pub code: String, pub expires_in_seconds: u64 }
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GroupSummary { pub id: ChatId, pub name: String, pub member_count: usize }
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct CallRecord { pub id: String, pub contact_id: ContactId, pub timestamp: String, pub video: bool, pub missed: bool }
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct CallOptions { pub video: bool, pub muted: bool }
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct AppSettings { pub theme: ThemePreference, pub notifications_enabled: bool }
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ThemePreference { Light, Dark, System }
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum CallState { Ringing, Connected, Ended }
#[derive(Clone, Debug, Error, PartialEq, Eq)]
pub enum ServiceError {
    #[error("simulated offline failure")]
    Offline,
    #[error("the requested fixture was not found")]
    NotFound,
    #[error("simulated operation failure: {0}")]
    Failed(String),
}

#[async_trait::async_trait]
pub trait AuthService: Send + Sync {
    async fn start_qr_pairing(&self) -> Result<PairingStatus, ServiceError>;
    async fn start_phone_linking(&self, phone_number: &str) -> Result<PairingStatus, ServiceError>;
    async fn poll_link_status(&self) -> Result<ConnectionState, ServiceError>;
    async fn logout(&self) -> Result<(), ServiceError>;
}

#[async_trait::async_trait]
pub trait ChatService: Send + Sync {
    async fn list_chats(&self) -> Result<Vec<ChatSummary>, ServiceError>;
    async fn list_messages(&self, chat_id: &ChatId) -> Result<Vec<String>, ServiceError>;
    async fn send(&self, draft: MessageDraft) -> Result<MessageId, ServiceError>;
    async fn edit(&self, message_id: &MessageId, text: &str) -> Result<(), ServiceError>;
    async fn delete(&self, message_id: &MessageId) -> Result<(), ServiceError>;
    async fn forward(&self, message_id: &MessageId, target: &ChatId) -> Result<MessageId, ServiceError>;
    async fn react(&self, message_id: &MessageId, reaction: &str) -> Result<(), ServiceError>;
    async fn search(&self, chat_id: Option<&ChatId>, query: &str) -> Result<Vec<String>, ServiceError>;
}

#[async_trait::async_trait]
pub trait ContactService: Send + Sync {
    async fn list_contacts(&self) -> Result<Vec<ContactSummary>, ServiceError>;
    async fn search_contacts(&self, query: &str) -> Result<Vec<ContactSummary>, ServiceError>;
    async fn get_contact(&self, id: &ContactId) -> Result<ContactSummary, ServiceError>;
    async fn block(&self, id: &ContactId, blocked: bool) -> Result<(), ServiceError>;
}

#[async_trait::async_trait]
pub trait GroupService: Send + Sync {
    async fn create(&self, name: &str, members: &[ContactId]) -> Result<GroupSummary, ServiceError>;
    async fn update(&self, id: &ChatId, name: &str) -> Result<GroupSummary, ServiceError>;
    async fn members(&self, id: &ChatId) -> Result<Vec<ContactSummary>, ServiceError>;
    async fn set_permissions(&self, id: &ChatId, admins_only: bool) -> Result<(), ServiceError>;
    async fn invite(&self, id: &ChatId, contact_id: &ContactId) -> Result<(), ServiceError>;
}

#[async_trait::async_trait]
pub trait MediaService: Send + Sync {
    async fn list_media(&self, chat_id: &ChatId) -> Result<Vec<MediaItem>, ServiceError>;
    async fn upload(&self, chat_id: &ChatId, name: &str) -> Result<MediaId, ServiceError>;
    async fn choose_files(&self) -> Result<Vec<MediaItem>, ServiceError>;
    async fn download(&self, id: &MediaId) -> Result<Vec<u8>, ServiceError>;
    async fn media_index(&self, chat_id: &ChatId) -> Result<Vec<MediaItem>, ServiceError>;
}

#[async_trait::async_trait]
pub trait CallService: Send + Sync {
    async fn history(&self) -> Result<Vec<CallRecord>, ServiceError>;
    async fn place(&self, contact_id: &ContactId, options: CallOptions) -> Result<CallState, ServiceError>;
    async fn accept(&self) -> Result<CallState, ServiceError>;
    async fn reject(&self) -> Result<(), ServiceError>;
    async fn hangup(&self) -> Result<CallState, ServiceError>;
    async fn set_mute(&self, muted: bool) -> Result<(), ServiceError>;
    async fn set_video(&self, enabled: bool) -> Result<(), ServiceError>;
    async fn share(&self) -> Result<(), ServiceError>;
}

#[async_trait::async_trait]
pub trait SettingsStore: Send + Sync {
    async fn load(&self) -> Result<AppSettings, ServiceError>;
    async fn update(&self, settings: AppSettings) -> Result<(), ServiceError>;
    async fn set_theme(&self, theme: ThemePreference) -> Result<(), ServiceError>;
}

#[async_trait::async_trait]
pub trait NotificationService: Send + Sync {
    async fn show_in_app(&self, title: &str, body: &str) -> Result<(), ServiceError>;
    async fn dismiss(&self, id: &str) -> Result<(), ServiceError>;
}

#[cfg(test)]
mod stable_id_tests {
    use super::*;

    #[test]
    fn ids_accept_only_stable_lowercase_tokens() {
        assert_eq!(ChatId::new("maya-chat").unwrap().as_str(), "maya-chat");
        assert!(ContactId::new("").is_err());
        assert!(MessageId::new("Message 001").is_err());
        assert!(MediaId::new("brief_v2").is_err());
    }
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub enum ConnectionState {
    Disconnected,
    Connecting,
    WaitingForPairing,
    Connected,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub enum BackendCommand {
    StartQrPairing,
    SendTextMessage { chat_id: String, text: String },
    Disconnect,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub enum BackendEvent {
    ConnectionChanged(ConnectionState),
    QrPairingReady {
        code: String,
        valid_for_seconds: u64,
    },
    PairingCodeReady {
        code: String,
        valid_for_seconds: u64,
    },
    PairingSucceeded {
        account_id: String,
    },
    MessageReceived {
        chat_id: String,
        sender_id: String,
        text: String,
        timestamp_millis: i64,
    },
    MessageSent {
        chat_id: String,
        text: String,
    },
    Failure {
        message: String,
    },
}

#[derive(Debug, Error, PartialEq, Eq)]
pub enum BackendError {
    #[error("backend event receiver has closed")]
    EventReceiverClosed,
}

pub trait BackendService: Send + Sync {
    fn dispatch(&self, command: BackendCommand) -> Result<(), BackendError>;
    fn subscribe(&self) -> Receiver<BackendEvent>;
}
