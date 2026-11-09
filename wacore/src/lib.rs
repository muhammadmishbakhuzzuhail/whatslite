extern crate self as wacore;

pub use aes_gcm;
pub use wacore_appstate as appstate;
pub use wacore_noise as noise;

// Re-export derive macros
pub use wacore_derive::{EmptyNode, ProtocolNode, StringEnum, WireEnum};

pub mod crypto;
pub mod diagnostics;
pub mod event;
pub mod runtime;
pub mod stats;
pub mod sync_marker;
pub mod telemetry;
pub mod time;

pub mod adv;
pub mod client;
pub mod client_profile;
pub mod companion_reg;
pub mod download;
pub mod identity_verification;
pub mod iq;
pub mod protocol;
pub use wacore_noise::framing;
pub mod handshake;
pub mod history_sync;
pub mod ib;
pub use wacore_libsignal as libsignal;
pub mod bot_message;
pub mod comment;
pub mod media_retry;
pub mod message_edit;
pub mod messages;
#[cfg(feature = "mobile")]
pub mod mobile;
pub mod msg_secret;
pub mod net;
pub mod pair;
pub mod pair_code;
pub mod prekeys;
pub mod proto_helpers;
pub mod reaction;
pub mod reporting_token;
pub mod request;
pub mod secret_enc_addon;
pub mod send;
pub mod session_engine;
pub mod shortcake;
pub mod stanza;
pub mod sticker_pack;
pub mod store;
pub mod sync_engine;
pub mod types;
pub mod upload;
pub mod usync;
pub mod version;
#[cfg(feature = "voip")]
pub mod voip;
#[cfg(feature = "voip-control")]
pub mod voip_control;
pub mod webp;
pub mod xml;
pub mod zip;
