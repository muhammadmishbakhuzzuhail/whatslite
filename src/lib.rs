extern crate self as whatsapp_rust;

pub use wacore;
pub use wacore::{iq::privacy as privacy_settings, proto_helpers, store::traits};
pub use wacore_binary;
pub use wacore_binary::builder::NodeBuilder;
pub use wacore_binary::jid::Jid;
pub use waproto;

pub mod http;
pub mod types;

pub mod cache;
pub mod cache_config;
pub mod cache_store;
pub mod error;
pub use error::{ErrorChainExt, ServerRejection, Sources};
pub mod client;
pub mod portable_cache;
pub use client::Client;
pub use client::builder::{ClientBuild, ClientBuilder, ClientBuilderError, ClientOptions};
pub use client::lifecycle::{ConnectError, ConnectStage};
pub use client::node_io::{NodeFilter, NodeWaiter};
pub use types::connect_admission::ConnectAdmission;
pub use types::decrypt_failure_admission::DecryptFailureAdmission;
pub use types::durability_hook::{HistorySyncCaptureHook, InboundDurabilityHook};
pub use types::history_sync_admission::HistorySyncAdmission;
pub use types::retry_admission::RetryAdmission;
pub mod download;
pub mod handlers;
pub use handlers::chatstate::ChatStateEvent;
pub mod handshake;
pub mod jid_utils;
pub mod keepalive;
pub mod media;
pub mod mediaconn;
pub mod message;
pub mod pair;
pub mod pair_code;
#[cfg(feature = "passkey")]
pub mod passkey;
pub mod request;
pub mod send;
pub use send::{RevokeType, SendOptions};
pub mod session;
pub mod socket;
pub mod store;
pub mod transport;
pub mod upload;
#[cfg(any(feature = "voip", feature = "voip-runtime"))]
pub mod voip;
#[cfg(feature = "voip-control")]
pub mod voip_control;

pub mod pdo;
pub mod prekeys;
pub mod receipt;
pub mod retry;
pub mod unified_session;

pub mod appstate_sync;
pub mod flush_scope;
pub mod history_sync;
pub mod inbound_queue;
pub mod msg_secret_buffer;
pub mod runtime_impl;
pub mod signal_flush;
pub mod usync;

pub mod features;
#[cfg(feature = "mobile-core")]
pub mod mobile;
#[cfg(feature = "mobile-core")]
pub mod mobile_session;
pub use features::{
    Blocking, BlocklistEntry, ChatStateType, Chatstate, ContactInfo, Contacts, CreateGroupResult,
    GroupCreateOptions, GroupDescription, GroupMetadata, GroupParticipant, GroupParticipantOptions,
    GroupSubject, Groups, IsOnWhatsAppResult, MemberAddMode, MemberLinkMode,
    MembershipApprovalMode, Mex, MexError, MexErrorExtensions, MexRequest, MexResponse,
    ParticipantChangeResponse, Presence, PresenceStatus, ProfilePicture, TcToken, UserInfo,
};

pub mod bot;
pub mod lid_pn_cache;
pub mod spam_report;
pub mod sync_task;
pub mod version;

pub use spam_report::{SpamFlow, SpamReportRequest, SpamReportResult};

#[cfg(any(test, feature = "sqlite-storage"))]
pub mod test_utils;
